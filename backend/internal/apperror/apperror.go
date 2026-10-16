// Package apperror mendefinisikan error domain aplikasi dan cara
// merepresentasikannya sebagai Problem Details (RFC 9457).
//
// Prinsip: setiap error yang muncul di dalam aplikasi adalah *domain error* dengan
// kode yang stabil. Layer HTTP_then_hanya_menerjemahkannya menjadi respons - bukan
// menentukan sendiri status code atau pesan.
//
// Kegunaan:
//   - Klien selalu mendapat bentuk error yang konsisten, apa pun yang terjadi di server
//   - Kode error stabil sehingga frontend bisa bereaksi secara terprogram
//   - Detail internal (SQL error, path file) tidak pernah bocor ke klien
package apperror

import (
	"errors"
	"fmt"
	"net/http"
)

// Kind mengklasifikasikan error. Ini yang menentukan HTTP status code.
type Kind int

const (
	// KindInternal adalah kesalahan tak terduga. Detailnya TIDAK dikirim ke klien.
	KindInternal Kind = iota
	// KindValidation berarti input tidak valid. Detail per-field dikirim.
	KindValidation
	// KindUnauthorized berarti belum login / token tidak valid.
	KindUnauthorized
	// KindForbidden berarti login tapi tidak berwenang.
	KindForbidden
	// KindNotFound berarti resource tidak ada.
	KindNotFound
	// KindConflict berarti state tidak mengizinkan operasi (mis. transisi status ilegal).
	KindConflict
	// KindUnprocessable berarti syntaksnya benar tapi secara bisnis tidak valid.
	KindUnprocessable
	// KindRateLimited berarti terlalu banyak permintaan.
	KindRateLimited
	// KindInternalDependency berarti serviço di luar (mis. database) bermasalah.
	KindInternalDependency
	// KindNotImplemented berarti endpoint ada tapi belum dikerjakan.
	//
	// Dipisahkan dari KindUnprocessable karena maknanya berbeda bagi klien:
	// 422 berarti "request-nya salah, perbaiki", sedangkan 501 berarti "server
	// belum bisa layani ini". Frontend bisa menampilkan pesan yang berbeda.
	KindNotImplemented
)

// String mengembalikan nama Kind dalam bentuk yang stabil (dipakai juga di log).
func (k Kind) String() string {
	switch k {
	case KindInternal:
		return "internal"
	case KindValidation:
		return "validation"
	case KindUnauthorized:
		return "unauthorized"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindUnprocessable:
		return "unprocessable"
	case KindRateLimited:
		return "rate_limited"
	case KindInternalDependency:
		return "dependency"
	case KindNotImplemented:
		return "not_implemented"
	default:
		return "unknown"
	}
}

// HTTPStatus memetakan Kind ke status code HTTP.
func (k Kind) HTTPStatus() int {
	switch k {
	case KindValidation:
		return http.StatusUnprocessableEntity // 422
	case KindUnauthorized:
		return http.StatusUnauthorized // 401
	case KindForbidden:
		return http.StatusForbidden // 403
	case KindNotFound:
		return http.StatusNotFound // 404
	case KindConflict:
		return http.StatusConflict // 409
	case KindUnprocessable:
		return http.StatusUnprocessableEntity // 422
	case KindRateLimited:
		return http.StatusTooManyRequests // 429
	case KindInternalDependency:
		return http.StatusBadGateway // 502
	case KindNotImplemented:
		return http.StatusNotImplemented // 501
	case KindInternal:
		return http.StatusInternalServerError // 500
	default:
		return http.StatusInternalServerError
	}
}

// FieldError menjelaskan satu kesalahan validasi pada field tertentu.
type FieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
	// Rule opsional: nama aturan yang dilanggar (mis. "required", "maxlen").
	// Membantu frontend menampilkan pesan yang tepat.
	Rule string `json:"rule,omitempty"`
}

// Error adalah domain error aplikasi.
type Error struct {
	Kind Kind
	// Code stabil, mis. "payment_receipt.not_found". Dipakai klien untuk logika.
	Code string
	// Message aman untuk dibaca pengguna akhir (Bahasa Indonesia).
	Message string
	// Fields hanya untuk KindValidation.
	Fields []FieldError
	// Err adalah error yang mendasari. TIDAK pernah dikirim ke klien,
	// hanya dipakai untuk log dan errors.Is/As.
	Err error
	// RetryAfterDetik hanya untuk KindRateLimited.
	RetryAfterDetik int
}

// Error mengimplementasikan interface error.
func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s (%s): %v", e.Code, e.Kind, e.Err)
	}
	return fmt.Sprintf("%s (%s): %s", e.Code, e.Kind, e.Message)
}

// Unwrap memungkinkan errors.Is / errors.As menelusuri ke error asli.
func (e *Error) Unwrap() error { return e.Err }

// WithField menambahkan detail field pada error validasi.
func (e *Error) WithField(field, message, rule string) *Error {
	if e.Fields == nil {
		e.Fields = make([]FieldError, 0, 4)
	}
	e.Fields = append(e.Fields, FieldError{Field: field, Message: message, Rule: rule})
	return e
}

// Is memungkinkan perbandingan berdasarkan Code, bukan Kind. Dua error dengan
// Kind sama tapi kode berbeda harus tetap bisa dibedakan.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// --- Konstruktor ----------------------------------------------------------

// New membuat domain error dengan Kind, Code, dan Message.
func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

