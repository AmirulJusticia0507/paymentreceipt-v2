package router

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// minPasswordLen adalah aturan minimum panjang password lewat reset admin.
// Sistem tidak memaksa kombinasi karakter khusus - panjang dianggap cukup
// untuk aplikasi internal dengan akun yang dikontrol admin.
const minPasswordLen = 8

type userHandler struct {
	q      *dbgen.Queries
	params auth.Params
}

// userDTO memetakan baris user ke bentuk respons API. Kolomnya sudah aman
// (query tanpa password_hash) - tidak ada field rahasia yang perlu disaring
// di layer ini.
func userDTO(u dbgen.ListUsersRow) map[string]any {
	return map[string]any{
		"id":            u.ID,
		"username":      u.Username,
		"email":         u.Email,
		"full_name":     u.FullName,
		"role":          u.Role,
		"is_active":     u.IsActive,
		"last_login_at": timestamptzOrNull(u.LastLoginAt),
		"created_at":    u.CreatedAt,
		"updated_at":    u.UpdatedAt,
	}
}

// timestamptzOrNull mengubah kolom timestamp nullable menjadi nilai JSON:
// waktu atau null.
func timestamptzOrNull(t pgtype.Timestamptz) any {
	if !t.Valid {
		return nil
	}
	return t.Time
}

// mountUsers memasang resource /users. SELURUH route di bawahnya admin-only:
// daftar user memuat email dan riwayat login, bukan data untuk semua role.
func mountUsers(r chi.Router, a *auth.Authenticator, cfg config.AuthConfig, q *dbgen.Queries) {
	h := &userHandler{q: q, params: auth.ParamsFrom(cfg)}

	r.Route("/users", func(r chi.Router) {
		r.Use(a.RequireRole(auth.RoleAdmin))

		r.Get("/", h.list)
		r.Post("/{id}/reset-password", h.resetPassword)
	})
}

func (h *userHandler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.q.ListUsers(r.Context())
	if err != nil {
		httputil.Problemf(w, r, apperror.Internal(err))
		return
	}

	// Tanpa paginasi: jumlah user organisasi kecil, dan halaman admin perlu
	// seluruh daftar sekaligus untuk pencarian lokal.
	dtos := make([]map[string]any, 0, len(users))
	for _, u := range users {
		dtos = append(dtos, userDTO(u))
	}
	httputil.JSONData(w, http.StatusOK, dtos)
}

// resetPassword menetapkan password baru untuk satu user dan mencabut semua
// refresh token miliknya - reset password mengeluarkan user dari seluruh
// perangkat, penting saat reset dipakai meredam akun yang dicurigai dicuri.
//
// Access token yang sudah terbit tetap berlaku sampai kedaluwarsa (maksimal
// AUTH_ACCESS_TTL); batas itu yang membatasi sisa sesi lama.
func (h *userHandler) resetPassword(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	// Password sengaja TIDAK di-trim: spasi adalah karakter valid, dan
	// memangkasnya di sini sementara login tidak memangkas akan membuat
	// password yang benar-benar memuat spasi mustahil dipakai.
	if len(body.Password) < minPasswordLen {
		httputil.Problemf(w, r, apperror.FieldValidation("password",
			fmt.Sprintf("Password minimal %d karakter", minPasswordLen), "minlen"))
		return
	}

	user, err := h.q.GetUserSafeByID(r.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.Problemf(w, r, apperror.NotFound("User"))
			return
		}
		httputil.Problemf(w, r, apperror.Internal(err))
		return
	}

	hash, err := auth.Hash(body.Password, h.params)
	if err != nil {
		httputil.Problemf(w, r, apperror.Internalf("auth.password_hash", err))
		return
	}

	if err := h.q.UpdateUserPassword(r.Context(), dbgen.UpdateUserPasswordParams{
		PasswordHash: hash,
		ID:           id,
	}); err != nil {
		httputil.Problemf(w, r, apperror.Internal(err))
		return
	}

	if err := h.q.RevokeAllUserRefreshTokens(r.Context(), id); err != nil {
		httputil.Problemf(w, r, apperror.Internal(err))
		return
	}

	httputil.JSONData(w, http.StatusOK, map[string]any{
		"id":       user.ID,
		"username": user.Username,
		"reset":    true,
	})
}
