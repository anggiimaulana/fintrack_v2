package validator

import (
	"testing"
)

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"plain text", "hello world", "hello world"},
		{"with HTML tags", "<script>alert('xss')</script>", "scriptalert(xss)/script"},
		{"with angle brackets", "<b>bold</b>", "bbold/b"},
		{"with quotes", `he said "hello"`, "he said hello"},
		{"normal text", "Hello, World! How are you?", "Hello, World! How are you?"},
		{"with ampersand", "foo & bar", "foo  bar"},
		{"with special chars only", "<>", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizeString(tt.input)
			if got != tt.want {
				t.Errorf("SanitizeString(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeMap(t *testing.T) {
	data := map[string]interface{}{
		"name":        "<script>alert(1)</script>",
		"description": "safe text",
		"count":       42,
	}

	SanitizeMap(data)

	if data["name"].(string) != "scriptalert(1)/script" {
		t.Errorf("XSS not sanitized in name field")
	}
	if data["description"].(string) != "safe text" {
		t.Errorf("Safe text should remain unchanged")
	}
	if data["count"].(int) != 42 {
		t.Errorf("Non-string fields should not be modified")
	}
}
