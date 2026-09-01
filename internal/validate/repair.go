package validate

import (
	"fmt"
	"path/filepath"

	"github.com/weldnor/backlog/internal/store"
	"github.com/weldnor/backlog/internal/task"
)

// repair fixes the findings that have a single unambiguous correction:
// renaming a file whose slug drifted from its title, adding a missing format
// version, writing down the default priority where none was recorded,
// normalising timestamp formatting, and de-duplicating tags.
//
// Everything that needs a judgement is left alone. Two tasks sharing an
// identifier cannot be renumbered without deciding which one keeps the number,
// frontmatter that does not parse cannot be rewritten without guessing what it
// was meant to say, and a priority outside the permitted set is a value someone
// typed on purpose rather than an omission to fill in. A declined task with no
// reason and a reason recorded on a task that is not declined are the same
// kind of case from the other direction: one needs prose written and the other
// needs prose deleted, and neither is text a tool can supply or discard.
func repair(st *store.Store) ([]string, error) {
	entries, err := st.Entries()
	if err != nil {
		return nil, err
	}

	duplicated := map[int]int{}
	for _, e := range entries {
		if e.Task != nil && e.Task.ID > 0 {
			duplicated[e.Task.ID]++
		}
	}

	var actions []string
	for _, e := range entries {
		if e.Err != nil {
			continue // unparseable: nothing to rewrite from
		}
		t := e.Task
		if t.ID <= 0 || duplicated[t.ID] > 1 || t.HasBlockingIssue() {
			continue
		}
		if nameID, ok := task.IDFromFileName(e.Name); ok && nameID != t.ID {
			continue // which of the two identifiers is right is a judgement
		}

		var changes []string
		if t.Meta.Schema == 0 {
			t.Meta.Schema = task.SchemaVersion
			changes = append(changes, fmt.Sprintf("set metadata.schema to %d", task.SchemaVersion))
		}
		if normalized, changed := task.NormalizeCreated(t.Meta.Created); changed {
			t.Meta.Created = normalized
			changes = append(changes, fmt.Sprintf("normalised metadata.created to %s", normalized))
		}
		if t.HasRepairableIssue() {
			// Reading already dropped duplicate tags and recovered a missing
			// identifier; writing the file back is what makes that stick.
			changes = append(changes, "rewrote the frontmatter")
		}

		want := t.FileName()
		if want != e.Name {
			changes = append(changes, fmt.Sprintf("renamed to %s", want))
		}
		if len(changes) == 0 {
			continue
		}

		if err := st.Save(t); err != nil {
			return nil, err
		}
		for _, ch := range changes {
			actions = append(actions, fmt.Sprintf("%s: %s", relTo(st.Project, e.Path), ch))
		}
	}
	return actions, nil
}

func relTo(base, path string) string {
	r, err := filepath.Rel(base, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	return filepath.ToSlash(r)
}
