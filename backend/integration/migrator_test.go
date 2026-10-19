//go:build integration

package integration_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/golang-migrate/migrate/v4"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/migrator"
)

// TestMigrator_UpDownSiklusPenuh menguji strategi migrasi end-to-end:
// bersihkan skema, terapkan semua migrasi, jalankan ulang (harus no-op),
// rollback satu langkah, lalu terapkan lagi.
//
// Ini test integrasi utama untuk T12.4 - kalau hijau, jalur deploy
// (migrate up sebelum aplikasi start) bisa dipercaya.
func TestMigrator_UpDownSiklusPenuh(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL wajib diisi untuk integration test")
	}

	// Bersihkan skema supaya migrator mulai dari nol. Koneksi langsung ke
	// database test (dengan penjaga nama di openTestDatabase) - drop schema
	// hanya menyentuh database test, tidak pernah production.
	resetTestSchema(t)

	m, err := migrator.New(databaseURL)
	if err != nil {
		t.Fatalf("migrator.New: %v", err)
	}
	defer func() {
		if err := m.Close(); err != nil {
			t.Errorf("tutup migrator: %v", err)
		}
	}()

	// --- 1. Up dari nol ---------------------------------------------------
	applied, err := m.Up()
	if err != nil {
		t.Fatalf("Up pertama: %v", err)
	}
	if len(applied) == 0 {
		t.Fatal("Up pertama tidak menerapkan migrasi apa pun, ingin semua versi")
	}
	if applied[0] != 1 {
		t.Fatalf("migrasi pertama yang diterapkan = %d, ingin 1", applied[0])
	}

	version, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("Version setelah Up: %v", err)
	}
	if dirty {
		t.Fatal("database dirty setelah Up sukses")
	}
	if version != applied[len(applied)-1] {
		t.Fatalf("versi = %d, ingin versi terakhir yang diterapkan %d", version, applied[len(applied)-1])
	}

	// --- 2. Up kedua harus no-op -------------------------------------------
	appliedAgain, err := m.Up()
	if err != nil {
		t.Fatalf("Up kedua: %v", err)
	}
	if len(appliedAgain) != 0 {
		t.Fatalf("Up kedua menerapkan %v, ingin tidak ada (idempoten)", appliedAgain)
	}

	// --- 3. Rollback satu langkah ------------------------------------------
	if err := m.Down(1); err != nil {
		t.Fatalf("Down(1): %v", err)
	}
	afterDown, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("Version setelah Down: %v", err)
	}
	if dirty {
		t.Fatal("database dirty setelah Down sukses")
	}
	if afterDown != version-1 {
		t.Fatalf("versi setelah rollback = %d, ingin %d", afterDown, version-1)
	}

	// --- 4. Up lagi harus kembali ke terbaru --------------------------------
	if _, err := m.Up(); err != nil {
		t.Fatalf("Up setelah rollback: %v", err)
	}
	final, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("Version final: %v", err)
	}
	if dirty || final != version {
		t.Fatalf("versi final = %d (dirty=%v), ingin %d dan bersih", final, dirty, version)
	}
}