// Newf sama seperti New dengan format string.
func Newf(kind Kind, code, format string, args ...any) *Error {
	return &Error{Kind: kind, Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wrap membungkus error yang sudah ada menjadi domain error.
//
// Err asli disimpan untuk errors.Is/As dan log, tapi TIDAK muncul di respons.
// Detail teknis (SQL error, path) tidak boleh sampai ke user.
func Wrap(kind Kind, code, message string, err error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Err: err}
}

// --- Error yang sering dipakai -------------------------------------------
// Kode dan pesan ini jadi kontrak - diubah berarti mengubah perilaku frontend.

// Validation mengembalikan error validasi umum.
func Validation(message string) *Error {
	return New(KindValidation, "validation_failed", message)
}

// Validationf mengembalikan error validasi dengan format string.
func Validationf(format string, args ...any) *Error {
	return Newf(KindValidation, "validation_failed", format, args...)
}

// FieldValidation mengembalikan error validasi untuk sebuah field.
func FieldValidation(field, message, rule string) *Error {
	return Validation(message).WithField(field, message, rule)
}

// Unauthorized mengembalikan error "belum login".
func Unauthorized(message string) *Error {
	return New(KindUnauthorized, "unauthorized", message)
}

// Forbidden mengembalikan error "tidak berwenang".
func Forbidden(message string) *Error {
	return New(KindForbidden, "forbidden", message)
}

// NotFound mengembalikan error "resource tidak ada".
//
// pesan harus menyebut nama resource ("Payment receipt tidak ditemukan"), bukan
// "404". Ini membantu user tanpa membocorkan struktur internal.
func NotFound(resource string) *Error {
	return New(KindNotFound, "not_found", resource+" tidak ditemukan")
}

// Conflict mengembalikan error "state tidak mengizinkan".
func Conflict(message string) *Error {
	return New(KindConflict, "conflict", message)
}

// Conflictf mengembalikan error conflict dengan format string.
func Conflictf(format string, args ...any) *Error {
	return Newf(KindConflict, "conflict", format, args...)
}

// Unprocessable mengembalikan error validasi bisnis.
func Unprocessable(message string) *Error {
	return New(KindUnprocessable, "unprocessable", message)
}

// Unprocessablef mengembalikan error unprocessable dengan format string.
func Unprocessablef(format string, args ...any) *Error {
	return Newf(KindUnprocessable, "unprocessable", format, args...)
}

// RateLimited mengembalikan error terlalu banyak permintaan.
func RateLimited(message string, retryAfterDetik int) *Error {
	return &Error{
		Kind:            KindRateLimited,
		Code:            "rate_limited",
		Message:         message,
		RetryAfterDetik: retryAfterDetik,
	}
}

// NotImplemented mengembalikan error 501.
//
// Dipakai sebagai placeholder untuk endpoint yang ada di router tapi belum punya
// implementasi. Status ini jujur dan tidak tertukar dengan "endpoint tidak ada".
func NotImplemented(message string) *Error {
	return New(KindNotImplemented, "not_implemented", message)
}

// NotImplementedf mengembalikan error 501 dengan format string.
func NotImplementedf(format string, args ...any) *Error {
	return Newf(KindNotImplemented, "not_implemented", format, args...)
}

// Internal membungkus error tak terduga.
func Internal(err error) *Error {
	return Wrap(KindInternal, "internal_error", "Terjadi kesalahan internal", err)
}

// Internalf membungkus error tak terduga dengan pesan internal khusus.
func Internalf(code string, err error) *Error {
	return Wrap(KindInternal, code, "Terjadi kesalahan internal", err)
}

// --- Helper ---------------------------------------------------------------

// AsAPPError mengekstrak *Error dari rantai error.
// Mengembalikan (nil, false) kalau error itu bukan domain error.
func AsAPPError(err error) (*Error, bool) {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// KindOf mengembalikan Kind dari error. Error non-domain dianggap KindInternal.
func KindOf(err error) Kind {
	if appErr, ok := AsAPPError(err); ok {
		return appErr.Kind
	}
	return KindInternal
}

// IsNotFound melaporkan apakah err adalah error 404.
func IsNotFound(err error) bool { return KindOf(err) == KindNotFound }

// IsConflict melaporkan apakah err adalah error 409.
func IsConflict(err error) bool { return KindOf(err) == KindConflict }

// IsValidation melaporkan apakah err adalah error validasi.
func IsValidation(err error) bool {
	k := KindOf(err)
	return k == KindValidation || k == KindUnprocessable
}

// IsUniqueViolation melaporkan apakah err adalah error unique constraint dari
// database. Dipakai repository untuk mengubahnya jadi error Conflict yang berguna.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	// Diperiksa lewat string karena pgconn.PgError tidak selalu terbungkus rapi.
	msg := err.Error()
	return contains(msg, "SQLSTATE 23505") || contains(msg, "duplicate key value")
}

// IsForeignKeyViolation melaporkan apakah err adalah error FK constraint.
func IsForeignKeyViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "SQLSTATE 23503") || contains(msg, "foreign key")
}

// IsCheckViolation melaporkan apakah err adalah error CHECK constraint.
func IsCheckViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return contains(msg, "SQLSTATE 23514") || contains(msg, "violates check constraint")
}

func contains(s, sub string) bool {
	return len(sub) > 0 && len(s) >= len(sub) && indexOf(s, sub) >= 0
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
