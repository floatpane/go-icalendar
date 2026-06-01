package icalendar

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Frequency is an RRULE FREQ value (RFC 5545 §3.3.10).
type Frequency string

const (
	Secondly Frequency = "SECONDLY"
	Minutely Frequency = "MINUTELY"
	Hourly   Frequency = "HOURLY"
	Daily    Frequency = "DAILY"
	Weekly   Frequency = "WEEKLY"
	Monthly  Frequency = "MONTHLY"
	Yearly   Frequency = "YEARLY"
)

// WeekDay is one BYDAY entry: a weekday with an optional ordinal. Ord is 0 for a
// plain weekday ("every Monday"), positive for the nth from the start of the
// period ("2MO" = second Monday), or negative for the nth from the end
// ("-1FR" = last Friday).
type WeekDay struct {
	Ord int
	Day time.Weekday
}

// RRule is a parsed RRULE recurrence (RFC 5545 §3.3.10). A zero Interval is
// treated as 1. Count and Until are mutually exclusive bounds; a zero value for
// each means "unbounded by that mechanism".
type RRule struct {
	Freq       Frequency
	Interval   int
	Count      int
	Until      time.Time
	ByMonth    []time.Month
	ByMonthDay []int // day-of-month; negative counts from the end (-1 = last)
	ByDay      []WeekDay
	ByHour     []int
	ByMinute   []int
	BySetPos   []int // 1-based position within a period; negative from the end
	WeekStart  time.Weekday
}

// maxRecurIterations caps recurrence expansion so a malformed or unbounded rule
// can't spin forever. It bounds the number of candidate periods examined.
const maxRecurIterations = 100_000

var weekdayCode = map[string]time.Weekday{
	"SU": time.Sunday, "MO": time.Monday, "TU": time.Tuesday, "WE": time.Wednesday,
	"TH": time.Thursday, "FR": time.Friday, "SA": time.Saturday,
}
var codeForWeekday = map[time.Weekday]string{
	time.Sunday: "SU", time.Monday: "MO", time.Tuesday: "TU", time.Wednesday: "WE",
	time.Thursday: "TH", time.Friday: "FR", time.Saturday: "SA",
}

// ParseRRule parses an RRULE value such as
// "FREQ=WEEKLY;INTERVAL=2;BYDAY=MO,WE;COUNT=10". A leading "RRULE:" prefix is
// tolerated. WeekStart defaults to Monday (RFC 5545's default).
func ParseRRule(s string) (*RRule, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "RRULE:")
	r := &RRule{Interval: 1, WeekStart: time.Monday}

	for _, part := range strings.Split(s, ";") {
		if part == "" {
			continue
		}
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			return nil, fmt.Errorf("rrule: malformed segment %q", part)
		}
		key := strings.ToUpper(strings.TrimSpace(k))
		val := strings.TrimSpace(v)

		switch key {
		case "FREQ":
			r.Freq = Frequency(strings.ToUpper(val))
		case "INTERVAL":
			n, err := strconv.Atoi(val)
			if err != nil || n < 1 {
				return nil, fmt.Errorf("rrule: bad INTERVAL %q", val)
			}
			r.Interval = n
		case "COUNT":
			n, err := strconv.Atoi(val)
			if err != nil || n < 0 {
				return nil, fmt.Errorf("rrule: bad COUNT %q", val)
			}
			r.Count = n
		case "UNTIL":
			t, ok := parseValue(val, "", len(val) == 8)
			if !ok {
				return nil, fmt.Errorf("rrule: bad UNTIL %q", val)
			}
			r.Until = t
		case "BYMONTH":
			for _, p := range splitCommaList(val) {
				n, err := strconv.Atoi(p)
				if err != nil || n < 1 || n > 12 {
					return nil, fmt.Errorf("rrule: bad BYMONTH %q", p)
				}
				r.ByMonth = append(r.ByMonth, time.Month(n))
			}
		case "BYMONTHDAY":
			ints, err := parseIntList(val)
			if err != nil {
				return nil, fmt.Errorf("rrule: bad BYMONTHDAY: %w", err)
			}
			r.ByMonthDay = ints
		case "BYHOUR":
			ints, err := parseIntList(val)
			if err != nil {
				return nil, fmt.Errorf("rrule: bad BYHOUR: %w", err)
			}
			r.ByHour = ints
		case "BYMINUTE":
			ints, err := parseIntList(val)
			if err != nil {
				return nil, fmt.Errorf("rrule: bad BYMINUTE: %w", err)
			}
			r.ByMinute = ints
		case "BYSETPOS":
			ints, err := parseIntList(val)
			if err != nil {
				return nil, fmt.Errorf("rrule: bad BYSETPOS: %w", err)
			}
			r.BySetPos = ints
		case "BYDAY":
			for _, p := range splitCommaList(val) {
				wd, err := parseWeekDay(p)
				if err != nil {
					return nil, err
				}
				r.ByDay = append(r.ByDay, wd)
			}
		case "WKST":
			wd, ok := weekdayCode[strings.ToUpper(val)]
			if !ok {
				return nil, fmt.Errorf("rrule: bad WKST %q", val)
			}
			r.WeekStart = wd
		default:
			// Unknown parts (e.g. BYWEEKNO, BYYEARDAY) are ignored rather than
			// rejected, so a rule we can't fully expand still round-trips.
		}
	}

	if r.Freq == "" {
		return nil, fmt.Errorf("rrule: missing FREQ")
	}
	return r, nil
}