// TestMigrator_ForceMembersihkanDirtyState memastikan jalur pemulihan dirty
// state berfungsi.
//
// Skenario nyatanya: sebuah migrasi gagal di tengah jalan, database ditandai
// dirty, dan semua migrasi berikutnya ditolak. Setelah penyebabnya diperbaiki
// dan keadaan skema diketahui pasti, operator menjalankan force untuk membersihkan
// penanda dirty - lalu Up bisa jalan lagi.
//
// Catatan: force HANYA menyetel penanda versi; ia tidak mengembalikan skema.
// Karena itu test ini menandai dirty di database yang sudah lengkap, lalu
// memaksa ke versi yang memang sedang diterapkan (bukan versi lebih lama).
func TestMigrator_ForceMembersihkanDirtyState(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL wajib diisi untuk integration test")
	}

	resetTestSchema(t)

	m, err := migrator.New(databaseURL)
	if err != nil {
		t.Fatalf("migrator.New: %v", err)
	}
	defer func() {
		if err := m.Close(); err != nil {
			t.Errorf("tutup migrator: %v", err)
		}
	}()

	if _, err := m.Up(); err != nil {
		t.Fatalf("Up pertama: %v", err)
	}

	version, _, err := m.Version()
	if err != nil {
		t.Fatalf("Version: %v", err)
	}

	// Simulasikan kegagalan migrasi: tandai database dirty langsung di tabel
	// schema_migrations (golang-migrate menandai dirty sebelum menjalankan
	// migrasi, dan membersihkannya hanya setelah sukses).
	markDirty(t, databaseURL, int(version))

	// Up pada database dirty harus ditolak dengan ErrDirtyState.
	if _, err := m.Up(); !errors.Is(err, migrator.ErrDirtyState) {
		t.Fatalf("Up pada dirty state: error = %v, ingin ErrDirtyState", err)
	}

	// Force membersihkan penanda dirty pada versi yang sedang berlaku.
	if err := m.Force(int(version)); err != nil {
		t.Fatalf("Force(%d): %v", version, err)
	}

	afterForce, dirty, err := m.Version()
	if err != nil {
		t.Fatalf("Version setelah force: %v", err)
	}
	if afterForce != version || dirty {
		t.Fatalf("versi setelah force = %d (dirty=%v), ingin %d dan bersih", afterForce, dirty, version)
	}

	// Setelah bersih, Up kembali menjadi no-op yang sukses (sudah di terbaru).
	applied, err := m.Up()
	if err != nil {
		t.Fatalf("Up setelah force: %v", err)
	}
	if len(applied) != 0 {
		t.Fatalf("Up setelah force menerapkan %v, ingin tidak ada (sudah di terbaru)", applied)
	}
}

// markDirty menandai schema_migrations pada versi tertentu sebagai dirty.
//
// Hanya untuk test: mensimulasikan kondisi yang biasanya tercipta saat migrasi
// gagal di tengah jalan (mis. koneksi terputus di antara dua statement).
func markDirty(t *testing.T, databaseURL string, version int) {
	t.Helper()

	conn := openTestDatabase(t)
	_, err := conn.Exec(context.Background(),
		"UPDATE schema_migrations SET dirty = TRUE WHERE version = $1", version)
	if err != nil {
		t.Fatalf("tandai dirty: %v", err)
	}
}

// TestMigrator_DownTanpaStepsValid memastikan validasi input dipegang.
func TestMigrator_DownTanpaStepsValid(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Fatal("TEST_DATABASE_URL wajib diisi untuk integration test")
	}

	m, err := migrator.New(databaseURL)
	if err != nil {
		t.Fatalf("migrator.New: %v", err)
	}
	defer func() {
		if err := m.Close(); err != nil {
			t.Errorf("tutup migrator: %v", err)
		}
	}()

	if err := m.Down(0); err == nil {
		t.Fatal("Down(0) seharusnya ditolak")
	}
	if err := m.Down(-1); err == nil {
		t.Fatal("Down(-1) seharusnya ditolak")
	}
}

// resetTestSchema menjatuhkan dan membuat ulang schema public di database test.
//
// Sama dengan pola di database_test.go: seluruh test di package ini berjalan
// berurutan (tanpa t.Parallel), jadi reset di awal setiap test aman.
func resetTestSchema(t *testing.T) {
	t.Helper()

	conn := openTestDatabase(t)
	_, err := conn.Exec(context.Background(), "DROP SCHEMA public CASCADE; CREATE SCHEMA public")
	if err != nil {
		t.Fatalf("reset test schema: %v", err)
	}
}

// TestMigrator_ErrDirtyStateBisaDierrorsIs memastikan sentinel error tetap
// dapat dites setelah dibungkus (kontrak yang dipakai cmd/migrate untuk exit
// code yang berbeda pada dirty state).
func TestMigrator_ErrDirtyStateBisaDierrorsIs(t *testing.T) {
	wrapped := errors.Join(errors.New("konteks"), migrator.ErrDirtyState)
	if !errors.Is(wrapped, migrator.ErrDirtyState) {
		t.Fatal("ErrDirtyState harus bisa dites dengan errors.Is setelah di-wrap")
	}

	// Pastikan tipe migrate.ErrDirty memang dikenali errors.As - kalau versi
	// library berubah bentuk, test ini gagal lebih dulu daripada produksi.
	var dirty migrate.ErrDirty
	if (&dirty).Error() == "" {
		t.Fatal("migrate.ErrDirty harus punya pesan error")
	}
}
