package authservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
)

// RequesterRegistrationStore persists self-registration and lists selectable departments.
type RequesterRegistrationStore interface {
	RegisterRequester(context.Context, dbgen.RegisterRequesterParams) (int64, error)
	ListDepartments(context.Context, dbgen.ListDepartmentsParams) ([]dbgen.Department, error)
}

// RegisterRequester creates an inactive requester account pending admin approval.
func RegisterRequester(ctx context.Context, store RequesterRegistrationStore, params auth.Params, username, password, phone string, departmentID int64) error {
	username = strings.TrimSpace(username)
	phone = strings.TrimSpace(phone)
	if len(username) < 3 || len(username) > 50 {
		return apperror.FieldValidation("username", "Username harus 3–50 karakter", "length")
	}
	for _, ch := range username {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || strings.ContainsRune("._-", ch)) {
			return apperror.FieldValidation("username", "Username hanya boleh berisi huruf, angka, titik, garis bawah, dan tanda hubung", "format")
		}
	}
	if len(password) < 8 || len(password) > 200 {
		return apperror.FieldValidation("password", "Password harus 8–200 karakter", "length")
	}
	if len(phone) < 8 || len(phone) > 30 {
		return apperror.FieldValidation("phone", "Nomor HP harus 8–30 karakter", "length")
	}
	for _, ch := range phone {
		if !(ch >= '0' && ch <= '9' || strings.ContainsRune("+()- ", ch)) {
			return apperror.FieldValidation("phone", "Format nomor HP tidak valid", "format")
		}
	}
	if departmentID <= 0 {
		return apperror.FieldValidation("department_id", "Department wajib dipilih", "required")
	}
	hash, err := auth.Hash(password, params)
	if err != nil {
		return apperror.Internalf("auth.password_hash", err)
	}
	rows, err := store.RegisterRequester(ctx, dbgen.RegisterRequesterParams{
		Username: username, Phone: pgtype.Text{String: phone, Valid: true},
		PasswordHash: hash, FullName: username, DepartmentID: departmentID,
	})
	if err != nil {
		var pgErr interface{ SQLState() string }
		if errors.As(err, &pgErr) && pgErr.SQLState() == "23505" {
			return apperror.Conflict("Username sudah digunakan")
		}
		return apperror.Internalf("auth.requester_registration_failed", fmt.Errorf("register requester: %w", err))
	}
	if rows == 0 {
		return apperror.FieldValidation("department_id", "Department tidak ditemukan atau tidak aktif", "not_found")
	}
	return nil
}
