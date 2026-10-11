package payment

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
)

// ManualGateway implements Gateway for bank-transfer/QRIS payments: it returns
// transfer instructions and waits for the user to upload proof which a super
// admin reviews.
type ManualGateway struct {
	// BankAccount / instructions shown to the customer.
	BankName      string
	AccountNumber string
	AccountHolder string
	QRISURL       string
}

// NewManualGateway builds the manual gateway.
func NewManualGateway(bankName, accountNumber, accountHolder string) *ManualGateway {
	return &ManualGateway{BankName: bankName, AccountNumber: accountNumber, AccountHolder: accountHolder}
}

// Name returns the gateway identifier.
func (g *ManualGateway) Name() string { return "manual" }

// CreateTransaction returns transfer instructions; status stays pending until
// proof is uploaded and approved.
func (g *ManualGateway) CreateTransaction(_ context.Context, req CreateTxRequest) (*CreateTxResponse, error) {
	return &CreateTxResponse{
		GatewayRef: req.OrderID,
		Status:     PaymentPending,
		Raw: map[string]any{
			"instructions": map[string]any{
				"bank_name":      g.BankName,
				"account_number": g.AccountNumber,
				"account_holder": g.AccountHolder,
				"amount_idr":     req.AmountIDR,
				"qris_url":       g.QRISURL,
			},
		},
	}, nil
}

// VerifyWebhook is not used for manual payments (there is no callback).
func (g *ManualGateway) VerifyWebhook(context.Context, []byte, map[string]string) (*WebhookResult, error) {
	return nil, fmt.Errorf("manual gateway does not support webhooks")
}

// CheckStatus is not used for manual payments (status is set by admin review).
func (g *ManualGateway) CheckStatus(context.Context, string) (*StatusResult, error) {
	return nil, fmt.Errorf("manual gateway does not support status pull")
}

// CancelTransaction is a no-op (admin can void the invoice instead).
func (g *ManualGateway) CancelTransaction(context.Context, string) error { return nil }

// ─── Proof validation ─────────────────────────────────────────

// AllowedProofMIME is the set of accepted proof image types.
var AllowedProofMIME = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// MaxProofBytes caps proof uploads at 5 MB.
const MaxProofBytes = 5 << 20

// DetectImageMIME sniffs magic bytes to determine the image type. Returns "" when
// the content is not a supported image (rejects PDF/EXE/etc.).
func DetectImageMIME(b []byte) string {
	if len(b) < 4 {
		return ""
	}
	switch {
	case b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF:
		return "image/jpeg"
	case b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47:
		return "image/png"
	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return "image/webp"
	}
	return ""
}

// ValidateProof validates size and magic-byte type, returning the detected MIME
// and a safe file extension.
func ValidateProof(content []byte, _ string) (mime, ext string, err error) {
	if len(content) == 0 {
		return "", "", ErrInvalidProof
	}
	if len(content) > MaxProofBytes {
		return "", "", ErrProofTooLarge
	}
	mime = DetectImageMIME(content)
	if mime == "" || !AllowedProofMIME[mime] {
		return "", "", ErrInvalidProof
	}
	switch mime {
	case "image/jpeg":
		ext = ".jpg"
	case "image/png":
		ext = ".png"
	case "image/webp":
		ext = ".webp"
	}
	return mime, ext, nil
}

// SafeProofFilename builds a UUID-based filename with the validated extension.
func SafeProofFilename(id string, ext string) string {
	clean := strings.TrimSuffix(filepath.Base(id), filepath.Ext(id))
	return "payment-proofs/" + clean + ext
}

var _ Gateway = (*ManualGateway)(nil)
