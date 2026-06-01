// Package icalendar is an ergonomic wrapper around the iCalendar (RFC 5545)
// and iMIP/iTIP (RFC 6047 / RFC 5546) formats for Go email clients and
// schedulers.
//
// It turns the low-level property bag exposed by the underlying parser
// (github.com/arran4/golang-ical) into a flat, easy-to-render [Event] struct,
// and provides the glue most mail clients end up writing by hand:
//
//   - Parse an .ics attachment into one [Event] ([ParseICS]) or a whole
//     [Calendar] of them ([Parse]).
//   - Build outgoing invites: REQUEST ([NewRequest]), CANCEL ([NewCancel]) and
//     REPLY ([Event.Reply]) calendars, serialized back to RFC-compliant bytes.
//   - Generate an RFC 6047 RSVP reply from a received invite ([GenerateRSVP])
//     — the dance Google Calendar and Outlook require to register attendance.
//   - Expand recurring events: parse an RRULE ([ParseRRule]) and enumerate the
//     concrete occurrences in a window ([Event.Occurrences]).
//   - Compute availability: merge events into busy intervals and the free gaps
//     between them ([FreeBusy], [BusyPeriods]).
//
// The package name is icalendar and the import path is
// github.com/floatpane/go-icalendar.
package icalendar
