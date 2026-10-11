package main

import (
	"os"
	"testing"
	"time"
)

func TestShutdownTimeout_DefaultAndOverride(t *testing.T) {
	os.Unsetenv("SHUTDOWN_TIMEOUT")
	if got := shutdownTimeout(); got != 30*time.Second {
		t.Fatalf("default = %v, want 30s", got)
	}

	os.Setenv("SHUTDOWN_TIMEOUT", "45")
	defer os.Unsetenv("SHUTDOWN_TIMEOUT")
	if got := shutdownTimeout(); got != 45*time.Second {
		t.Fatalf("override = %v, want 45s", got)
	}

	// Invalid / non-positive values fall back to the default.
	for _, bad := range []string{"abc", "0", "-5"} {
		os.Setenv("SHUTDOWN_TIMEOUT", bad)
		if got := shutdownTimeout(); got != 30*time.Second {
			t.Fatalf("%q: got %v, want default 30s", bad, got)
		}
	}
}
