// Package authhandler berisi HTTP handler untuk autentikasi.
package authhandler

import (
	"net/http"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/authservice"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// Handler menangani endpoint autentikasi.
type Handler struct {
	svc     *authservice.Service
	cookies auth.CookieOptions
	// refreshTTL dipakai untuk menghitung masa hidup cookie refresh saat menulis.
	// Access token expires lebih dulu, jadi expiry refresh dihitung dari situ
	// ditambah TTL refresh - kalau tidak, cookie refresh akan ikut kedaluwarsa
	// bersamaan dengan access token dan user akan sering ter-logout.
	refreshTTL time.Duration
}

// New membuat Handler.
func New(svc *authservice.Service, cookies auth.CookieOptions, refreshTTL time.Duration) *Handler {
	return &Handler{svc: svc, cookies: cookies, refreshTTL: refreshTTL}
}

// loginRequest adalah body POST /api/v1/auth/login.
type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// updateProfileRequest adalah body PATCH /api/v1/auth/me.
type updateProfileRequest struct {
	FullName string `json:"full_name"`
}

// Login menangani POST /api/v1/auth/login.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	// Username dikembalikan apa adanya supaya pesan errornya jujur. Password
	// justru tidak boleh diberi pesan khusus - kalau ada, penyerang bisa
	// membedakan "format salah" dari "kredensial salah" dan belajar dari
	// perbedaan itu.
	req.Username = strings.TrimSpace(req.Username)

	if req.Username == "" {
		httputil.Problemf(w, r, apperror.FieldValidation("username", "Username wajib diisi", "required"))
		return
	}
	if req.Password == "" {
		httputil.Problemf(w, r, apperror.FieldValidation("password", "Password wajib diisi", "required"))
		return
	}

	result, err := h.svc.Login(r.Context(), req.Username, req.Password, clientIP(r), r.UserAgent())
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	h.writeLoginResponse(w, result, http.StatusOK)
}

type requesterRegistrationRequest struct {
	Username     string `json:"username"`
	Password     string `json:"password"`
	Phone        string `json:"phone"`
	DepartmentID int64  `json:"department_id"`
}

// SignupDepartments lists active departments without exposing administrative fields.
func (h *Handler) SignupDepartments(w http.ResponseWriter, r *http.Request) {
	departments, err := h.svc.DepartmentsForSignup(r.Context())
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	items := make([]map[string]any, 0, len(departments))
	for _, department := range departments {
		items = append(items, map[string]any{"id": department.ID, "kode": department.Kode, "nama": department.Nama})
	}
	httputil.JSONData(w, http.StatusOK, items)
}

// SignupRequester accepts public requester registrations and never creates an active session.
func (h *Handler) SignupRequester(w http.ResponseWriter, r *http.Request) {
	var req requesterRegistrationRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	if err := h.svc.RegisterRequester(r.Context(), req.Username, req.Password, req.Phone, req.DepartmentID); err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	httputil.JSONData(w, http.StatusCreated, map[string]string{
		"message": "Pendaftaran berhasil dikirim. Akun menunggu persetujuan admin.",
	})
}

// Refresh menangani POST /api/v1/auth/refresh.
//
// Tidak butuh access token: justru INI yang dikembalikan. Frontend memanggilnya
// saat access token expired.
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	refreshToken := auth.RefreshTokenFromRequest(r, h.cookies.RefreshName)
	if refreshToken == "" {
		httputil.Problemf(w, r, apperror.Unauthorized("Sesi tidak ditemukan"))
		return
	}

	result, err := h.svc.Refresh(r.Context(), refreshToken, clientIP(r), r.UserAgent())
	if err != nil {
		// Refresh gagal: cookie lama dibersihkan supaya browser berhenti
		// mengirimnya. Kalau dibiarkan, frontend akan mencoba refresh
		// berulang-ulang dengan token yang pasti ditolak.
		h.cookies.ClearRefreshCookie(w)
		h.cookies.ClearAccessCookie(w)
		httputil.Problemf(w, r, err)
		return
	}

	h.writeLoginResponse(w, result, http.StatusOK)
}

// Logout menangani POST /api/v1/auth/logout.
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	refreshToken := auth.RefreshTokenFromRequest(r, h.cookies.RefreshName)

	// Logout selalu sukses, bahkan kalau token sudah tidak ada. Kalau tidak,
	// logout dua kali akan mengembalikan error padahal tidak ada masalah -
	// dan user akan bingung kenapa.
	if err := h.svc.Logout(r.Context(), refreshToken); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	h.cookies.ClearAccessCookie(w)
	h.cookies.ClearRefreshCookie(w)

	httputil.JSONData(w, http.StatusOK, map[string]any{"message": "Berhasil keluar"})
}

// Me menangani GET /api/v1/auth/me.
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		httputil.Problemf(w, r, apperror.Unauthorized("Anda harus masuk terlebih dahulu"))
		return
	}

	info, err := h.svc.Me(r.Context(), identity.UserID)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, info)
}

// UpdateMe menangani PATCH /api/v1/auth/me.
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		httputil.Problemf(w, r, apperror.Unauthorized("Anda harus masuk terlebih dahulu"))
		return
	}

	var req updateProfileRequest
	if err := httputil.DecodeJSON(w, r, &req); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	info, err := h.svc.UpdateFullName(r.Context(), identity.UserID, req.FullName)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, info)
}

// writeLoginResponse menulis token ke cookie dan mengembalikan profil user di body.
//
// Token TIDAK pernah dikembalikan di body - hanya di cookie HttpOnly. Kalau
// dikembalikan di body, frontend cenderung menyimpannya di localStorage, dan
// itu justru menghilangkan perlindungan HttpOnly.
func (h *Handler) writeLoginResponse(w http.ResponseWriter, result *authservice.LoginResult, status int) {
	h.cookies.SetAccessCookie(w, result.AccessToken, result.ExpiresAt)
	h.cookies.SetRefreshCookie(w, result.RefreshToken,
		result.ExpiresAt.Add(h.refreshTTL))

	httputil.JSONData(w, status, map[string]any{
		"user":       result.User,
		"expires_at": result.ExpiresAt,
	})
}

// clientIP mengambil IP klien. RemoteAddr sudah benar karena middleware RealIP
// menormalisasikannya lebih dulu.
func clientIP(r *http.Request) string {
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i > 0 && !strings.Contains(host[i:], "]") {
		host = host[:i]
	}
	return strings.Trim(host, "[]")
}
