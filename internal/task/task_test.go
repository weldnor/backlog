package task

import "testing"

func TestValidStatus(t *testing.T) {
	for _, s := range []string{StatusNew, StatusTodo, StatusDoing, StatusDone, StatusDeclined} {
		if !ValidStatus(s) {
			t.Errorf("ValidStatus(%q) = false, want true", s)
		}
	}
	for _, s := range []string{"", "Declined", "wontfix", "closed"} {
		if ValidStatus(s) {
			t.Errorf("ValidStatus(%q) = true, want false", s)
		}
	}
}

func TestIsTerminal(t *testing.T) {
	// The two terminal statuses are what the archive directory holds; the
	// other two, and anything unrecognised, are not.
	for _, s := range []string{StatusDone, StatusDeclined} {
		if !IsTerminal(s) {
			t.Errorf("IsTerminal(%q) = false, want true", s)
		}
	}
	for _, s := range []string{StatusNew, StatusTodo, StatusDoing, "", "wontfix"} {
		if IsTerminal(s) {
			t.Errorf("IsTerminal(%q) = true, want false", s)
		}
	}
}

func TestValidPriority(t *testing.T) {
	for _, p := range []string{PriorityHigh, PriorityMedium, PriorityLow} {
		if !ValidPriority(p) {
			t.Errorf("ValidPriority(%q) = false, want true", p)
		}
	}
	for _, p := range []string{"", "urgent", "High", "0"} {
		if ValidPriority(p) {
			t.Errorf("ValidPriority(%q) = true, want false", p)
		}
	}
}

func TestPriorityRankOrdersBySeverity(t *testing.T) {
	if !(PriorityRank(PriorityHigh) < PriorityRank(PriorityMedium) &&
		PriorityRank(PriorityMedium) < PriorityRank(PriorityLow)) {
		t.Fatalf("ranks are not descending by severity: high=%d medium=%d low=%d",
			PriorityRank(PriorityHigh), PriorityRank(PriorityMedium), PriorityRank(PriorityLow))
	}
	// A value the CLI does not recognise still needs a defined place, after
	// every permitted one, so that a listing has a total order.
	for _, p := range []string{"", "urgent"} {
		if PriorityRank(p) <= PriorityRank(PriorityLow) {
			t.Errorf("PriorityRank(%q) = %d, want greater than low", p, PriorityRank(p))
		}
	}
}

func TestDefaultPriorityIsMedium(t *testing.T) {
	if DefaultPriority != PriorityMedium {
		t.Errorf("DefaultPriority = %q, want %q", DefaultPriority, PriorityMedium)
	}
}

func TestSortByPriorityThenID(t *testing.T) {
	tasks := []*Task{
		{ID: 4, Priority: PriorityLow},
		{ID: 1, Priority: PriorityMedium},
		{ID: 7, Priority: "urgent"},
		{ID: 3, Priority: PriorityHigh},
		{ID: 2, Priority: PriorityMedium},
		{ID: 5, Priority: PriorityHigh},
	}
	SortByPriorityThenID(tasks)

	var got []int
	for _, task := range tasks {
		got = append(got, task.ID)
	}
	want := []int{3, 5, 1, 2, 4, 7}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestSortByPriorityThenIDIsStable(t *testing.T) {
	// Two runs over the same input must agree, since the listing order is part
	// of what the CLI promises.
	build := func() []*Task {
		return []*Task{
			{ID: 9, Priority: PriorityMedium},
			{ID: 2, Priority: PriorityMedium},
			{ID: 6, Priority: PriorityHigh},
		}
	}
	a, b := build(), build()
	SortByPriorityThenID(a)
	SortByPriorityThenID(b)
	for i := range a {
		if a[i].ID != b[i].ID {
			t.Fatalf("runs disagree at %d: %d vs %d", i, a[i].ID, b[i].ID)
		}
	}
}

func TestBlockers(t *testing.T) {
	blocker := &Task{ID: 1, Status: StatusTodo}
	unrelated := &Task{ID: 2, Status: StatusTodo}
	all := []*Task{blocker, unrelated}

	t.Run("an unfinished blocked-by target blocks", func(t *testing.T) {
		tk := &Task{ID: 3, Links: []Link{{Type: LinkBlockedBy, ID: blocker.ID}}}
		got := Blockers(tk, all)
		if len(got) != 1 || got[0].ID != blocker.ID {
			t.Fatalf("Blockers = %+v, want [blocker]", got)
		}
	})

	t.Run("a done blocked-by target does not block", func(t *testing.T) {
		done := &Task{ID: 1, Status: StatusDone}
		tk := &Task{ID: 3, Links: []Link{{Type: LinkBlockedBy, ID: done.ID}}}
		if got := Blockers(tk, []*Task{done}); len(got) != 0 {
			t.Fatalf("Blockers = %+v, want none", got)
		}
	})

	t.Run("a declined blocked-by target does not block", func(t *testing.T) {
		declined := &Task{ID: 1, Status: StatusDeclined}
		tk := &Task{ID: 3, Links: []Link{{Type: LinkBlockedBy, ID: declined.ID}}}
		if got := Blockers(tk, []*Task{declined}); len(got) != 0 {
			t.Fatalf("Blockers = %+v, want none", got)
		}
	})

	t.Run("other link types do not block", func(t *testing.T) {
		tk := &Task{ID: 3, Links: []Link{
			{Type: LinkBlocks, ID: blocker.ID},
			{Type: LinkRelated, ID: unrelated.ID},
		}}
		if got := Blockers(tk, all); len(got) != 0 {
			t.Fatalf("Blockers = %+v, want none", got)
		}
	})

	t.Run("a dangling target does not block", func(t *testing.T) {
		tk := &Task{ID: 3, Links: []Link{{Type: LinkBlockedBy, ID: 99}}}
		if got := Blockers(tk, all); len(got) != 0 {
			t.Fatalf("Blockers = %+v, want none", got)
		}
	})

	t.Run("no links means no blockers", func(t *testing.T) {
		tk := &Task{ID: 3}
		if got := Blockers(tk, all); len(got) != 0 {
			t.Fatalf("Blockers = %+v, want none", got)
		}
	})
}

func TestDescribeBlockers(t *testing.T) {
	got := DescribeBlockers([]*Task{{ID: 1, Status: StatusTodo}})
	if want := "blocked by task 1 (todo)"; got != want {
		t.Errorf("DescribeBlockers = %q, want %q", got, want)
	}

	got = DescribeBlockers([]*Task{{ID: 1, Status: StatusTodo}, {ID: 2, Status: StatusDoing}})
	if want := "blocked by tasks 1 (todo), 2 (doing)"; got != want {
		t.Errorf("DescribeBlockers = %q, want %q", got, want)
	}
}
