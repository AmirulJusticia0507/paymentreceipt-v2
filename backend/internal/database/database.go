// Package database menangani koneksi PostgreSQL: pool, ping, dan transaksi.
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
)

// DB membungkus pgxpool agar package lain tidak perlu mengurussi detail driver.
type DB struct {
	pool *pgxpool.Pool
}

// New membuat pool koneksi baru dan memverifikasi koneksinya.
//
// Ping dilakukan saat startup supaya aplikasi gagal keras kalau database tidak
// terjangkau - lebih baik kelihatan sekarang daripada muncul sebagai 500 di setiap
// request nanti.
func New(ctx context.Context, cfg config.DBConfig) (*DB, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse DB_URL: %w", err)
	}

	poolCfg.MaxConns = cfg.MaxConns
	poolCfg.MinConns = cfg.MinConns
	poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	poolCfg.MaxConnIdleTime = cfg.MaxConnIdleTime
	poolCfg.ConnConfig.ConnectTimeout = cfg.ConnectTimeout

	// describe -> cache prepared statement per-koneksi (aman untuk PgBouncer
	// transaction mode, dan tahan perubahan skema tanpa perlu reconnect)
	// prepare  -> prepared statement di sisi server, sedikit lebih cepat
	switch cfg.StatementCacheMode {
	case "prepare":
		poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheStatement
	case "describe":
		poolCfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeCacheDescribe
	default:
		return nil, fmt.Errorf("DB_STATEMENT_CACHE_MODE: %q tidak dikenal (pilihan: describe, prepare)",
			cfg.StatementCacheMode)
	}

	// Health check koneksi baru. Koneksi yang rusak dikembalikan oleh driver
	// alih-alih dipakai lagi pada request berikutnya.
	poolCfg.BeforeConnect = func(ctx context.Context, cfg *pgx.ConnConfig) error {
		acquireCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
		defer cancel()

		conn, err := pgx.ConnectConfig(acquireCtx, cfg)
		if err != nil {
			return err
		}
		defer func() { _ = conn.Close(context.Background()) }()

		return conn.Ping(acquireCtx)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("buat pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, cfg.ConnectTimeout)
	defer cancel()

	if err := pool.Ping(pingCtx); err != nil {
		pool.Close() // jangan sampai ada pool yang menggantung kalau gagal
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Pool mengembalikan pool mentah, untuk kasus yang butuh akses pgx langsung
// (misalnya sqlc atau laporan analitik). Dipakai seminimal mungkin.
func (db *DB) Pool() *pgxpool.Pool { return db.pool }

// Close menutup pool dan menunggu koneksi yang sedang dipakai selesai.
func (db *DB) Close() {
	if db == nil || db.pool == nil {
		return
	}
	db.pool.Close()
}

// Ping memeriksa koneksi masih hidup.
func (db *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	return db.pool.Ping(ctx)
}

// Stats mengembalikan statistik pool untuk monitoring.
func (db *DB) Stats() *pgxpool.Stat {
	return db.pool.Stat()
}

// Querier adalah subset dari pgx yang dipakai repository.
//
// Dipisah sebagai interface supaya repository bisa diuji dengan mock tanpa
// membuka koneksi database sungguhan.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Tx adalah handle transaksi berurutan.
type Tx interface {
	Querier
	Commit(ctx context.Context) error
	Rollback(ctx context.Context) error
}

// WithTx menjalankan fn di dalam transaksi.
//
// perilaku:
//   - fn mengembalikan nil  -> commit
//   - fn mengembalikan error-> rollback, error dikembalikan ke pemanggil
//   - fn panic             -> rollback, lalu panic diteruskan
//
// Rollback dilakukan dengan context terpisah karena context asli bisa sudah
// cancelled - dengan begitu rollback tetap dijamin terjadi.
func WithTx(ctx context.Context, db *DB, fn func(tx Tx) error) error {
	pgxTx, err := db.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("mulai transaksi: %w", err)
	}

	committed := false
	defer func() {
		if !committed {
			// Context baru supaya rollback tetap jalan walau ctx sudah dibatalkan.
			rollbackCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			defer cancel()
			_ = pgxTx.Rollback(rollbackCtx)
		}
	}()

	if err := fn(pgxTx); err != nil {
		return err
	}

	if err := pgxTx.Commit(ctx); err != nil {
		return fmt.Errorf("commit transaksi: %w", err)
	}

	committed = true
	return nil
}