func parseWeekDay(s string) (WeekDay, error) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) < 2 {
		return WeekDay{}, fmt.Errorf("rrule: bad BYDAY %q", s)
	}
	code := s[len(s)-2:]
	day, ok := weekdayCode[code]
	if !ok {
		return WeekDay{}, fmt.Errorf("rrule: bad BYDAY weekday %q", s)
	}
	ord := 0
	if prefix := s[:len(s)-2]; prefix != "" && prefix != "+" {
		n, err := strconv.Atoi(prefix)
		if err != nil {
			return WeekDay{}, fmt.Errorf("rrule: bad BYDAY ordinal %q", s)
		}
		ord = n
	}
	return WeekDay{Ord: ord, Day: day}, nil
}

func parseIntList(s string) ([]int, error) {
	var out []int
	for _, p := range splitCommaList(s) {
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("%q", p)
		}
		out = append(out, n)
	}
	return out, nil
}

// String renders the rule back to RFC 5545 form with a stable field order.
func (r *RRule) String() string {
	if r == nil {
		return ""
	}
	var b []string
	b = append(b, "FREQ="+string(r.Freq))
	if r.Interval > 1 {
		b = append(b, "INTERVAL="+strconv.Itoa(r.Interval))
	}
	if r.Count > 0 {
		b = append(b, "COUNT="+strconv.Itoa(r.Count))
	}
	if !r.Until.IsZero() {
		b = append(b, "UNTIL="+r.Until.UTC().Format("20060102T150405Z"))
	}
	if len(r.ByMonth) > 0 {
		months := make([]string, len(r.ByMonth))
		for i, m := range r.ByMonth {
			months[i] = strconv.Itoa(int(m))
		}
		b = append(b, "BYMONTH="+strings.Join(months, ","))
	}
	if len(r.ByMonthDay) > 0 {
		b = append(b, "BYMONTHDAY="+joinInts(r.ByMonthDay))
	}
	if len(r.ByDay) > 0 {
		days := make([]string, len(r.ByDay))
		for i, wd := range r.ByDay {
			if wd.Ord != 0 {
				days[i] = strconv.Itoa(wd.Ord) + codeForWeekday[wd.Day]
			} else {
				days[i] = codeForWeekday[wd.Day]
			}
		}
		b = append(b, "BYDAY="+strings.Join(days, ","))
	}
	if len(r.ByHour) > 0 {
		b = append(b, "BYHOUR="+joinInts(r.ByHour))
	}
	if len(r.ByMinute) > 0 {
		b = append(b, "BYMINUTE="+joinInts(r.ByMinute))
	}
	if len(r.BySetPos) > 0 {
		b = append(b, "BYSETPOS="+joinInts(r.BySetPos))
	}
	if r.WeekStart != time.Monday {
		b = append(b, "WKST="+codeForWeekday[r.WeekStart])
	}
	return strings.Join(b, ";")
}

