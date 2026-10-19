package migrator

import (
	"errors"
	"testing"
	"testing/fstest"

	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/db/migrations"
)

func TestVersionsAfter(t *testing.T) {
	tests := []struct {
		name   string
		before uint
		after  uint
		want   []uint
	}{
		{name: "dari nol ke terbaru", before: 0, after: 3, want: []uint{1, 2, 3}},
		{name: "tidak ada perubahan", before: 3, after: 3, want: nil},
		{name: "turun versi (force)", before: 5, after: 2, want: nil},
		{name: "database kosong", before: 0, after: 0, want: nil},
		{name: "satu langkah", before: 6, after: 7, want: []uint{7}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := versionsAfter(tt.before, tt.after)
			if len(got) != len(tt.want) {
				t.Fatalf("versionsAfter(%d, %d) = %v, ingin %v", tt.before, tt.after, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("versionsAfter(%d, %d) = %v, ingin %v", tt.before, tt.after, got, tt.want)
				}
			}
		})
	}
}

// TestNewFromFS_MenolakFSKosong memastikan migrator gagal jelas (bukan panic)
// saat sumber migrasi tidak berisi berkas apa pun.
func TestNewFromFS_MenolakFSKosong(t *testing.T) {
	_, err := NewFromFS(fstest.MapFS{}, "postgres://user:pass@localhost:5432/nowhere?sslmode=disable")
	if err == nil {
		t.Fatal("NewFromFS dengan FS kosong seharusnya gagal")
	}
}

// TestNewFromFS_MenolakURLInvalid memastikan URL database yang tidak valid
// ditolak dengan error yang jelas.
func TestNewFromFS_MenolakURLInvalid(t *testing.T) {
	fsys := fstest.MapFS{
		"0001_contoh.up.sql":   &fstest.MapFile{Data: []byte("SELECT 1;")},
		"0001_contoh.down.sql": &fstest.MapFile{Data: []byte("SELECT 1;")},
	}
	_, err := NewFromFS(fsys, "bukan-url-postgres")
	if err == nil {
		t.Fatal("NewFromFS dengan URL invalid seharusnya gagal")
	}
}

// TestEmbeddedFS_Lengkap memastikan seluruh migrasi di repo punya pasangan
// up dan down yang valid, dengan penamaan yang bisa dibaca golang-migrate.
//
// Ini guard terhadap kejadian nyata: file 0012_status_history.down.sql pernah
// hilang, sehingga rollback berhenti di migrasi 0011 tanpa pesan yang jelas.
func TestEmbeddedFS_Lengkap(t *testing.T) {
	// iofs adalah cara resmi golang-migrate membaca fs.FS - kalau Init sukses,
	// berarti seluruh penamaan berkas memenuhi pola yang dimigrator.
	drv, err := iofs.New(migrations.FS, ".")
	if err != nil {
		t.Fatalf("iofs.Init gagal terhadap migrations.FS: %v", err)
	}

	first, err := drv.First()
	if err != nil {
		t.Fatalf("First(): %v", err)
	}
	if first != 1 {
		t.Fatalf("First() = %d, ingin 1 (migrasi pertama)", first)
	}

	// Telusuri semua versi sampai habis; setiap langkah Next() harus sukses
	// dan ReadUp/ReadDown harus mengembalikan isi.
	version := first
	for {
		up, _, err := drv.ReadUp(version)
		if err != nil {
			t.Fatalf("ReadUp(%d): %v", version, err)
		}
		if err := up.Close(); err != nil {
			t.Fatalf("tutup ReadUp(%d): %v", version, err)
		}

		down, _, err := drv.ReadDown(version)
		if err != nil {
			t.Fatalf("ReadDown(%d): %v - setiap migrasi wajib punya berkas .down.sql", version, err)
		}
		if err := down.Close(); err != nil {
			t.Fatalf("tutup ReadDown(%d): %v", version, err)
		}

		next, err := drv.Next(version)
		if err != nil {
			// ErrNotFound berarti sudah di versi terakhir.
			break
		}
		if next != version+1 {
			t.Fatalf("Next(%d) = %d, ingin %d (versi harus berurutan tanpa lompatan)",
				version, next, version+1)
		}
		version = next
	}

	if version < 14 {
		t.Fatalf("versi terakhir = %d, ingin >= 14 (cek berkas migrasi terbaru)", version)
	}
}

func TestErrDirtyState_BisaDierrorsIs(t *testing.T) {
	err := errors.Join(errors.New("wrap"), ErrDirtyState)
	if !errors.Is(err, ErrDirtyState) {
		t.Fatal("ErrDirtyState harus bisa dites dengan errors.Is setelah di-wrap")
	}
}
