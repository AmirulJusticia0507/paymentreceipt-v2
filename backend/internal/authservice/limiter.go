package authservice

import (
	"math"
	"sync"
	"time"
)

// LoginLimiter membatasi percobaan login gagal untukentitas tertentu.
//
// Entity = kombinasi username + IP. Membatasi per username saja tidak cukup
// (penyerang bisa mencoba banyak username dari satu IP), dan per IP saja juga
// tidak cukup (satu IP NAT dipakai banyak orang - memblokir satu orang tidak
// boleh menyandera yang lain). Kombinasi keduanya menutup kedua kasus.
//
// Implementasi memakai in-memory map dengan TTL. Cukup untuk satu instance.
// Kalau nanti perlu horizontal scale, ganti backing store ke Redis - interface
// -nya sengaja dibuat kecil supaya penggantiannya tidak-invasive.
type LoginLimiter struct {
	mu       sync.Mutex
	attempts map[string]*loginAttempt
	// maxAttempts jumlah kegagalan yang tolerated sebelum diblokir.
	maxAttempts int
	// window durasidihitung dihitung sejak kegagalan pertama.
	window time.Duration
	// now bisa di-override untuk pengujian.
	now func() time.Time
}

type loginAttempt struct {
	count        int
	firstAt      time.Time
	blockedUntil time.Time
}

// NewLoginLimiter membuat limiter.
//
// maxAttempts = 5, window = 15 menit: sesuaiOWASP cheat sheet untuk
// Zombie lockout yang menyeimbangkan keamanan dan inconvenience.
func NewLoginLimiter(maxAttempts int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{
		attempts:    make(map[string]*loginAttempt),
		maxAttempts: maxAttempts,
		window:      window,
		now:         time.Now,
	}
}

// Check mengembalikan sisa detik pemblokiran, atau 0 bila entity boleh mencoba.
//
// Window dihitung sejak kegagalan PERTAMA, bukan sejak kegagalan terakhir.
// Kalau dihitung sejak yang terakhir, penyerang bisa mencoba selamanya selama
// masih ada jeda di antara percobaan.
func (l *LoginLimiter) Check(username, ip string) int {
	key := entityKey(username, ip)

	l.mu.Lock()
	defer l.mu.Unlock()

	att, ok := l.attempts[key]
	if !ok {
		return 0
	}

	l.pruneLocked(key, att)

	if att.blockedUntil.After(l.now()) {
		remaining := att.blockedUntil.Sub(l.now())
		// Bulatkan ke atas, bukan truncate, dan minimal 1 detik.
		//
		// Kalau di-truncate, sisa 0,4 detik jadi 0 - dan pemanggil memakai
		// nilai 0 sebagai sinyal "tidak diblokir", sehingga blokir dilewati.
		// Bulatkan ke atas supaya Retry-After tidak pernah lebih pendek dari
		// waktu blokir yang sebenarnya.
		secs := int(math.Ceil(remaining.Seconds()))
		if secs < 1 {
			secs = 1
		}
		return secs
	}

	// Sudah lewat masa blokir tapi masih ada jejak kegagalan lama - mulai hitung
	// ulang dari nol agar user yang sudah login berhasil tidak ikut terkunci.
	if !att.blockedUntil.IsZero() {
		delete(l.attempts, key)
	}

	return 0
}

// Fail mencatat satu percobaan gagal.
func (l *LoginLimiter) Fail(username, ip string) {
	key := entityKey(username, ip)

	l.mu.Lock()
	defer l.mu.Unlock()

	att, ok := l.attempts[key]
	if !ok {
		att = &loginAttempt{firstAt: l.now()}
		l.attempts[key] = att
	}

	l.pruneLocked(key, att)

	att.count++
	if att.count >= l.maxAttempts {
		att.blockedUntil = l.now().Add(l.window)
	}
}

// Success menghapus jejak kegagalan setelah login berhasil.
//
// Tanpa ini, user yang sempat salah ketik beberapa kali lalu berhasil tetap
// akan terkunci padahal dia sudah melakukan login yang benar.
func (l *LoginLimiter) Success(username, ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, entityKey(username, ip))
}

// pruneLocked membersihkan entri yang sudah kedaluwarsa.
//
// HARUS dipanggil dengan lock sudah dipegang.
func (l *LoginLimiter) pruneLocked(key string, att *loginAttempt) {
	if !att.firstAt.IsZero() && l.now().Sub(att.firstAt) > l.window {
		delete(l.attempts, key)
		*att = loginAttempt{firstAt: l.now()}
	}
}

// entityKey menyusun kunci unik untuk sebuah kombinasi username dan IP.
func entityKey(username, ip string) string {
	return username + "|" + ip
}

// Cleanup menghapus seluruh entri yang sudah lewat jendela waktunya.
//
// Dijadwalkan berkala supaya map tidak tumbuh tanpa batas di server yang
// hidup. Mengembalikan jumlah entri yang dihapus.
func (l *LoginLimiter) Cleanup() int {
	l.mu.Lock()
	defer l.mu.Unlock()

	removed := 0
	for key, att := range l.attempts {
		if l.now().Sub(att.firstAt) > l.window {
			delete(l.attempts, key)
			removed++
		}
	}
	return removed
}

// Len mengembalikan jumlah entri yang sedang dilacak. Untuk pengujian.
func (l *LoginLimiter) Len() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.attempts)
}

var _ = time.Second
