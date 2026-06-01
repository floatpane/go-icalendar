package icalendar

import (
	"sort"
	"time"
)

// Period is a half-open time interval [Start, End).
type Period struct {
	Start time.Time
	End   time.Time
}

// Duration returns End-Start.
func (p Period) Duration() time.Duration { return p.End.Sub(p.Start) }

// Contains reports whether t lies within [Start, End).
func (p Period) Contains(t time.Time) bool {
	return !t.Before(p.Start) && t.Before(p.End)
}

// BusyPeriods expands every event (including recurrences) into its occurrences
// within [from, to), clips each to the window, and merges overlapping or
// touching intervals into a sorted, non-overlapping list of busy time.
//
// Events with zero duration, and all-day events, are honored: an all-day event
// contributes its full DTSTART–DTEND span. CANCELLED events are skipped.
func BusyPeriods(events []*Event, from, to time.Time) []Period {
	var busy []Period
	for _, e := range events {
		if e == nil || e.Start.IsZero() {
			continue
		}
		if e.Status == string(StatusCancelled) {
			continue
		}
		dur := e.Duration()
		if dur < 0 {
			dur = 0
		}
		// Widen the lower bound by the event duration so an occurrence that
		// starts before the window but runs into it is still caught.
		for _, start := range e.Occurrences(from.Add(-dur), to, 0) {
			end := start.Add(dur)
			// Clip to the window.
			if start.Before(from) {
				start = from
			}
			if end.After(to) {
				end = to
			}
			if end.After(start) {
				busy = append(busy, Period{Start: start, End: end})
			}
		}
	}
	return mergePeriods(busy)
}

// FreeBusy returns the busy intervals within [from, to) (as [BusyPeriods]) and
// the free gaps between them that fill the rest of the window.
func FreeBusy(events []*Event, from, to time.Time) (busy, free []Period) {
	busy = BusyPeriods(events, from, to)
	cursor := from
	for _, b := range busy {
		if b.Start.After(cursor) {
			free = append(free, Period{Start: cursor, End: b.Start})
		}
		if b.End.After(cursor) {
			cursor = b.End
		}
	}
	if cursor.Before(to) {
		free = append(free, Period{Start: cursor, End: to})
	}
	return busy, free
}

// mergePeriods sorts intervals and coalesces any that overlap or touch.
func mergePeriods(ps []Period) []Period {
	if len(ps) == 0 {
		return nil
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Start.Before(ps[j].Start) })

	merged := []Period{ps[0]}
	for _, p := range ps[1:] {
		last := &merged[len(merged)-1]
		if !p.Start.After(last.End) { // overlap or adjacency
			if p.End.After(last.End) {
				last.End = p.End
			}
			continue
		}
		merged = append(merged, p)
	}
	return merged
}
