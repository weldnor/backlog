package browse

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/weldnor/backlog/internal/hooks"
	"github.com/weldnor/backlog/internal/store"
	"github.com/weldnor/backlog/internal/task"
	"github.com/weldnor/backlog/internal/taskview"
)

// hookDiag returns the writer hook diagnostics go to: opts.Log if the caller
// gave one, discarded otherwise. A hook's own success or failure is not
// something an HTTP client can be handed inline, so it goes wherever browse's
// other best-effort diagnostics go.
func hookDiag(opts Options) io.Writer {
	if opts.Log == nil {
		return io.Discard
	}
	return opts.Log.Writer()
}

// deref reads a patchRequest string pointer, treating "not supplied" the same
// as "supplied empty" - fine for a hook's BACKLOG_NEW_* variable, which only
// distinguishes those for fields applyPatch itself validates.
func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefSlice(s *[]string) []string {
	if s == nil {
		return nil
	}
	return *s
}

func derefLinks(s *[]linkJSON) []linkJSON {
	if s == nil {
		return nil
	}
	return *s
}

// linkJSON is the JSON shape a create or patch request uses for one link,
// mirroring taskview.LinkView.
type linkJSON struct {
	Type string `json:"type"`
	ID   int    `json:"id"`
}

// toLinks converts request-supplied links to the internal model, rejecting
// an unknown type or a non-positive id the same way applyPatch rejects an
// empty ref.
func toLinks(raw []linkJSON) ([]task.Link, error) {
	out := make([]task.Link, 0, len(raw))
	for _, l := range raw {
		if !task.ValidLinkType(l.Type) {
			return nil, fmt.Errorf("unknown link type %q, expected one of %s", l.Type, strings.Join(task.LinkTypes, ", "))
		}
		if l.ID <= 0 {
			return nil, fmt.Errorf("link %s:%d is not a task identifier", l.Type, l.ID)
		}
		out = append(out, task.Link{Type: l.Type, ID: l.ID})
	}
	return task.NormalizeLinks(out), nil
}

func linkStrings(links []task.Link) string {
	parts := make([]string, 0, len(links))
	for _, l := range links {
		parts = append(parts, fmt.Sprintf("%s:%d", l.Type, l.ID))
	}
	return strings.Join(parts, ",")
}

// newMux builds the server's routes: the JSON API under /api and the
// embedded web UI everywhere else.
func newMux(st *store.Store, opts Options) (*http.ServeMux, error) {
	assets, err := webAssets()
	if err != nil {
		return nil, err
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/repo", handleRepoInfo(st, opts))
	mux.HandleFunc("GET /api/tasks", handleListTasks(st))
	mux.HandleFunc("POST /api/tasks", handleCreateTask(st, opts))
	mux.HandleFunc("GET /api/tasks/{id}", handleGetTask(st))
	mux.HandleFunc("PATCH /api/tasks/{id}", handlePatchTask(st, opts))
	mux.HandleFunc("DELETE /api/tasks/{id}", handleDeleteTask(st, opts))
	mux.Handle("/", http.FileServer(http.FS(assets)))
	return mux, nil
}

// repoInfo is the JSON shape of GET /api/repo: just enough for the UI's top
// bar chip (project name, git branch) and version string, none of which is
// per-task.
type repoInfo struct {
	Name    string `json:"name"`
	Branch  string `json:"branch"`
	Version string `json:"version"`
}

func handleRepoInfo(st *store.Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, repoInfo{
			Name:    filepath.Base(st.Project),
			Branch:  store.Provenance(st.Project).Branch,
			Version: opts.Version,
		})
	}
}

func handleListTasks(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tasks, err := selectTasks(st, r.URL.Query())
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		task.SortByPriorityThenID(tasks)
		writeJSON(w, taskview.Views(tasks))
	}
}

func handleGetTask(st *store.Store) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		t, err := st.Find(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeJSON(w, taskview.View(t))
	}
}

