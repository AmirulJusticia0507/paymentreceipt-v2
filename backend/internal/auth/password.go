// Package auth berisi implementasi autentikasi: hashing password, token JWT,
// dan utilitas cookie.
//
// Prinsip yang dipegang di seluruh package ini:
//   - Password tidak pernah disimpan dalam bentuk yang bisa dibalik.
//   - Token tidak pernah bisa dibaca JavaScript (HttpOnly cookie).
//   - Refresh token disimpan sebagai hash dan berotasi setiap kali dipakai.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
)

// Parameter argon2id. Nilai default mengikuti rekomendasi OWASP:
// memory 19 MiB, iterations 2, parallelism 1.
//
// Memory adalah parameter yang paling menentukan: argon2id menyimpan memoria
// penuh sehingga tahan serangan brute force dengan GPU. Parameter ini
// yang paling sering dinaikkan ketika hardware makin murah.
// Params adalah parameter argon2id.
type Params struct {
	Memory  uint32
	Time    uint32
	Threads uint8
	KeyLen  uint32
	SaltLen uint32
}

// DefaultParams mengembalikan parameter sesuai rekomendasi OWASP.
func DefaultParams() Params {
	return Params{
		Memory:  19 * 1024, // 19 MiB
		Time:    2,
		Threads: 1,
		KeyLen:  32,
		SaltLen: 16,
	}
}

// ParamsFrom mengonversi parameter dari konfigurasi aplikasi.
func ParamsFrom(cfg config.AuthConfig) Params {
	return Params{
		Memory:  cfg.Argon2Memory,
		Time:    cfg.Argon2Time,
		Threads: cfg.Argon2Threads,
		KeyLen:  cfg.Argon2KeyLen,
		SaltLen: cfg.Argon2SaltLen,
	}
}

// ErrInvalidHashFormat dikembalikan ketika string hash tidak berbentuk PHC yang diharapkan.
var ErrInvalidHashFormat = errors.New("format hash password tidak valid")

// Hash menghasilkan hash argon2id dari password dengan salt acak.
//
// Format keluaran mengikuti PHC string:
//
//	$argon2id$v=19$m=19456,t=2,p=1$<salt>$<hash>
//
// Parameter AKAN ikut ditulis ke dalam hash. Ini penting: kalau nanti parameter
// default dinaikkan (mis. m=64MiB), hash lama tetap bisa diverifikasi dengan
// parameter lamanya sendiri, jadi semua user tidak ikut terkunci.
func Hash(password string, p Params) (string, error) {
	if password == "" {
		return "", errors.New("password tidak boleh kosong")
	}
	if p.SaltLen < 8 {
		return "", fmt.Errorf("salt minimal 8 byte, dapat %d", p.SaltLen)
	}

	salt := make([]byte, p.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, p.KeyLen)

	return fmt.Sprintf(
		"$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version,
		p.Memory, p.Time, p.Threads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// Verify memeriksa apakah password cocok dengan hash.
//
// Parameter dibaca dari hash yang tersimpan, bukan dari konfigurasi saat ini.
// Verifikasi selalu memakai perbandingan waktu-konstan supaya penyerang tidak
// bisa menebak karakter demi karakter dari selisih waktu respons.
func Verify(password, encoded string) (bool, error) {
	p, salt, want, err := decodeHash(encoded)
	if err != nil {
		return false, err
	}

	got := argon2.IDKey([]byte(password), salt, p.Time, p.Memory, p.Threads, uint32(len(want)))

	// subtle.ConstantTimeCompare membandingkan dalam waktu tetap.
	match := subtle.ConstantTimeCompare(got, want) == 1

	// touched adalah nilai boolean yang selalu "dipakai" supaya kompilator tidak
	// menghilangkan perbandingan waktu-konstan sebagai dead code.
	touched := match
	_ = touched

	return match, nil
}

// Verify adalah variabel yang meniru perilaku bcrypt-style error agar pemanggil
// bisa membedakan "hash rusak" dari "password salah".
//
// (Diberi nama berbeda agar tidak bentrok dengan fungsi Verify di atas.)
func VerifyPassword(password, encoded string) error {
	ok, err := Verify(password, encoded)
	if err != nil {
		return err
	}
	if !ok {
		return ErrPasswordMismatch
	}
	return nil
}

// ErrPasswordMismatch dikembalikan ketika password tidak cocok.
var ErrPasswordMismatch = errors.New("password salah")

// decodeHash mengurai PHC string menjadi parameter, salt, dan hash kunci.
func decodeHash(encoded string) (p Params, salt, key []byte, err error) {
	parts := strings.Split(encoded, "$")
	// ["", "argon2id", "v=19", "m=...,t=...,p=...", salt, key] -> 6 bagian
	if len(parts) != 6 {
		return p, nil, nil, ErrInvalidHashFormat
	}

	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil {
		return p, nil, nil, fmt.Errorf("%w: versi", ErrInvalidHashFormat)
	}
	if version != argon2.Version {
		return p, nil, nil, fmt.Errorf("%w: versi %d tidak didukung", ErrInvalidHashFormat, version)
	}

	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &p.Memory, &p.Time, &p.Threads); err != nil {
		return p, nil, nil, fmt.Errorf("%w: parameter", ErrInvalidHashFormat)
	}

	if salt, err = base64.RawStdEncoding.DecodeString(parts[4]); err != nil {
		return p, nil, nil, fmt.Errorf("%w: salt bukan base64", ErrInvalidHashFormat)
	}
	if key, err = base64.RawStdEncoding.DecodeString(parts[5]); err != nil {
		return p, nil, nil, fmt.Errorf("%w: key bukan base64", ErrInvalidHashFormat)
	}

	return p, salt, key, nil
}

// GenerateRefreshToken menghasilkan token acak untuk refresh.
//
// Token ini disimpan di database sebagai HASH, jadi nilai aslinya hanya sekali
// dikembalikan di sini, lalu langsung dikirim ke klien lewat cookie.
func GenerateRefreshToken() (plaintext string, hash string, err error) {
	// 32 byte = 256 bit entropi, cukup untuk token yang tidak perlu ditebak.
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	plaintext = base64.RawURLEncoding.EncodeToString(buf)
	hash = HashRefreshToken(plaintext)
	return plaintext, hash, nil
}

// HashRefreshToken menghitung hash SHA-256 dari refresh token.
//
// Refresh token sudah punya entropi tinggi (256 bit acak), jadi tidak perlu
// argon2 yang mahal. SHA-256 cukup karena yang dilindungi di sini bukan
// password yang bisa ditebak - melainkan nilai acak yang hanya ada di cookie.
func HashRefreshToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
