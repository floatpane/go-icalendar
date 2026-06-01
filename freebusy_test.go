package icalendar

import (
	"testing"
	"time"
)

func TestBusyPeriodsMerge(t *testing.T) {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	mk := func(h0, h1 int) *Event {
		s := base.Add(time.Duration(h0) * time.Hour)
		return &Event{UID: "x", Start: s, End: base.Add(time.Duration(h1) * time.Hour)}
	}
	events := []*Event{
		mk(9, 10),
		mk(9, 11),  // overlaps previous -> merge to 9-11
		mk(11, 12), // adjacent -> merge to 9-12
		mk(14, 15), // separate
	}

	busy := BusyPeriods(events, base, base.AddDate(0, 0, 1))
	if len(busy) != 2 {
		t.Fatalf("busy = %d periods %v, want 2", len(busy), busy)
	}
	if !busy[0].Start.Equal(base.Add(9*time.Hour)) || !busy[0].End.Equal(base.Add(12*time.Hour)) {
		t.Errorf("busy[0] = %v", busy[0])
	}
	if !busy[1].Start.Equal(base.Add(14*time.Hour)) || !busy[1].End.Equal(base.Add(15*time.Hour)) {
		t.Errorf("busy[1] = %v", busy[1])
	}
}

func TestBusyPeriodsSkipsCancelled(t *testing.T) {
	base := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	events := []*Event{
		{UID: "a", Start: base, End: base.Add(time.Hour), Status: string(StatusCancelled)},
	}
	if got := BusyPeriods(events, base.AddDate(0, 0, -1), base.AddDate(0, 0, 1)); len(got) != 0 {
		t.Errorf("cancelled event counted busy: %v", got)
	}
}

func TestBusyPeriodsExpandsRecurrence(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{UID: "r", Start: start, End: start.Add(time.Hour), Recurrence: mustRRule(t, "FREQ=DAILY;COUNT=3")}
	busy := BusyPeriods([]*Event{ev}, start, start.AddDate(0, 0, 10))
	if len(busy) != 3 {
		t.Fatalf("busy = %d, want 3", len(busy))
	}
}

func TestFreeBusyGaps(t *testing.T) {
	from := time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 18, 0, 0, 0, time.UTC)
	mtg := &Event{UID: "m", Start: from.Add(2 * time.Hour), End: from.Add(3 * time.Hour)} // 10-11

	busy, free := FreeBusy([]*Event{mtg}, from, to)
	if len(busy) != 1 {
		t.Fatalf("busy = %v", busy)
	}
	// Free: 08-10 and 11-18.
	if len(free) != 2 {
		t.Fatalf("free = %d %v, want 2", len(free), free)
	}
	if !free[0].Start.Equal(from) || !free[0].End.Equal(from.Add(2*time.Hour)) {
		t.Errorf("free[0] = %v", free[0])
	}
	if !free[1].Start.Equal(from.Add(3*time.Hour)) || !free[1].End.Equal(to) {
		t.Errorf("free[1] = %v", free[1])
	}
}

func TestFreeBusyClipsToWindow(t *testing.T) {
	from := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	to := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	// Event spans 09-13, wider than the window -> busy clipped to 10-12, no free.
	ev := &Event{UID: "w", Start: from.Add(-time.Hour), End: to.Add(time.Hour)}
	busy, free := FreeBusy([]*Event{ev}, from, to)
	if len(busy) != 1 || !busy[0].Start.Equal(from) || !busy[0].End.Equal(to) {
		t.Errorf("busy = %v", busy)
	}
	if len(free) != 0 {
		t.Errorf("free = %v, want none", free)
	}
}
