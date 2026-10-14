package handler

import (
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/payment"
	"github.com/fintrack/backend/internal/storage"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/internal/user"
	"github.com/fintrack/backend/pkg/response"
)

const (
	accessCookieMaxAge  = 900     // 15 minutes
	refreshCookieMaxAge = 2592000 // 30 days

	// MaxAvatarBytes caps avatar uploads at 2 MB.
	MaxAvatarBytes = 2 << 20
)

type AuthHandler struct {
	authUsecase  *usecase.AuthUsecase
	cookieSecure bool
	anonymizeSvc *user.AnonymizeService
	store        *storage.Provider
}

func NewAuthHandler(authUsecase *usecase.AuthUsecase, cookieSecure bool) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase, cookieSecure: cookieSecure}
}

func (h *AuthHandler) WithAnonymize(anonymizeSvc *user.AnonymizeService) *AuthHandler {
	h.anonymizeSvc = anonymizeSvc
	return h
}

// WithStorage attaches the object store used for avatar uploads/serving.
func (h *AuthHandler) WithStorage(p *storage.Provider) *AuthHandler {
	h.store = p
	return h
}

func (h *AuthHandler) setCookies(c *gin.Context, pair *usecase.TokenPair) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessCookieName, pair.AccessToken, accessCookieMaxAge, "/", "", h.cookieSecure, true)
	c.SetCookie(middleware.RefreshCookieName, pair.RefreshToken, refreshCookieMaxAge, "/", "", h.cookieSecure, true)
}

func (h *AuthHandler) clearCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(middleware.AccessCookieName, "", -1, "/", "", h.cookieSecure, true)
	c.SetCookie(middleware.RefreshCookieName, "", -1, "/", "", h.cookieSecure, true)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
		FullName string `json:"full_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	profile, err := h.authUsecase.Register(c.Request.Context(), req.Email, req.Password, req.FullName)
	switch {
	case errors.Is(err, domain.ErrEmailTaken):
		response.Conflict(c, "Email sudah terdaftar")
		return
	case errors.Is(err, usecase.ErrInvalidEmail):
		response.BadRequest(c, "Format email tidak valid", "invalid_email")
		return
	case errors.Is(err, usecase.ErrWeakPassword):
		response.BadRequest(c, "Password minimal 8 karakter", "weak_password")
		return
	case err != nil:
		response.InternalError(c, "Gagal mendaftar")
		return
	}

	response.Created(c, "Registrasi berhasil", profile)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	pair, err := h.authUsecase.Login(c.Request.Context(), req.Email, req.Password, c.Request.UserAgent())
	switch {
	case errors.Is(err, usecase.ErrInvalidCredentials):
		response.Error(c, http.StatusUnauthorized, "Email atau password salah", "invalid_credentials")
		return
	case errors.Is(err, usecase.ErrAccountInactive):
		response.Error(c, http.StatusUnauthorized, "Akun tidak aktif", "account_inactive")
		return
	case errors.Is(err, usecase.ErrPasswordReset):
		response.Error(c, http.StatusUnauthorized, "Akun perlu reset password", "password_reset_required")
		return
	case err != nil:
		response.InternalError(c, "Gagal login")
		return
	}

	h.setCookies(c, pair)
	response.OK(c, "Login berhasil", pair.User)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	raw, err := c.Cookie(middleware.RefreshCookieName)
	if err != nil || raw == "" {
		response.Unauthorized(c, "Refresh token tidak ditemukan")
		return
	}

	pair, err := h.authUsecase.Refresh(c.Request.Context(), raw, c.Request.UserAgent())
	if err != nil {
		h.clearCookies(c)
		response.Unauthorized(c, "Refresh token tidak valid")
		return
	}

	h.setCookies(c, pair)
	response.OK(c, "Token diperbarui", pair.User)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	raw, _ := c.Cookie(middleware.RefreshCookieName)
	_ = h.authUsecase.Logout(c.Request.Context(), raw)
	h.clearCookies(c)
	response.OK(c, "Logout berhasil", nil)
}

func (h *AuthHandler) GetProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	profile, err := h.authUsecase.Me(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Gagal mengambil profil")
		return
	}
	if profile == nil {
		response.NotFound(c, "Profil tidak ditemukan")
		return
	}

	response.OK(c, "Profil berhasil diambil", profile)
}

func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		FullName  string  `json:"full_name"`
		Phone     *string `json:"phone"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	fullName := strings.TrimSpace(req.FullName)
	if fullName == "" {
		response.BadRequest(c, "Nama lengkap wajib diisi", "validation")
		return
	}
	// An empty phone clears the field (NULL) instead of storing a blank string.
	phone := req.Phone
	if phone != nil {
		if trimmed := strings.TrimSpace(*phone); trimmed == "" {
			phone = nil
		} else {
			phone = &trimmed
		}
	}

	profile, err := h.authUsecase.UpdateProfile(c.Request.Context(), userID, fullName, phone, req.AvatarURL)
	if err != nil {
		response.InternalError(c, "Gagal memperbarui profil")
		return
	}
	if profile == nil {
		response.NotFound(c, "Profil tidak ditemukan")
		return
	}

	response.OK(c, "Profil berhasil diperbarui", profile)
}

