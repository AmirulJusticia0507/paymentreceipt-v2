// Command api adalah entrypoint server Payment Receipt v2.
//
// File ini sengaja tipis: hanya merangkai config, database, router, dan server,
// lalu mengurus graceful shutdown. Seluruh logika bisnis ada di internal/.
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/database"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/logging"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/router"
)

// version diisi saat build via -ldflags.
var version = "dev"

func main() {
	showVersion := flag.Bool("version", false, "tampilkan versi lalu keluar")
	flag.Parse()

	if *showVersion {
		fmt.Printf("paymentreceipt-v2 api %s\n", version)
		return
	}

	if err := run(); err != nil {
		// Logger mungkin belum siap, jadi pakai slog default untuk pesan terakhir.
		slog.Error("aplikasi berhenti dengan error", slog.Any("error", err))
		os.Exit(1)
	}
}

func run() error {
	// --- Config -----------------------------------------------------------
	cfg, err := config.Load()
	if err != nil {
		// Logger belum ada, pakai default sementara.
		return fmt.Errorf("konfigurasi: %w", err)
	}

	// --- Logger -----------------------------------------------------------
	logger := logging.New(cfg.Log.Level, cfg.Log.Format)
	slog.SetDefault(logger)

	logger.Info("memulai aplikasi",
		slog.String("nama", cfg.App.Name),
		slog.String("env", cfg.App.Env),
		slog.String("versi", version),
	)

	// --- Database ---------------------------------------------------------
	// Context terpisah: koneksi database harus selesai di awal, tidak mengikuti
	// lifecycle server.
	dbCtx, cancelDB := context.WithTimeout(context.Background(), 30*time.Second)
	db, err := database.New(dbCtx, cfg.DB)
	cancelDB()
	if err != nil {
		return fmt.Errorf("database: %w", err)
	}
	defer db.Close()

	logger.Info("database terhubung",
		slog.Int("max_conns", int(cfg.DB.MaxConns)),
		slog.Int("min_conns", int(cfg.DB.MinConns)),
	)

	// --- Router -----------------------------------------------------------
	handler := router.New(cfg, db, logger)

	// --- Server -----------------------------------------------------------
	srv := &http.Server{
		Addr:    cfg.Server.Addr(),
		Handler: handler,
		// Timeout per-koneksi. Tanpa ini, satu koneksi yang lambat bisa menahan
		// goroutine selamanya (slowloris).
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       cfg.Server.ReadTimeout,
		WriteTimeout:      cfg.Server.WriteTimeout,
		IdleTimeout:       cfg.Server.IdleTimeout,
		// BaseContext dipakai agar handler yang sedang berjalan bisa dibatalkan
		// saat shutdown.
		BaseContext: func(_ net.Listener) context.Context { return context.Background() },
		ErrorLog:    slog.NewLogLogger(logger.Handler(), slog.LevelWarn),
	}

	// --- Listen dalam goroutine ------------------------------------------
	serverErr := make(chan error, 1)

	go func() {
		logger.Info("server listen", slog.String("addr", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- fmt.Errorf("listen: %w", err)
			return
		}
		serverErr <- nil
	}()

	// --- Tunggu sinyal shutdown ------------------------------------------
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-serverErr:
		// Server gagal start (mis. port sudah dipakai) - langsung keluar.
		if err != nil {
			return err
		}
		return nil

	case <-ctx.Done():
		logger.Info("menerima sinyal shutdown, menutup server")
	}

	// --- Graceful shutdown ------------------------------------------------
	// Request yang sedang diproses diberi waktu selesai supaya user tidak
	// melihat koneksi terpotong di tengah jalan.
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("graceful shutdown gagal, memaksa keluar",
			slog.Any("error", err))
		return fmt.Errorf("shutdown: %w", err)
	}

	if err := <-serverErr; err != nil {
		return err
	}

	logger.Info("server berhenti bersih")
	return nil
}
