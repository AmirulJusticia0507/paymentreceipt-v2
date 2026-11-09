package router

import (
	"net/http"
	"strconv"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
)

// pathID membaca dan memvalidasi parameter {id} dari path.
//
// Terpisah dari padanannya di package masterdata karena fungsi itu unexported
// dan sengaja tidak diekspor: memvalidasi id adalah urusan layer HTTP, bukan
// urusan domain. Dua implementasi kecil ini lebih baik daripada satu diekspor
// hanya untuk dipakai satu package lain.
func pathID(r *http.Request) (int64, error) {
	// r.PathValue tersedia sejak Go 1.22 lewat pola route {id} milik chi.
	raw := r.PathValue("id")
	if raw == "" {
		return 0, apperror.FieldValidation("id", "ID wajib diisi", "required")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		// Pesannya menyebut format yang diharapkan, bukan "invalid syntax".
		return 0, apperror.FieldValidation("id", "ID harus berupa angka", "numeric")
	}
	if id <= 0 {
		return 0, apperror.FieldValidation("id", "ID harus lebih besar dari 0", "range")
	}

	return id, nil
}

// queryInt64 membaca parameter query sebagai int64.
//
// Nilai yang tidak valid menghasilkan 0, yang berarti "tanpa filter" - bukan
// error. Alasannya parameter ini hanya mempersempit hasil (filter per
// department), jadi input rusak sebaiknya mengabaikan filter, bukan menolak
// seluruh request.
func queryInt64(r *http.Request, name string) int64 {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return 0
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func departmentScope(r *http.Request) (int64, error) {
	identity, ok := auth.IdentityFrom(r.Context())
	if !ok || identity.Role != auth.RoleRequester {
		return 0, nil
	}
	if identity.DepartmentID <= 0 {
		return 0, apperror.Forbidden("Akun requester belum ditetapkan ke department")
	}
	return identity.DepartmentID, nil
}

func checkDepartmentAccess(r *http.Request, resourceDepartmentID int64) error {
	departmentID, err := departmentScope(r)
	if err != nil {
		return err
	}
	if departmentID > 0 && departmentID != resourceDepartmentID {
		return apperror.NotFound("Data")
	}
	return nil
}