func joinInts(ns []int) string {
	parts := make([]string, len(ns))
	for i, n := range ns {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

// Between enumerates occurrence start times of a rule anchored at dtstart,
// returning those in the half-open window [from, to). dtstart is always the
// first occurrence. If limit > 0, at most limit times are returned.
//
// Expansion covers the common, real-world rules: FREQ DAILY/WEEKLY/MONTHLY/
// YEARLY (and the sub-day frequencies), INTERVAL, COUNT/UNTIL, BYMONTH,
// BYMONTHDAY, BYDAY (including ordinals like 2MO / -1FR), and BYSETPOS.
// Unsupported parts (BYWEEKNO, BYYEARDAY) are ignored.
func (r *RRule) Between(dtstart, from, to time.Time, limit int) []time.Time {
	if r == nil || dtstart.IsZero() || !to.After(from) {
		return nil
	}
	interval := r.Interval
	if interval < 1 {
		interval = 1
	}

	var out []time.Time
	emitted := 0 // total occurrences from dtstart, for COUNT

	for p := 0; p < maxRecurIterations; p++ {
		periodStart, ok := r.periodStart(dtstart, interval, p)
		if !ok {
			break
		}
		// Once the whole period sits past the window, no later period can
		// contribute to it either — times only increase.
		if periodStart.After(to) {
			break
		}
		if !r.Until.IsZero() && periodStart.After(r.Until) && periodStart.Year() > r.Until.Year() {
			break
		}

		for _, c := range r.candidates(dtstart, periodStart) {
			if c.Before(dtstart) {
				continue
			}
			if r.Count > 0 && emitted >= r.Count {
				return out
			}
			if !r.Until.IsZero() && c.After(r.Until) {
				return out
			}
			emitted++
			if !c.Before(from) && c.Before(to) {
				out = append(out, c)
				if limit > 0 && len(out) >= limit {
					return out
				}
			}
		}
	}
	return out
}

// periodStart returns the anchor time of the p-th period. For month- and
// year-based frequencies it uses calendar arithmetic on year/month so that, for
// example, Jan 31 + 1 month lands in February rather than drifting into March.
func (r *RRule) periodStart(dtstart time.Time, interval, p int) (time.Time, bool) {
	switch r.Freq {
	case Yearly:
		return dtstart.AddDate(interval*p, 0, 0), true
	case Monthly:
		// Anchor to the first of the target month; candidate days are filled in
		// by candidates(). This avoids day-overflow from AddDate on long months.
		y := dtstart.Year()
		m := int(dtstart.Month()) - 1 + interval*p
		y += m / 12
		m = m % 12
		if m < 0 {
			m += 12
			y--
		}
		return time.Date(y, time.Month(m+1), 1,
			dtstart.Hour(), dtstart.Minute(), dtstart.Second(), dtstart.Nanosecond(), dtstart.Location()), true
	case Weekly:
		return dtstart.AddDate(0, 0, 7*interval*p), true
	case Daily:
		return dtstart.AddDate(0, 0, interval*p), true
	case Hourly:
		return dtstart.Add(time.Duration(interval*p) * time.Hour), true
	case Minutely:
		return dtstart.Add(time.Duration(interval*p) * time.Minute), true
	case Secondly:
		return dtstart.Add(time.Duration(interval*p) * time.Second), true
	default:
		return time.Time{}, false
	}
}

// candidates returns the sorted occurrence times contributed by a single
// period, after applying BY* filters and BYSETPOS.
func (r *RRule) candidates(dtstart, periodStart time.Time) []time.Time {
	var set []time.Time

	switch r.Freq {
	case Monthly:
		set = r.monthlyDays(dtstart, periodStart)
	case Yearly:
		set = r.yearlyDays(dtstart, periodStart)
	case Weekly:
		set = r.weeklyDays(dtstart, periodStart)
	default:
		// DAILY and sub-day: the period anchor itself, gated by BY* filters.
		if r.dayMatches(periodStart) {
			set = []time.Time{periodStart}
		}
	}

	set = r.applyByMonth(set)
	sort.Slice(set, func(i, j int) bool { return set[i].Before(set[j]) })
	return r.applySetPos(set)
}

// weeklyDays expands a WEEKLY period into its BYDAY weekdays (or the dtstart
// weekday when BYDAY is absent).
func (r *RRule) weeklyDays(dtstart, periodStart time.Time) []time.Time {
	if len(r.ByDay) == 0 {
		return []time.Time{periodStart}
	}
	// Start of the week containing periodStart, per WeekStart.
	offset := (int(periodStart.Weekday()) - int(r.WeekStart) + 7) % 7
	weekStart := periodStart.AddDate(0, 0, -offset)

	var out []time.Time
	for _, wd := range r.ByDay {
		d := (int(wd.Day) - int(r.WeekStart) + 7) % 7
		out = append(out, withClock(weekStart.AddDate(0, 0, d), dtstart))
	}
	return out
}

// monthlyDays expands a MONTHLY period via BYMONTHDAY and/or BYDAY; with neither
// it keeps dtstart's day-of-month.
func (r *RRule) monthlyDays(dtstart, periodStart time.Time) []time.Time {
	year, month := periodStart.Year(), periodStart.Month()
	var out []time.Time

	// periodStart is anchored to the 1st of the month, so the original
	// day-of-month and time-of-day come from dtstart.
	if len(r.ByMonthDay) == 0 && len(r.ByDay) == 0 {
		return monthDayIfValid(year, month, dtstart.Day(), dtstart)
	}

	for _, md := range r.ByMonthDay {
		day := md
		if md < 0 {
			day = daysInMonth(year, month) + md + 1
		}
		out = append(out, monthDayIfValid(year, month, day, dtstart)...)
	}
	for _, wd := range r.ByDay {
		out = append(out, weekdayOccurrences(year, month, wd, dtstart)...)
	}
	return out
}

// yearlyDays expands a YEARLY period. Months default to dtstart's month; within
// each month, BYMONTHDAY/BYDAY apply, defaulting to dtstart's day.
func (r *RRule) yearlyDays(dtstart, periodStart time.Time) []time.Time {
	year := periodStart.Year()
	months := r.ByMonth
	if len(months) == 0 {
		months = []time.Month{dtstart.Month()}
	}

	var out []time.Time
	for _, month := range months {
		switch {
		case len(r.ByDay) > 0:
			for _, wd := range r.ByDay {
				out = append(out, weekdayOccurrences(year, month, wd, dtstart)...)
			}
		case len(r.ByMonthDay) > 0:
			for _, md := range r.ByMonthDay {
				day := md
				if md < 0 {
					day = daysInMonth(year, month) + md + 1
				}
				out = append(out, monthDayIfValid(year, month, day, dtstart)...)
			}
		default:
			out = append(out, monthDayIfValid(year, month, dtstart.Day(), dtstart)...)
		}
	}
	return out
}

// weekdayOccurrences returns the dates in (year, month) matching a BYDAY entry.
// Ord 0 yields every matching weekday; positive/negative select the nth from the
// start/end of the month.
func weekdayOccurrences(year int, month time.Month, wd WeekDay, clock time.Time) []time.Time {
	var matches []time.Time
	last := daysInMonth(year, month)
	for day := 1; day <= last; day++ {
		d := withClock(time.Date(year, month, day, 0, 0, 0, 0, clock.Location()), clock)
		if d.Weekday() == wd.Day {
			matches = append(matches, d)
		}
	}
	if wd.Ord == 0 {
		return matches
	}
	idx := wd.Ord - 1
	if wd.Ord < 0 {
		idx = len(matches) + wd.Ord
	}
	if idx < 0 || idx >= len(matches) {
		return nil
	}
	return []time.Time{matches[idx]}
}

// monthDayIfValid returns the single date (year, month, day) with clock's
// time-of-day, or nil when the day doesn't exist in that month (e.g. Feb 30).
func monthDayIfValid(year int, month time.Month, day int, clock time.Time) []time.Time {
	if day < 1 || day > daysInMonth(year, month) {
		return nil
	}
	return []time.Time{withClock(time.Date(year, month, day, 0, 0, 0, 0, clock.Location()), clock)}
}

// applyByMonth drops candidates whose month isn't in BYMONTH (when set).
func (r *RRule) applyByMonth(set []time.Time) []time.Time {
	if len(r.ByMonth) == 0 {
		return set
	}
	allowed := make(map[time.Month]bool, len(r.ByMonth))
	for _, m := range r.ByMonth {
		allowed[m] = true
	}
	var out []time.Time
	for _, t := range set {
		if allowed[t.Month()] {
			out = append(out, t)
		}
	}
	return out
}

// applySetPos selects the BYSETPOS-th entries from a period's sorted candidates.
func (r *RRule) applySetPos(set []time.Time) []time.Time {
	if len(r.BySetPos) == 0 {
		return set
	}
	var out []time.Time
	for _, pos := range r.BySetPos {
		idx := pos - 1
		if pos < 0 {
			idx = len(set) + pos
		}
		if idx >= 0 && idx < len(set) {
			out = append(out, set[idx])
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Before(out[j]) })
	return out
}

// dayMatches reports whether t passes the BYDAY/BYMONTHDAY filters used by
// DAILY and the sub-day frequencies (where each period is a single instant).
func (r *RRule) dayMatches(t time.Time) bool {
	if len(r.ByDay) > 0 {
		ok := false
		for _, wd := range r.ByDay {
			if wd.Ord == 0 && wd.Day == t.Weekday() {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if len(r.ByMonthDay) > 0 {
		ok := false
		dim := daysInMonth(t.Year(), t.Month())
		for _, md := range r.ByMonthDay {
			day := md
			if md < 0 {
				day = dim + md + 1
			}
			if day == t.Day() {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	return true
}

// withClock returns date with the time-of-day (and location) of clock.
func withClock(date, clock time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(),
		clock.Hour(), clock.Minute(), clock.Second(), clock.Nanosecond(), clock.Location())
}

func daysInMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// Occurrences returns the concrete start times of this event within the
// half-open window [from, to), merging the RRULE expansion with any RDATE and
// removing any EXDATE. A non-recurring event yields its single start if it falls
// in the window. Results are sorted and de-duplicated; pass limit > 0 to cap the
// count (0 means unlimited, subject to an internal safety bound).
func (e *Event) Occurrences(from, to time.Time, limit int) []time.Time {
	if e.Start.IsZero() || !to.After(from) {
		return nil
	}

	var times []time.Time
	if e.Recurrence != nil {
		times = append(times, e.Recurrence.Between(e.Start, from, to, 0)...)
	} else if !e.Start.Before(from) && e.Start.Before(to) {
		times = append(times, e.Start)
	}
	for _, rd := range e.RDates {
		if !rd.Before(from) && rd.Before(to) {
			times = append(times, rd)
		}
	}

	excluded := make(map[int64]bool, len(e.ExDates))
	for _, ex := range e.ExDates {
		excluded[ex.Unix()] = true
	}

	sort.Slice(times, func(i, j int) bool { return times[i].Before(times[j]) })
	out := times[:0]
	var prev int64 = -1
	for _, t := range times {
		u := t.Unix()
		if excluded[u] || u == prev {
			continue
		}
		prev = u
		out = append(out, t)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out
}
