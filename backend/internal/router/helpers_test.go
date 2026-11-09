package router

import (
	"net/http/httptest"
	"testing"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
)

func TestDepartmentScope_RequesterIsScoped(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{Role: auth.RoleRequester, DepartmentID: 12}))
	id, err := departmentScope(r)
	if err != nil || id != 12 {
		t.Fatalf("departmentScope() = %d, %v; want 12, nil", id, err)
	}
}

func TestDepartmentScope_UnassignedRequesterIsRejected(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{Role: auth.RoleRequester}))
	if _, err := departmentScope(r); err == nil {
		t.Fatal("departmentScope() error = nil, want unassigned requester rejection")
	}
}

func TestCheckDepartmentAccess_HidesOtherDepartment(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{Role: auth.RoleRequester, DepartmentID: 12}))
	if err := checkDepartmentAccess(r, 13); err == nil {
		t.Fatal("checkDepartmentAccess() error = nil, want cross-department rejection")
	}
}
