package scheduler

import (
	"testing"
	"time"
)

func TestParseCron_BasicExpression(t *testing.T) {
	cs, err := ParseCron("0 2 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Minutes) != 1 || cs.Minutes[0] != 0 {
		t.Errorf("Minutes = %v, want [0]", cs.Minutes)
	}
	if len(cs.Hours) != 1 || cs.Hours[0] != 2 {
		t.Errorf("Hours = %v, want [2]", cs.Hours)
	}
	if len(cs.DaysOfMonth) != 31 {
		t.Errorf("DaysOfMonth should have 31 entries for *, got %d", len(cs.DaysOfMonth))
	}
	if len(cs.Months) != 12 {
		t.Errorf("Months should have 12 entries for *, got %d", len(cs.Months))
	}
	if len(cs.DaysOfWeek) != 7 {
		t.Errorf("DaysOfWeek should have 7 entries for *, got %d", len(cs.DaysOfWeek))
	}
}

func TestParseCron_SpecialAliases(t *testing.T) {
	tests := []struct {
		alias string
		equiv string
	}{
		{"@hourly", "0 * * * *"},
		{"@daily", "0 0 * * *"},
		{"@midnight", "0 0 * * *"},
		{"@weekly", "0 0 * * 0"},
		{"@monthly", "0 0 1 * *"},
		{"@yearly", "0 0 1 1 *"},
		{"@annually", "0 0 1 1 *"},
	}
	for _, tt := range tests {
		t.Run(tt.alias, func(t *testing.T) {
			cs1, err := ParseCron(tt.alias)
			if err != nil {
				t.Fatal(err)
			}
			cs2, err := ParseCron(tt.equiv)
			if err != nil {
				t.Fatal(err)
			}
			if !intSliceEqual(cs1.Minutes, cs2.Minutes) {
				t.Errorf("Minutes differ: %v vs %v", cs1.Minutes, cs2.Minutes)
			}
			if !intSliceEqual(cs1.Hours, cs2.Hours) {
				t.Errorf("Hours differ: %v vs %v", cs1.Hours, cs2.Hours)
			}
		})
	}
}

func TestParseCron_Ranges(t *testing.T) {
	cs, err := ParseCron("0-30 9-17 * * *")
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Minutes) != 31 {
		t.Errorf("Minutes should have 31 entries (0-30), got %d", len(cs.Minutes))
	}
	if cs.Minutes[0] != 0 || cs.Minutes[30] != 30 {
		t.Error("Minutes should be 0 to 30")
	}
	if len(cs.Hours) != 9 {
		t.Errorf("Hours should have 9 entries (9-17), got %d", len(cs.Hours))
	}
}

func TestParseCron_Steps(t *testing.T) {
	cs, err := ParseCron("*/15 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	// */15 means 0, 15, 30, 45
	expected := []int{0, 15, 30, 45}
	if !intSliceEqual(cs.Minutes, expected) {
		t.Errorf("Minutes = %v, want %v", cs.Minutes, expected)
	}
}

func TestParseCron_CommaList(t *testing.T) {
	cs, err := ParseCron("0 2,14 * * *")
	if err != nil {
		t.Fatal(err)
	}
	expected := []int{2, 14}
	if !intSliceEqual(cs.Hours, expected) {
		t.Errorf("Hours = %v, want %v", cs.Hours, expected)
	}
}

func TestParseCron_DayNames(t *testing.T) {
	cs, err := ParseCron("0 0 * * MON-FRI")
	if err != nil {
		t.Fatal(err)
	}
	// Mon=1 through Fri=5
	expected := []int{1, 2, 3, 4, 5}
	if !intSliceEqual(cs.DaysOfWeek, expected) {
		t.Errorf("DaysOfWeek = %v, want %v", cs.DaysOfWeek, expected)
	}
}

func TestParseCron_MonthNames(t *testing.T) {
	cs, err := ParseCron("0 0 1 JAN,JUN *")
	if err != nil {
		t.Fatal(err)
	}
	expected := []int{1, 6}
	if !intSliceEqual(cs.Months, expected) {
		t.Errorf("Months = %v, want %v", cs.Months, expected)
	}
}

func TestParseCron_InvalidExpressions(t *testing.T) {
	tests := []struct {
		name string
		expr string
	}{
		{"empty", ""},
		{"too few fields", "0 2 *"},
		{"too many fields", "0 2 * * * *"},
		{"invalid minute", "60 * * * *"},
		{"invalid hour", "0 25 * * *"},
		{"invalid day", "0 0 32 * *"},
		{"invalid month", "0 0 * 13 *"},
		{"invalid dow", "0 0 * * 8"},
		{"bad range", "5-3 * * * *"},
		{"bad step", "*/0 * * * *"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseCron(tt.expr)
			if err == nil {
				t.Errorf("ParseCron(%q) should return error", tt.expr)
			}
		})
	}
}

