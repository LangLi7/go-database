package scheduler

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// nextTime computes the next time a cron expression fires after the given time
// Format: "minute hour day-of-month month day-of-week"
// Supports: *, */N, N, N,M (comma list)
func nextTime(expr string, after time.Time) (time.Time, error) {
	fields := strings.Fields(expr)
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("cron: expected 5 fields, got %d", len(fields))
	}

	minSet, err := parseField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron minute: %w", err)
	}
	hourSet, err := parseField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron hour: %w", err)
	}
	daySet, err := parseField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron day: %w", err)
	}
	monthSet, err := parseField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron month: %w", err)
	}
	dowSet, err := parseField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, fmt.Errorf("cron dow: %w", err)
	}

	t := after.Add(time.Minute)
	// Limit search to 2 years
	end := after.AddDate(2, 0, 0)

	for t.Before(end) {
		if !monthSet[int(t.Month())] {
			t = t.AddDate(0, 1, 0)
			t = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
			continue
		}
		if !daySet[t.Day()] {
			t = t.AddDate(0, 0, 1)
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			continue
		}
		if !dowSet[int(t.Weekday())] {
			t = t.AddDate(0, 0, 1)
			t = time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
			continue
		}
		if !hourSet[t.Hour()] {
			t = t.Add(time.Hour)
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, t.Location())
			continue
		}
		if !minSet[t.Minute()] {
			t = t.Add(time.Minute)
			continue
		}
		return t, nil
	}

	return time.Time{}, fmt.Errorf("cron: no matching time within 2 years")
}

func parseField(field string, min, max int) (map[int]bool, error) {
	set := make(map[int]bool)

	// Comma-separated list
	for _, part := range strings.Split(field, ",") {
		part = strings.TrimSpace(part)
		if part == "*" {
			for i := min; i <= max; i++ {
				set[i] = true
			}
			return set, nil
		}
		if strings.Contains(part, "/") {
			parts := strings.SplitN(part, "/", 2)
			base := parts[0]
			step, err := strconv.Atoi(parts[1])
			if err != nil || step <= 0 {
				return nil, fmt.Errorf("invalid step: %s", parts[1])
			}
			var start int
			if base == "*" {
				start = min
			} else {
				start, err = strconv.Atoi(base)
				if err != nil {
					return nil, fmt.Errorf("invalid value: %s", base)
				}
			}
			for i := start; i <= max; i += step {
				set[i] = true
			}
			return set, nil
		}
		val, err := strconv.Atoi(part)
		if err != nil {
			return nil, fmt.Errorf("invalid value: %s", part)
		}
		if val < min || val > max {
			return nil, fmt.Errorf("value %d out of range [%d,%d]", val, min, max)
		}
		set[val] = true
	}
	return set, nil
}
