package cli

import (
	"strconv"
	"strings"

	"github.com/weldnor/backlog/internal/task"
)

// parseLink parses one --link value in "type:id" form, e.g. "blocks:5".
func parseLink(raw string) (task.Link, error) {
	typ, idStr, ok := strings.Cut(raw, ":")
	if !ok {
		return task.Link{}, usagef("link %q must be type:id, e.g. blocks:5", raw)
	}
	typ = strings.TrimSpace(typ)
	if !task.ValidLinkType(typ) {
		return task.Link{}, usagef("unknown link type %q, expected one of %s", typ, strings.Join(task.LinkTypes, ", "))
	}
	id, err := strconv.Atoi(strings.TrimSpace(idStr))
	if err != nil || id <= 0 {
		return task.Link{}, usagef("link %q: %q is not a task identifier", raw, idStr)
	}
	return task.Link{Type: typ, ID: id}, nil
}

// parseLinks parses every --link value, in order, failing on the first bad
// one.
func parseLinks(raw []string) ([]task.Link, error) {
	out := make([]task.Link, 0, len(raw))
	for _, r := range raw {
		l, err := parseLink(r)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, nil
}

// linkStrings renders links back to "type:id" form, for a hook's
// BACKLOG_NEW_LINKS environment variable.
func linkStrings(links []task.Link) []string {
	out := make([]string, 0, len(links))
	for _, l := range links {
		out = append(out, l.Type+":"+strconv.Itoa(l.ID))
	}
	return out
}
