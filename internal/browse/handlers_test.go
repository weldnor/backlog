package browse

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/weldnor/backlog/internal/store"
	"github.com/weldnor/backlog/internal/task"
	"github.com/weldnor/backlog/internal/taskview"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Init(t.TempDir())
	if err != nil {
		t.Fatalf("store.Init: %v", err)
	}
	return st
}

func newTestMux(t *testing.T, st *store.Store) http.Handler {
	t.Helper()
	mux, err := newMux(st, Options{Version: "test"})
	if err != nil {
		t.Fatalf("newMux: %v", err)
	}
	return mux
}

func addTask(t *testing.T, st *store.Store, title, status, priority string, tags []string) *task.Task {
	t.Helper()
	tk := task.New(title, "", tags, nil, nil, task.AuthorAgent, priority, task.Source{}, time.Now())
	if err := st.Create(tk); err != nil {
		t.Fatalf("Create(%q): %v", title, err)
	}
	if status != task.StatusTodo {
		tk.Status = status
		if status == task.StatusDeclined {
			tk.Reason = "because"
		}
		if err := st.Save(tk); err != nil {
			t.Fatalf("Save(%q): %v", title, err)
		}
	}
	return tk
}

func doJSON(t *testing.T, h http.Handler, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var r *http.Request
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal request body: %v", err)
		}
		r = httptest.NewRequest(method, path, bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func decodeBody[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var v T
	if err := json.Unmarshal(w.Body.Bytes(), &v); err != nil {
		t.Fatalf("decode response body %q: %v", w.Body.String(), err)
	}
	return v
}

func TestListTasksDefaultsToNonTerminal(t *testing.T) {
	st := newTestStore(t)
	addTask(t, st, "in progress", task.StatusDoing, task.PriorityMedium, nil)
	addTask(t, st, "waiting", task.StatusTodo, task.PriorityHigh, nil)
	addTask(t, st, "finished", task.StatusDone, task.PriorityLow, nil)
	addTask(t, st, "declined one", task.StatusDeclined, task.PriorityLow, nil)

	h := newTestMux(t, st)
	w := doJSON(t, h, http.MethodGet, "/api/tasks", nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeBody[[]taskview.TaskView](t, w)
	if len(got) != 2 {
		t.Fatalf("got %d tasks, want 2 (todo+doing only): %+v", len(got), got)
	}
	// descending priority, then ascending identifier
	if got[0].Title != "waiting" || got[1].Title != "in progress" {
		t.Errorf("order = %q, %q; want waiting (high) before in progress (medium)", got[0].Title, got[1].Title)
	}
}

func TestListTasksAllIncludesArchive(t *testing.T) {
	st := newTestStore(t)
	addTask(t, st, "waiting", task.StatusTodo, task.PriorityMedium, nil)
	addTask(t, st, "finished", task.StatusDone, task.PriorityMedium, nil)
	addTask(t, st, "declined one", task.StatusDeclined, task.PriorityMedium, nil)

	h := newTestMux(t, st)
	w := doJSON(t, h, http.MethodGet, "/api/tasks?all=1", nil)
	got := decodeBody[[]taskview.TaskView](t, w)
	if len(got) != 3 {
		t.Fatalf("got %d tasks, want 3 with ?all=1: %+v", len(got), got)
	}
}

func TestListTasksFiltersByStatusTagPriority(t *testing.T) {
	st := newTestStore(t)
	addTask(t, st, "a", task.StatusTodo, task.PriorityHigh, []string{"bug"})
	addTask(t, st, "b", task.StatusTodo, task.PriorityLow, []string{"bug", "ux"})
	addTask(t, st, "c", task.StatusDoing, task.PriorityHigh, []string{"ux"})

	h := newTestMux(t, st)

	w := doJSON(t, h, http.MethodGet, "/api/tasks?status=todo&priority=high", nil)
	got := decodeBody[[]taskview.TaskView](t, w)
	if len(got) != 1 || got[0].Title != "a" {
		t.Fatalf("status=todo&priority=high = %+v, want just %q", got, "a")
	}

	w = doJSON(t, h, http.MethodGet, "/api/tasks?all=1&tag=bug&tag=ux", nil)
	got = decodeBody[[]taskview.TaskView](t, w)
	if len(got) != 1 || got[0].Title != "b" {
		t.Fatalf("tag=bug&tag=ux (AND) = %+v, want just %q", got, "b")
	}
}

func TestListTasksRejectsUnknownFilter(t *testing.T) {
	h := newTestMux(t, newTestStore(t))
	w := doJSON(t, h, http.MethodGet, "/api/tasks?status=bogus", nil)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var env errorEnvelope
	if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Error == "" {
		t.Fatalf("body = %q, want a JSON error envelope", w.Body.String())
	}
}

func TestGetTaskNotFound(t *testing.T) {
	h := newTestMux(t, newTestStore(t))
	w := doJSON(t, h, http.MethodGet, "/api/tasks/42", nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404, body %s", w.Code, w.Body.String())
	}
}

func TestGetTaskFound(t *testing.T) {
	st := newTestStore(t)
	tk := addTask(t, st, "findable", task.StatusTodo, task.PriorityMedium, nil)
	h := newTestMux(t, st)
	w := doJSON(t, h, http.MethodGet, "/api/tasks/"+itoa(tk.ID), nil)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeBody[taskview.TaskView](t, w)
	if got.Title != "findable" {
		t.Errorf("title = %q, want %q", got.Title, "findable")
	}
}

func TestCreateTaskMinimal(t *testing.T) {
	st := newTestStore(t)
	h := newTestMux(t, st)
	w := doJSON(t, h, http.MethodPost, "/api/tasks", createRequest{Title: "captured from the UI"})
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeBody[taskview.TaskView](t, w)
	if got.Status != task.StatusTodo || got.Priority != task.DefaultPriority || got.Metadata.Author != task.AuthorHuman {
		t.Errorf("created task = %+v, want status todo, priority %s, author %s", got, task.DefaultPriority, task.AuthorHuman)
	}

	onDisk, err := st.Find(got.ID)
	if err != nil {
		t.Fatalf("task not on disk: %v", err)
	}
	if onDisk.Title != "captured from the UI" {
		t.Errorf("on-disk title = %q", onDisk.Title)
	}
}

func TestCreateTaskFullContext(t *testing.T) {
	st := newTestStore(t)
	h := newTestMux(t, st)
	req := createRequest{
		Title: "full", Description: "body text", Tags: []string{"a", "b"},
		Priority: task.PriorityHigh, Files: []string{"main.go"}, Refs: []string{"issue:1"},
	}
	w := doJSON(t, h, http.MethodPost, "/api/tasks", req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeBody[taskview.TaskView](t, w)
	if got.Priority != task.PriorityHigh || got.Description != "body text" ||
		len(got.Tags) != 2 || len(got.Metadata.Source.Files) != 1 || len(got.Metadata.Refs) != 1 {
		t.Errorf("created task = %+v, missing supplied fields", got)
	}
}

func TestCreateTaskRejectsEmptyTitle(t *testing.T) {
	st := newTestStore(t)
	h := newTestMux(t, st)
	w := doJSON(t, h, http.MethodPost, "/api/tasks", createRequest{Title: "   "})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	tasks, _ := st.Tasks()
	if len(tasks) != 0 {
		t.Errorf("a task was created despite the empty title: %+v", tasks)
	}
}

func TestCreateTaskRejectsInvalidPriority(t *testing.T) {
	h := newTestMux(t, newTestStore(t))
	w := doJSON(t, h, http.MethodPost, "/api/tasks", createRequest{Title: "x", Priority: "urgent"})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestCreateTaskConcurrentGetsDistinctIDs(t *testing.T) {
	st := newTestStore(t)
	h := newTestMux(t, st)

	const n = 12
	var wg sync.WaitGroup
	ids := make([]int, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			w := doJSON(t, h, http.MethodPost, "/api/tasks", createRequest{Title: "concurrent"})
			if w.Code != http.StatusCreated {
				t.Errorf("create %d: status = %d, body %s", i, w.Code, w.Body.String())
				return
			}
			ids[i] = decodeBody[taskview.TaskView](t, w).ID
		}(i)
	}
	wg.Wait()

	seen := map[int]bool{}
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if seen[id] {
			t.Fatalf("identifier %d claimed by more than one concurrent create: %v", id, ids)
		}
		seen[id] = true
	}
	if len(seen) != n {
		t.Fatalf("got %d distinct identifiers, want %d: %v", len(seen), n, ids)
	}
}

func TestPatchDescriptionOnlyLeavesOtherFieldsUnchanged(t *testing.T) {
	st := newTestStore(t)
	tk := addTask(t, st, "keep me", task.StatusDoing, task.PriorityHigh, []string{"bug"})
	h := newTestMux(t, st)

	desc := "a new description"
	w := doJSON(t, h, http.MethodPatch, "/api/tasks/"+itoa(tk.ID), patchRequest{Description: &desc})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeBody[taskview.TaskView](t, w)
	if got.Title != "keep me" || got.Status != task.StatusDoing || got.Priority != task.PriorityHigh ||
		len(got.Tags) != 1 || got.Tags[0] != "bug" || got.Description != desc {
		t.Errorf("patched task = %+v, only description should have changed", got)
	}
}

func TestPatchDeclineRequiresReason(t *testing.T) {
	st := newTestStore(t)
	tk := addTask(t, st, "will decline", task.StatusTodo, task.PriorityMedium, nil)
	h := newTestMux(t, st)

	status := task.StatusDeclined
	w := doJSON(t, h, http.MethodPatch, "/api/tasks/"+itoa(tk.ID), patchRequest{Status: &status})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 declining without a reason, body %s", w.Code, w.Body.String())
	}
	onDisk, _ := st.Find(tk.ID)
	if onDisk.Status != task.StatusTodo {
		t.Errorf("task status changed to %q despite the rejected request", onDisk.Status)
	}
}

func TestPatchDeclineWithReasonMovesToArchive(t *testing.T) {
	st := newTestStore(t)
	tk := addTask(t, st, "will decline", task.StatusTodo, task.PriorityMedium, nil)
	h := newTestMux(t, st)

	status, reason := task.StatusDeclined, "not worth it"
	w := doJSON(t, h, http.MethodPatch, "/api/tasks/"+itoa(tk.ID), patchRequest{Status: &status, Reason: &reason})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeBody[taskview.TaskView](t, w)
	if got.Status != task.StatusDeclined || got.Reason != reason {
		t.Errorf("patched task = %+v, want declined with the given reason", got)
	}
}

func TestPatchReopenClearsReason(t *testing.T) {
	st := newTestStore(t)
	tk := addTask(t, st, "was declined", task.StatusDeclined, task.PriorityMedium, nil)
	h := newTestMux(t, st)

	status, reason := task.StatusTodo, ""
	w := doJSON(t, h, http.MethodPatch, "/api/tasks/"+itoa(tk.ID), patchRequest{Status: &status, Reason: &reason})
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	got := decodeBody[taskview.TaskView](t, w)
	if got.Status != task.StatusTodo || got.Reason != "" {
		t.Errorf("patched task = %+v, want todo with the reason cleared", got)
	}
}

func TestPatchReasonOnNonDeclinedIsRejected(t *testing.T) {
	st := newTestStore(t)
	tk := addTask(t, st, "still open", task.StatusTodo, task.PriorityMedium, nil)
	h := newTestMux(t, st)

	reason := "why though"
	w := doJSON(t, h, http.MethodPatch, "/api/tasks/"+itoa(tk.ID), patchRequest{Reason: &reason})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400 for a reason on a non-declined task, body %s", w.Code, w.Body.String())
	}
}

