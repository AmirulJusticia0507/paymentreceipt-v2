// Command migrate menjalankan migrasi skema database Payment Receipt v2.
//
// Dipakai saat deploy (lihat docker-compose.yml, service "migrate") dan untuk
// operasi skema manual:
//
//	migrate up              # terapkan semua migrasi tertunda (perintah default)
//	migrate down [-steps N] # rollback N migrasi terakhir (default 1)
//	migrate version         # cetak versi skema saat ini
//	migrate force VERSION   # perbaiki dirty state (lihat internal/migrator)
//
// Konfigurasi cukup satu env: DB_URL. Perintah ini tidak memuat konfigurasi
// aplikasi penuh supaya bisa dijalankan tanpa AUTH_JWT_SECRET dan sejenisnya -
// migrasi skema tidak butuh autentikasi.
package main

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/logging"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/migrator"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		// Logger bisa sudah terpasang (untuk pesan konteks) ataupun belum.
		slog.Error("migrasi gagal", slog.Any("error", err))
		fmt.Fprintln(os.Stderr, "migrasi gagal:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	logger := logging.New(envOr("LOG_LEVEL", "info"), envOr("LOG_FORMAT", "text"))
	slog.SetDefault(logger)

	cmd := "up"
	if len(args) > 0 {
		cmd = args[0]
		args = args[1:]
	}

	databaseURL := os.Getenv("DB_URL")
	if databaseURL == "" {
		return errors.New("DB_URL wajib diisi")
	}

	m, err := migrator.New(databaseURL)
	if err != nil {
		return err
	}
	defer func() {
		if err := m.Close(); err != nil {
			logger.Warn("gagal menutup migrator", slog.Any("error", err))
		}
	}()

	switch cmd {
	case "up":
		applied, err := m.Up()
		if err != nil {
			return err
		}
		if len(applied) == 0 {
			logger.Info("tidak ada migrasi tertunda")
			return nil
		}
		logger.Info("migrasi diterapkan", slog.Any("versi", applied))
		return nil

	case "down":
		fs := flag.NewFlagSet("down", flag.ContinueOnError)
		steps := fs.Int("steps", 1, "jumlah migrasi yang di-rollback")
		if err := fs.Parse(args); err != nil {
			return err
		}
		if err := m.Down(*steps); err != nil {
			return err
		}
		version, dirty, err := m.Version()
		if err != nil {
			return err
		}
		logger.Info("rollback selesai",
			slog.Int("steps", *steps),
			slog.Uint64("versi_sekarang", uint64(version)),
			slog.Bool("dirty", dirty),
		)
		return nil

	case "version":
		version, dirty, err := m.Version()
		if err != nil {
			return err
		}
		logger.Info("versi skema",
			slog.Uint64("versi", uint64(version)),
			slog.Bool("dirty", dirty),
		)
		return nil

	case "force":
		fs := flag.NewFlagSet("force", flag.ContinueOnError)
		if err := fs.Parse(args); err != nil {
			return err
		}
		if fs.NArg() != 1 {
			return errors.New("force membutuhkan tepat satu argumen: versi target")
		}
		var version int
		if _, err := fmt.Sscanf(fs.Arg(0), "%d", &version); err != nil {
			return fmt.Errorf("versi %q bukan angka", fs.Arg(0))
		}
		if err := m.Force(version); err != nil {
			return err
		}
		logger.Warn("versi database dipaksa",
			slog.Int("versi", version),
			slog.String("catatan", "force tidak menjalankan migrasi; hanya menyetel penanda versi"),
		)
		return nil

	default:
		return fmt.Errorf("perintah tidak dikenal: %q (pilihan: up, down, version, force)", cmd)
	}
}

func envOr(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
