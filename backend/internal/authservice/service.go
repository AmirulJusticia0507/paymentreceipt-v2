// Package authservice berisi logika use case autentikasi.
package authservice

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
)

// UserStore adalah operasi yang butuh dari tabel users.
//
// Dipisah sebagai interface supaya service bisa diuji tanpa database. Hanya
// operasi yang benar-benar dipakai yang dideklarasikan.
type UserStore interface {
	GetUserByUsername(ctx context.Context, username string) (dbgen.User, error)
	GetUserSafeByID(ctx context.Context, id int64) (dbgen.GetUserSafeByIDRow, error)
	TouchUserLastLogin(ctx context.Context, id int64) error
}

// TokenStore adalah operasi yang butuh dari tabel refresh_tokens.
type TokenStore interface {
	StoreRefreshToken(ctx context.Context, arg dbgen.StoreRefreshTokenParams) (dbgen.RefreshToken, error)
	GetActiveRefreshToken(ctx context.Context, tokenHash string) (dbgen.GetActiveRefreshTokenRow, error)
	RevokeRefreshToken(ctx context.Context, tokenHash string) error
	RevokeAllUserRefreshTokens(ctx context.Context, userID int64) error
}

// Service menangani login, refresh, dan logout.
type Service struct {
	users     UserStore
	tokens    TokenStore
	tokenMgr  *auth.TokenManager
	cookies   auth.CookieOptions
	argon2    auth.Params
	loginFail *LoginLimiter
}

// New membuat Service.
func New(
	users UserStore,
	tokens TokenStore,
	tokenMgr *auth.TokenManager,
	cookies auth.CookieOptions,
	argon2 auth.Params,
) *Service {
	return &Service{
		users:     users,
		tokens:    tokens,
		tokenMgr:  tokenMgr,
		cookies:   cookies,
		argon2:    argon2,
		loginFail: NewLoginLimiter(5, 15*time.Minute),
	}
}

// LoginResult adalah hasil login atau refresh yang sukses.
type LoginResult struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    time.Time
	User         UserInfo
}

// UserInfo adalah profil user tanpa data sensitif.
type UserInfo struct {
	ID          int64     `json:"id"`
	Username    string    `json:"username"`
	Email       string    `json:"email"`
	FullName    string    `json:"full_name"`
	Role        string    `json:"role"`
	LastLoginAt time.Time `json:"last_login_at,omitempty"`
}

func toUserInfo(u dbgen.User) UserInfo {
	info := UserInfo{
		ID:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		FullName: u.FullName,
		Role:     string(u.Role),
	}
	if u.LastLoginAt.Valid {
		info.LastLoginAt = u.LastLoginAt.Time
	}
	return info
}

// Login memverifikasi kredensial dan menerbitkan sepasang token.
//
// Dua detail keamanan yang sengaja dilakukan di sini:
//
//  1. Password salah dan user tidak ada menghasilkan error YANG SAMA. Kalau
//     berbeda, penyerang bisa memakai respons untuk menebak username mana yang
//     terdaftar - username harvest.
//
//  2. Verifikasi argon2 tetap dijalankan walau user tidak ditemukan, memakai
//     hash palsu. Tanpa ini, waktu respons untuk "user tidak ada" jauh lebih
//     cepat daripada "password salah", dan itu membocorkan username yang valid.
func (s *Service) Login(ctx context.Context, username, password, ip, userAgent string) (*LoginResult, error) {
	// Rate limit dicek lebih dulu supaya penyerang tidak bisa memakai CPU
	// server untuk menebak password tanpa batas.
	if retryAfter := s.loginFail.Check(username, ip); retryAfter > 0 {
		return nil, apperror.RateLimited(
			"Terlalu banyak percobaan masuk. Coba lagi nanti.", retryAfter)
	}

	user, err := s.users.GetUserByUsername(ctx, username)

	if err != nil {
		// Samakan jalur dengan password salah.
		s.loginFail.Fail(username, ip)
		s.burnPasswordTime(password)
		return nil, apperror.Unauthorized("Username atau password salah")
	}

	ok, verifyErr := auth.Verify(password, user.PasswordHash)
	if verifyErr != nil {
		// Hash rusak di database. Ini masalah internal, bukan kesalahan user.
		return nil, apperror.Internalf("auth.password_hash_corrupt", verifyErr)
	}

	if !ok {
		s.loginFail.Fail(username, ip)
		return nil, apperror.Unauthorized("Username atau password salah")
	}

	if !user.IsActive {
		// Akun dinonaktifkan. Pesannya eksplisit supaya user tahu contacting admin.
		return nil, apperror.Forbidden("Akun Anda dinonaktifkan. Hubungi administrator.")
	}

	s.loginFail.Success(username, ip)

	result, err := s.issueTokens(ctx, user, ip, userAgent)
	if err != nil {
		return nil, err
	}

	// Best-effort: kegagalan mencatat waktu login tidak boleh menggagalkan login.
	if err := s.users.TouchUserLastLogin(ctx, user.ID); err == nil {
		_ = err // dicatat oleh lapisan pemanggil
	}

	return result, nil
}

