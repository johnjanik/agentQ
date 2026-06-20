package crud

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/robfig/cron/v3"
)

// ValidateCronGranularity validates that a cron schedule is syntactically valid
// and not finer than the supported granularity.
//
// For RECURRING schedules (dom or month field is "*"), the minimum granularity is
// hourly: the minute field must be a single fixed integer 0-59 (no wildcards, steps,
// ranges, or comma-lists), because sub-hourly recurring tasks are not supported.
// For ONE-TIME schedules (both dom and month are fixed integers, e.g. "30 14 25 4 *"),
// any fixed minute value 0-59 is accepted — enabling minute-level precision.
//
// This is the single source of truth for cron granularity rules; every entry point
// that accepts a cron schedule (MCP tool handlers and the REST CRUD paths) must
// route through it so the rules cannot drift between transports.
func ValidateCronGranularity(schedule string) error {
	fields := strings.Fields(schedule)
	if len(fields) != 5 {
		return fmt.Errorf("cron schedule must have exactly 5 fields (minute hour dom month dow)")
	}

	minuteField := fields[0]

	// Reject wildcards, steps (*/n), ranges (a-b), and comma-lists in the minute field.
	if minuteField == "*" ||
		strings.Contains(minuteField, "/") ||
		strings.Contains(minuteField, "-") ||
		strings.Contains(minuteField, ",") {
		return fmt.Errorf("cron schedule granularity too fine: minute field must be a single fixed value (0-59), not %q — only hourly or coarser schedules are allowed", minuteField)
	}

	minute, err := strconv.Atoi(minuteField)
	if err != nil || minute < 0 || minute > 59 {
		return fmt.Errorf("cron schedule minute field must be a valid integer between 0 and 59, got %q", minuteField)
	}

	// Validate overall syntax using the standard 5-field parser.
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	if _, err := parser.Parse(schedule); err != nil {
		return fmt.Errorf("invalid cron schedule: %w", err)
	}

	return nil
}
