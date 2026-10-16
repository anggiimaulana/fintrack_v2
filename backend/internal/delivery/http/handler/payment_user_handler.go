package handler

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/billing"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/storage"
	"github.com/fintrack/backend/pkg/pagination"
	"github.com/fintrack/backend/pkg/response"
)

// ─── Wiring (P0/P1 user billing surface) ──────────────────────

// WithUserService attaches the user-scoped payment service (history + proof).
func (h *PaymentHandler) WithUserService(u *payment.UserService) *PaymentHandler {
	h.userSvc = u
	return h
}

// WithCatalog attaches the public plan catalog service.
func (h *PaymentHandler) WithCatalog(c *payment.CatalogService) *PaymentHandler {
	h.catalog = c
	return h
}

// WithSubscriptionService attaches the user subscription self-service.
func (h *PaymentHandler) WithSubscriptionService(s *payment.SubscriptionService, subs billing.SubscriptionRepository) *PaymentHandler {
	h.subSvc = s
	h.subRepo = subs
	return h
}

// WithStorage attaches the proof object store (for serving uploaded proofs).
func (h *PaymentHandler) WithStorage(ps *storage.ProofStore) *PaymentHandler {
	h.proofs = ps
	return h
}

// ─── Catalog ──────────────────────────────────────────────────

// ListPublicPlans handles GET /api/v1/plans (any authenticated user). Returns
// the active public catalog with features/limits so the pricing UI is dynamic.
func (h *PaymentHandler) ListPublicPlans(c *gin.Context) {
	if h.catalog == nil {
		response.OK(c, "Katalog paket tidak tersedia", []any{})
		return
	}
	plans, err := h.catalog.ListPublicPlans(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Gagal memuat katalog paket")
		return
	}
	if plans == nil {
		plans = []*payment.PublicPlan{}
	}
	response.OK(c, "Katalog paket berhasil dimuat", plans)
}

// ─── User payment history & invoices ──────────────────────────

// ListMyPayments handles GET /api/v1/payments (authenticated user). Supports
// table params (page, per_page, sort, order, q, filter_status).
func (h *PaymentHandler) ListMyPayments(c *gin.Context) {
	if h.userSvc == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	pq := pagination.Parse(c, pagination.Options{
		AllowedSort:  []string{"created_at", "amount_idr", "status", "total_idr"},
		DefaultSort:  "created_at",
		DefaultOrder: "desc",
	})
	list, total, err := h.userSvc.ListPayments(c.Request.Context(), userID, &pq)
	if err != nil {
		response.InternalError(c, "Gagal memuat riwayat pembayaran")
		return
	}
	if list == nil {
		list = []*payment.UserPayment{}
	}
	response.Paginated(c, "Riwayat pembayaran berhasil dimuat", list, total, pq.Page, pq.PerPage)
}

// GetMyPayment handles GET /api/v1/payments/:id (owner only).
func (h *PaymentHandler) GetMyPayment(c *gin.Context) {
	if h.userSvc == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID pembayaran tidak valid", "invalid_id")
		return
	}
	pay, err := h.userSvc.GetPayment(c.Request.Context(), userID, id)
	if err != nil {
		response.InternalError(c, "Gagal memuat pembayaran")
		return
	}
	if pay == nil {
		response.NotFound(c, "Pembayaran tidak ditemukan")
		return
	}
	response.OK(c, "Pembayaran berhasil dimuat", pay)
}

// GetMyPaymentInstructions handles GET /api/v1/payments/:id/instructions.
func (h *PaymentHandler) GetMyPaymentInstructions(c *gin.Context) {
	if h.userSvc == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID pembayaran tidak valid", "invalid_id")
		return
	}
	instr, err := h.userSvc.PaymentInstructions(c.Request.Context(), userID, id)
	if err != nil {
		if errors.Is(err, payment.ErrPaymentNotFound) {
			response.NotFound(c, "Pembayaran tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal memuat instruksi pembayaran")
		return
	}
	response.OK(c, "Instruksi pembayaran berhasil dimuat", instr)
}

// ListMyInvoices handles GET /api/v1/invoices (authenticated user). Supports
// table params (page, per_page, sort, order, q, filter_status).
func (h *PaymentHandler) ListMyInvoices(c *gin.Context) {
	if h.userSvc == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	pq := pagination.Parse(c, pagination.Options{
		AllowedSort:  []string{"created_at", "issued_at", "due_at", "total_idr", "status", "invoice_number"},
		DefaultSort:  "created_at",
		DefaultOrder: "desc",
	})
	list, total, err := h.userSvc.ListInvoices(c.Request.Context(), userID, &pq)
	if err != nil {
		response.InternalError(c, "Gagal memuat daftar tagihan")
		return
	}
	if list == nil {
		list = []*payment.Invoice{}
	}
	response.Paginated(c, "Daftar tagihan berhasil dimuat", list, total, pq.Page, pq.PerPage)
}

