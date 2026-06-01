package icalendar

import (
	"bytes"
	"fmt"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

// GenerateRSVP turns a received invite (originalData) into an RFC 6047 (iMIP)
// reply on behalf of userEmail. response must be one of "ACCEPTED", "DECLINED"
// or "TENTATIVE" (the [PartStatAccepted]/[PartStatDeclined]/[PartStatTentative]
// values).
//
// It reproduces exactly what Google Calendar and Outlook expect from a reply:
//
//   - METHOD:REPLY at the calendar level;
//   - only the responding attendee left in each VEVENT (all others removed);
//   - that attendee's PARTSTAT set to response and RSVP=TRUE;
//   - a fresh DTSTAMP.
//
// Working from the original bytes (rather than a re-serialized [Event]) keeps
// the UID, SEQUENCE, recurrence and organizer of the invite byte-for-byte, which
// is what lets the organizer's calendar match the reply to the original event.
func GenerateRSVP(originalData []byte, userEmail, response string) ([]byte, error) {
	cal, err := ics.ParseCalendar(bytes.NewReader(originalData))
	if err != nil {
		return nil, fmt.Errorf("parse calendar: %w", err)
	}

	cal.SetMethod(ics.MethodReply)
	userEmail = strings.ToLower(strings.TrimSpace(userEmail))

	for _, vevent := range cal.Events() {
		vevent.SetDtStampTime(time.Now().UTC())

		// Find the responding attendee among the originals.
		var matched *ics.Attendee
		for _, attendee := range vevent.Attendees() {
			ae := strings.ToLower(extractEmail(attendee.Email()))
			if strings.Contains(ae, userEmail) || strings.Contains(userEmail, ae) {
				matched = attendee
				break
			}
		}

		// Drop every ATTENDEE, then re-add only the responder.
		vevent.RemoveProperty(ics.ComponentPropertyAttendee)

		if matched != nil {
			matched.ICalParameters[string(ics.ParameterParticipationStatus)] = []string{response}
			matched.ICalParameters["RSVP"] = []string{"TRUE"}
			vevent.Properties = append(vevent.Properties, matched.IANAProperty)
		} else {
			// Responder wasn't listed — add ourselves outright.
			vevent.AddAttendee("mailto:"+userEmail,
				ics.WithRSVP(true),
				ics.ParticipationStatusNeedsAction,
				ics.CalendarUserTypeIndividual,
				ics.ParticipationRoleReqParticipant,
			)
			for _, att := range vevent.Attendees() {
				att.ICalParameters[string(ics.ParameterParticipationStatus)] = []string{response}
			}
		}
	}

	return []byte(cal.Serialize()), nil
}

// Reply builds a METHOD:REPLY calendar for this event on behalf of userEmail,
// from the parsed [Event] rather than the original bytes. Prefer [GenerateRSVP]
// when you still hold the invite's raw .ics; use Reply when you only have an
// Event in hand. status is the responder's [PartStat] (typically Accepted,
// Declined or Tentative).
func (e *Event) Reply(userEmail string, status PartStat) *Calendar {
	userEmail = strings.ToLower(strings.TrimSpace(userEmail))

	reply := *e // shallow copy; we replace the attendee slice below
	reply.Method = string(MethodReply)
	reply.Stamp = time.Now().UTC()

	var name string
	for _, a := range e.Attendees {
		if strings.EqualFold(extractEmail(a.Email), userEmail) {
			name = a.Name
			break
		}
	}
	reply.Attendees = []Attendee{{
		Email:    userEmail,
		Name:     name,
		PartStat: status,
		RSVP:     true,
	}}

	return methodCalendar(MethodReply, &reply)
}
