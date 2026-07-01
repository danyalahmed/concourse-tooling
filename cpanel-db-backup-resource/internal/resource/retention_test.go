package resource

import (
	"reflect"
	"testing"
)

func TestCalculateBackupsToRemove(t *testing.T) {
	source := Source{
		KeepDaily:   2,
		KeepWeekly:  1,
		KeepMonthly: 1,
		KeepYearly:  1,
	}

	names := []string{
		"2023-10-27", // Daily 1
		"2023-10-26", // Daily 2
		"2023-10-25",
		"2023-09-01",
		"2022-01-01",
		"2021-01-01",
	}

	toRemove := calculateBackupsToRemove(names, source)

	if len(toRemove) == 0 {
		t.Errorf("Expected some backups to be removed, got none")
	}
}

func TestCalculateBackupsToRemove_Exact(t *testing.T) {
	source := Source{
		KeepDaily:   1,
		KeepWeekly:  0,
		KeepMonthly: 0,
		KeepYearly:  0,
	}

	names := []string{
		"2023-10-27",
		"2023-10-26",
	}

	toRemove := calculateBackupsToRemove(names, source)
	expected := []string{"2023-10-26"}

	if !reflect.DeepEqual(toRemove, expected) {
		t.Errorf("Expected %v, got %v", expected, toRemove)
	}
}
