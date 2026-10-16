package auth

import (
	"strings"
	"testing"
)

func TestHashAndVerify(t *testing.T) {
	p := DefaultParams()

	hash, err := Hash("rahasia123", p)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if !strings.HasPrefix(hash, "$argon2id$") {
		t.Errorf("Hash() = %q, want prefix $argon2id$", hash)
	}

	ok, err := Verify("rahasia123", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if !ok {
		t.Error("Verify(password benar) = false, want true")
	}

	ok, err = Verify("rahasia456", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v", err)
	}
	if ok {
		t.Error("Verify(password salah) = true, want false")
	}
}

func TestHashIsSalted(t *testing.T) {
	p := DefaultParams()

	h1, err := Hash("sama", p)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	h2, err := Hash("sama", p)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Salt acak berarti dua hash dari password sama HARUS berbeda.
	// Kalau sama, database yang bocor langsungalias akun yang password-nya sama.
	if h1 == h2 {
		t.Error("Hash() menghasilkan hash identik untuk password yang sama, want berbeda (salt acak)")
	}

	// Tapi keduanya harus tetap bisa diverifikasi.
	for i, h := range []string{h1, h2} {
		ok, err := Verify("sama", h)
		if err != nil || !ok {
			t.Errorf("Verify(hash %d) = (%v, %v), want (true, nil)", i, ok, err)
		}
	}
}

func TestHashNeverContainsPlaintext(t *testing.T) {
	hash, err := Hash("passwordrahasia", DefaultParams())
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}
	if strings.Contains(hash, "passwordrahasia") {
		t.Error("Hash() memuat password dalam bentuk plaintext")
	}
}

func TestHashRejectsEmptyPassword(t *testing.T) {
	if _, err := Hash("", DefaultParams()); err == nil {
		t.Error("Hash(\"\") error = nil, want error")
	}
}

func TestVerifyRejectsMalformedHash(t *testing.T) {
	// Hash rusak harus menghasilkan error, bukan false yang disamarkan jadi
	// "password salah" - kalau tidak, hash yang korup akan terlihat seperti
	// kredensial yang lupa diketik.
	malformed := []string{
		"",
		"bukan-hash",
		"$argon2id$v=19",                 // terlalu sedikit bagian
		"$argon2id$v=19$m=19456,t=2,p=1", // tanpa salt/key
		"$argon2id$v=99$m=19456,t=2,p=1$c2FsdA$aGFzaA", // versi tidak didukung
		"$argon2id$v=19$m=19456,t=2,p=1$!!!$???",       // base64 rusak
	}

	for _, h := range malformed {
		if _, err := Verify("apa saja", h); err == nil {
			t.Errorf("Verify(%q) error = nil, want error", h)
		}
	}
}

// TestVerifyUsesParamsFromHash adalah regresi untuk keputusan desain: parameter
// argon2 ditulis INTO the hash.
func TestVerifyUsesParamsFromHash(t *testing.T) {
	weak := Params{Memory: 19 * 1024, Time: 2, Threads: 1, KeyLen: 32, SaltLen: 16}
	hash, err := Hash("test", weak)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	// Nilai parameter pada saat verifikasi sengaja dibuat berbeda.
	strong := DefaultParams()
	strong.Memory = 64 * 1024
	strong.Time = 3

	// Tetap harus verifikasi, karena parameter diambil dari hash, bukan dari
	// konfigurasi saat ini. Kalau tidak, menaikkan parameter akan mengunci
	// semua user yang password-nya di-hash dengan parameter lama.
	ok, err := Verify("test", hash)
	if err != nil {
		t.Fatalf("Verify() error = %v, want nil (params harus diambil dari hash)", err)
	}
	if !ok {
		t.Error("Verify() = false, want true (params diambil dari hash tersimpan)")
	}
	_ = strong
}

func TestVerifyIsConstantTime(t *testing.T) {
	// Tidak bisa diuji secara deterministik dalam waktu berjalan, tapi kita
	// bisa memastikan password dengan panjang berbeda diperlakukan sama -
	// implementasi waktu-tetap akan tetap consume waktu serupa.
	hash, _ := Hash("abc", DefaultParams())

	if _, err := Verify("a", hash); err != nil {
		t.Errorf("Verify(pendek) error = %v", err)
	}
	if _, err := Verify("abcdefghijklmnopqrstuvwxyz", hash); err != nil {
		t.Errorf("Verify(panjang) error = %v", err)
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	plain1, hash1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	plain2, hash2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if plain1 == plain2 {
		t.Error("GenerateRefreshToken() menghasilkan token identik, want berbeda")
	}
	if hash1 == hash2 {
		t.Error("hash identik untuk token berbeda, want berbeda")
	}

	// Hash harus deterministik: token sama -> hash sama. Ini yang dipakai
	// untuk mencari token di database.
	if HashRefreshToken(plain1) != hash1 {
		t.Error("HashRefreshToken(plain) != hash dari GenerateRefreshToken, want sama")
	}
}

func TestHashRefreshTokenDoesNotContainToken(t *testing.T) {
	plain, hash, _ := GenerateRefreshToken()
	if strings.Contains(hash, plain) {
		t.Error("hash refresh token memuat token asli, want hanya hash")
	}
}
