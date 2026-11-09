package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
)

type identityUserStore struct{ user dbgen.GetUserSafeByIDRow }

func (s identityUserStore) GetUserSafeByID(context.Context, int64) (dbgen.GetUserSafeByIDRow, error) {
	return s.user, nil
}

func TestAuthenticator_UsesCurrentRoleAndDepartmentFromDatabase(t *testing.T) {
	tokens := testManager(time.Minute)
	access, _, err := tokens.IssueAccess(9, "stale-name", RoleAdmin, 4)
	if err != nil {
		t.Fatal(err)
	}
	authenticator := NewAuthenticator(tokens, CookieOptions{AccessName: "access"}, identityUserStore{
		user: dbgen.GetUserSafeByIDRow{
			ID: 9, Username: "requester", Role: dbgen.UserRoleRequester, IsActive: true,
			DepartmentID: pgtype.Int8{Int64: 12, Valid: true},
		},
	})
	var got Identity
	endpoint := authenticator.Middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ = IdentityFrom(r.Context())
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/payment-receipts", nil)
	request.Header.Set("Authorization", "Bearer "+access)
	response := httptest.NewRecorder()
	endpoint.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got.Role != RoleRequester || got.DepartmentID != 12 || got.Username != "requester" {
		t.Fatalf("identity = %#v, want current database identity", got)
	}
}

func TestAuthenticator_RequireRoleRejectsRoleOutsideModule(t *testing.T) {
	authenticator := NewAuthenticator(testManager(time.Minute), CookieOptions{})
	called := false
	endpoint := authenticator.RequireRole(RoleAdmin, RoleGAStaff)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "/api/v1/sources", nil)
	request = request.WithContext(WithIdentity(request.Context(), Identity{UserID: 8, Role: RoleRequester}))
	response := httptest.NewRecorder()
	endpoint.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
	if called {
		t.Fatal("protected handler was called for requester")
	}
}
