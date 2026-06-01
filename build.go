package icalendar

import (
	"fmt"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

// defaultProdID identifies this library as the producer in generated calendars.
const defaultProdID = "-//floatpane//go-icalendar//EN"

// NewCalendar returns an empty PUBLISH calendar ready for [Calendar.Add].
func NewCalendar() *Calendar {
	return &Calendar{
		Method:  string(MethodPublish),
		ProdID:  defaultProdID,
		Version: "2.0",
	}
}

// Add appends events to the calendar and returns it, for chaining.
func (c *Calendar) Add(events ...*Event) *Calendar {
	c.Events = append(c.Events, events...)
	return c
}

// NewRequest builds a METHOD:REQUEST calendar inviting attendees to e — the
// message an organizer sends to schedule a meeting. The event's Status defaults
// to CONFIRMED when unset.
func NewRequest(e *Event) *Calendar {
	if e.Status == "" {
		e.Status = string(StatusConfirmed)
	}
	return methodCalendar(MethodRequest, e)
}

// NewCancel builds a METHOD:CANCEL calendar for e, marking it CANCELLED and
// bumping its SEQUENCE so recipients treat it as a newer update than the
// original invite (RFC 5546 §3.2.5).
func NewCancel(e *Event) *Calendar {
	e.Status = string(StatusCancelled)
	e.Sequence++
	return methodCalendar(MethodCancel, e)
}

func methodCalendar(m Method, e *Event) *Calendar {
	return &Calendar{
		Method:  string(m),
		ProdID:  defaultProdID,
		Version: "2.0",
		Events:  []*Event{e},
	}
}

// Serialize renders the calendar as RFC 5545 bytes (CRLF-folded). It fills in
// sensible defaults for an empty ProdID/Version, and stamps any event missing a
// DTSTAMP with the current time.
func (c *Calendar) Serialize() ([]byte, error) {
	cal := ics.NewCalendar()
	cal.SetProductId(orDefault(c.ProdID, defaultProdID))
	cal.SetVersion(orDefault(c.Version, "2.0"))
	if c.Method != "" {
		cal.SetMethod(ics.Method(c.Method))
	}
	for _, e := range c.Events {
		if err := writeVEvent(cal, e); err != nil {
			return nil, err
		}
	}
	return []byte(cal.Serialize()), nil
}

// writeVEvent materializes one Event onto the ics calendar.
func writeVEvent(cal *ics.Calendar, e *Event) error {
	if e.UID == "" {
		return fmt.Errorf("event %q: UID is required", e.Summary)
	}
	ve := cal.AddEvent(e.UID)

	stamp := e.Stamp
	if stamp.IsZero() {
		stamp = time.Now().UTC()
	}
	ve.SetDtStampTime(stamp)

	if !e.Start.IsZero() {
		if e.AllDay {
			ve.SetAllDayStartAt(e.Start)
		} else {
			ve.SetStartAt(e.Start)
		}
	}
	if !e.End.IsZero() {
		if e.AllDay {
			ve.SetAllDayEndAt(e.End)
		} else {
			ve.SetEndAt(e.End)
		}
	}

	if e.Summary != "" {
		ve.SetSummary(e.Summary)
	}
	if e.Description != "" {
		ve.SetDescription(e.Description)
	}
	if e.Location != "" {
		ve.SetLocation(e.Location)
	}
	if e.URL != "" {
		ve.SetURL(e.URL)
	}
	if e.Status != "" {
		ve.SetStatus(ics.ObjectStatus(strings.ToUpper(e.Status)))
	}
	if e.Sequence > 0 {
		ve.SetSequence(e.Sequence)
	}
	if len(e.Categories) > 0 {
		ve.SetProperty(ics.ComponentPropertyCategories, strings.Join(e.Categories, ","))
	}
	if !e.Created.IsZero() {
		ve.SetCreatedTime(e.Created)
	}
	if !e.Modified.IsZero() {
		ve.SetModifiedAt(e.Modified)
	}

	if e.Organizer != "" {
		if e.OrganizerName != "" {
			ve.SetOrganizer("mailto:"+e.Organizer, ics.WithCN(e.OrganizerName))
		} else {
			ve.SetOrganizer("mailto:" + e.Organizer)
		}
	}
	for _, a := range e.Attendees {
		var params []ics.PropertyParameter
		if a.Name != "" {
			params = append(params, ics.WithCN(a.Name))
		}
		if a.Role != "" {
			params = append(params, &ics.KeyValues{Key: "ROLE", Value: []string{a.Role}})
		}
		if a.PartStat != "" {
			params = append(params, &ics.KeyValues{Key: "PARTSTAT", Value: []string{string(a.PartStat)}})
		}
		if a.RSVP {
			params = append(params, ics.WithRSVP(true))
		}
		ve.AddAttendee("mailto:"+a.Email, params...)
	}

	if e.Recurrence != nil {
		ve.AddRrule(e.Recurrence.String())
	}
	for _, t := range e.RDates {
		ve.AddRdate(t.UTC().Format("20060102T150405Z"))
	}
	for _, t := range e.ExDates {
		ve.SetProperty(ics.ComponentPropertyExdate, t.UTC().Format("20060102T150405Z"))
	}
	return nil
}

func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}
