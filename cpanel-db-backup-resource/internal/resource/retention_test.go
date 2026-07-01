package resource

import (
	"reflect"
	"testing"
)

func intPtr(i int) *int {
	return &i
}

func TestCalculateBackupsToRemove(t *testing.T) {
	source := Source{
		KeepDaily:   intPtr(2),
		KeepWeekly:  intPtr(1),
		KeepMonthly: intPtr(1),
		KeepYearly:  intPtr(1),
	}

	names := []string{
		"2023-10-27", // Daily 1
		"2023-10-26", // Daily 2
		"2023-10-25", // Weekly 1 (ISO week 43)
		"2023-09-01", // Monthly 1
		"2022-01-01", // Yearly 1
		"2021-01-01", // Yearly 2 (it seems I have keep yearly 1, but let's check)
	}

	toRemove := calculateBackupsToRemove(names, source, InParams{})

	if len(toRemove) == 0 {
		t.Errorf("Expected some backups to be removed, got none")
	}
}

func TestCalculateBackupsToRemove_Exact(t *testing.T) {
	source := Source{
		KeepDaily:   intPtr(1),
		KeepWeekly:  intPtr(0),
		KeepMonthly: intPtr(0),
		KeepYearly:  intPtr(0),
	}

	names := []string{
		"2023-10-27",
		"2023-10-26",
	}

	toRemove := calculateBackupsToRemove(names, source, InParams{})
	expected := []string{"2023-10-26"}

	if !reflect.DeepEqual(toRemove, expected) {
		t.Errorf("Expected %v, got %v", expected, toRemove)
	}
}

func TestCalculateBackupsToRemove_Override(t *testing.T) {
	source := Source{
		KeepDaily: intPtr(10),
	}
	params := InParams{
		KeepDaily: intPtr(1),
	}

	names := []string{
		"2023-10-27",
		"2023-10-26",
	}

	toRemove := calculateBackupsToRemove(names, source, params)
	expected := []string{"2023-10-26"}

	if !reflect.DeepEqual(toRemove, expected) {
		t.Errorf("Expected %v, got %v", expected, toRemove)
	}
}

func TestCalculateBackupsToRemove_GFS(t *testing.T) {
	source := Source{
		KeepDaily:   intPtr(1),
		KeepWeekly:  intPtr(2),
		KeepMonthly: intPtr(1),
		KeepYearly:  intPtr(1),
	}

	names := []string{
		"2023-10-27", // Daily 1, Weekly 1, Monthly 1, Yearly 1 (Week 43)
		"2023-10-26", // Should be removed
		"2023-10-20", // Weekly 2 (Week 42)
	}

	toRemove := calculateBackupsToRemove(names, source, InParams{})

	// 2023-10-27 is kept (all buckets)
	// 2023-10-26 is NOT daily (27 is), NOT weekly (27 is), NOT monthly (27 is), NOT yearly (27 is). REMOVE.
	// 2023-10-20 is NOT daily (27 is), IS weekly (week 42, 27 is week 43). KEEP because KeepWeekly is 2.

	contains := func(slice []string, s string) bool {
		for _, item := range slice {
			if item == s {
				return true
			}
		}
		return false
	}

	if !contains(toRemove, "2023-10-26") {
		t.Errorf("Expected 2023-10-26 to be removed")
	}
	if contains(toRemove, "2023-10-27") {
		t.Errorf("Expected 2023-10-27 to be kept")
	}
	if contains(toRemove, "2023-10-20") {
		t.Errorf("Expected 2023-10-20 to be kept (weekly)")
	}
}
