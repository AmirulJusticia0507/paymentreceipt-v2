// Package middleware berisi middleware HTTP yang dipasang pada router.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// RequestIDHeader adalah header tempat request ID dikembalikan ke klien.
const RequestIDHeader = "X-Request-ID"

// Fungsi context di bawah hanya alias ke httputil. Kunci context HARUS sama
// antara middleware dan handler, jadi keduanya memakai definisi yang sama -
// bukan implementasi terpisah.
func WithRequestID(ctx context.Context, id string) context.Context {
	return httputil.ContextWithRequestID(ctx, id)
}

func RequestID(ctx context.Context) string {
	return httputil.RequestIDFromContext(ctx)
}

func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return httputil.ContextWithLogger(ctx, l)
}

func LoggerFrom(ctx context.Context) *slog.Logger {
	return httputil.LoggerFromContext(ctx)
}

// newRequestID membuat request ID acak dalam format hex 16 byte.
func newRequestID() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		// crypto/rand praktis tidak pernah gagal di platform yang didukung.
		// Kalau sampai gagal, pakai timestamp supaya request tetap punya ID.
		return "ts-" + time.Now().UTC().Format("20060102T150405.000000000")
	}
	return hex.EncodeToString(buf[:])
}

// statusRecorder membungkus ResponseWriter untuk mencatat status dan ukuran.
// Harus mengimplementasikan Flush, Hijack, dan Unwrap supaya middleware lain
// (mis. gzip atau websocket) tetap bisa mengambil ResponseWriter aslinya.
type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
	wrote  bool
}

func (w *statusRecorder) WriteHeader(code int) {
	if !w.wrote {
		w.status = code
		w.wrote = true
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusRecorder) Write(b []byte) (int, error) {
	// Tanpa WriteHeader eksplisit, net/http memakai 200. Disalin supaya log
	// mencerminkan apa yang benar-benar dikirim ke klien.
	if !w.wrote {
		w.status = http.StatusOK
		w.wrote = true
	}
	n, err := w.ResponseWriter.Write(b)
	w.bytes += n
	return n, err
}

// Unwrap memberi akses ke ResponseWriter asli (dipakai http.ResponseController).
func (w *statusRecorder) Unwrap() http.ResponseWriter { return w.ResponseWriter }

// Flush mengimplementasikan http.Flusher.
func (w *statusRecorder) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// RequestLogger mencatat setiap request sebagai satu baris terstruktur,
// lengkap dengan request ID, durasi, status, dan ukuran respons.
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			id := newRequestID()

			ctx := WithRequestID(r.Context(), id)
			ctx = WithLogger(ctx, base.With(slog.String("request_id", id)))

			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			next.ServeHTTP(rec, r.WithContext(ctx))

			duration := time.Since(start)

			attrs := []any{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Int("bytes", rec.bytes),
				slog.Duration("duration", duration),
				slog.String("remote_addr", r.RemoteAddr),
			}

			// Endpoint yang lambat layak diberi perhatian lebih.
			if duration > 1*time.Second {
				base.WarnContext(ctx, "request lambat", attrs...)
			} else {
				base.InfoContext(ctx, "request", attrs...)
			}

			// Request ID dikembalikan supaya user bisa melaporkan error dengan
			// menyebut ID yang sama, dan kita bisa mencarinya di log.
			w.Header().Set(RequestIDHeader, id)
		})
	}
}

// Recoverer mengembalikan aplikasi ke kondisi stabil setelah panic.
//
// Tanpa ini, satu panic akan menghentikan seluruh proses dan semua request lain
// ikut mati. Response 500 dikirim, detail panic hanya ditulis ke log.
func Recoverer(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					// Stack trace dibutuhkan untuk debug. type assertion ke error
					// memberi stack asli; selain itu kita cetak value-nya.
					err, isErr := rec.(error)
					if !isErr {
						base.ErrorContext(r.Context(), "panic",
							slog.Any("value", rec),
							slog.String("method", r.Method),
							slog.String("path", r.URL.Path),
						)
						http.Error(w, "internal server error", http.StatusInternalServerError)
						return
					}

					base.ErrorContext(r.Context(), "panic",
						slog.Any("error", err),
						slog.String("method", r.Method),
						slog.String("path", r.URL.Path),
					)
					http.Error(w, "internal server error", http.StatusInternalServerError)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