// DeleteAccount handles permanent deletion and anonymization (UU PDP Pasal 43).
func (h *AuthHandler) DeleteAccount(c *gin.Context) {
	if h.anonymizeSvc == nil {
		response.InternalError(c, "Modul kepatuhan PDP belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)

	var req struct {
		Password string `json:"password" binding:"required"`
		Reason   string `json:"reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Password konfirmasi wajib diisi", "validation")
		return
	}

	valid, err := h.anonymizeSvc.VerifyPassword(c.Request.Context(), userID, req.Password)
	if err != nil || !valid {
		response.Unauthorized(c, "Password konfirmasi salah")
		return
	}

	res, err := h.anonymizeSvc.AnonymizeUserData(c.Request.Context(), userID, &userID, req.Reason)
	if err != nil {
		response.InternalError(c, "Gagal memproses penghapusan akun. Silakan coba beberapa saat lagi.")
		return
	}

	h.clearCookies(c)
	response.OK(c, "Akun berhasil dihapus dan data pribadi dianonimkan sesuai UU PDP No. 27/2022", res)
}

// ExportData handles personal data export for Data Portability (UU PDP Pasal 21).
func (h *AuthHandler) ExportData(c *gin.Context) {
	if h.anonymizeSvc == nil {
		response.InternalError(c, "Modul kepatuhan PDP belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)

	data, err := h.anonymizeSvc.ExportUserData(c.Request.Context(), userID)
	if err != nil {
		response.InternalError(c, "Gagal mengekspor data pengguna. Silakan coba beberapa saat lagi.")
		return
	}

	c.Header("Content-Disposition", "attachment; filename=fintrack-data-export.json")
	response.OK(c, "Data pribadi berhasil diekspor", data)
}

// ─── Avatar upload + serving ──────────────────────────────────

// UploadAvatar handles POST /api/v1/me/avatar (multipart form field "file").
// The image is validated by magic bytes, stored under a per-user key, and the
// resulting URL is persisted on the profile.
func (h *AuthHandler) UploadAvatar(c *gin.Context) {
	if h.store == nil {
		response.InternalError(c, "Penyimpanan berkas belum dikonfigurasi")
		return
	}
	userID := c.MustGet("user_id").(uuid.UUID)

	file, err := c.FormFile("file")
	if err != nil {
		response.BadRequest(c, "Berkas gambar wajib diunggah", "invalid_payload")
		return
	}
	if file.Size > MaxAvatarBytes {
		response.BadRequest(c, "Ukuran berkas melebihi 2 MB", "avatar_too_large")
		return
	}
	content, err := readUpload(file)
	if err != nil {
		response.BadRequest(c, "Gagal membaca berkas", "invalid_payload")
		return
	}
	// Magic-byte validation: only real JPG/PNG/WEBP images pass, so a renamed
	// script or PDF is rejected regardless of its declared content type.
	ext, err := payment.ValidateAvatar(content)
	if err != nil {
		response.BadRequest(c, "Format gambar tidak didukung (gunakan JPG, PNG, atau WEBP)", "invalid_image")
		return
	}

	store, err := h.store.Active(c.Request.Context())
	if err != nil {
		response.InternalError(c, "Penyimpanan berkas tidak tersedia")
		return
	}
	filename := userID.String() + ext
	if err := store.Put(c.Request.Context(), storage.AvatarKey(filename), content, storage.ContentTypeFor(filename)); err != nil {
		response.InternalError(c, "Gagal menyimpan foto profil")
		return
	}

	url := avatarURLFor(filename)
	if _, err := h.authUsecase.SetAvatarURL(c.Request.Context(), userID, url); err != nil {
		response.InternalError(c, "Gagal menyimpan foto profil")
		return
	}

	response.OK(c, "Foto profil berhasil diperbarui", gin.H{"avatar_url": url})
}

// ServeAvatar handles GET /api/v1/avatars/:file, where :file is
// "<uuid>.<ext>". It is intentionally public so an <img> tag can load the
// avatar without the auth cookie; the UUID is unguessable, and the lookup is
// scoped to the owning user so a missing file is a 404 rather than a leak.
func (h *AuthHandler) ServeAvatar(c *gin.Context) {
	if h.store == nil {
		response.NotFound(c, "Berkas tidak ditemukan")
		return
	}
	filename := c.Param("file")
	idPart := strings.TrimSuffix(filename, filepath.Ext(filename))
	userID, err := uuid.Parse(idPart)
	if err != nil {
		response.NotFound(c, "Berkas tidak ditemukan")
		return
	}
	profile, err := h.authUsecase.Me(c.Request.Context(), userID)
	if err != nil || profile == nil || profile.AvatarURL == nil || *profile.AvatarURL == "" {
		response.NotFound(c, "Berkas tidak ditemukan")
		return
	}
	// Only serve the file the profile actually points at (defends against
	// probing other extensions for the same user). The stored URL carries a
	// cache-busting `?v=` query that must be stripped before comparing.
	storedPath := *profile.AvatarURL
	if i := strings.IndexByte(storedPath, '?'); i >= 0 {
		storedPath = storedPath[:i]
	}
	if filepath.Base(storedPath) != filename {
		response.NotFound(c, "Berkas tidak ditemukan")
		return
	}

	obj, err := h.store.Open(c.Request.Context(), storage.AvatarKey(filename))
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			response.NotFound(c, "Berkas tidak ditemukan")
			return
		}
		response.InternalError(c, "Gagal membaca berkas")
		return
	}
	defer obj.Body.Close()

	if obj.ContentType != "" {
		c.Header("Content-Type", obj.ContentType)
	}
	if obj.Size > 0 {
		c.Header("Content-Length", strconv.FormatInt(obj.Size, 10))
	}
	c.Header("Cache-Control", "public, max-age=300")
	c.Header("Content-Disposition", "inline")
	c.Status(http.StatusOK)
	_, _ = io.Copy(c.Writer, obj.Body)
}

// avatarURLFor builds the public, cache-busted URL for an avatar filename.
func avatarURLFor(filename string) string {
	return "/api/v1/avatars/" + filename + "?v=" + strconv.FormatInt(time.Now().Unix(), 10)
}

// readUpload reads a bounded multipart file into memory.
func readUpload(fh *multipart.FileHeader) ([]byte, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, MaxAvatarBytes+1))
}
