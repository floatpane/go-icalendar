<div align="center">

# go-icalendar

**Ergonomic iCalendar (RFC 5545) and iMIP (RFC 6047) for Go. Parse invites, build REQUEST/REPLY/CANCEL, expand RRULE recurrences, compute free/busy.**

[![Go Version](https://img.shields.io/github/go-mod/go-version/floatpane/go-icalendar)](https://golang.org)
[![Go Reference](https://pkg.go.dev/badge/github.com/floatpane/go-icalendar.svg)](https://pkg.go.dev/github.com/floatpane/go-icalendar)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/floatpane/go-icalendar)](https://github.com/floatpane/go-icalendar/releases)
[![CI](https://github.com/floatpane/go-icalendar/actions/workflows/ci.yml/badge.svg)](https://github.com/floatpane/go-icalendar/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

`go-icalendar` turns the low-level property bag of an `.ics` file into a flat,
render-ready `Event` struct, and provides the scheduling glue an email client
otherwise writes by hand: parsing invites, building outgoing REQUEST/CANCEL
messages, generating the fiddly iMIP replies Google Calendar and Outlook
require, expanding recurrence rules, and computing availability.

It was extracted from [matcha](https://github.com/floatpane/matcha)'s mail
reader, where it powers the meeting-invite card and Accept / Decline / Tentative
replies.

## Features

- **Flat `Event`.** Summary, times, organizer, attendees, status, recurrence —
  all on one struct, no property lookups at the call site.
- **Parse one or many.** `ParseICS` for the first VEVENT (what mail clients
  want), `Parse` for the whole calendar.
- **Correct timestamps.** UTC, floating, `TZID`-qualified and `VALUE=DATE`
  all-day values — including the real-world quirk of a `TZID` wrongly attached to
  a date-only value, which is ignored rather than silently shifting the day.
- **iMIP-correct replies.** `GenerateRSVP` reproduces exactly what schedulers
  expect: `METHOD:REPLY`, only the responding attendee, updated `PARTSTAT`,
  `RSVP=TRUE`, fresh `DTSTAMP`, and a preserved UID.
- **Build outgoing invites.** `NewRequest` / `NewCancel` / `Event.Reply` →
  `Serialize()` to RFC-compliant bytes.
- **A real recurrence engine.** DAILY/WEEKLY/MONTHLY/YEARLY with INTERVAL,
  COUNT/UNTIL, BYMONTH, BYMONTHDAY, BYDAY (ordinals like `2MO` / `-1FR`) and
  BYSETPOS, plus RDATE/EXDATE merging.
- **Free/busy.** Merge events (recurrences expanded) into busy intervals and the
  free gaps between them.
- **Single dependency.** Only `github.com/arran4/golang-ical`.

## Install

```bash
go get github.com/floatpane/go-icalendar
```

Requires Go 1.26+.

## Usage

### Parse an invite

```go
package main

import (
    "fmt"
    "log"
    "os"

    icalendar "github.com/floatpane/go-icalendar"
)

func main() {
    data, err := os.ReadFile("invite.ics")
    if err != nil {
        log.Fatal(err)
    }

    ev, err := icalendar.ParseICS(data)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(ev.Summary, ev.Start.Local(), "->", ev.End.Local())
    for _, a := range ev.Attendees {
        fmt.Printf("  %s (%s)\n", a.Email, a.PartStat)
    }
}
```

### Reply (RSVP)

```go
// response: "ACCEPTED", "DECLINED", "TENTATIVE"
reply, err := icalendar.GenerateRSVP(data, "me@example.com", "ACCEPTED")
// Send reply as text/calendar; method=REPLY back to the organizer.
```

### Build and send an invite

```go
start := time.Date(2026, 5, 1, 9, 0, 0, 0, time.UTC)
ev := &icalendar.Event{
    UID:       "kickoff@example.com",
    Summary:   "Project kickoff",
    Location:  "Room 1",
    Start:     start,
    End:       start.Add(time.Hour),
    Organizer: "me@example.com",
    Attendees: []icalendar.Attendee{
        {Email: "you@example.com", PartStat: icalendar.PartStatNeedsAction, RSVP: true},
    },
}
ics, err := icalendar.NewRequest(ev).Serialize()

// Later — call it off (bumps SEQUENCE, sets STATUS:CANCELLED):
cancelICS, err := icalendar.NewCancel(ev).Serialize()
```

### Expand a recurring event

```go
ev, _ := icalendar.ParseICS(data)
for _, t := range ev.Occurrences(time.Now(), time.Now().AddDate(0, 1, 0), 0) {
    fmt.Println(t.Local())
}

// Or work with a rule directly:
r, _ := icalendar.ParseRRule("FREQ=MONTHLY;BYDAY=-1FR") // last Friday monthly
times := r.Between(start, from, to, 0)
```

### Compute free/busy

```go
busy, free := icalendar.FreeBusy(events, from, to)
for _, gap := range free {
    if gap.Duration() >= 30*time.Minute {
        fmt.Println("can meet at", gap.Start)
        break
    }
}
```

## Supported RRULE parts

| Part | Status |
|------|--------|
| `FREQ` (SECONDLY → YEARLY), `INTERVAL`, `COUNT`, `UNTIL` | ✅ |
| `BYMONTH`, `BYMONTHDAY` (incl. negatives), `BYDAY` (incl. ordinals), `BYSETPOS`, `WKST` | ✅ |
| `BYWEEKNO`, `BYYEARDAY` | parsed, ignored during expansion |

## Documentation

Full API reference: [pkg.go.dev/github.com/floatpane/go-icalendar](https://pkg.go.dev/github.com/floatpane/go-icalendar)

Guides and diagrams: see [`docs/`](docs/).

## Sister projects

| Project | Role |
|---------|------|
| [floatpane/matcha](https://github.com/floatpane/matcha) | Reference consumer — renders invites and sends RSVPs. |
| [floatpane/go-secretbox](https://github.com/floatpane/go-secretbox) | Sibling extraction — password-based encryption at rest. |

## Contributing

PRs welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).

## Security

Report vulnerabilities privately via [SECURITY.md](SECURITY.md).

## License

MIT. See [LICENSE](LICENSE).
