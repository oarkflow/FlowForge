package scheduler

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

// CronSchedule represents a parsed 5-field cron expression.
type CronSchedule struct {
	Expression string
	Minutes    []int // 0-59
	Hours      []int // 0-23
	DaysOfMonth []int // 1-31
	Months     []int // 1-12
	DaysOfWeek []int // 0-6 (0 = Sunday)
}

// specialAliases maps shorthand strings to standard cron expressions.
var specialAliases = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

// dayNames maps three-letter day abbreviations to their numeric value (0-6).
var dayNames = map[string]int{
	"sun": 0, "mon": 1, "tue": 2, "wed": 3,
	"thu": 4, "fri": 5, "sat": 6,
}

// monthNames maps three-letter month abbreviations to their numeric value (1-12).
var monthNames = map[string]int{
	"jan": 1, "feb": 2, "mar": 3, "apr": 4,
	"may": 5, "jun": 6, "jul": 7, "aug": 8,
	"sep": 9, "oct": 10, "nov": 11, "dec": 12,
}

// ParseCron parses a cron expression and returns a CronSchedule.
// Supports standard 5-field format and special aliases (@hourly, @daily, etc.).
func ParseCron(expression string) (*CronSchedule, error) {
	expr := strings.TrimSpace(expression)
	if expr == "" {
		return nil, fmt.Errorf("empty cron expression")
	}

	// Check for special aliases
	if alias, ok := specialAliases[strings.ToLower(expr)]; ok {
		expr = alias
	}

	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return nil, fmt.Errorf("cron expression must have 5 fields (minute hour day_of_month month day_of_week), got %d", len(fields))
	}

	minutes, err := parseField(fields[0], 0, 59, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid minute field %q: %w", fields[0], err)
	}

	hours, err := parseField(fields[1], 0, 23, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid hour field %q: %w", fields[1], err)
	}

	daysOfMonth, err := parseField(fields[2], 1, 31, nil)
	if err != nil {
		return nil, fmt.Errorf("invalid day_of_month field %q: %w", fields[2], err)
	}

	months, err := parseField(fields[3], 1, 12, monthNames)
	if err != nil {
		return nil, fmt.Errorf("invalid month field %q: %w", fields[3], err)
	}

	daysOfWeek, err := parseField(fields[4], 0, 6, dayNames)
	if err != nil {
		return nil, fmt.Errorf("invalid day_of_week field %q: %w", fields[4], err)
	}

	return &CronSchedule{
		Expression:  expression,
		Minutes:     minutes,
		Hours:       hours,
		DaysOfMonth: daysOfMonth,
		Months:      months,
		DaysOfWeek:  daysOfWeek,
	}, nil
}

// ValidateCron validates a cron expression without returning the schedule.
func ValidateCron(expression string) error {
	_, err := ParseCron(expression)
	return err
}

// Next computes the next run time after from.
func (cs *CronSchedule) Next(from time.Time) time.Time {
	// Start from the next full minute after 'from'
	t := from.Truncate(time.Minute).Add(time.Minute)

	// Search up to 4 years ahead to prevent infinite loops
	maxTime := from.Add(4 * 365 * 24 * time.Hour)

	for t.Before(maxTime) {
		// Check month
		if !contains(cs.Months, int(t.Month())) {
			// Advance to next month
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Check day of month and day of week
		domMatch := contains(cs.DaysOfMonth, t.Day())
		dowMatch := contains(cs.DaysOfWeek, int(t.Weekday()))

		// If both DOM and DOW are restricted (not *), match either (union).
		// If only one is restricted, match that one.
		domRestricted := len(cs.DaysOfMonth) < 31
		dowRestricted := len(cs.DaysOfWeek) < 7

		dayMatch := false
		if domRestricted && dowRestricted {
			dayMatch = domMatch || dowMatch
		} else if domRestricted {
			dayMatch = domMatch
		} else if dowRestricted {
			dayMatch = dowMatch
		} else {
			dayMatch = true // both unrestricted
		}

		if !dayMatch {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, t.Location())
			continue
		}

		// Check hour
		if !contains(cs.Hours, t.Hour()) {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, t.Location())
			continue
		}

		// Check minute
		if !contains(cs.Minutes, t.Minute()) {
			t = t.Add(time.Minute)
			continue
		}

		return t
	}

	// Fallback: should never reach here for valid schedules
	return maxTime
}

// NextN computes the next n run times after from.
func (cs *CronSchedule) NextN(from time.Time, n int) []time.Time {
	times := make([]time.Time, 0, n)
	current := from
	for i := 0; i < n; i++ {
		next := cs.Next(current)
		times = append(times, next)
		current = next
	}
	return times
}

// parseField parses a single cron field and returns the matching values.
// names is an optional map from name strings to numeric values (e.g., month/day names).
func parseField(field string, min, max int, names map[string]int) ([]int, error) {
	parts := strings.Split(field, ",")
	valueSet := make(map[int]bool)

	for _, part := range parts {
		values, err := parsePart(strings.TrimSpace(part), min, max, names)
		if err != nil {
			return nil, err
		}
		for _, v := range values {
			valueSet[v] = true
		}
	}

	result := make([]int, 0, len(valueSet))
	for v := range valueSet {
		result = append(result, v)
	}
	sort.Ints(result)
	return result, nil
}

// parsePart parses a single comma-separated part of a cron field.
func parsePart(part string, min, max int, names map[string]int) ([]int, error) {
	// Handle step: e.g., "*/5" or "1-30/2"
	var step int
	var base string
	if idx := strings.Index(part, "/"); idx >= 0 {
		var err error
		step, err = strconv.Atoi(part[idx+1:])
		if err != nil || step <= 0 {
			return nil, fmt.Errorf("invalid step value in %q", part)
		}
		base = part[:idx]
	} else {
		step = 1
		base = part
	}

	var rangeMin, rangeMax int

	if base == "*" {
		rangeMin = min
		rangeMax = max
	} else if idx := strings.Index(base, "-"); idx >= 0 {
		// Range: e.g., "1-5"
		low, err := resolveValue(base[:idx], names, min, max)
		if err != nil {
			return nil, err
		}
		high, err := resolveValue(base[idx+1:], names, min, max)
		if err != nil {
			return nil, err
		}
		if low > high {
			return nil, fmt.Errorf("invalid range %d-%d", low, high)
		}
		rangeMin = low
		rangeMax = high
	} else {
		// Single value
		val, err := resolveValue(base, names, min, max)
		if err != nil {
			return nil, err
		}
		if step == 1 {
			return []int{val}, nil
		}
		rangeMin = val
		rangeMax = max
	}

	// Generate values with step
	var values []int
	for v := rangeMin; v <= rangeMax; v += step {
		values = append(values, v)
	}
	return values, nil
}

// resolveValue converts a string to an integer, supporting named values.
func resolveValue(s string, names map[string]int, min, max int) (int, error) {
	s = strings.TrimSpace(s)
	if names != nil {
		if v, ok := names[strings.ToLower(s)]; ok {
			return v, nil
		}
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid value %q", s)
	}
	if val < min || val > max {
		return 0, fmt.Errorf("value %d out of range [%d, %d]", val, min, max)
	}
	return val, nil
}

// contains checks if a sorted slice contains a value.
func contains(sorted []int, val int) bool {
	idx := sort.SearchInts(sorted, val)
	return idx < len(sorted) && sorted[idx] == val
}
