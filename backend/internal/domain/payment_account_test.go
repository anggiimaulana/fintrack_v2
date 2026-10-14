package domain

import (
	"errors"
	"testing"
)

func TestPaymentAccountKind_Valid(t *testing.T) {
	for _, k := range []PaymentAccountKind{PaymentAccountBank, PaymentAccountEwallet, PaymentAccountQRIS} {
		if !k.Valid() {
			t.Errorf("%q should be valid", k)
		}
	}
	for _, k := range []PaymentAccountKind{"", "crypto", "BANK"} {
		if k.Valid() {
			t.Errorf("%q should be invalid", k)
		}
	}
}

func TestUpsertPaymentAccountRequest_Validate(t *testing.T) {
	active := true
	cases := []struct {
		name    string
		req     UpsertPaymentAccountRequest
		wantErr error
	}{
		{
			name: "valid bank",
			req:  UpsertPaymentAccountRequest{Kind: PaymentAccountBank, Name: "BCA", AccountNumber: "123"},
		},
		{
			name: "valid ewallet",
			req:  UpsertPaymentAccountRequest{Kind: PaymentAccountEwallet, Name: "DANA", AccountNumber: "0812"},
		},
		{
			name: "valid qris needs only a url",
			req:  UpsertPaymentAccountRequest{Kind: PaymentAccountQRIS, Name: "QRIS", QRISURL: "https://x/y.png"},
		},
		{
			name:    "unknown kind",
			req:     UpsertPaymentAccountRequest{Kind: "crypto", Name: "BTC", AccountNumber: "1"},
			wantErr: ErrPaymentAccountInvalidKind,
		},
		{
			name:    "empty name",
			req:     UpsertPaymentAccountRequest{Kind: PaymentAccountBank, AccountNumber: "1"},
			wantErr: ErrPaymentAccountNameEmpty,
		},
		{
			name:    "bank without number",
			req:     UpsertPaymentAccountRequest{Kind: PaymentAccountBank, Name: "BCA"},
			wantErr: ErrPaymentAccountNumberEmpty,
		},
		{
			name:    "qris without url",
			req:     UpsertPaymentAccountRequest{Kind: PaymentAccountQRIS, Name: "QRIS"},
			wantErr: ErrPaymentAccountQRISURL,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.req.Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Fatalf("expected valid, got %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("expected %v, got %v", tc.wantErr, err)
			}
		})
	}
	_ = active
}
