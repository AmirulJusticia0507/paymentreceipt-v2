// Package migrator menjalankan migrasi skema database dari berkas SQL yang
// di-embed di dalam binary.
//
// Kenapa tidak memakai CLI `migrate` eksternal saat deploy: image produksi
// memakai distroless yang tidak punya shell apalagi binary pihak ketiga.
// Dengan meng-embed migrasi ke dalam binary sendiri, strategi deploy menjadi
// satu binary yang menjalankan migrasi lalu start aplikasi - tidak ada risiko
// versi CLI berbeda antar mesin, dan versi migrasi selalu sinkron dengan kode
// yang di-build.
package migrator

import (
	"errors"
	"fmt"
	"io/fs"

	"github.com/golang-migrate/migrate/v4"
	pgxv5 "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/db/migrations"
)

// ErrDirtyState dilaporkan saat tabel schema_migrations menandai database
// dalam kondisi kotor - sebuah migrasi gagal di tengah jalan dan perlu
// intervensi manual (force) sebelum migrasi lain boleh dijalankan.
var ErrDirtyState = errors.New("database dalam kondisi dirty state")

// Migrator menjalankan migrasi terhadap satu database.
type Migrator struct {
	m *migrate.Migrate
}

// New membuat Migrator untuk database tertentu.
//
// databaseURL memakai format PostgreSQL yang sama dengan DB_URL aplikasi.
func New(databaseURL string) (*Migrator, error) {
	return NewFromFS(migrations.FS, databaseURL)
}

// NewFromFS membuat Migrator dari fs.FS yang berisi berkas migrasi.
//
// Dipisah dari New supaya test bisa memberi sumber migrasi buatan sendiri
// tanpa menyentuh database yang sebenarnya.
func NewFromFS(fsys fs.FS, databaseURL string) (*Migrator, error) {
	src, err := iofs.New(fsys, ".")
	if err != nil {
		return nil, fmt.Errorf("baca sumber migrasi: %w", err)
	}

	db, err := (&pgxv5.Postgres{}).Open(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("buka koneksi database untuk migrasi: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", src, "pgx5", db)
	if err != nil {
		// migrate.NewWithInstance tidak menutup instance yang diberikan saat
		// gagal, jadi tutup manual supaya koneksi tidak bocor.
		_ = src.Close()
		_ = db.Close()
		return nil, fmt.Errorf("inisialisasi migrator: %w", err)
	}

	return &Migrator{m: m}, nil
}

// Up menjalankan seluruh migrasi yang belum diterapkan sampai versi terbaru.
//
// Mengembalikan daftar versi yang baru diterapkan (kosong kalau tidak ada
// migrasi tertunda) supaya pemanggil bisa mencatatnya ke log.
func (m *Migrator) Up() ([]uint, error) {
	// migrate tidak mengekspos daftar migrasi yang dijalankan, jadi catat
	// versi sebelum dan sesudah untuk menghitung yang berubah.
	before, _, err := m.m.Version()
	if err != nil && !errors.Is(err, migrate.ErrNilVersion) {
		return nil, fmt.Errorf("baca versi sebelum migrasi: %w", err)
	}

	if err := m.m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			return nil, nil
		}
		var dirty migrate.ErrDirty
		if errors.As(err, &dirty) {
			return nil, fmt.Errorf("%w pada versi %d: jalankan 'force <versi>' setelah memperbaiki penyebab kegagalan",
				ErrDirtyState, dirty.Version)
		}
		return nil, fmt.Errorf("jalankan migrasi: %w", err)
	}

	after, _, err := m.m.Version()
	if err != nil {
		return nil, fmt.Errorf("baca versi sesudah migrasi: %w", err)
	}

	return versionsAfter(before, after), nil
}

// Down mengembalikan database ke n versi sebelumnya.
//
// Hanya untuk rollback darurat. Migrasi down yang tidak tersedia (berkas
// .down.sql hilang) akan mengembalikan error - lebih baik gagal eksplisit
// daripada diam-diam meninggalkan skema setengah jalan.
func (m *Migrator) Down(steps int) error {
	if steps < 1 {
		return fmt.Errorf("steps harus >= 1, dapat %d", steps)
	}
	if err := m.m.Steps(-steps); err != nil {
		return fmt.Errorf("jalankan rollback %d langkah: %w", steps, err)
	}
	return nil
}

// Version mengembalikan versi skema saat ini dan apakah database dirty.
//
// dirty=true berarti ada migrasi yang gagal di tengah jalan dan database
// mungkin berada dalam skema campuran.
func (m *Migrator) Version() (version uint, dirty bool, err error) {
	v, d, err := m.m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			// Belum ada satu pun migrasi yang diterapkan.
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("baca versi migrasi: %w", err)
	}
	return v, d, nil
}

// Force menandai database pada versi tertentu tanpa menjalankan migrasi.
//
// Hanya untuk pemulihan dirty state: gunakan setelah penyebab kegagalan
// diperbaiki dan keadaan skema diketahui pasti.
func (m *Migrator) Force(version int) error {
	if version < 0 {
		return fmt.Errorf("version tidak boleh negatif, dapat %d", version)
	}
	if err := m.m.Force(version); err != nil {
		return fmt.Errorf("force versi %d: %w", version, err)
	}
	return nil
}

// Close menutup koneksi database migrator.
func (m *Migrator) Close() error {
	srcErr, dbErr := m.m.Close()
	if srcErr != nil {
		return fmt.Errorf("tutup sumber migrasi: %w", srcErr)
	}
	if dbErr != nil {
		return fmt.Errorf("tutup koneksi database migrasi: %w", dbErr)
	}
	return nil
}

// versionsAfter menghitung versi yang diterapkan antara before dan after.
//
// before=0 (belum ada migrasi) berarti semua versi 1..after dianggap baru
// diterapkan.
func versionsAfter(before, after uint) []uint {
	if after == 0 || after <= before {
		return nil
	}
	out := make([]uint, 0, after-before)
	for v := before + 1; v <= after; v++ {
		out = append(out, v)
	}
	return out
}
