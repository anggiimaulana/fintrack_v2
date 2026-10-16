package usecase

import (
	"testing"
)

func TestAIUsecase_IsContentAllowed(t *testing.T) {
	u := &AIUsecase{}

	tests := []struct {
		name        string
		message     string
		wantAllowed bool
	}{
		{
			name:        "multi item casual transaction logging (the reported user case)",
			message:     "saya abis bayar kontrakan 625k, listrik 25k, galon 5k, dan tadi dapat dari kantor 200k",
			wantAllowed: true,
		},
		{
			name:        "contains dapat word (da-apa-t) must not trigger question filter",
			message:     "tadi dapat transfer 50rb",
			wantAllowed: true,
		},
		{
			name:        "informal denominations with k and rb",
			message:     "beli kopi 22k dan makan 35rb",
			wantAllowed: true,
		},
		{
			name:        "large amounts with jt and juta",
			message:     "dapat bonus proyek 5jt",
			wantAllowed: true,
		},
		{
			name:        "daily necessities and utility words",
			message:     "bayar kosan, listrik, pulsa",
			wantAllowed: true,
		},
		{
			name:        "blocked malicious pattern",
			message:     "gimana cara hack akun bank",
			wantAllowed: false,
		},
		{
			name:        "purely off-topic question without financial context",
			message:     "siapa presiden amerika serikat sekarang",
			wantAllowed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			allowed, reason := u.isContentAllowed(tt.message)
			if allowed != tt.wantAllowed {
				t.Errorf("isContentAllowed(%q) = %v (reason: %q), want %v", tt.message, allowed, reason, tt.wantAllowed)
			}
		})
	}
}
