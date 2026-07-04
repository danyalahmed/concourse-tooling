package resource

import (
	"reflect"
	"testing"
)

func TestBuildKeepArgs(t *testing.T) {
	tests := []struct {
		name     string
		source   Source
		expected []string
	}{
		{
			name:   "defaults",
			source: Source{},
			expected: []string{
				"--keep-daily", "7",
				"--keep-weekly", "4",
				"--keep-monthly", "12",
				"--keep-yearly", "3",
			},
		},
		{
			name: "overrides",
			source: Source{
				KeepDaily:   1,
				KeepWeekly:  2,
				KeepMonthly: 3,
				KeepYearly:  4,
			},
			expected: []string{
				"--keep-daily", "1",
				"--keep-weekly", "2",
				"--keep-monthly", "3",
				"--keep-yearly", "4",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildKeepArgs(tt.source)
			if !reflect.DeepEqual(got, tt.expected) {
				t.Errorf("got %v, want %v", got, tt.expected)
			}
		})
	}
}
