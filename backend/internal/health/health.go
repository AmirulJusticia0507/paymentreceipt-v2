// Package health menyediakan endpoint pemeriksaan kesehatan untuk load balancer
// dan monitoring.
package health

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/database"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// Status adalah hasil pemeriksaan kesehatan sebuah komponen.
type Status string

const (
	// StatusOK berarti komponen siap.
	StatusOK Status = "ok"
	// StatusDown berarti komponen tidak siap.
	StatusDown Status = "down"
)

// Report adalah hasil pemeriksaan satu komponen.
type Report struct {
	Component string `json:"component"`
	Status    Status `json:"status"`
	// DurationMs durasi pemeriksaan dalam milidetik.
	DurationMs int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

// Checker memeriksa satu dependensi.
type Checker struct {
	// Component nama komponen yang dicek, mis. "database".
	Component string
	// Check melakukan pemeriksaan. Error yang dikembalikan berarti tidak sehat.
	Check func(ctx context.Context) error
}

// DBChecker membuat Checker untuk koneksi database.
func DBChecker(db *database.DB) Checker {
	return Checker{
		Component: "database",
		Check:     db.Ping,
	}
}

// Run menjalankan seluruh checker secara paralel dan mengembalikan laporan.
//
// Paralel supaya satu dependensi yang lambat tidak menunda pemeriksaan yang lain -
// kalau berurutan, total waktu = jumlah semua timeout.
func Run(ctx context.Context, checkers []Checker) []Report {
	reports := make([]Report, len(checkers))
	var wg sync.WaitGroup

	for i, c := range checkers {
		wg.Add(1)
		go func(i int, c Checker) {
			defer wg.Done()

			start := time.Now()
			err := c.Check(ctx)

			r := Report{
				Component:  c.Component,
				Status:     StatusOK,
				DurationMs: time.Since(start).Milliseconds(),
			}
			if err != nil {
				r.Status = StatusDown
				r.Error = err.Error()
			}
			reports[i] = r
		}(i, c)
	}

	wg.Wait()
	return reports
}

// OverallStatus menggabungkan laporan menjadi satu status.
//
// Semuanya harus OK supaya hasilnya OK. Kalau ada satu saja yang down, seluruh
// instance dianggap tidak siap menerima traffic.
func OverallStatus(reports []Report) Status {
	for _, r := range reports {
		if r.Status != StatusOK {
			return StatusDown
		}
	}
	return StatusOK
}

// Liveness melaporkan apakah proses masih hidup.
//
// Sengaja TIDAK mengecek dependensi. Kalau database down, restart aplikasi tidak
// akan menolong - masalahnya ada di luar proses ini. Justru mematikan pod di
// situation itu hanya memperburuk: tidak ada yang bisa melayani traffic, dan yang
// baru start akan gagal juga.
//
// Yang perlu dijawab liveness hanya satu: apakah prosesnya masih hidup?
func Liveness() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		httputil.JSON(w, http.StatusOK, map[string]any{
			"status": string(StatusOK),
			"uptime": time.Since(processStart).Round(time.Second).String(),
			"time":   time.Now().UTC(),
		})
	}
}

// processStart diisi saat package di-init.
var processStart = time.Now()

// Readiness melaporkan apakah instance siap menerima traffic.
//
// Berbeda dari liveness, ini memeriksa dependensi. Kalau database tidak terjangkau,
// load balancer sebaiknya menarik instance ini dari rotasi sampai sehat lagi.
func Readiness(db *database.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
		defer cancel()

		reports := Run(ctx, []Checker{DBChecker(db)})
		status := OverallStatus(reports)

		code := http.StatusOK
		if status != StatusOK {
			code = http.StatusServiceUnavailable
		}

		httputil.JSON(w, code, map[string]any{
			"status": string(status),
			"checks": reports,
			"time":   time.Now().UTC(),
		})
	}
}
