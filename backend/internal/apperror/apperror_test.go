package apperror

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestKindHTTPStatus(t *testing.T) {
	tests := []struct {
		kind Kind
		want int
	}{
		{KindValidation, http.StatusUnprocessableEntity},
		{KindUnauthorized, http.StatusUnauthorized},
		{KindForbidden, http.StatusForbidden},
		{KindNotFound, http.StatusNotFound},
		{KindConflict, http.StatusConflict},
		{KindUnprocessable, http.StatusUnprocessableEntity},
		{KindRateLimited, http.StatusTooManyRequests},
		{KindInternalDependency, http.StatusBadGateway},
		{KindInternal, http.StatusInternalServerError},
		{KindNotImplemented, http.StatusNotImplemented},
	}

	for _, tt := range tests {
		t.Run(tt.kind.String(), func(t *testing.T) {
			if got := tt.kind.HTTPStatus(); got != tt.want {
				t.Errorf("HTTPStatus() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestKindString(t *testing.T) {
	tests := []struct {
		kind Kind
		want string
	}{
		{KindValidation, "validation"},
		{KindUnauthorized, "unauthorized"},
		{KindForbidden, "forbidden"},
		{KindNotFound, "not_found"},
		{KindConflict, "conflict"},
		{KindUnprocessable, "unprocessable"},
		{KindRateLimited, "rate_limited"},
		{KindInternalDependency, "dependency"},
		{KindNotImplemented, "not_implemented"},
		{KindInternal, "internal"},
	}

	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("Kind(%d).String() = %q, want %q", tt.kind, got, tt.want)
		}
	}
}

func TestErrorIsComparesByCode(t *testing.T) {
	// Dua error dengan Kind sama harus tetap bisa dibedakan lewat Code.
	a := New(KindNotFound, "department.not_found", "Department tidak ditemukan")
	b := New(KindNotFound, "base.not_found", "Base tidak ditemukan")

	if errors.Is(a, b) {
		t.Error("errors.Is(a, b) = true, want false (kode berbeda harus dibedakan)")
	}
	if !errors.Is(a, New(KindNotFound, "department.not_found", "pesan lain")) {
		t.Error("errors.Is = false, want true (kode sama harus cocok walau pesan berbeda)")
	}
}

func TestErrorUnwrapPreservesCause(t *testing.T) {
	cause := errors.New("koneksi database ditolak")
	err := Internal(cause)

	if !errors.Is(err, cause) {
		t.Error("errors.Is(err, cause) = false, want true (harus bisa ditelusuri ke error asli)")
	}
}

func TestErrorMessageNeverLeaksCause(t *testing.T) {
	// Detail internal tidak boleh bocor lewat Message yang dikirim ke klien.
	cause := errors.New("pq: password authentication failed for user \"admin\"")
	err := Internal(cause)

	if contains(err.Message, "pq:") || contains(err.Message, "password") {
		t.Errorf("Message %q membocorkan detail internal", err.Message)
	}
}

func TestValidationWithFields(t *testing.T) {
	err := Validation("data tidak valid").
		WithField("nama", "wajib diisi", "required").
		WithField("qty", "harus lebih dari 0", "gt")

	if len(err.Fields) != 2 {
		t.Fatalf("len(Fields) = %d, want 2", len(err.Fields))
	}
	if err.Fields[0].Field != "nama" || err.Fields[0].Rule != "required" {
		t.Errorf("Fields[0] = %+v, want field=nama rule=required", err.Fields[0])
	}
	if err.Fields[1].Field != "qty" {
		t.Errorf("Fields[1].Field = %q, want qty", err.Fields[1].Field)
	}
}

func TestAsAPPError(t *testing.T) {
	domain := NotFound("Payment receipt")

	got, ok := AsAPPError(domain)
	if !ok {
		t.Fatal("AsAPPError() = false, want true")
	}
	if got != domain {
		t.Error("AsAPPError() mengembalikan error berbeda")
	}

	// Error yang sudah di-wrap pun harus ditemukan.
	wrapped := fmt.Errorf("layer service: %w", domain)
	got2, ok2 := AsAPPError(wrapped)
	if !ok2 {
		t.Fatal("AsAPPError(wrapped) = false, want true")
	}
	if got2.Code != "not_found" {
		t.Errorf("AsAPPError(wrapped).Code = %q, want not_found", got2.Code)
	}

	// Error biasa bukan domain error.
	if _, ok3 := AsAPPError(errors.New("biasa")); ok3 {
		t.Error("AsAPPError(error biasa) = true, want false")
	}
}

func TestKindOfFallsBackToInternal(t *testing.T) {
	// Error yang bukan domain error dianggap internal - supaya tidak pernah
	// bocor sebagai pesan yang lebih spesifik dari yang seharusnya.
	if got := KindOf(errors.New("biasa")); got != KindInternal {
		t.Errorf("KindOf(biasa) = %v, want KindInternal", got)
	}
	if got := KindOf(NotFound("x")); got != KindNotFound {
		t.Errorf("KindOf(NotFound) = %v, want KindNotFound", got)
	}
}

func TestPredicates(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		isNotFound bool
		isConflict bool
		isValid    bool
	}{
		{"not found", NotFound("X"), true, false, false},
		{"conflict", Conflict("X"), false, true, false},
		{"validation", Validation("X"), false, false, true},
		{"unprocessable", Unprocessable("X"), false, false, true},
		{"internal", Internal(errors.New("x")), false, false, false},
		{"nil", nil, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.isNotFound {
				t.Errorf("IsNotFound = %v, want %v", got, tt.isNotFound)
			}
			if got := IsConflict(tt.err); got != tt.isConflict {
				t.Errorf("IsConflict = %v, want %v", got, tt.isConflict)
			}
			if got := IsValidation(tt.err); got != tt.isValid {
				t.Errorf("IsValidation = %v, want %v", got, tt.isValid)
			}
		})
	}
}

func TestPostgresViolationDetectors(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		unique bool
		fk     bool
		check  bool
	}{
		{"unique constraint", errors.New(`ERROR: duplicate key value violates unique constraint "x" (SQLSTATE 23505)`), true, false, false},
		{"fk violation", errors.New(`ERROR: insert or update violates foreign key constraint (SQLSTATE 23503)`), false, true, false},
		{"check violation", errors.New(`ERROR: new row violates check constraint "y" (SQLSTATE 23514)`), false, false, true},
		{"unrelated", errors.New("connection refused"), false, false, false},
		{"nil", nil, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsUniqueViolation(tt.err); got != tt.unique {
				t.Errorf("IsUniqueViolation = %v, want %v", got, tt.unique)
			}
			if got := IsForeignKeyViolation(tt.err); got != tt.fk {
				t.Errorf("IsForeignKeyViolation = %v, want %v", got, tt.fk)
			}
			if got := IsCheckViolation(tt.err); got != tt.check {
				t.Errorf("IsCheckViolation = %v, want %v", got, tt.check)
			}
		})
	}
}

func TestRateLimitedCarriesRetryAfter(t *testing.T) {
	err := RateLimited("Terlalu banyak percobaan", 900)
	if err.RetryAfterDetik != 900 {
		t.Errorf("RetryAfterDetik = %d, want 900", err.RetryAfterDetik)
	}
	if err.Kind != KindRateLimited {
		t.Errorf("Kind = %v, want KindRateLimited", err.Kind)
	}
}

func TestNotFoundMentionsResource(t *testing.T) {
	// Pesan harus menyebut resource, bukan sekadar "404" - supaya user tahu
	// apa yang hilang tanpa membocorkan struktur internal.
	err := NotFound("Payment receipt")
	if !contains(err.Message, "Payment receipt") {
		t.Errorf("Message = %q, want it to name the resource", err.Message)
	}
}

func TestErrorStringIncludesCodeAndKind(t *testing.T) {
	err := New(KindConflict, "payment.invalid_transition", "transisi tidak valid")
	s := err.Error()
	if !contains(s, "payment.invalid_transition") || !contains(s, "conflict") {
		t.Errorf("Error() = %q, want it to include code and kind", s)
	}
}
