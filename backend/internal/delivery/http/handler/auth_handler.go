package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/fintrack/backend/internal/delivery/http/middleware"
	"github.com/fintrack/backend/internal/domain"
	"github.com/fintrack/backend/internal/usecase"
	"github.com/fintrack/backend/internal/user"
	"github.com/fintrack/backend/pkg/response"
)

const (
	accessCookieMaxAge  = 900     // 15 minutes
	refreshCookieMaxAge = 2592000 // 30 days
)

type AuthHandler struct {
	authUsecase  *usecase.AuthUsecase
	cookieSecure bool
	anonymizeSvc *user.AnonymizeService
}

func NewAuthHandler(authUsecase *usecase.AuthUsecase, cookieSecure bool) *AuthHandler {
	return &AuthHandler{authUsecase: authUsecase, cookieSecure: cookieSecure}
}

func (h *AuthHandler) WithAnonymize(anonymizeSvc *user.AnonymizeService) *AuthHandler {
	h.anonymizeSvc = anonymizeSvc
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
		FullName  *string `json:"full_name"`
		AvatarURL *string `json:"avatar_url"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request body", err.Error())
		return
	}

	profile, err := h.authUsecase.UpdateProfile(c.Request.Context(), userID, req.FullName, req.AvatarURL)
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
		response.InternalError(c, "Gagal memproses penghapusan akun: "+err.Error())
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
		response.InternalError(c, "Gagal mengekspor data pengguna: "+err.Error())
		return
	}

	c.Header("Content-Disposition", "attachment; filename=fintrack-data-export.json")
	response.OK(c, "Data pribadi berhasil diekspor", data)
}
