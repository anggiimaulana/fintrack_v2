package audit

import "strings"

// sensitiveKeys are field names whose values must never be persisted verbatim.
// Matching is case-insensitive and substring-based so "password_hash",
// "api_key", "access_token", etc. are all covered.
var sensitiveKeys = []string{
	"password",
	"passwd",
	"secret",
	"token",
	"api_key",
	"apikey",
	"authorization",
	"cookie",
	"encryption_key",
	"private_key",
	"card",
	"cvv",
	"pan",
}

// redacted is the replacement value for masked fields.
const redacted = "[REDACTED]"

// Mask returns a copy of the map with sensitive values replaced. Nested maps
// are masked recursively. The original map is not modified.
func Mask(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		if isSensitiveKey(k) {
			out[k] = redacted
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			out[k] = Mask(val)
		case []any:
			out[k] = maskSlice(val)
		default:
			out[k] = v
		}
	}
	return out
}

func maskSlice(in []any) []any {
	out := make([]any, len(in))
	for i, v := range in {
		switch val := v.(type) {
		case map[string]any:
			out[i] = Mask(val)
		case []any:
			out[i] = maskSlice(val)
		default:
			out[i] = v
		}
	}
	return out
}

func isSensitiveKey(key string) bool {
	lower := strings.ToLower(key)
	for _, s := range sensitiveKeys {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
