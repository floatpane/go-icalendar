package icalendar

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

// Calendar is a parsed (or constructed) VCALENDAR: a method, some bookkeeping
// headers, and the events it carries.
type Calendar struct {
	Method  string // METHOD (REQUEST, REPLY, CANCEL, ...)
	ProdID  string // PRODID
	Version string // VERSION (almost always "2.0")
	Events  []*Event
}

// Parse reads .ics data and returns every VEVENT it contains, along with the
// calendar-level method and headers. Unlike [ParseICS] it does not require any
// events to be present — an empty calendar yields a Calendar with no Events and
// a nil error.
func Parse(data []byte) (*Calendar, error) {
	cal, err := ics.ParseCalendar(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("parse calendar: %w", err)
	}

	out := &Calendar{
		Method:  calProp(cal, ics.PropertyMethod),
		ProdID:  calProp(cal, ics.PropertyProductId),
		Version: calProp(cal, ics.PropertyVersion),
	}
	for _, ve := range cal.Events() {
		out.Events = append(out.Events, fromVEvent(ve, out.Method))
	}
	return out, nil
}

// ParseICS extracts the first VEVENT from .ics data. It is the convenience
// entry point for mail clients, which overwhelmingly deal with single-event
// invites; use [Parse] when a calendar may hold several events.
func ParseICS(data []byte) (*Event, error) {
	cal, err := Parse(data)
	if err != nil {
		return nil, err
	}
	if len(cal.Events) == 0 {
		return nil, fmt.Errorf("no VEVENT found")
	}
	return cal.Events[0], nil
}

// fromVEvent flattens a parser VEVENT into our Event. method is the
// calendar-level METHOD, copied onto every event for convenience.
func fromVEvent(ve *ics.VEvent, method string) *Event {
	e := &Event{
		UID:         prop(ve, ics.ComponentPropertyUniqueId),
		Summary:     prop(ve, ics.ComponentPropertySummary),
		Description: prop(ve, ics.ComponentPropertyDescription),
		Location:    prop(ve, ics.ComponentPropertyLocation),
		URL:         prop(ve, ics.ComponentPropertyUrl),
		Status:      prop(ve, ics.ComponentPropertyStatus),
		Method:      method,
	}

	org := ve.GetProperty(ics.ComponentPropertyOrganizer)
	if org != nil {
		e.Organizer = extractEmail(org.Value)
		if cn := org.ICalParameters["CN"]; len(cn) > 0 {
			e.OrganizerName = cn[0]
		}
	}

	if seq := prop(ve, ics.ComponentPropertySequence); seq != "" {
		e.Sequence, _ = strconv.Atoi(seq)
	}
	if cats := prop(ve, ics.ComponentPropertyCategories); cats != "" {
		e.Categories = splitCommaList(cats)
	}

	e.Start, e.AllDay, _ = parseTimestamp(ve, ics.ComponentPropertyDtStart)
	var endAllDay bool
	e.End, endAllDay, _ = parseTimestamp(ve, ics.ComponentPropertyDtEnd)
	e.AllDay = e.AllDay || endAllDay

	// If DTEND is absent, derive it from DURATION, falling back to a zero-length
	// (all-day: one-day) event — RFC 5545 §3.6.1.
	if e.End.IsZero() && !e.Start.IsZero() {
		if d := prop(ve, ics.ComponentPropertyDuration); d != "" {
			if dur, ok := parseICalDuration(d); ok {
				e.End = e.Start.Add(dur)
			}
		}
		if e.End.IsZero() && e.AllDay {
			e.End = e.Start.AddDate(0, 0, 1)
		}
	}

	e.Stamp, _, _ = parseTimestamp(ve, ics.ComponentPropertyDtstamp)
	e.Created, _, _ = parseTimestamp(ve, ics.ComponentPropertyCreated)
	e.Modified, _, _ = parseTimestamp(ve, ics.ComponentPropertyLastModified)

	for _, att := range ve.Attendees() {
		a := Attendee{
			Email:    extractEmail(att.Email()),
			Name:     firstParam(att.ICalParameters, "CN"),
			Role:     firstParam(att.ICalParameters, "ROLE"),
			PartStat: PartStat(firstParam(att.ICalParameters, "PARTSTAT")),
			RSVP:     strings.EqualFold(firstParam(att.ICalParameters, "RSVP"), "TRUE"),
		}
		e.Attendees = append(e.Attendees, a)
	}

	if rr := prop(ve, ics.ComponentPropertyRrule); rr != "" {
		if parsed, err := ParseRRule(rr); err == nil {
			e.Recurrence = parsed
		}
	}
	e.RDates = multiTimestamp(ve, ics.ComponentPropertyRdate)
	e.ExDates = multiTimestamp(ve, ics.ComponentPropertyExdate)

	return e
}

// prop returns a VEVENT property value, or "" when absent.
func prop(ve *ics.VEvent, p ics.ComponentProperty) string {
	if got := ve.GetProperty(p); got != nil {
		return got.Value
	}
	return ""
}