func TestValidateCron(t *testing.T) {
	if err := ValidateCron("0 2 * * *"); err != nil {
		t.Errorf("valid cron should not error: %v", err)
	}
	if err := ValidateCron("invalid"); err == nil {
		t.Error("invalid cron should return error")
	}
}

func TestCronSchedule_Next(t *testing.T) {
	cs, _ := ParseCron("0 2 * * *") // every day at 2:00
	from := time.Date(2026, 3, 27, 1, 30, 0, 0, time.UTC)
	next := cs.Next(from)

	expected := time.Date(2026, 3, 27, 2, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestCronSchedule_Next_CrossDay(t *testing.T) {
	cs, _ := ParseCron("30 1 * * *") // every day at 1:30
	from := time.Date(2026, 3, 27, 2, 0, 0, 0, time.UTC)
	next := cs.Next(from)

	expected := time.Date(2026, 3, 28, 1, 30, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestCronSchedule_Next_CrossMonth(t *testing.T) {
	cs, _ := ParseCron("0 0 1 * *") // first of every month
	from := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	next := cs.Next(from)

	expected := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestCronSchedule_Next_EveryFiveMinutes(t *testing.T) {
	cs, _ := ParseCron("*/5 * * * *")
	from := time.Date(2026, 3, 27, 10, 3, 0, 0, time.UTC)
	next := cs.Next(from)

	// Next :05 minute
	expected := time.Date(2026, 3, 27, 10, 5, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestCronSchedule_Next_DOWFilter(t *testing.T) {
	cs, _ := ParseCron("0 9 * * 1") // Mondays at 9:00
	// March 27, 2026 is a Friday
	from := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	next := cs.Next(from)

	// Next Monday is March 30
	expected := time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("Next() = %v, want %v", next, expected)
	}
}

func TestCronSchedule_NextN(t *testing.T) {
	cs, _ := ParseCron("0 * * * *") // every hour
	from := time.Date(2026, 3, 27, 10, 0, 0, 0, time.UTC)
	times := cs.NextN(from, 3)

	if len(times) != 3 {
		t.Fatalf("NextN(3) returned %d times", len(times))
	}

	expected := []time.Time{
		time.Date(2026, 3, 27, 11, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 27, 13, 0, 0, 0, time.UTC),
	}
	for i, exp := range expected {
		if !times[i].Equal(exp) {
			t.Errorf("NextN[%d] = %v, want %v", i, times[i], exp)
		}
	}
}

func TestContains(t *testing.T) {
	sorted := []int{1, 5, 10, 15, 20}
	tests := []struct {
		val  int
		want bool
	}{
		{1, true},
		{5, true},
		{20, true},
		{2, false},
		{0, false},
		{25, false},
	}
	for _, tt := range tests {
		if got := contains(sorted, tt.val); got != tt.want {
			t.Errorf("contains(%v, %d) = %v, want %v", sorted, tt.val, got, tt.want)
		}
	}
}

func intSliceEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