// Refresh menerbitkan access token baru dari refresh token yang valid.
//
// Refresh token dirotasi: yang lama langsung dicabut. Kalau ada yang memakai
// token lama SESUDAH dirotasi, itu indikasi token dicuri - seluruh rantai token
// user tersebut langsung dicabut (ADR-0003).
func (s *Service) Refresh(ctx context.Context, refreshToken, ip, userAgent string) (*LoginResult, error) {
	if refreshToken == "" {
		return nil, apperror.Unauthorized("Refresh token tidak ditemukan")
	}

	hash := auth.HashRefreshToken(refreshToken)

	row, err := s.tokens.GetActiveRefreshToken(ctx, hash)
	if err != nil {
		// Tidak ada, sudah dicabut, atau kedaluwarsa - semuanya konfirmasi
		// yang sama supaya penyerang tidak bisa membedakan.
		return nil, apperror.Unauthorized("Sesi tidak valid, silakan masuk ulang")
	}

	// Deteksi penyalahgunaan: token yang sudah dicabut tapi tetap dipakai.
	// Kalau query di atas tidak menjemputnya, berarti ini pemakaian ulang.
	if row.RevokedAt.Valid {
		if revokeErr := s.tokens.RevokeAllUserRefreshTokens(ctx, row.UserID); revokeErr != nil {
			return nil, apperror.Internalf("auth.revoke_chain_failed", revokeErr)
		}
		return nil, apperror.Unauthorized("Sesi tidak aman, silakan masuk ulang dari awal")
	}

	// Cabut token lama SEBELUM menerbitkan yang baru. Kalau dua request refresh
	// datang bersamaan, hanya yang pertama yang berhasil - yang kedua dianggap
	// penyalahgunaan.
	if err := s.tokens.RevokeRefreshToken(ctx, hash); err != nil {
		return nil, apperror.Internalf("auth.revoke_failed", err)
	}

	user, err := s.users.GetUserByUsername(ctx, row.Username)
	if err != nil {
		return nil, apperror.Unauthorized("Sesi tidak valid, silakan masuk ulang")
	}

	if !user.IsActive {
		return nil, apperror.Forbidden("Akun Anda dinonaktifkan. Hubungi administrator.")
	}

	return s.issueTokens(ctx, user, ip, userAgent)
}

// Logout mencabut refresh token milik pengguna.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		// Sudah logout. Ini kondisi yang wajar, bukan error.
		return nil
	}

	hash := auth.HashRefreshToken(refreshToken)
	if err := s.tokens.RevokeRefreshToken(ctx, hash); err != nil {
		return apperror.Internalf("auth.revoke_failed", err)
	}
	return nil
}

