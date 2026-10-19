package middleware

import (
	"context"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/config"
)

// CORS mengatur kebijakan cross-origin.
//
// Aplikasi ini menyediakan API untuk frontend terpisah (dev server Vite di
// port berbeda), jadi CORS memang diperlukan di pengembangan. Di production
// frontend di-serving dari origin yang sama, jadi daftar origin dibatasi.
func CORS(appCfg config.AppConfig) func(http.Handler) http.Handler {
	// Mode development: izinkan origin Vite.
	// Production: hanya origin yang sama dengan APP_BASE_URL.
	allowed := map[string]bool{}

	if appCfg.IsDevelopment() || appCfg.IsProduction() {
		// Selalu izinkan origin dari APP_BASE_URL.
		allowed[strings.TrimRight(appCfg.BaseURL, "/")] = true
	}
	if appCfg.IsDevelopment() {
		for _, o := range []string{
			"http://localhost:5173", "http://127.0.0.1:5173",
			"http://localhost:4173", "http://127.0.0.1:4173",
		} {
			allowed[o] = true
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Header Vary wajib supaya cache tidak memakai respons CORS untuk
			// origin yang berbeda.
			w.Header().Add("Vary", "Origin")

			if origin != "" {
				if allowed[origin] {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					w.Header().Set("Access-Control-Allow-Credentials", "true")
					w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
					w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
					w.Header().Set("Access-Control-Expose-Headers", RequestIDHeader)
					w.Header().Set("Access-Control-Max-Age", "600")
				}
				// Origin yang tidak diizinkan sengaja tidak diberi header apa pun.
				// Browser yang akan menolak - kita tidak perlu mengirim 403 karena
				// request tetap aman (server menolak juga).
			}

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// SecureHeaders memasang header keamanan dasar.
func SecureHeaders(isProduction bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")

			if isProduction {
				// HSTS hanya bermakna (dan aman) lewat HTTPS.
				h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// Timeout membatasi durasi pemrosesan satu request.
//
// Karena context dari net/http sudah di-cancel ketika handler selesai, context
// yang di-cancel oleh context.WithTimeout akan otomatis membatalkan query
// database yang sedang berjalan.
func Timeout(d time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// BodyLimit membatasi ukuran body request. Dicek lewat MaxBytesReader supaya
// request besar ditolak saat dibaca, bukan setelah seluruh body sudah di memori.
//
// overrides memetakan path persis ke batas khusus untuk path itu - dipakai
// untuk endpoint yang memang butuh body lebih besar dari batas default (mis.
// unggahan gambar AI). Pemilihan batas harus terjadi SEBELUM body dibungkus:
// menumpuk MaxBytesReader berbatas lebih besar di atas pembungkus yang lebih
// ketat tidak menaikkan apa pun, karena pembungkus luar tetap lebih dulu
// menolak bacaan yang melewati batasnya.
func BodyLimit(defaultMax int64, overrides map[string]int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Body == nil {
				next.ServeHTTP(w, r)
				return
			}
			max := defaultMax
			if v, ok := overrides[r.URL.Path]; ok && v > 0 {
				max = v
			}
			r.Body = http.MaxBytesReader(w, r.Body, max)
			next.ServeHTTP(w, r)
		})
	}
}

// RealIP menuliskan IP asli ke RemoteAddr, tetapi HANYA kalau request datang dari
// proxy yang dipercaya.
//
// Tanpa ini, semua request terlihat berasal dari IP proxy dan rate limit per-IP
// jadi tidak berguna. Dengan patching RemoteHost(), seluruh kode di bawah tidak
// perlu tahu soal proxy sama sekali.
func RealIP(trustedProxies []string) func(http.Handler) http.Handler {
	trusted := make([]*net.IPNet, 0, len(trustedProxies))
	for _, c := range trustedProxies {
		if _, ipnet, err := net.ParseCIDR(c); err == nil {
			trusted = append(trusted, ipnet)
		} else if ip := net.ParseIP(c); ip != nil {
			bits := 32
			if ip.To4() == nil {
				bits = 128
			}
			trusted = append(trusted, &net.IPNet{IP: ip, Mask: net.CIDRMask(bits, bits)})
		}
	}

	isTrusted := func(ip net.IP) bool {
		for _, n := range trusted {
			if n.Contains(ip) {
				return true
			}
		}
		return false
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(trusted) == 0 {
				next.ServeHTTP(w, r)
				return
			}

			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				host = r.RemoteAddr
			}
			peerIP := net.ParseIP(host)
			if peerIP == nil || !isTrusted(peerIP) {
				// Bukan proxy terpercaya - header dari client tidak boleh dipercaya.
				next.ServeHTTP(w, r)
				return
			}

			// X-Forwarded-For berisi rantai "client, proxy1, proxy2". Elemen paling
			// kiri paling mudah dipalsukan client, jadi kita telusuri dari kanan
			// dan mengambil entri pertama yang bukan proxy - itulah client asli
			// menurut rantai yang bisa kita percaya.
			if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
				parts := strings.Split(xff, ",")
				for i := len(parts) - 1; i >= 0; i-- {
					ip := net.ParseIP(strings.TrimSpace(parts[i]))
					if ip != nil && !isTrusted(ip) {
						setRemoteAddr(r, ip.String())
						break
					}
				}
			} else if xr := r.Header.Get("X-Real-IP"); xr != "" {
				if ip := net.ParseIP(strings.TrimSpace(xr)); ip != nil && !isTrusted(ip) {
					setRemoteAddr(r, ip.String())
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func setRemoteAddr(r *http.Request, ip string) {
	if port := portOf(r.RemoteAddr); port != "" {
		r.RemoteAddr = net.JoinHostPort(ip, port)
		return
	}
	r.RemoteAddr = ip
}

func portOf(remoteAddr string) string {
	_, port, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return ""
	}
	if _, err := strconv.Atoi(port); err != nil {
		return ""
	}
	return port
}
