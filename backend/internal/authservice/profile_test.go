package authservice

import (
	"context"
	"testing"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
)

type profileUserStore struct {
	updated dbgen.UpdateUserFullNameParams
}

func (s *profileUserStore) GetUserByUsername(context.Context, string) (dbgen.User, error) {
	return dbgen.User{}, nil
}

func (s *profileUserStore) GetUserSafeByID(context.Context, int64) (dbgen.GetUserSafeByIDRow, error) {
	return dbgen.GetUserSafeByIDRow{}, nil
}

func (s *profileUserStore) TouchUserLastLogin(context.Context, int64) error { return nil }

func (s *profileUserStore) UpdateUserFullName(
	_ context.Context,
	arg dbgen.UpdateUserFullNameParams,
) (dbgen.UpdateUserFullNameRow, error) {
	s.updated = arg
	return dbgen.UpdateUserFullNameRow{
		ID:       arg.ID,
		Username: "ga_staff",
		Email:    "ga@example.test",
		FullName: arg.FullName,
		Role:     dbgen.UserRoleGaStaff,
		IsActive: true,
	}, nil
}

func TestServiceUpdateFullName_ValidatesAndTrimsName(t *testing.T) {
	store := &profileUserStore{}
	service := New(store, nil, nil, auth.CookieOptions{}, auth.Params{})

	user, err := service.UpdateFullName(context.Background(), 7, "  Susi Pudjiastuti  ")
	if err != nil {
		t.Fatalf("UpdateFullName() error = %v", err)
	}
	if store.updated.ID != 7 || store.updated.FullName != "Susi Pudjiastuti" {
		t.Fatalf("query params = %#v, want trimmed name for user 7", store.updated)
	}
	if user.FullName != "Susi Pudjiastuti" {
		t.Fatalf("FullName = %q, want trimmed name", user.FullName)
	}
}

func TestServiceUpdateFullName_RejectsBlankAndOverlongName(t *testing.T) {
	service := New(&profileUserStore{}, nil, nil, auth.CookieOptions{}, auth.Params{})

	for _, name := range []string{"   ", string(make([]rune, 151))} {
		if _, err := service.UpdateFullName(context.Background(), 7, name); err == nil {
			t.Fatalf("UpdateFullName(%q) error = nil, want validation error", name)
		}
	}
}
