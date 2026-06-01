package icalendar

import "time"

// Method is an iTIP/iMIP method (RFC 5546) declared at the VCALENDAR level.
type Method string

const (
	MethodRequest Method = "REQUEST"
	MethodReply   Method = "REPLY"
	MethodCancel  Method = "CANCEL"
	MethodPublish Method = "PUBLISH"
	MethodRefresh Method = "REFRESH"
	MethodCounter Method = "COUNTER"
)

// Status is a VEVENT status (RFC 5545 STATUS).
type Status string

const (
	StatusConfirmed Status = "CONFIRMED"
	StatusTentative Status = "TENTATIVE"
	StatusCancelled Status = "CANCELLED"
)

// PartStat is an attendee participation status (PARTSTAT). The Accepted /
// Declined / Tentative values are also the legal responses to [GenerateRSVP]
// and [Event.Reply].
type PartStat string

const (
	PartStatNeedsAction PartStat = "NEEDS-ACTION"
	PartStatAccepted    PartStat = "ACCEPTED"
	PartStatDeclined    PartStat = "DECLINED"
	PartStatTentative   PartStat = "TENTATIVE"
	PartStatDelegated   PartStat = "DELEGATED"
)

// Attendee is a single ATTENDEE on a VEVENT.
type Attendee struct {
	Email    string   // address, without the "mailto:" scheme
	Name     string   // CN parameter, if any
	Role     string   // ROLE parameter (e.g. REQ-PARTICIPANT)
	PartStat PartStat // PARTSTAT parameter
	RSVP     bool     // RSVP parameter
}

// Event is a parsed (or hand-built) VEVENT, flattened for easy rendering.
//
// Status and Method are kept as plain strings (rather than the typed [Status]
// and [Method]) because they are read straight from the wire and may carry
// values an older or non-conforming producer emitted; compare them against the
// typed constants with string conversion, e.g. ev.Status == string(StatusConfirmed).
type Event struct {
	UID         string
	Summary     string // event title (SUMMARY)
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool // DTSTART/DTEND were VALUE=DATE (no time component)

	Organizer     string // organizer email
	OrganizerName string // organizer CN, if any
	Attendees     []Attendee

	Status     string // CONFIRMED, TENTATIVE, CANCELLED
	Method     string // REQUEST, REPLY, CANCEL (mirrors the calendar METHOD)
	Sequence   int    // SEQUENCE; bump on every update to an existing UID
	URL        string
	Categories []string

	Stamp    time.Time // DTSTAMP
	Created  time.Time // CREATED
	Modified time.Time // LAST-MODIFIED

	// Recurrence is the parsed RRULE, or nil for a one-off event.
	Recurrence *RRule
	// RDates and ExDates are explicit additional / excluded occurrence starts.
	RDates  []time.Time
	ExDates []time.Time
}

// Duration is the wall-clock length of the event. It is zero when either
// endpoint is unset.
func (e *Event) Duration() time.Duration {
	if e.Start.IsZero() || e.End.IsZero() {
		return 0
	}
	return e.End.Sub(e.Start)
}

// IsRecurring reports whether the event carries an RRULE or any RDATE.
func (e *Event) IsRecurring() bool {
	return e.Recurrence != nil || len(e.RDates) > 0
}
