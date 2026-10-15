package entitlement

import "testing"

// TestCatalog_KeysMatchResolvers guards the invariant that every limit key the
// middleware can enforce is present in the catalog, and vice-versa — F5/T-ADM-CAT-01.
func TestCatalog_KeysMatchResolvers(t *testing.T) {
	// These are the keys the usage resolvers/middleware actually enforce today.
	enforced := []string{
		"ai_tokens_per_cycle",
		"ai_requests_per_cycle",
		"max_transactions",
		"ocr_scans_per_cycle",
		"voice_requests_per_day",
	}
	for _, key := range enforced {
		if !IsKnownLimit(key) {
			t.Errorf("enforced limit key %q missing from catalog", key)
		}
	}

	// Catalogued-as-enforced keys must be exactly the enforced set.
	for _, l := range LimitCatalog {
		if l.Enforced && !contains(enforced, l.Key) {
			t.Errorf("catalog marks %q enforced but no resolver exists", l.Key)
		}
	}
}

// TestCatalog_FeatureLookup verifies feature membership checks.
func TestCatalog_FeatureLookup(t *testing.T) {
	if !IsKnownFeature("ai_chat") {
		t.Error("ai_chat should be a known feature")
	}
	if IsKnownFeature("ai_chatt") {
		t.Error("typo'd key must NOT be known (this is the bug the catalog prevents)")
	}
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