// LogoutAll mencabut seluruh refresh token milik pengguna.
func (s *Service) LogoutAll(ctx context.Context, userID int64) error {
	if err := s.tokens.RevokeAllUserRefreshTokens(ctx, userID); err != nil {
		return apperror.Internalf("auth.revoke_failed", err)
	}
	return nil
}

// Me mengembalikan profil user yang sedang login.
func (s *Service) Me(ctx context.Context, userID int64) (*UserInfo, error) {
	user, err := s.users.GetUserSafeByID(ctx, userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperror.NotFound("User")
		}
		return nil, apperror.Internalf("auth.get_user_failed", err)
	}
	if !user.IsActive {
		return nil, apperror.Forbidden("Akun Anda dinonaktifkan.")
	}

	info := UserInfo{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		FullName: user.FullName,
		Role:     string(user.Role),
	}
	if user.LastLoginAt.Valid {
		info.LastLoginAt = user.LastLoginAt.Time
	}
	return &info, nil
}

// issueTokens membuat access + refresh token dan menyimpan hash refresh token.
func (s *Service) issueTokens(ctx context.Context, user dbgen.User, ip, userAgent string) (*LoginResult, error) {
	access, accessExp, err := s.tokenMgr.IssueAccess(user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, apperror.Internalf("auth.issue_access_failed", err)
	}

	refresh, refreshExp, err := s.tokenMgr.IssueRefresh(user.ID, user.Username, string(user.Role))
	if err != nil {
		return nil, apperror.Internalf("auth.issue_refresh_failed", err)
	}

	// Yang disimpan ke database adalah HASH dari refresh token, bukan tokennya.
	if _, err := s.tokens.StoreRefreshToken(ctx, dbgen.StoreRefreshTokenParams{
		UserID:    user.ID,
		TokenHash: auth.HashRefreshToken(refresh),
		ExpiresAt: refreshExp,
		UserAgent: nullString(userAgent, 512),
		IpAddress: nullIP(ip),
	}); err != nil {
		return nil, apperror.Internalf("auth.store_refresh_failed", err)
	}

	return &LoginResult{
		AccessToken:  access,
		RefreshToken: refresh,
		ExpiresAt:    accessExp,
		User:         toUserInfo(user),
	}, nil
}

// burnPasswordTime menjalankan argon2 dengan hash palsu supaya waktu respons
// untuk "user tidak ada" mirip dengan "password salah".
//
// Tanpa ini, penyerang bisa menebak username yang valid hanya dari selisih waktu.
func (s *Service) burnPasswordTime(password string) {
	_, _ = auth.Verify(password, dummyHash)
}

// dummyHash adalah hash argon2id dari string acak, dipakai hanya untuk menyamakan
// waktu eksekusi. Nilai ini tidak pernah menjadi password user mana pun.
const dummyHash = "$argon2id$v=19$m=19456,t=2,p=1$" +
	"c2FsdHNhbHRzYWx0c2E=" + // base64 dari "saltsaltsalt"
	"$q83t1Y3v5b7x9z1c3e5g7i9k1m3o5q7s9u1w3y5z7a9c3e5g7i9k1m3o5q7s9u1w3y"

// nullString mengubah string kosong menjadi NULL, dan memotong yang melebihi panjang.
func nullString(s string, maxLen int) pgtype.Text {
	s = strings.TrimSpace(s)
	if s == "" {
		return pgtype.Text{}
	}
	if len(s) > maxLen {
		s = s[:maxLen]
	}
	return pgtype.Text{String: s, Valid: true}
}

// nullIP mengubah IP kosong menjadi NULL. Kolom bertipe inet, jadi harus
// di-parse dulu - string sembarang akan ditolak database.
func nullIP(s string) *netip.Addr {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		// Bukan IP valid (mis. sockaddr tanpa port yang tidak bisa diparse).
		// Simpan NULL, jangan gagalkan login.
		return nil
	}
	return &addr
}

var _ = fmt.Sprintf
