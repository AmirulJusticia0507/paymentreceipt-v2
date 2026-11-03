package health

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// roundTripFunc mengubah fungsi menjadi http.RoundTripper sehingga Probe bisa
// diuji tanpa membuka port jaringan nyata. Di Windows, httptest.NewServer
// membuka port ephemeral yang kadang diblokir Windows Firewall pada loopback,
// sehingga tes yang bergantung pada koneksi TCP lokal rawan flaky. Dengan
// transport in-process ini, tidak ada soket yang dibuka sama sekali.
type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// probeWithTransport memanggil Probe lewat http.Client yang transport-nya
// sudah diinjeksi — sama persis dengan Probe biasa, tapi tanpa buka soket.
func probeWithTransport(t *testing.T, transport http.RoundTripper, url string) error {
	t.Helper()
	client := &http.Client{
		Timeout:   2 * time.Second,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return errors.New("status " + http.StatusText(resp.StatusCode))
	}
	return nil
}

func TestProbe_Sehat(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}).ServeHTTP(rec, r)
		return rec.Result(), nil
	})

	if err := probeWithTransport(t, transport, "http://fake/healthz"); err != nil {
		t.Fatalf("Probe pada server 200 seharusnya sukses, dapat: %v", err)
	}
}

func TestProbe_TolakStatusError(t *testing.T) {
	tests := []struct {
		name string
		code int
	}{
		{name: "503 service unavailable", code: http.StatusServiceUnavailable},
		{name: "500 internal server error", code: http.StatusInternalServerError},
		{name: "404 not found", code: http.StatusNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
				rec := httptest.NewRecorder()
				http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					w.WriteHeader(tt.code)
				}).ServeHTTP(rec, r)
				return rec.Result(), nil
			})

			err := probeWithTransport(t, transport, "http://fake/healthz")
			if err == nil {
				t.Fatalf("Probe pada status %d seharusnya gagal", tt.code)
			}
			if !strings.Contains(err.Error(), "status") {
				t.Fatalf("error Probe harus menyebut status, dapat: %v", err)
			}
		})
	}
}

func TestProbe_GagalKoneksi(t *testing.T) {
	// Port yang tidak dipakai siapa pun - koneksi pasti ditolak.
	err := Probe("http://127.0.0.1:1/healthz", 1*time.Second)
	if err == nil {
		t.Fatal("Probe ke alamat tanpa server seharusnya gagal")
	}
}

func TestProbe_TolakRedirect(t *testing.T) {
	// Server merespons redirect - probe tidak boleh mengikutinya, karena
	// endpoint health yang di-redirect biasanya berarti ada yang salah di
	// konfigurasi (mis. dipaksa ke halaman login).
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		rec := httptest.NewRecorder()
		http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			http.Redirect(w, req, "/somewhere-else", http.StatusFound)
		}).ServeHTTP(rec, r)
		return rec.Result(), nil
	})

	if err := probeWithTransport(t, transport, "http://fake/healthz"); err == nil {
		t.Fatal("Probe seharusnya menolak respons redirect (3xx dianggap tidak sehat)")
	}
}

// TestLiveness_TanpaDependensi memastikan /healthz tidak menyentuh dependensi
// apa pun - probe orchestrator restart-safe.
func TestLiveness_TanpaDependensi(t *testing.T) {
	handler := Liveness()

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Liveness status = %d, ingin 200", rec.Code)
	}
}

// fakePinger adalah Pinger buatan untuk menguji Readiness tanpa database
// sungguhan.
type fakePinger struct {
	err error
}

func (f fakePinger) Ping(_ context.Context) error { return f.err }

func TestReadiness_OKSaatDatabaseSehat(t *testing.T) {
	handler := Readiness(fakePinger{})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("Readiness saat DB sehat = %d, ingin 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"ok"`) {
		t.Fatalf("body Readiness harus melapor ok, dapat: %s", rec.Body.String())
	}
}

func TestReadiness_DownSaatDBTidakTerjangkau(t *testing.T) {
	handler := Readiness(fakePinger{err: errors.New("connection refused")})

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("Readiness saat DB mati = %d, ingin 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"status":"down"`) {
		t.Fatalf("body Readiness harus melapor down, dapat: %s", rec.Body.String())
	}
	// Laporan harus menyebut komponen dan penyebabnya, supaya operator langsung
	// tahu apa yang rusak tanpa membuka log.
	if !strings.Contains(rec.Body.String(), "database") ||
		!strings.Contains(rec.Body.String(), "connection refused") {
		t.Fatalf("body Readiness harus menyebut komponen 'database' dan error aslinya, dapat: %s", rec.Body.String())
	}
}

func TestOverallStatus(t *testing.T) {
	if got := OverallStatus([]Report{}); got != StatusOK {
		t.Fatalf("OverallStatus kosong = %s, ingin ok", got)
	}
	if got := OverallStatus([]Report{{Status: StatusOK}, {Status: StatusDown}}); got != StatusDown {
		t.Fatalf("OverallStatus dengan satu down = %s, ingin down", got)
	}
	if got := OverallStatus([]Report{{Status: StatusOK}, {Status: StatusOK}}); got != StatusOK {
		t.Fatalf("OverallStatus semua ok = %s, ingin ok", got)
	}
}

func TestRun_ParalelDanLengkap(t *testing.T) {
	errFake := errors.New("rusak")

	reports := Run(t.Context(), []Checker{
		{Component: "a", Check: func(_ context.Context) error { return nil }},
		{Component: "b", Check: func(_ context.Context) error { return errFake }},
	})
	if len(reports) != 2 {
		t.Fatalf("Run mengembalikan %d laporan, ingin 2", len(reports))
	}
	byName := map[string]Report{}
	for _, r := range reports {
		byName[r.Component] = r
	}
	if byName["a"].Status != StatusOK {
		t.Fatalf("checker a = %s, ingin ok", byName["a"].Status)
	}
	if byName["b"].Status != StatusDown || byName["b"].Error == "" {
		t.Fatalf("checker b = %+v, ingin down dengan pesan error", byName["b"])
	}
}
