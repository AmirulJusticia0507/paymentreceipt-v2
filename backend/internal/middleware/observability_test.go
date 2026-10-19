package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSanitizeRequestID(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "hex standar diterima", in: "0af3c2e1d4b5a6978c1f2e3d4b5a6978", want: "0af3c2e1d4b5a6978c1f2e3d4b5a6978"},
		{name: "dash dan underscore diterima", in: "req_id-ABC-123", want: "req_id-ABC-123"},
		{name: "persis 8 karakter", in: "12345678", want: "12345678"},
		{name: "persis 64 karakter", in: strings.Repeat("a", 64), want: strings.Repeat("a", 64)},
		{name: "terlalu pendek ditolak", in: "1234567", want: ""},
		{name: "terlalu panjang ditolak", in: strings.Repeat("a", 65), want: ""},
		{name: "kosong ditolak", in: "", want: ""},
		// Skenario log injection: baris baru + JSON palsu. Harus dibuang semua,
		// jadi logger tidak pernah menuliskan baris palsu ke agregator.
		{name: "newline injection ditolak", in: "abc\n{\"level\":\"info\"}", want: ""},
		{name: "carriage return ditolak", in: "abcdefg\rhij", want: ""},
		{name: "tab ditolak", in: "abcdef\tg", want: ""},
		{name: "spasi ditolak", in: "abc def", want: ""},
		{name: "karakter non-ascii ditolak", in: "abcé123", want: ""},
		{name: "quote ditolak", in: `abc"def`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizeRequestID(tt.in); got != tt.want {
				t.Fatalf("sanitizeRequestID(%q) = %q, ingin %q", tt.in, got, tt.want)
			}
		})
	}
}

// TestRequestLogger_MenghormatiHeaderValid memastikan request ID dari proxy
// (mis. nginx) dipakai apa adanya - korelasi log edge dengan log aplikasi.
func TestRequestLogger_MenghormatiHeaderValid(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contoh", nil)
	req.Header.Set(RequestIDHeader, "edge-abc-12345")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	// ID yang sama harus muncul di response header DAN di log.
	if got := rec.Header().Get(RequestIDHeader); got != "edge-abc-12345" {
		t.Fatalf("response header %s = %q, ingin \"edge-abc-12345\"", RequestIDHeader, got)
	}
	if !strings.Contains(buf.String(), "edge-abc-12345") {
		t.Fatalf("log harus memuat request ID dari header, dapat: %s", buf.String())
	}
}

// TestRequestLogger_MenggantiHeaderBerbahaya memastikan header yang tidak
// valid (injection) diganti dengan ID buatan sendiri, dan nilai berbahaya
// tidak pernah masuk ke log.
func TestRequestLogger_MenggantiHeaderBerbahaya(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/v1/contoh", nil)
	req.Header.Set(RequestIDHeader, "abc\nfake-log-entry")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	got := rec.Header().Get(RequestIDHeader)
	if got == "" || got == "abc\nfake-log-entry" || strings.Contains(got, "\n") {
		t.Fatalf("response header %s = %q, ingin ID buatan yang bersih", RequestIDHeader, got)
	}

	// Log harus berisi tepat SATU baris JSON yang valid - kalau injection
	// berhasil, parser akan gagal atau ada lebih dari satu objek.
	dec := json.NewDecoder(&buf)
	var entry map[string]any
	if err := dec.Decode(&entry); err != nil {
		t.Fatalf("log harus tetap JSON valid satu objek: %v (log: %s)", err, buf.String())
	}
	if dec.More() {
		t.Fatalf("log injection terdeteksi: ada entri kedua setelah baris pertama (log: %s)", buf.String())
	}
	if strings.Contains(buf.String(), "fake-log-entry") {
		t.Fatalf("nilai header berbahaya tidak boleh muncul di log: %s", buf.String())
	}
}

// TestRequestLogger_HeaderDikirimSebelumBody memastikan X-Request-ID benar-benar
// terkirim ke client (diset sebelum handler menulis body).
func TestRequestLogger_HeaderDikirimSebelumBody(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		// Menulis body langsung - tanpa WriteHeader eksplisit. Ini kondisi
		// umum yang membuat header setelah-serve tidak terkirim.
		_, _ = w.Write([]byte("halo"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get(RequestIDHeader) == "" {
		t.Fatal("X-Request-ID harus sudah terpasang sebelum handler menulis body")
	}
}

// TestRequestLogger_MencatatFieldLengkap memastikan setiap request tercatat
// dengan method, path, status, bytes, duration, dan remote_addr - kontrak
// yang diandalkan agregator log.
func TestRequestLogger_MencatatFieldLengkap(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := RequestLogger(logger)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte("isi"))
	}))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/payment-receipts", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log harus JSON valid: %v (log: %s)", err, buf.String())
	}

	for _, key := range []string{"method", "path", "status", "bytes", "duration", "remote_addr", "request_id"} {
		if _, ok := entry[key]; !ok {
			t.Errorf("log harus memuat field %q, dapat: %v", key, entry)
		}
	}
	if entry["method"] != "POST" {
		t.Errorf("method = %v, ingin POST", entry["method"])
	}
	if entry["path"] != "/api/v1/payment-receipts" {
		t.Errorf("path = %v, ingin /api/v1/payment-receipts", entry["path"])
	}
	if entry["status"] != float64(http.StatusCreated) {
		t.Errorf("status = %v, ingin 201", entry["status"])
	}
}
