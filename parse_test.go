package icalendar

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestParseICS_Simple(t *testing.T) {
	data, err := os.ReadFile("testdata/simple.ics")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	event, err := ParseICS(data)
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}

	checks := []struct {
		name, got, want string
	}{
		{"UID", event.UID, "test-event-123@example.com"},
		{"Summary", event.Summary, "Q2 Planning Meeting"},
		{"Location", event.Location, "Conference Room A"},
		{"Organizer", event.Organizer, "alice@company.com"},
		{"Status", event.Status, "CONFIRMED"},
		{"Method", event.Method, "REQUEST"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", c.name, c.got, c.want)
		}
	}

	wantStart := time.Date(2026, 4, 21, 14, 0, 0, 0, time.UTC)
	if !event.Start.Equal(wantStart) {
		t.Errorf("Start = %v, want %v", event.Start, wantStart)
	}
	wantEnd := time.Date(2026, 4, 21, 15, 30, 0, 0, time.UTC)
	if !event.End.Equal(wantEnd) {
		t.Errorf("End = %v, want %v", event.End, wantEnd)
	}

	if got := event.Duration(); got != 90*time.Minute {
		t.Errorf("Duration = %v, want 90m", got)
	}
	if len(event.Attendees) != 2 {
		t.Fatalf("Attendees = %d, want 2", len(event.Attendees))
	}
	if event.Attendees[0].Name != "Bob Smith" || event.Attendees[0].Email != "bob@company.com" {
		t.Errorf("attendee[0] = %+v", event.Attendees[0])
	}
	if event.Attendees[0].PartStat != PartStatNeedsAction {
		t.Errorf("attendee[0] PARTSTAT = %q", event.Attendees[0].PartStat)
	}
}

func TestParse_MultiEvent(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//T//EN\r\nMETHOD:PUBLISH\r\n" +
		"BEGIN:VEVENT\r\nUID:a@x\r\nDTSTAMP:20260101T000000Z\r\nDTSTART:20260101T100000Z\r\nSUMMARY:A\r\nEND:VEVENT\r\n" +
		"BEGIN:VEVENT\r\nUID:b@x\r\nDTSTAMP:20260101T000000Z\r\nDTSTART:20260102T100000Z\r\nSUMMARY:B\r\nEND:VEVENT\r\n" +
		"END:VCALENDAR\r\n")

	cal, err := Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if cal.Method != "PUBLISH" {
		t.Errorf("Method = %q", cal.Method)
	}
	if len(cal.Events) != 2 {
		t.Fatalf("Events = %d, want 2", len(cal.Events))
	}
	if cal.Events[1].Summary != "B" {
		t.Errorf("Events[1].Summary = %q", cal.Events[1].Summary)
	}
}

func TestParseICS_NoEvent(t *testing.T) {
	_, err := ParseICS([]byte("BEGIN:VCALENDAR\nVERSION:2.0\nPRODID:-//T//T//EN\nEND:VCALENDAR"))
	if err == nil || !strings.Contains(err.Error(), "no VEVENT") {
		t.Errorf("want 'no VEVENT' error, got %v", err)
	}
}

func TestParseICS_Malformed(t *testing.T) {
	if _, err := ParseICS([]byte("INVALID ICAL DATA")); err == nil {
		t.Error("want error for malformed data")
	}
}

// buildICS wraps DTSTART/DTEND lines into a minimal VCALENDAR.
func buildICS(dtstart, dtend string) []byte {
	return []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//Test//Test//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:date-only@example.com\r\nDTSTAMP:20260415T120000Z\r\n" +
		dtstart + "\r\n" + dtend + "\r\nSUMMARY:Test\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n")
}

func TestParseICS_DateOnly(t *testing.T) {
	wantStart := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	wantEnd := time.Date(2026, 4, 22, 0, 0, 0, 0, time.UTC)

	tests := []struct{ name, dtstart, dtend string }{
		{"VALUE=DATE without TZID", "DTSTART;VALUE=DATE:20260421", "DTEND;VALUE=DATE:20260422"},
		{"VALUE=DATE with TZID is ignored", "DTSTART;TZID=America/New_York;VALUE=DATE:20260421", "DTEND;TZID=America/New_York;VALUE=DATE:20260422"},
		{"YYYYMMDD shape with TZID treated as date-only", "DTSTART;TZID=America/Los_Angeles:20260421", "DTEND;TZID=America/Los_Angeles:20260422"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event, err := ParseICS(buildICS(tt.dtstart, tt.dtend))
			if err != nil {
				t.Fatalf("ParseICS: %v", err)
			}
			if !event.Start.Equal(wantStart) {
				t.Errorf("Start = %v, want %v", event.Start.UTC(), wantStart)
			}
			if !event.End.Equal(wantEnd) {
				t.Errorf("End = %v, want %v", event.End.UTC(), wantEnd)
			}
			if !event.AllDay {
				t.Error("AllDay = false, want true")
			}
		})
	}
}

func TestParseICS_TimedWithTZID(t *testing.T) {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		t.Skipf("America/New_York unavailable: %v", err)
	}
	event, err := ParseICS(buildICS(
		"DTSTART;TZID=America/New_York:20260421T140000",
		"DTEND;TZID=America/New_York:20260421T153000",
	))
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}
	if want := time.Date(2026, 4, 21, 14, 0, 0, 0, loc); !event.Start.Equal(want) {
		t.Errorf("Start = %v, want %v", event.Start, want)
	}
}

func TestParseICS_DurationDerivesEnd(t *testing.T) {
	data := []byte("BEGIN:VCALENDAR\r\nVERSION:2.0\r\nPRODID:-//T//T//EN\r\n" +
		"BEGIN:VEVENT\r\nUID:d@x\r\nDTSTAMP:20260101T000000Z\r\nDTSTART:20260101T100000Z\r\n" +
		"DURATION:PT1H30M\r\nSUMMARY:Dur\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n")
	event, err := ParseICS(data)
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}
	if want := time.Date(2026, 1, 1, 11, 30, 0, 0, time.UTC); !event.End.Equal(want) {
		t.Errorf("End = %v, want %v", event.End, want)
	}
}

func TestExtractEmail(t *testing.T) {
	tests := []struct{ in, want string }{
		{"mailto:user@example.com", "user@example.com"},
		{"MAILTO:user@example.com", "user@example.com"},
		{"CN=John Doe:user@example.com", "user@example.com"},
		{"user@example.com", "user@example.com"},
		{"  user@example.com  ", "user@example.com"},
	}
	for _, tt := range tests {
		if got := extractEmail(tt.in); got != tt.want {
			t.Errorf("extractEmail(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestParseICalDuration(t *testing.T) {
	tests := []struct {
		in   string
		want time.Duration
		ok   bool
	}{
		{"PT1H30M", 90 * time.Minute, true},
		{"P2D", 48 * time.Hour, true},
		{"PT15M", 15 * time.Minute, true},
		{"-PT15M", -15 * time.Minute, true},
		{"P1W", 7 * 24 * time.Hour, true},
		{"P1M", 0, false}, // months aren't a fixed duration
		{"junk", 0, false},
	}
	for _, tt := range tests {
		got, ok := parseICalDuration(tt.in)
		if ok != tt.ok || (ok && got != tt.want) {
			t.Errorf("parseICalDuration(%q) = %v,%v want %v,%v", tt.in, got, ok, tt.want, tt.ok)
		}
	}
}
