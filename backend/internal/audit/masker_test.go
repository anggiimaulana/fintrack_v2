package audit

import (
	"reflect"
	"testing"
)

// TestMask_RedactsSensitiveKeys verifies sensitive fields are replaced while
// non-sensitive fields survive.
func TestMask_RedactsSensitiveKeys(t *testing.T) {
	in := map[string]any{
		"email":         "user@example.com",
		"password":      "s3cr3t",
		"api_key":       "sk-123",
		"access_token":  "tok",
		"Authorization": "Bearer x",
		"plan":          "pro",
	}

	out := Mask(in)

	if out["email"] != "user@example.com" {
		t.Fatalf("email should be preserved, got %v", out["email"])
	}
	if out["plan"] != "pro" {
		t.Fatalf("plan should be preserved, got %v", out["plan"])
	}
	for _, k := range []string{"password", "api_key", "access_token", "Authorization"} {
		if out[k] != redacted {
			t.Fatalf("key %q should be %q, got %v", k, redacted, out[k])
		}
	}
}

// TestMask_Nested verifies recursive masking of nested maps and slices.
func TestMask_Nested(t *testing.T) {
	in := map[string]any{
		"user": map[string]any{
			"name":     "Budi",
			"password": "x",
		},
		"keys": []any{
			map[string]any{"api_key": "k1", "alias": "a1"},
			map[string]any{"secret": "s2"},
		},
	}

	out := Mask(in)

	user := out["user"].(map[string]any)
	if user["name"] != "Budi" {
		t.Fatalf("nested name should be preserved, got %v", user["name"])
	}
	if user["password"] != redacted {
		t.Fatalf("nested password should be masked, got %v", user["password"])
	}

	keys := out["keys"].([]any)
	k0 := keys[0].(map[string]any)
	if k0["api_key"] != redacted {
		t.Fatalf("nested api_key should be masked, got %v", k0["api_key"])
	}
	if k0["alias"] != "a1" {
		t.Fatalf("nested alias should be preserved, got %v", k0["alias"])
	}
}

// TestMask_DoesNotMutateInput verifies the source map is left untouched.
func TestMask_DoesNotMutateInput(t *testing.T) {
	in := map[string]any{"password": "orig"}
	_ = Mask(in)
	if in["password"] != "orig" {
		t.Fatal("Mask must not mutate its input")
	}
}

// TestMask_Nil verifies nil handling.
func TestMask_Nil(t *testing.T) {
	if Mask(nil) != nil {
		t.Fatal("Mask(nil) should be nil")
	}
	if !reflect.DeepEqual(Mask(map[string]any{}), map[string]any{}) {
		t.Fatal("Mask(empty) should be empty")
	}
}
