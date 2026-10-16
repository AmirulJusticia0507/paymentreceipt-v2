package authservice

import (
	"testing"
	"time"
)

// newTestLimiter membuat limiter dengan jam yang bisa dikontrol.
func newTestLimiter(max int, window time.Duration) (*LoginLimiter, *time.Time) {
	now := time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)
	l := NewLoginLimiter(max, window)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestLimiterBlocksAfterMaxAttempts(t *testing.T) {
	l, _ := newTestLimiter(5, 15*time.Minute)

	// Lima kegagalan pertama belum memblokir.
	for i := 0; i < 5; i++ {
		if retry := l.Check("budi", "1.2.3.4"); retry > 0 {
			t.Fatalf("Check() setelah %d kegagalan = %d, want 0", i+1, retry)
		}
		l.Fail("budi", "1.2.3.4")
	}

	// Failure kelima memicu blokir.
	if retry := l.Check("budi", "1.2.3.4"); retry <= 0 {
		t.Error("Check() setelah 5 kegagalan = 0, want > 0 (harus diblokir)")
	}
}

func TestLimiterCountsFromFirstFailure(t *testing.T) {
	// Window dihitung sejak kegagalan PERTAMA. Kalau dihitung sejak yang
	// terakhir, penyerang bisa mencoba selamanya selama menyisipkan jeda.
	l, now := newTestLimiter(3, 10*time.Minute)

	l.Fail("budi", "1.2.3.4")
	*now = now.Add(4 * time.Minute)
	l.Fail("budi", "1.2.3.4")
	l.Fail("budi", "1.2.3.4") // kegagalan ketiga -> diblokir

	if l.Check("budi", "1.2.3.4") <= 0 {
		t.Error("Check() = 0, want diblokir")
	}

	// Lewat 10 menit sejak kegagalan pertama: jendela selesai, harus bebas lagi.
	*now = now.Add(11 * time.Minute)
	if retry := l.Check("budi", "1.2.3.4"); retry > 0 {
		t.Errorf("Check() setelah jendela berakhir = %d, want 0", retry)
	}
}

func TestLimiterSuccessResetsCount(t *testing.T) {
	// Tanpa reset, user yang sempat salah ketik lalu berhasil akan tetap
	// terkunci padahal login yang berhasilnya sudah benar.
	l, _ := newTestLimiter(3, 15*time.Minute)

	l.Fail("budi", "1.2.3.4")
	l.Fail("budi", "1.2.3.4")
	l.Success("budi", "1.2.3.4")

	l.Fail("budi", "1.2.3.4")
	l.Fail("budi", "1.2.3.4")

	// Dua kegagalan lagi setelah reset = total 2, belum sampai batas 3.
	if retry := l.Check("budi", "1.2.3.4"); retry > 0 {
		t.Errorf("Check() = %d, want 0 (harus di-reset oleh Success)", retry)
	}
}

func TestLimiterIsPerEntity(t *testing.T) {
	// Membatasi per username saja tidak cukup (satu IP bisa mencoba banyak
	// username), dan per IP saja juga tidak cukup (satu IP NAT dipakai banyak
	// orang). Kombinasi keduanya menutup kedua kasus.
	l, _ := newTestLimiter(3, 15*time.Minute)

	for i := 0; i < 5; i++ {
		l.Fail("budi", "1.2.3.4")
	}

	// Username lain dari IP yang sama: tidak boleh ikut terkunci.
	if retry := l.Check("siti", "1.2.3.4"); retry > 0 {
		t.Errorf("Check(user lain, IP sama) = %d, want 0", retry)
	}

	// Username sama dari IP lain: juga tidak boleh ikut terkunci.
	if retry := l.Check("budi", "5.6.7.8"); retry > 0 {
		t.Errorf("Check(user sama, IP lain) = %d, want 0", retry)
	}
}

func TestLimiterUnblocksAfterWindow(t *testing.T) {
	l, now := newTestLimiter(2, 5*time.Minute)

	l.Fail("budi", "1.2.3.4")
	l.Fail("budi", "1.2.3.4")

	retry := l.Check("budi", "1.2.3.4")
	if retry <= 0 {
		t.Fatal("Check() = 0, want diblokir")
	}
	// Retry-After tidak boleh melebihi jendela.
	if retry > 5*60 {
		t.Errorf("Check() = %d, want <= 300", retry)
	}

	*now = now.Add(6 * time.Minute)
	if retry := l.Check("budi", "1.2.3.4"); retry > 0 {
		t.Errorf("Check() setelah jendela = %d, want 0", retry)
	}
}

func TestLimiterCleanup(t *testing.T) {
	// Map harus bisa dibersihkan supaya tidak tumbuh tanpa batas di server
	// yang lama hidup.
	l, now := newTestLimiter(3, 10*time.Minute)

	l.Fail("budi", "1.2.3.4")
	l.Fail("siti", "5.6.7.8")
	l.Fail("andi", "9.9.9.9")

	if l.Len() != 3 {
		t.Fatalf("Len() = %d, want 3", l.Len())
	}

	*now = now.Add(11 * time.Minute)

	if removed := l.Cleanup(); removed != 3 {
		t.Errorf("Cleanup() = %d, want 3", removed)
	}
	if l.Len() != 0 {
		t.Errorf("Len() setelah Cleanup = %d, want 0", l.Len())
	}
}

func TestLimiterDoesNotGrowUnboundedAfterExpiry(t *testing.T) {
	// Entri yang sudah lewat jendela harus hilang saat dicek, bukan menumpuk.
	l, now := newTestLimiter(5, time.Minute)

	for i := 0; i < 3; i++ {
		l.Fail("budi", "1.2.3.4")
	}
	*now = now.Add(2 * time.Minute)
	l.Check("budi", "1.2.3.4")

	if l.Len() != 0 {
		t.Errorf("Len() = %d, want 0 (entri kedaluwarsa should've been dropped)", l.Len())
	}
}
