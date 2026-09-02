package cli

import (
	"strings"
	"testing"
)

func TestStatsCountsByStatusPriorityAndTag(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	seed(t, h) // One(low), Two(high, bug), Three(medium), Four(high)
	h.mustRun("set", "1", "done")
	h.mustRun("set", "2", "declined", "--reason", "not worth it")

	var got StatsView
	decode(t, h.mustRun("stats", "--json"), &got)

	if got.Total != 4 {
		t.Fatalf("Total = %d, want 4", got.Total)
	}
	if got.ByStatus["done"] != 1 || got.ByStatus["declined"] != 1 || got.ByStatus["new"] != 2 {
		t.Errorf("ByStatus = %+v", got.ByStatus)
	}
	if got.ByPriority["high"] != 2 || got.ByPriority["medium"] != 1 || got.ByPriority["low"] != 1 {
		t.Errorf("ByPriority = %+v", got.ByPriority)
	}
	if got.ByTag["bug"] != 1 {
		t.Errorf("ByTag = %+v", got.ByTag)
	}
}

func TestStatsOpenAverageAgeCountsOnlyNonTerminal(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "Still open")
	h.mustRun("add", "Finished")
	h.mustRun("set", "2", "done")

	var got StatsView
	decode(t, h.mustRun("stats", "--json"), &got)

	if got.OpenAvgAgeDays == nil {
		t.Fatal("OpenAvgAgeDays is nil, want a value for the one open task")
	}
	if *got.OpenAvgAgeDays < 0 {
		t.Errorf("OpenAvgAgeDays = %v, want >= 0", *got.OpenAvgAgeDays)
	}
}

func TestStatsEmptyBacklogHasNoAge(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()

	var got StatsView
	decode(t, h.mustRun("stats", "--json"), &got)

	if got.Total != 0 {
		t.Errorf("Total = %d, want 0", got.Total)
	}
	if got.OpenAvgAgeDays != nil {
		t.Errorf("OpenAvgAgeDays = %v, want nil", *got.OpenAvgAgeDays)
	}
}

func TestStatsHumanOutput(t *testing.T) {
	h := newHarness(t)
	h.initBacklog()
	h.mustRun("add", "One", "--priority", "high", "--tag", "bug")

	out := h.mustRun("stats")
	for _, want := range []string{"total 1", "by status", "by priority", "by tag", "bug"} {
		if !strings.Contains(out, want) {
			t.Errorf("stats output missing %q:\n%s", want, out)
		}
	}
}
