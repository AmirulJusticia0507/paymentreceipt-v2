package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

// Problem adalah representasi error sesuai RFC 9457
// (Problem Details for HTTP APIs).
//
// Bentuk ini dipilih karena memberi field yang seragam dan bermakna:
// "type" untuk identifikasi programatik, "title" untuk ditampilkan di UI,
// "detail" untuk pesan spesifik.
type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Detail string `json:"detail"`
	// Instance adalah request ID supaya user bisa melapor error secara spesifik.
	Instance string `json:"instance,omitempty"`
	// Code stabil untuk logika frontend.
	Code string `json:"code,omitempty"`
	// Errors berisi detail per-field, hanya untuk KindValidation.
	Errors []apperror.FieldError `json:"errors,omitempty"`
	// RetryAfter hanya untuk rate limit.
	RetryAfter int `json:"retry_after,omitempty"`
}

// jsonResponse adalah envelope respons sukses.
type jsonResponse struct {
	Data any `json:"data"`
	Meta any `json:"meta,omitempty"`
}

// JSON menulis respons JSON dengan status tertentu.
func JSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if body == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// Pada titik ini header sudah terkirim, jadi tidak ada yang bisa dilakukan
		// selain mencatat. Klien akan melihat respons tidak lengkap, yang
		// diperlakukan sebagai error oleh frontend.
		slog.Error("gagal menulis JSON response", slog.Any("error", err))
	}
}

// JSONData menulis respons sukses berisi satu payload.
func JSONData(w http.ResponseWriter, status int, data any) {
	JSON(w, status, jsonResponse{Data: data})
}

// JSONList menulis respons sukses berisi daftar dan metadata paginasi.
func JSONList(w http.ResponseWriter, items any, p Pagination) {
	JSON(w, http.StatusOK, jsonResponse{Data: items, Meta: p})
}

// Problemf menulis respons error berdasarkan error yang lewat di.
//
// Semua error di aplikasi ini adalah domain error. Error yang bukan domain error
// dianggap bug dan diperlakukan sebagai 500 dengan pesan generik - detail aslinya
// hanya masuk ke log.
func Problemf(w http.ResponseWriter, r *http.Request, err error) {
	logger := loggerFrom(r)

	appErr, ok := apperror.AsAPPError(err)
	if !ok {
		// Error tidak dikenal = bug. Log lengkap, kirim generik.
		logger.Error("unhandled error",
			slog.Any("error", err),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
		appErr = apperror.Internal(err)
	}

	status := appErr.Kind.HTTPStatus()
	problem := Problem{
		Type:     "https://httpstatuses.io/" + fmt.Sprint(status),
		Title:    titleFor(appErr.Kind),
		Status:   status,
		Detail:   appErr.Message,
		Instance: RequestIDFromContext(r.Context()),
		Code:     appErr.Code,
		Errors:   appErr.Fields,
	}

	if appErr.Kind == apperror.KindRateLimited && appErr.RetryAfterDetik > 0 {
		problem.RetryAfter = appErr.RetryAfterDetik
		w.Header().Set("Retry-After", fmt.Sprint(appErr.RetryAfterDetik))
	}

	// Error 5xx itu bug atau masalah infrastruktur: catat dengan konteks lengkap.
	// Error 4xx adalah kesalahanKlien yang wajar - catat ringkas saja supaya log
	// tidak tenggelam oleh percobaan brute force.
	if status >= 500 {
		logger.Error("request gagal",
			slog.String("code", appErr.Code),
			slog.String("kind", appErr.Kind.String()),
			slog.Any("cause", appErr.Unwrap()),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	} else {
		logger.Warn("request ditolak",
			slog.String("code", appErr.Code),
			slog.String("kind", appErr.Kind.String()),
			slog.String("message", appErr.Message),
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
		)
	}

	JSON(w, status, problem)
}

// titleFor mengembalikan judul generik sesuai jenis error.
func titleFor(k apperror.Kind) string {
	switch k {
	case apperror.KindValidation:
		return "Data tidak valid"
	case apperror.KindUnauthorized:
		return "Belum terautentikasi"
	case apperror.KindForbidden:
		return "Akses ditolak"
	case apperror.KindNotFound:
		return "Data tidak ditemukan"
	case apperror.KindConflict:
		return "Konflik dengan data yang ada"
	case apperror.KindUnprocessable:
		return "Permintaan tidak dapat diproses"
	case apperror.KindRateLimited:
		return "Terlalu banyak permintaan"
	case apperror.KindInternalDependency:
		return "Layanan sedang bermasalah"
	default:
		return "Terjadi kesalahan"
	}
}

// DecodeJSON membaca dan memvalidasi body JSON ke struct v.
//
// Content-Type dicek eksplisit karena menerima JSON dari content type lain bisa
// membuka celah CSRF lewat form yang menyamar.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	ct := r.Header.Get("Content-Type")
	if ct != "" {
		// tolerate parameter charset
		if len(ct) >= len("application/json") && ct[:len("application/json")] == "application/json" {
			// ok
		} else {
			return apperror.Validationf("Content-Type harus application/json, diterima %q", ct)
		}
	}

	dec := json.NewDecoder(r.Body)
	// DisallowUnknownFields menolak field yang tidak dikenal. Tanpa ini, typo di
	// nama field akan lolos tanpa terdiam.
	dec.DisallowUnknownFields()

	if err := dec.Decode(v); err != nil {
		return translateDecodeError(err)
	}

	// Hanya boleh ada satu JSON value di body.
	if dec.More() {
		return apperror.Validation("Body harus berisi tepat satu objek JSON")
	}
	return nil
}

// translateDecodeError mengubah error encoding/json menjadi pesan yang berguna.
func translateDecodeError(err error) error {
	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		field := typeErr.Field
		if field == "" {
			field = "body"
		}
		return apperror.Validationf("Field %q harus berupa %s", field, typeErr.Type)
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return apperror.Validationf("JSON tidak valid (kesalahan di posisi %d)", syntaxErr.Offset)
	}

	// json:unknown field "xyz"
	msg := err.Error()
	if len(msg) > 15 && msg[:15] == "json: unknown f" {
		// Ambil nama field dari pesan agar user tahu field mana yang salah.
		name := msg[len("json: unknown field "):]
		return apperror.Validationf("Field %s tidak dikenal", name)
	}

	if msg == "EOF" || msg == "unexpected EOF" {
		return apperror.Validation("Body kosong, harus berisi JSON")
	}

	// Jangan bocorkan detail internal ke klien.
	return apperror.Validation("Format JSON tidak valid")
}
