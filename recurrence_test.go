package icalendar

import (
	"testing"
	"time"
)

func mustRRule(t *testing.T, s string) *RRule {
	t.Helper()
	r, err := ParseRRule(s)
	if err != nil {
		t.Fatalf("ParseRRule(%q): %v", s, err)
	}
	return r
}

func datesEqual(t *testing.T, got, want []time.Time) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d times %v, want %d %v", len(got), got, len(want), want)
	}
	for i := range got {
		if !got[i].Equal(want[i]) {
			t.Errorf("times[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestParseRRuleRoundTrip(t *testing.T) {
	tests := []string{
		"FREQ=DAILY",
		"FREQ=WEEKLY;INTERVAL=2;COUNT=10;BYDAY=MO,WE,FR",
		"FREQ=MONTHLY;BYDAY=-1FR",
		"FREQ=YEARLY;BYMONTH=3;BYMONTHDAY=15",
		"FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=-1",
	}
	for _, s := range tests {
		r := mustRRule(t, s)
		if got := r.String(); got != s {
			t.Errorf("round-trip %q -> %q", s, got)
		}
	}
}

func TestParseRRuleErrors(t *testing.T) {
	for _, s := range []string{"INTERVAL=2", "FREQ=DAILY;INTERVAL=0", "FREQ=DAILY;BYMONTH=13", "FREQ=DAILY;BYDAY=XX"} {
		if _, err := ParseRRule(s); err == nil {
			t.Errorf("ParseRRule(%q): want error", s)
		}
	}
}

func TestDailyCount(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{UID: "d@x", Start: start, End: start.Add(time.Hour), Recurrence: mustRRule(t, "FREQ=DAILY;COUNT=3")}
	got := ev.Occurrences(start, start.AddDate(0, 1, 0), 0)
	datesEqual(t, got, []time.Time{
		start,
		start.AddDate(0, 0, 1),
		start.AddDate(0, 0, 2),
	})
}

func TestDailyInterval(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=DAILY;INTERVAL=2;COUNT=3")}
	got := ev.Occurrences(start, start.AddDate(0, 1, 0), 0)
	datesEqual(t, got, []time.Time{start, start.AddDate(0, 0, 2), start.AddDate(0, 0, 4)})
}

func TestWeeklyByDay(t *testing.T) {
	// Thursday 2026-01-01. Want MO & WE, 4 occurrences.
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=WEEKLY;BYDAY=MO,WE;COUNT=4")}
	got := ev.Occurrences(start, start.AddDate(0, 2, 0), 0)
	want := []time.Time{
		time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC),  // Mon
		time.Date(2026, 1, 7, 9, 0, 0, 0, time.UTC),  // Wed
		time.Date(2026, 1, 12, 9, 0, 0, 0, time.UTC), // Mon
		time.Date(2026, 1, 14, 9, 0, 0, 0, time.UTC), // Wed
	}
	datesEqual(t, got, want)
}

func TestUntilBound(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=DAILY;UNTIL=20260103T090000Z")}
	got := ev.Occurrences(start, start.AddDate(0, 1, 0), 0)
	datesEqual(t, got, []time.Time{start, start.AddDate(0, 0, 1), start.AddDate(0, 0, 2)})
}

func TestMonthlyByMonthDay(t *testing.T) {
	start := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=MONTHLY;COUNT=3")}
	got := ev.Occurrences(start, start.AddDate(1, 0, 0), 0)
	datesEqual(t, got, []time.Time{
		time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 15, 10, 0, 0, 0, time.UTC),
	})
}

func TestMonthlySkipsMissingDay(t *testing.T) {
	// 31st: Feb and Apr have no 31st, so they're skipped (RFC 5545).
	start := time.Date(2026, 1, 31, 8, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=MONTHLY;BYMONTHDAY=31;COUNT=3")}
	got := ev.Occurrences(start, start.AddDate(1, 0, 0), 0)
	datesEqual(t, got, []time.Time{
		time.Date(2026, 1, 31, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 31, 8, 0, 0, 0, time.UTC),
		time.Date(2026, 5, 31, 8, 0, 0, 0, time.UTC),
	})
}

func TestMonthlyLastFriday(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=MONTHLY;BYDAY=-1FR;COUNT=3")}
	got := ev.Occurrences(start, start.AddDate(1, 0, 0), 0)
	want := []time.Time{
		time.Date(2026, 1, 30, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 27, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
	}
	datesEqual(t, got, want)
}

func TestMonthlySecondMonday(t *testing.T) {
	start := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=MONTHLY;BYDAY=2MO;COUNT=2")}
	got := ev.Occurrences(start, start.AddDate(1, 0, 0), 0)
	want := []time.Time{
		time.Date(2026, 1, 12, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
	}
	datesEqual(t, got, want)
}

func TestYearly(t *testing.T) {
	start := time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=YEARLY;COUNT=3")}
	got := ev.Occurrences(start, start.AddDate(5, 0, 0), 0)
	datesEqual(t, got, []time.Time{
		time.Date(2026, 3, 15, 9, 0, 0, 0, time.UTC),
		time.Date(2027, 3, 15, 9, 0, 0, 0, time.UTC),
		time.Date(2028, 3, 15, 9, 0, 0, 0, time.UTC),
	})
}

func TestBySetPosLastWeekday(t *testing.T) {
	// Last weekday (MO-FR) of the month.
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=MONTHLY;BYDAY=MO,TU,WE,TH,FR;BYSETPOS=-1;COUNT=2")}
	got := ev.Occurrences(start, start.AddDate(1, 0, 0), 0)
	want := []time.Time{
		time.Date(2026, 1, 30, 9, 0, 0, 0, time.UTC), // Fri Jan 30
		time.Date(2026, 2, 27, 9, 0, 0, 0, time.UTC), // Fri Feb 27
	}
	datesEqual(t, got, want)
}

func TestWindowFiltersAndExDate(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{
		Start:      start,
		Recurrence: mustRRule(t, "FREQ=DAILY;COUNT=10"),
		ExDates:    []time.Time{start.AddDate(0, 0, 1)},
	}
	// Window covering days 0..4 (5 occurrences), minus the excluded day 1.
	got := ev.Occurrences(start, start.AddDate(0, 0, 5), 0)
	datesEqual(t, got, []time.Time{
		start,
		start.AddDate(0, 0, 2),
		start.AddDate(0, 0, 3),
		start.AddDate(0, 0, 4),
	})
}

func TestRDateMerge(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	extra := time.Date(2026, 1, 1, 15, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=DAILY;COUNT=1"), RDates: []time.Time{extra}}
	got := ev.Occurrences(start, start.AddDate(0, 0, 1), 0)
	datesEqual(t, got, []time.Time{start, extra})
}

func TestNonRecurringOccurrences(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{Start: start}
	if got := ev.Occurrences(start, start.AddDate(0, 0, 1), 0); len(got) != 1 || !got[0].Equal(start) {
		t.Errorf("non-recurring occurrences = %v", got)
	}
	// Outside window -> none.
	if got := ev.Occurrences(start.AddDate(0, 0, 1), start.AddDate(0, 0, 2), 0); len(got) != 0 {
		t.Errorf("out-of-window = %v, want none", got)
	}
}

func TestLimit(t *testing.T) {
	start := time.Date(2026, 1, 1, 9, 0, 0, 0, time.UTC)
	ev := &Event{Start: start, Recurrence: mustRRule(t, "FREQ=DAILY")}
	got := ev.Occurrences(start, start.AddDate(1, 0, 0), 5)
	if len(got) != 5 {
		t.Errorf("limit not honored: got %d", len(got))
	}
}