// handleDeleteTask removes a task's file exactly as `backlog rm` does, reusing
// Store.Remove, and reports the removed task's view. An unknown id surfaces as
// the same 404 handleGetTask returns, since Store.Remove calls Find internally.
func handleDeleteTask(st *store.Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		t, err := st.Find(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if err := hooks.RunPre(hookDiag(opts), st.Root, st.Project, hooks.PreRemove, t, nil); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if _, err := st.Remove(id); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		hooks.Run(hookDiag(opts), st.Root, st.Project, hooks.PostRemove, t, nil)
		writeJSON(w, taskview.View(t))
	}
}

// createRequest is the JSON body POST /api/tasks accepts, mirroring the
// fields `backlog add` accepts.
type createRequest struct {
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Tags        []string   `json:"tags"`
	Priority    string     `json:"priority"`
	Files       []string   `json:"files"`
	Refs        []string   `json:"refs"`
	Links       []linkJSON `json:"links"`
}

func handleCreateTask(st *store.Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "request body must be JSON: "+err.Error())
			return
		}

		title := strings.TrimSpace(req.Title)
		if title == "" {
			writeError(w, http.StatusBadRequest, "a title is required")
			return
		}
		priority := req.Priority
		if priority == "" {
			priority = task.DefaultPriority
		}
		if !task.ValidPriority(priority) {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("unknown priority %q, expected one of %s", priority, strings.Join(task.Priorities, ", ")))
			return
		}
		for _, ref := range req.Refs {
			if strings.TrimSpace(ref) == "" {
				writeError(w, http.StatusBadRequest, "a reference may not be empty")
				return
			}
		}
		links, err := toLinks(req.Links)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		// Everything created through the UI is a human sitting at a browser,
		// never an agent — see design.md's "Task creation always records
		// author: human".
		t := task.New(title, req.Description, req.Tags, req.Files, req.Refs,
			task.AuthorHuman, priority, store.Provenance(st.Project), time.Now())
		t.Links = links
		if err := hooks.RunPre(hookDiag(opts), st.Root, st.Project, hooks.PreAdd, t, nil); err != nil {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		if err := st.Create(t); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		hooks.Run(hookDiag(opts), st.Root, st.Project, hooks.PostAdd, t, nil)
		writeJSONStatus(w, http.StatusCreated, taskview.View(t))
	}
}

// patchRequest is the JSON body PATCH /api/tasks/{id} accepts. Every field is
// a pointer so the handler can tell "not supplied" (nil) apart from
// "supplied as the zero value" (a non-nil pointer to "" or an empty slice).
type patchRequest struct {
	Title       *string     `json:"title"`
	Description *string     `json:"description"`
	Tags        *[]string   `json:"tags"`
	Priority    *string     `json:"priority"`
	Status      *string     `json:"status"`
	Reason      *string     `json:"reason"`
	Refs        *[]string   `json:"refs"`
	Links       *[]linkJSON `json:"links"`
}

