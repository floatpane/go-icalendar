# Security Policy

## Supported Versions

Only the latest release of go-icalendar is supported with security updates.

## Reporting a Vulnerability

If you discover a security vulnerability in go-icalendar, please report it responsibly. **Do not open a public issue.**

Email us at [us@floatpane.com](mailto:us@floatpane.com) with:

- A description of the vulnerability
- Steps to reproduce the issue
- The potential impact
- Any suggested fixes (optional)

We will acknowledge your report within 48 hours and aim to provide a fix or mitigation plan within 7 days, depending on severity.

## Scope

This policy covers the go-icalendar codebase and its official releases.

Since this library parses untrusted `.ics` data straight off email attachments, of particular interest:

- **Parser denial-of-service** — crafted calendar input that triggers panics, unbounded memory growth, or pathological CPU use in `Parse` / `ParseICS`.
- **Recurrence blowups** — an `RRULE` that causes runaway expansion in `Occurrences` / `Between` despite the iteration cap, or that bypasses it.
- **Injection into generated output** — attendee, summary or other user-controlled fields that break out of their property in `Serialize`, producing forged headers or smuggled components.
- **Reply confusion** — `GenerateRSVP` matching the wrong attendee, leaking other attendees into a reply, or mis-setting `PARTSTAT`/`METHOD`.

Third-party dependencies (notably `github.com/arran4/golang-ical`) are outside our direct control, but we will work to address reported issues in them as quickly as possible.

## Disclosure

We ask that you give us reasonable time to address the issue before disclosing it publicly. We are committed to crediting reporters in release notes (unless you prefer to remain anonymous).
