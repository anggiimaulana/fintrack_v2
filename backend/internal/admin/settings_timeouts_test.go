package admin

import (
	"encoding/json"
	"testing"
)

// TestTimeoutsSettingsAreWritable verifies the AI timeout keys are accepted by
// the settings allowlist with sane bounds, so the Super Admin UI can persist
// them (a key missing from the allowlist is rejected with 400).
func TestTimeoutsSettingsAreWritable(t *testing.T) {
	keys := []string{
		"ai_timeout_chat_ms",
		"ai_timeout_vision_ms",
		"ai_timeout_voice_ms",
		"ai_timeout_embedding_ms",
		"ai_timeout_reasoning_ms",
		"ai_timeout_client_ms",
	}
	for _, k := range keys {
		spec, ok := writableSettings[k]
		if !ok {
			t.Fatalf("expected %q to be writable", k)
		}
		if spec.kind != "int" {
			t.Fatalf("expected %q to be an int setting, got %q", k, spec.kind)
		}
		if spec.min != 0 {
			t.Fatalf("expected %q min 0 (0 = fall back), got %d", k, spec.min)
		}
		if spec.max != 1_800_000 {
			t.Fatalf("expected %q max 1800000, got %d", k, spec.max)
		}
	}
}

// TestValidateTimeoutSetting covers the value validation the endpoint applies.
func TestValidateTimeoutSetting(t *testing.T) {
	spec := writableSettings["ai_timeout_vision_ms"]

	cases := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{"zero allowed", "0", false},
		{"mid allowed", "120000", false},
		{"max allowed", "1800000", false},
		{"negative rejected", "-1", true},
		{"above max rejected", "1800001", true},
		{"non-integer rejected", "\"abc\"", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateSetting(spec, json.RawMessage(tc.raw))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error for %s", tc.raw)
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error for %s: %v", tc.raw, err)
			}
		})
	}
}

// TestHasTimeoutsKey verifies the reload trigger only fires for timeout keys.
func TestHasTimeoutsKey(t *testing.T) {
	if !hasTimeoutsKey(map[string]json.RawMessage{"ai_timeout_vision_ms": json.RawMessage("120000")}) {
		t.Fatal("expected true for a timeout key")
	}
	if hasTimeoutsKey(map[string]json.RawMessage{"mail_host": json.RawMessage("\"x\"")}) {
		t.Fatal("expected false for a non-timeout key")
	}
	if hasTimeoutsKey(nil) {
		t.Fatal("expected false for nil map")
	}
}