// CancelMyInvoice handles POST /api/v1/invoices/:id/cancel. It lets a user
// cancel their own unpaid invoice so they can start a new checkout.
func (h *PaymentHandler) CancelMyInvoice(c *gin.Context) {
	if h.userSvc == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID tagihan tidak valid", "invalid_id")
		return
	}
	inv, err := h.userSvc.CancelInvoice(c.Request.Context(), userID, id)
	if err != nil {
		switch {
		case errors.Is(err, payment.ErrInvoiceNotFound):
			response.NotFound(c, "Tagihan tidak ditemukan")
		case errors.Is(err, payment.ErrInvoiceNotCancellable):
			response.Conflict(c, "Tagihan tidak dapat dibatalkan (sudah dibayar atau sudah ditutup)")
		default:
			response.InternalError(c, "Gagal membatalkan tagihan")
		}
		return
	}
	response.OK(c, "Tagihan berhasil dibatalkan", inv)
}

// ─── Manual proof upload + serving ────────────────────────────

// UploadMyProof handles POST /api/v1/payments/:id/proof (multipart or raw body).
// Accepts either a `file` form field or a raw request body, capped at 5 MB.
func (h *PaymentHandler) UploadMyProof(c *gin.Context) {
	if h.userSvc == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID pembayaran tidak valid", "invalid_id")
		return
	}

	// Reject oversized bodies up front (before the multipart parser truncates
	// them and loses the field, which would surface a confusing error).
	const overhead = 1 << 20 // multipart framing allowance
	if cl := c.Request.ContentLength; cl > int64(payment.MaxProofBytes)+overhead {
		response.BadRequest(c, "Ukuran berkas melebihi 5 MB", "proof_too_large")
		return
	}

	// Cap the body before reading anything into memory.
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, int64(payment.MaxProofBytes)+overhead)

	var (
		content  []byte
		filename string
	)
	ct := c.GetHeader("Content-Type")
	if isMultipart(ct) {
		fh, ferr := c.FormFile("file")
		if ferr != nil {
			response.BadRequest(c, "Berkas bukti wajib diunggah pada field 'file'", "file_required")
			return
		}
		if fh.Size > int64(payment.MaxProofBytes) {
			response.BadRequest(c, "Ukuran berkas melebihi 5 MB", "proof_too_large")
			return
		}
		f, oerr := fh.Open()
		if oerr != nil {
			response.InternalError(c, "Gagal membaca berkas bukti")
			return
		}
		defer f.Close()
		content = make([]byte, 0, fh.Size)
		buf := make([]byte, 32*1024)
		for {
			n, rerr := f.Read(buf)
			if n > 0 {
				content = append(content, buf[:n]...)
				if len(content) > payment.MaxProofBytes {
					response.BadRequest(c, "Ukuran berkas melebihi 5 MB", "proof_too_large")
					return
				}
			}
			if rerr != nil {
				break
			}
		}
		filename = fh.Filename
	} else {
		raw, rerr := c.GetRawData()
		if rerr != nil {
			response.BadRequest(c, "Gagal membaca berkas bukti", "invalid_body")
			return
		}
		content = raw
		filename = c.GetHeader("X-Filename")
	}

	pay, err := h.userSvc.UploadProof(c.Request.Context(), userID, id, content, filename)
	if err != nil {
		switch {
		case errors.Is(err, payment.ErrPaymentNotFound):
			response.NotFound(c, "Pembayaran tidak ditemukan")
		case errors.Is(err, payment.ErrProofTooLarge):
			response.BadRequest(c, "Ukuran berkas melebihi 5 MB", "proof_too_large")
		case errors.Is(err, payment.ErrInvalidProof):
			response.BadRequest(c, "Format bukti tidak didukung (gunakan JPG, PNG, atau WEBP)", "invalid_proof")
		case errors.Is(err, payment.ErrInvalidState):
			response.Conflict(c, "Pembayaran ini tidak dapat lagi diunggah buktinya")
		default:
			response.InternalError(c, "Gagal menyimpan bukti pembayaran")
		}
		return
	}
	response.OK(c, "Bukti pembayaran berhasil diunggah. Menunggu verifikasi admin.", gin.H{
		"id":     pay.ID,
		"status": pay.Status,
	})
}