// calProp returns a calendar-level property value, or "".
func calProp(cal *ics.Calendar, p ics.Property) string {
	for _, cp := range cal.CalendarProperties {
		if cp.IANAToken == string(p) {
			return cp.Value
		}
	}
	return ""
}

// multiTimestamp parses every value of a possibly-repeated date property
// (RDATE / EXDATE), each of which may itself be a comma-separated list.
func multiTimestamp(ve *ics.VEvent, p ics.ComponentProperty) []time.Time {
	var out []time.Time
	for _, prop := range ve.GetProperties(p) {
		tzid := firstParam(prop.ICalParameters, "TZID")
		for _, v := range splitCommaList(prop.Value) {
			if t, ok := parseValue(v, tzid, false); ok {
				out = append(out, t)
			}
		}
	}
	return out
}

// parseTimestamp parses a single date/date-time property (DTSTART, DTEND, ...),
// honoring TZID and VALUE=DATE. The bool reports whether the value was date-only.
func parseTimestamp(ve *ics.VEvent, p ics.ComponentProperty) (time.Time, bool, error) {
	got := ve.GetProperty(p)
	if got == nil {
		return time.Time{}, false, fmt.Errorf("property %s not found", p)
	}
	value := got.Value
	tzid := firstParam(got.ICalParameters, "TZID")

	dateOnly := strings.EqualFold(firstParam(got.ICalParameters, "VALUE"), "DATE")
	// RFC 5545 DATE form is YYYYMMDD (8 chars, no "T").
	if !dateOnly && len(value) == 8 && !strings.ContainsAny(value, "T") {
		dateOnly = true
	}

	t, ok := parseValue(value, tzid, dateOnly)
	if !ok {
		return time.Time{}, dateOnly, fmt.Errorf("parse timestamp %q", value)
	}
	return t, dateOnly, nil
}

// RFC 5545 date/date-time layouts, tried in order.
var icalLayouts = []string{
	"20060102T150405Z", // UTC
	"20060102T150405",  // floating / TZID-qualified
	"20060102",         // date only
	time.RFC3339,       // lenient fallback
}

// parseValue parses one RFC 5545 date/date-time string. For date-only values
// any TZID is ignored (RFC 5545 forbids it on VALUE=DATE; some producers emit
// it anyway). Timed values with a TZID are placed in that zone.
func parseValue(value, tzid string, dateOnly bool) (time.Time, bool) {
	var t time.Time
	var err error
	for _, layout := range icalLayouts {
		if t, err = time.Parse(layout, value); err == nil {
			break
		}
	}
	if err != nil {
		return time.Time{}, false
	}
	if tzid != "" && !dateOnly && !strings.HasSuffix(value, "Z") {
		if loc, locErr := time.LoadLocation(tzid); locErr == nil {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
		}
	}
	return t, true
}

// parseICalDuration parses an RFC 5545 DURATION (e.g. "PT1H30M", "P2D", "-PT15M").
func parseICalDuration(s string) (time.Duration, bool) {
	neg := false
	if strings.HasPrefix(s, "-") {
		neg, s = true, s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	if !strings.HasPrefix(s, "P") {
		return 0, false
	}
	s = s[1:]

	var total time.Duration
	inTime := false
	num := ""
	for _, r := range s {
		switch {
		case r >= '0' && r <= '9':
			num += string(r)
		case r == 'T':
			inTime = true
		default:
			n, err := strconv.Atoi(num)
			if err != nil {
				return 0, false
			}
			num = ""
			switch r {
			case 'W':
				total += time.Duration(n) * 7 * 24 * time.Hour
			case 'D':
				total += time.Duration(n) * 24 * time.Hour
			case 'H':
				total += time.Duration(n) * time.Hour
			case 'M':
				if inTime {
					total += time.Duration(n) * time.Minute
				} else {
					return 0, false // months are not a fixed duration
				}
			case 'S':
				total += time.Duration(n) * time.Second
			default:
				return 0, false
			}
		}
	}
	if num != "" {
		return 0, false // trailing number with no unit
	}
	if neg {
		total = -total
	}
	return total, true
}

// extractEmail strips a "mailto:" scheme and any "CN=Name:" prefix from an
// organizer/attendee value, returning the bare address.
func extractEmail(mailto string) string {
	email := strings.TrimPrefix(mailto, "mailto:")
	email = strings.TrimPrefix(email, "MAILTO:")
	if idx := strings.Index(email, ":"); idx != -1 {
		email = email[idx+1:]
	}
	return strings.TrimSpace(email)
}

// firstParam returns the first value of an iCal parameter, or "".
func firstParam(params map[string][]string, key string) string {
	if vs := params[key]; len(vs) > 0 {
		return vs[0]
	}
	return ""
}

// splitCommaList splits a comma-separated value and trims each element,
// dropping empties.
func splitCommaList(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}
