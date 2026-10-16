package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
)

// Claims adalah isi access token.
//
// - Token SINGKAT dan tidak bisa dicabut sebelum expired (lihat ADR-0003),
// jadi isinya harus seminimal mungkin. Jangan pernah menaruh data sensitif di
// sini: isi JWT hanya di-encode, bukan di-encode.
type Claims struct {
	jwt.RegisteredClaims
	Username string `json:"username"`
	Role     string `json:"role"`
	// TokenType membedakan access token dari refresh token kalau suatu saat
	// keduanya dipakai pada jalur yang sama.
	TokenType string `json:"typ"`
}

// TokenTypeAccess dan TokenTypeRefresh.
const (
	TokenTypeAccess  = "access"
	TokenTypeRefresh = "refresh"
)

// Error yang mungkin dikembalikan TokenManager.
var (
	ErrTokenExpired     = errors.New("token sudah kedaluwarsa")
	ErrTokenInvalid     = errors.New("token tidak valid")
	ErrTokenWrongType   = errors.New("tipe token tidak sesuai")
	ErrTokenNotYetValid = errors.New("token belum berlaku")
)

// TokenManager memb Issuing dan memverifikasi JWT.
type TokenManager struct {
	secret     []byte
	issuer     string
	accessTTL  time.Duration
	refreshTTL time.Duration
	// leeway menerima token yang "sedikit" kedaluwarsa atau belum valid.
	// 30 detik menutupi perbedaan jam antara server dan issuer.
	leeway time.Duration
}

// NewTokenManager membuat TokenManager dari konfigurasi.
func NewTokenManager(cfg config.AuthConfig) *TokenManager {
	return &TokenManager{
		secret:     cfg.JWTSecret,
		issuer:     cfg.Issuer,
		accessTTL:  cfg.AccessTTL,
		refreshTTL: cfg.RefreshTTL,
		leeway:     30 * time.Second,
	}
}

// TTLAccess mengembalikan masa berlaku access token.
func (m *TokenManager) TTLAccess() time.Duration { return m.accessTTL }

// TTLRefresh mengembalikan masa berlaku refresh token.
func (m *TokenManager) TTLRefresh() time.Duration { return m.refreshTTL }

// IssueAccess membuat access token baru untuk user.
func (m *TokenManager) IssueAccess(userID int64, username, role string) (string, time.Time, error) {
	return m.issue(userID, username, role, TokenTypeAccess, m.accessTTL)
}

// IssueRefresh membuat refresh token JWT.
//
// Refresh token disimpan sebagai hash di database, tapi yang dikirim ke klien
// tetap JWT supaya tidak perlu lookup database hanya untuk memvalidasi format.
func (m *TokenManager) IssueRefresh(userID int64, username, role string) (string, time.Time, error) {
	return m.issue(userID, username, role, TokenTypeRefresh, m.refreshTTL)
}

func (m *TokenManager) issue(userID int64, username, role, typ string, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(ttl)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   fmt.Sprintf("%d", userID),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			// ID unik per token. Berguna untuk log dan deteksi token replay.
			ID: fmt.Sprintf("%d-%d", userID, now.UnixNano()),
		},
		Username:  username,
		Role:      role,
		TokenType: typ,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign token: %w", err)
	}

	return signed, exp, nil
}

// Parse memverifikasi signature dan claim token, lalu mengembalikan isinya.
//
// Verifikasi algoritma WAJIB dibatasi ke HS256. Tanpa itu, penyerang bisa
// mengirim token beralgoritme "none" atau asymmetric dengan public key yang
// dia miliki sendiri - serangan klasik yang menjatuhkan banyak library JWT.
func (m *TokenManager) Parse(tokenString string) (*Claims, error) {
	claims := &Claims{}

	parsed, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(t *jwt.Token) (any, error) {
			// Hanya HS256 yang diterima.
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("%w: algoritme %v tidak diizinkan", ErrTokenInvalid, t.Header["alg"])
			}
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(m.issuer),
		jwt.WithLeeway(m.leeway),
		jwt.WithTimeFunc(func() time.Time { return time.Now() }),
	)

	if err != nil {
		switch {
		case errors.Is(err, jwt.ErrTokenExpired):
			return nil, ErrTokenExpired
		case errors.Is(err, jwt.ErrTokenNotValidYet):
			return nil, ErrTokenNotYetValid
		default:
			return nil, fmt.Errorf("%w: %v", ErrTokenInvalid, err)
		}
	}

	if !parsed.Valid {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// ParseAccess memverifikasi token dan memastikan tipenya access.
//
// Pemisahan ini penting: kalau refresh token bisa dipakai sebagai access token,
// // rantai refresh bisa dilewati begitu saja.
func (m *TokenManager) ParseAccess(tokenString string) (*Claims, error) {
	claims, err := m.Parse(tokenString)
	if err != nil {
		return nil, err
	}
	if claims.TokenType != TokenTypeAccess {
		return nil, ErrTokenWrongType
	}
	return claims, nil
}

// UserID mengembalikan ID user dari claims.
func (c *Claims) UserID() (int64, error) {
	if c.Subject == "" {
		return 0, fmt.Errorf("%w: subject kosong", ErrTokenInvalid)
	}

	var id int64
	if _, err := fmt.Sscanf(c.Subject, "%d", &id); err != nil {
		return 0, fmt.Errorf("%w: subject bukan angka: %v", ErrTokenInvalid, err)
	}
	if id <= 0 {
		return 0, fmt.Errorf("%w: user id tidak valid", ErrTokenInvalid)
	}

	return id, nil
}