// ServeProof handles GET /api/v1/payments/proofs/:id. It streams the stored
// proof to its owner, or to any super_admin. Auth is enforced by the caller's
// middleware; ownership is enforced here.
func (h *PaymentHandler) ServeProof(c *gin.Context) {
	if h.proofs == nil || h.userSvc == nil {
		response.InternalError(c, "Penyimpanan bukti belum dikonfigurasi")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID pembayaran tidak valid", "invalid_id")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	role, _ := c.Get("role")

	var key string
	if r, ok := role.(string); ok && r == "super_admin" {
		pay, perr := h.userSvc.GetPaymentForAdmin(c.Request.Context(), id)
		if perr != nil || pay == nil || pay.ProofURL == nil {
			response.NotFound(c, "Bukti pembayaran tidak ditemukan")
			return
		}
		key = *pay.ProofURL
	} else {
		pay, perr := h.userSvc.GetPayment(c.Request.Context(), userID, id)
		if perr != nil {
			response.InternalError(c, "Gagal memuat bukti pembayaran")
			return
		}
		if pay == nil || pay.ProofURL == nil || *pay.ProofURL == "" {
			response.NotFound(c, "Bukti pembayaran tidak ditemukan")
			return
		}
		key = *pay.ProofURL
	}

	if key == "" {
		response.NotFound(c, "Bukti pembayaran tidak ditemukan")
		return
	}
	obj, err := h.proofs.Open(c.Request.Context(), key)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			response.NotFound(c, "Berkas bukti tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal membaca berkas bukti")
		return
	}
	defer obj.Body.Close()

	if obj.ContentType != "" {
		c.Header("Content-Type", obj.ContentType)
	}
	if obj.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(obj.Size, 10))
	}
	c.Header("Cache-Control", "private, max-age=300")
	c.Header("Content-Disposition", "inline")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, obj.Body)
}

func isMultipart(ct string) bool {
	return len(ct) >= 19 && ct[:19] == "multipart/form-data"
}

// ─── Saved Card Management & Auto-Charge ───────────────────────

// ListMyCards handles GET /api/v1/payments/cards (authenticated user).
func (h *PaymentHandler) ListMyCards(c *gin.Context) {
	if h.payRepo == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	cards, err := h.payRepo.ListCardTokensByUser(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Gagal memuat daftar kartu tersimpan")
		return
	}
	if cards == nil {
		cards = []*payment.CardToken{}
	}
	response.OK(c, "Daftar kartu tersimpan berhasil dimuat", cards)
}

// DeleteMyCard handles DELETE /api/v1/payments/cards/:id (authenticated user).
func (h *PaymentHandler) DeleteMyCard(c *gin.Context) {
	if h.payRepo == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID kartu tidak valid", "invalid_id")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.payRepo.DeleteCardToken(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, payment.ErrCardTokenNotFound) {
			response.NotFound(c, "Kartu tersimpan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal menghapus kartu tersimpan")
		return
	}
	response.OK(c, "Kartu tersimpan berhasil dihapus", nil)
}

// SetMyDefaultCard handles POST /api/v1/payments/cards/:id/default (authenticated user).
func (h *PaymentHandler) SetMyDefaultCard(c *gin.Context) {
	if h.payRepo == nil {
		response.InternalError(c, "Layanan pembayaran belum dikonfigurasi")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID kartu tidak valid", "invalid_id")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	if err := h.payRepo.SetDefaultCardToken(c.Request.Context(), userID, id); err != nil {
		if errors.Is(err, payment.ErrCardTokenNotFound) {
			response.NotFound(c, "Kartu tersimpan tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal mengatur kartu utama")
		return
	}
	response.OK(c, "Kartu utama berhasil diperbarui", nil)
}

// AutoChargeInvoice handles POST /api/v1/invoices/:id/auto-charge (authenticated user).
func (h *PaymentHandler) AutoChargeInvoice(c *gin.Context) {
	if h.autoCharger == nil || h.invoices == nil {
		response.InternalError(c, "Layanan auto-charge belum dikonfigurasi")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "ID tagihan tidak valid", "invalid_id")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)
	inv, err := h.invoices.Get(c.Request.Context(), id)
	if err != nil || inv == nil {
		response.NotFound(c, "Tagihan tidak ditemukan")
		return
	}
	if inv.UserID != userID {
		response.Forbidden(c, "Akses ditolak")
		return
	}
	if !inv.Status.IsOpen() {
		response.BadRequest(c, "Tagihan sudah tidak dalam status aktif atau menunggu pembayaran", "invalid_state")
		return
	}

	res, err := h.autoCharger.ChargeInvoice(c.Request.Context(), inv, nil)
	if err != nil {
		if errors.Is(err, payment.ErrCardTokenNotFound) {
			response.BadRequest(c, "Tidak ada kartu tersimpan untuk akun Anda", "no_card_token")
			return
		}
		response.InternalError(c, "Gagal memproses pembayaran otomatis")
		return
	}

	if res.Settled {
		response.OK(c, "Pembayaran berhasil diproses menggunakan kartu tersimpan", res)
		return
	}

	response.BadRequest(c, "Pembayaran kartu tersimpan tidak dapat diselesaikan secara otomatis", "charge_not_settled")
}
