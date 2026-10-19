package middleware

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// readBodyHandler membaca seluruh body request dan menyimpan hasilnya ke
// *readErr supaya test bisa mengklaim batas tercapai.
func readBodyHandler(readErr *error, readN *int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		n, err := io.ReadAll(r.Body)
		*readErr, *readN = err, len(n)
		w.WriteHeader(http.StatusOK)
	}
}

// newBodyRequest membuat POST request ber-body berukuran bodyLen.
func newBodyRequest(path string, bodyLen int) *http.Request {
	return httptest.NewRequest(http.MethodPost, path, strings.NewReader(strings.Repeat("x", bodyLen)))
}

func TestBodyLimit_BatasDefaultUntukSemuaPath(t *testing.T) {
	var readErr error
	var readN int
	h := BodyLimit(10, map[string]int64{"/upload": 1000})(
		readBodyHandler(&readErr, &readN))

	h.ServeHTTP(httptest.NewRecorder(), newBodyRequest("/lain", 20))

	// Body 20 byte melewati batas default 10 - pembacaan harus gagal dengan
	// error khusus MaxBytesReader, bukan data terpotong yang diam-diam diterima.
	var maxErr *http.MaxBytesError
	if !errors.As(readErr, &maxErr) {
		t.Fatalf("read error = %v, ingin *http.MaxBytesError", readErr)
	}
}

func TestBodyLimit_OverrideMenaikkanBatasDiPathKhusus(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		bodyLen  int
		wantPass bool
	}{
		// 20 byte melewati default (10) tapi di bawah override (1000): diterima
		// hanya kalau override benar-benar dipakai untuk path ini.
		{name: "di bawah override diterima", path: "/upload", bodyLen: 20, wantPass: true},
		// Override bukan tanpa batas - melebihi tetap ditolak.
		{name: "melebihi override tetap ditolak", path: "/upload", bodyLen: 1001, wantPass: false},
		// Path lain tidak ikut terangkat oleh keberadaan override.
		{name: "path lain tetap kena default", path: "/lain", bodyLen: 20, wantPass: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var readErr error
			var readN int
			h := BodyLimit(10, map[string]int64{"/upload": 1000})(
				readBodyHandler(&readErr, &readN))

			h.ServeHTTP(httptest.NewRecorder(), newBodyRequest(tt.path, tt.bodyLen))

			if gotPass := readErr == nil; gotPass != tt.wantPass {
				t.Fatalf("readErr = %v (n=%d), wantPass = %v", readErr, readN, tt.wantPass)
			}
			if tt.wantPass && readN != tt.bodyLen {
				t.Fatalf("terbaca %d byte, ingin %d", readN, tt.bodyLen)
			}
		})
	}
}

// TestBodyLimit_OverridesKosongPertahankanPerilakuLama memastikan pemanggilan
// tanpa override (map nil) tetap membatasi semua path dengan nilai default -
// regresi terhadap signature lama BodyLimit(max).
func TestBodyLimit_OverridesKosongPertahankanPerilakuLama(t *testing.T) {
	var readErr error
	var readN int
	h := BodyLimit(10, nil)(readBodyHandler(&readErr, &readN))

	h.ServeHTTP(httptest.NewRecorder(), newBodyRequest("/upload", 20))

	var maxErr *http.MaxBytesError
	if !errors.As(readErr, &maxErr) {
		t.Fatalf("read error = %v, ingin *http.MaxBytesError", readErr)
	}
}
