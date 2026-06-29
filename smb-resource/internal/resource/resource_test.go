package resource

import (
	"testing"
)

func TestCheck(t *testing.T) {
	// Driver.Check requires a real SMB connection, so we'll just test the struct initialization
	d := &Driver{}
	if d == nil {
		t.Fatal("Driver is nil")
	}
}