func TestPatchRejectsEmptyTitle(t *testing.T) {
	st := newTestStore(t)
	tk := addTask(t, st, "keep this title", task.StatusTodo, task.PriorityMedium, nil)
	h := newTestMux(t, st)

	empty := "   "
	w := doJSON(t, h, http.MethodPatch, "/api/tasks/"+itoa(tk.ID), patchRequest{Title: &empty})
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	onDisk, _ := st.Find(tk.ID)
	if onDisk.Title != "keep this title" {
		t.Errorf("title changed to %q despite the rejected request", onDisk.Title)
	}
}

func TestPatchUnknownTaskNotFound(t *testing.T) {
	h := newTestMux(t, newTestStore(t))
	title := "x"
	w := doJSON(t, h, http.MethodPatch, "/api/tasks/999", patchRequest{Title: &title})
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestErrorEnvelopeStatusCodes(t *testing.T) {
	h := newTestMux(t, newTestStore(t))

	cases := []struct {
		name   string
		method string
		path   string
		body   any
		want   int
	}{
		{"validation", http.MethodPost, "/api/tasks", createRequest{Priority: "bogus"}, http.StatusBadRequest},
		{"not found", http.MethodGet, "/api/tasks/12345", nil, http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			w := doJSON(t, h, c.method, c.path, c.body)
			if w.Code != c.want {
				t.Fatalf("status = %d, want %d, body %s", w.Code, c.want, w.Body.String())
			}
			var env errorEnvelope
			if err := json.Unmarshal(w.Body.Bytes(), &env); err != nil || env.Error == "" {
				t.Fatalf("body = %q, want a non-empty JSON error envelope", w.Body.String())
			}
		})
	}
}

func itoa(id int) string { return strconv.Itoa(id) }