func handlePatchTask(st *store.Store, opts Options) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := pathID(r)
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		t, err := st.Find(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}

		var req patchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "request body must be JSON: "+err.Error())
			return
		}

		// A task cannot finish ahead of what it is blocked-by: that link says
		// another task's work must land first, and the only way around it is
		// to edit the link, not to race it to done. Checked against the
		// links the request would leave in place, so a request that clears
		// the blocking link in the same PATCH is not refused for it.
		if req.Status != nil && *req.Status == task.StatusDone {
			links := t.Links
			if req.Links != nil {
				parsed, err := toLinks(derefLinks(req.Links))
				if err != nil {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				links = parsed
			}
			all, err := st.Tasks()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err.Error())
				return
			}
			if blockers := task.Blockers(&task.Task{ID: t.ID, Links: links}, all); len(blockers) > 0 {
				writeError(w, http.StatusConflict, fmt.Sprintf("task %d cannot be marked done: %s; finish, decline, or unlink it first",
					t.ID, task.DescribeBlockers(blockers)))
				return
			}
		}

		prevStatus, prevPriority := t.Status, t.Priority
		// Pre-hooks run against t as it stands now, before applyPatch touches
		// it, mirroring the same split the CLI's `set` and `edit` make.
		if req.Status != nil || req.Priority != nil || req.Reason != nil || req.Refs != nil {
			if err := hooks.RunPre(hookDiag(opts), st.Root, st.Project, hooks.PreSet, t, map[string]string{
				"BACKLOG_NEW_STATUS":   deref(req.Status),
				"BACKLOG_NEW_PRIORITY": deref(req.Priority),
				"BACKLOG_NEW_REASON":   deref(req.Reason),
				"BACKLOG_NEW_REFS":     strings.Join(derefSlice(req.Refs), ","),
			}); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
		}
		if req.Title != nil || req.Description != nil || req.Tags != nil || req.Links != nil {
			newLinks, err := toLinks(derefLinks(req.Links))
			if err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if err := hooks.RunPre(hookDiag(opts), st.Root, st.Project, hooks.PreEdit, t, map[string]string{
				"BACKLOG_NEW_TITLE":       deref(req.Title),
				"BACKLOG_NEW_DESCRIPTION": deref(req.Description),
				"BACKLOG_NEW_TAGS":        strings.Join(derefSlice(req.Tags), ","),
				"BACKLOG_NEW_LINKS":       linkStrings(newLinks),
			}); err != nil {
				writeError(w, http.StatusConflict, err.Error())
				return
			}
		}
		if err := applyPatch(t, req); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if err := st.Save(t); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		// A PATCH can carry both workflow fields (status, priority, reason,
		// refs) and content fields (title, description, tags) in one
		// request, so it fires whichever of the CLI's two hooks match what
		// actually changed - the same events `set` and `edit` fire, just
		// possibly both at once here.
		if req.Status != nil || req.Priority != nil || req.Reason != nil || req.Refs != nil {
			hooks.Run(hookDiag(opts), st.Root, st.Project, hooks.PostSet, t, map[string]string{
				"BACKLOG_PREVIOUS_STATUS":   prevStatus,
				"BACKLOG_PREVIOUS_PRIORITY": prevPriority,
			})
		}
		if req.Title != nil || req.Description != nil || req.Tags != nil || req.Links != nil {
			hooks.Run(hookDiag(opts), st.Root, st.Project, hooks.PostEdit, t, nil)
		}
		writeJSON(w, taskview.View(t))
	}
}

// applyPatch validates req against t's would-be resulting state and, only if
// every check passes, applies it to t in place. Validation mirrors `add` and
// `set`: title must not be empty, status and priority must each be one of
// their permitted values, and a reason must be present exactly when the
// resulting status is declined. t is never partially mutated on the path
// that returns an error where it matters: nothing is persisted unless this
// returns nil.
func applyPatch(t *task.Task, req patchRequest) error {
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" {
			return errors.New("title must not be empty")
		}
	}
	if req.Status != nil && !task.ValidStatus(*req.Status) {
		return fmt.Errorf("unknown status %q, expected one of %s", *req.Status, strings.Join(task.Statuses, ", "))
	}
	if req.Priority != nil && !task.ValidPriority(*req.Priority) {
		return fmt.Errorf("unknown priority %q, expected one of %s", *req.Priority, strings.Join(task.Priorities, ", "))
	}
	if req.Refs != nil {
		for _, ref := range *req.Refs {
			if strings.TrimSpace(ref) == "" {
				return errors.New("a reference may not be empty")
			}
		}
	}
	var newLinks []task.Link
	if req.Links != nil {
		links, err := toLinks(*req.Links)
		if err != nil {
			return err
		}
		for _, l := range links {
			if l.ID == t.ID {
				return fmt.Errorf("task %d cannot link to itself", t.ID)
			}
		}
		newLinks = links
	}

	finalStatus := t.Status
	if req.Status != nil {
		finalStatus = *req.Status
	}
	// A reason on a task that will not be declined describes a state the
	// task is not in; reject it rather than write a reason that silently
	// stops meaning anything, matching `set`. An empty reason is not
	// rejected here: it is how the edit form clears one on the way out of
	// declined, below.
	if req.Reason != nil && strings.TrimSpace(*req.Reason) != "" && finalStatus != task.StatusDeclined {
		return fmt.Errorf("reason applies only to a %s task", task.StatusDeclined)
	}

	if req.Title != nil {
		t.Title = strings.TrimSpace(*req.Title)
	}
	if req.Status != nil {
		// Leaving declined drops the reason: it describes a state the task
		// is no longer in, and git keeps what it said.
		if t.Status == task.StatusDeclined && finalStatus != task.StatusDeclined {
			t.Reason = ""
		}
		t.Status = finalStatus
	}
	if req.Reason != nil {
		t.Reason = *req.Reason
	}
	// A decline nobody can audit is the state the status exists to
	// eliminate, so the reason is required rather than merely encouraged.
	if finalStatus == task.StatusDeclined && strings.TrimSpace(t.Reason) == "" {
		return errors.New("declining a task requires a reason, so that the decision can be read later")
	}

	if req.Priority != nil {
		t.Priority = *req.Priority
	}
	if req.Description != nil {
		t.SetBody(*req.Description)
	}
	if req.Tags != nil {
		t.Tags = task.NormalizeTags(*req.Tags)
	}
	if req.Refs != nil {
		refs := make([]string, 0, len(*req.Refs))
		refs = append(refs, (*req.Refs)...)
		t.Meta.Refs = refs
	}
	if req.Links != nil {
		t.Links = newLinks
	}
	return nil
}

