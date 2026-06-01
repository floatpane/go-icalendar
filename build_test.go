package icalendar

import (
	"strings"
	"testing"
	"time"
)

func TestSerializeRoundTrip(t *testing.T) {
	start := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
	orig := &Event{
		UID:         "rt@example.com",
		Summary:     "Roundtrip",
		Description: "desc",
		Location:    "Room 1",
		Start:       start,
		End:         start.Add(time.Hour),
		Organizer:   "alice@example.com",
		Status:      string(StatusConfirmed),
		Attendees: []Attendee{
			{Email: "bob@example.com", Name: "Bob", PartStat: PartStatNeedsAction, RSVP: true},
		},
	}

	data, err := NewRequest(orig).Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if !strings.Contains(string(data), "METHOD:REQUEST") {
		t.Error("missing METHOD:REQUEST")
	}

	got, err := ParseICS(data)
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}
	if got.UID != orig.UID || got.Summary != orig.Summary || got.Location != orig.Location {
		t.Errorf("fields lost: %+v", got)
	}
	if !got.Start.Equal(orig.Start) || !got.End.Equal(orig.End) {
		t.Errorf("times changed: %v–%v", got.Start, got.End)
	}
	if got.Organizer != orig.Organizer {
		t.Errorf("Organizer = %q", got.Organizer)
	}
	if len(got.Attendees) != 1 || got.Attendees[0].Email != "bob@example.com" {
		t.Errorf("attendees = %+v", got.Attendees)
	}
}

func TestNewCancel(t *testing.T) {
	ev := &Event{UID: "c@x", Summary: "Gone", Sequence: 2,
		Start: time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)}
	data, err := NewCancel(ev).Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	s := string(data)
	if !strings.Contains(s, "METHOD:CANCEL") {
		t.Error("missing METHOD:CANCEL")
	}
	if !strings.Contains(s, "STATUS:CANCELLED") {
		t.Error("missing STATUS:CANCELLED")
	}
	if !strings.Contains(s, "SEQUENCE:3") {
		t.Error("SEQUENCE not bumped to 3")
	}
}

func TestSerializeAllDay(t *testing.T) {
	ev := &Event{
		UID:     "ad@x",
		Summary: "Holiday",
		AllDay:  true,
		Start:   time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC),
		End:     time.Date(2026, 7, 5, 0, 0, 0, 0, time.UTC),
	}
	data, err := NewCalendar().Add(ev).Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	if !strings.Contains(string(data), "VALUE=DATE:20260704") {
		t.Errorf("all-day DTSTART not date-only:\n%s", data)
	}
}

func TestSerializeRequiresUID(t *testing.T) {
	if _, err := NewCalendar().Add(&Event{Summary: "no uid"}).Serialize(); err == nil {
		t.Error("want error for missing UID")
	}
}
