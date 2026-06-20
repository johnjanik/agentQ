package crud

import "testing"

func TestValidateCronGranularity(t *testing.T) {
	valid := []string{
		"0 * * * *",     // top of every hour
		"30 * * * *",    // half past every hour
		"0 9 * * *",     // 09:00 daily
		"15 9 * * 1",    // 09:15 every Monday
		"30 14 25 4 *",  // one-time: 14:30 on Apr 25 (fixed dom+month, minute precision)
		"59 23 31 12 *", // boundary minute/hour
	}
	for _, s := range valid {
		if err := ValidateCronGranularity(s); err != nil {
			t.Errorf("expected %q to be valid, got error: %v", s, err)
		}
	}

	invalid := []struct {
		name     string
		schedule string
	}{
		{"sub-hourly wildcard minute", "* * * * *"},
		{"every 5 minutes step", "*/5 * * * *"},
		{"minute range", "0-30 * * * *"},
		{"minute comma list", "0,30 * * * *"},
		{"minute out of range", "60 * * * *"},
		{"non-numeric minute", "abc * * * *"},
		{"too few fields", "0 9 * *"},
		{"too many fields", "0 9 * * * *"},
		{"empty", ""},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if err := ValidateCronGranularity(tc.schedule); err == nil {
				t.Errorf("expected %q to be rejected, got nil error", tc.schedule)
			}
		})
	}
}