// selectTasks reads a backlog and returns the tasks matching q's status, tag
// and priority filters, using the same selection rules `backlog list --all`
// applies: an explicit ?status is taken at face value; otherwise new, todo
// and doing — the non-terminal statuses — are in scope, with done and
// declined added by ?all=1. Multiple ?tag values must all match; multiple
// ?priority values match if any do.
func selectTasks(st *store.Store, q url.Values) ([]*task.Task, error) {
	statuses, err := selectedStatuses(q)
	if err != nil {
		return nil, err
	}
	priorities, err := selectedPriorities(q)
	if err != nil {
		return nil, err
	}
	tags := q["tag"]

	all, err := st.Tasks()
	if err != nil {
		return nil, err
	}
	var out []*task.Task
	for _, t := range all {
		if !statuses[t.Status] {
			continue
		}
		if priorities != nil && !priorities[t.Priority] {
			continue
		}
		if !hasAllTags(t, tags) {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func selectedStatuses(q url.Values) (map[string]bool, error) {
	out := map[string]bool{}
	if statuses := q["status"]; len(statuses) > 0 {
		for _, v := range statuses {
			v = strings.ToLower(strings.TrimSpace(v))
			if !task.ValidStatus(v) {
				return nil, fmt.Errorf("unknown status %q, expected one of %s", v, strings.Join(task.Statuses, ", "))
			}
			out[v] = true
		}
		return out, nil
	}
	out[task.StatusNew] = true
	out[task.StatusTodo] = true
	out[task.StatusDoing] = true
	if isTrue(q.Get("all")) {
		out[task.StatusDone] = true
		out[task.StatusDeclined] = true
	}
	return out, nil
}

func selectedPriorities(q url.Values) (map[string]bool, error) {
	priorities := q["priority"]
	if len(priorities) == 0 {
		return nil, nil
	}
	out := map[string]bool{}
	for _, v := range priorities {
		v = strings.ToLower(strings.TrimSpace(v))
		if !task.ValidPriority(v) {
			return nil, fmt.Errorf("unknown priority %q, expected one of %s", v, strings.Join(task.Priorities, ", "))
		}
		out[v] = true
	}
	return out, nil
}

func isTrue(v string) bool { return v == "1" || v == "true" }

func hasAllTags(t *task.Task, tags []string) bool {
	for _, tag := range tags {
		if !t.HasTag(tag) {
			return false
		}
	}
	return true
}

func pathID(r *http.Request) (int, error) {
	raw := r.PathValue("id")
	id, err := strconv.Atoi(strings.TrimLeft(raw, "#"))
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("%q is not a task identifier", raw)
	}
	return id, nil
}
