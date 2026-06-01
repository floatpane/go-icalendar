package icalendar

import (
	"os"
	"strings"
	"testing"
)

func TestGenerateRSVP(t *testing.T) {
	data, err := os.ReadFile("testdata/simple.ics")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	for _, response := range []string{"ACCEPTED", "DECLINED", "TENTATIVE"} {
		t.Run(response, func(t *testing.T) {
			out, err := GenerateRSVP(data, "bob@company.com", response)
			if err != nil {
				t.Fatalf("GenerateRSVP(%s): %v", response, err)
			}
			s := string(out)

			if !strings.Contains(s, "METHOD:REPLY") {
				t.Error("missing METHOD:REPLY")
			}
			if !strings.Contains(s, "PARTSTAT="+response) {
				t.Errorf("missing PARTSTAT=%s", response)
			}
			if n := strings.Count(s, "ATTENDEE"); n != 1 {
				t.Errorf("ATTENDEE count = %d, want 1", n)
			}
			if !strings.Contains(s, "bob@company.com") {
				t.Error("missing responder email")
			}
			if strings.Contains(s, "carol@company.com") {
				t.Error("other attendee leaked into reply")
			}
			if _, err := ParseICS(out); err != nil {
				t.Errorf("reply not valid iCalendar: %v", err)
			}
		})
	}
}

func TestEventReply(t *testing.T) {
	data, _ := os.ReadFile("testdata/simple.ics")
	ev, err := ParseICS(data)
	if err != nil {
		t.Fatalf("ParseICS: %v", err)
	}

	cal := ev.Reply("bob@company.com", PartStatAccepted)
	if cal.Method != string(MethodReply) {
		t.Errorf("Method = %q", cal.Method)
	}
	out, err := cal.Serialize()
	if err != nil {
		t.Fatalf("Serialize: %v", err)
	}
	s := string(out)
	if !strings.Contains(s, "METHOD:REPLY") {
		t.Error("missing METHOD:REPLY")
	}
	if !strings.Contains(s, "PARTSTAT=ACCEPTED") {
		t.Error("missing PARTSTAT=ACCEPTED")
	}
	if !strings.Contains(s, "bob@company.com") {
		t.Error("missing responder")
	}
	if strings.Contains(s, "carol@company.com") {
		t.Error("other attendee leaked")
	}
	// UID preserved so the organizer can match the reply.
	if !strings.Contains(s, "test-event-123@example.com") {
		t.Error("UID not preserved in reply")
	}
}
