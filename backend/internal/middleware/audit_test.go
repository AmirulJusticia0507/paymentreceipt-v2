package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

type auditWriterStub struct {
	params dbgen.CreateAuditLogParams
	calls  int
	err    error
}

func (s *auditWriterStub) CreateAuditLog(_ context.Context, params dbgen.CreateAuditLogParams) error {
	s.params = params
	s.calls++
	return s.err
}

func TestAuditMutations_RecordsSuccessfulMutationWithoutQuery(t *testing.T) {
	writer := &auditWriterStub{}
	handler := AuditMutations(writer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	r := httptest.NewRequest(http.MethodPatch, "/api/v1/payment-receipts/12/status?token=secret", nil)
	ctx := httputil.ContextWithRequestID(r.Context(), "req-12345678")
	r = r.WithContext(auth.WithIdentity(ctx, auth.Identity{UserID: 7, Role: auth.RoleApprover}))
	handler.ServeHTTP(httptest.NewRecorder(), r)

	if writer.calls != 1 {
		t.Fatalf("audit calls = %d, want 1", writer.calls)
	}
	if writer.params.ActorID != 7 || writer.params.HttpMethod != http.MethodPatch || writer.params.StatusCode != http.StatusNoContent {
		t.Fatalf("unexpected audit record: %#v", writer.params)
	}
	if writer.params.RequestPath != "/api/v1/payment-receipts/12/status" {
		t.Errorf("request path = %q, query must not be recorded", writer.params.RequestPath)
	}
	if writer.params.RequestID != "req-12345678" {
		t.Errorf("request ID = %q", writer.params.RequestID)
	}
}

func TestAuditMutations_SkipsReadsAndFailedMutations(t *testing.T) {
	for _, test := range []struct {
		name   string
		method string
		status int
	}{
		{name: "read", method: http.MethodGet, status: http.StatusOK},
		{name: "validation error", method: http.MethodPost, status: http.StatusUnprocessableEntity},
	} {
		t.Run(test.name, func(t *testing.T) {
			writer := &auditWriterStub{}
			handler := AuditMutations(writer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
			}))
			r := httptest.NewRequest(test.method, "/api/v1/payment-receipts", nil)
			r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{UserID: 7}))
			handler.ServeHTTP(httptest.NewRecorder(), r)
			if writer.calls != 0 {
				t.Errorf("audit calls = %d, want 0", writer.calls)
			}
		})
	}
}

func TestAuditMutations_DoesNotChangeResponseWhenStoreFails(t *testing.T) {
	writer := &auditWriterStub{err: errors.New("database unavailable")}
	handler := AuditMutations(writer)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/sources/2", nil)
	r = r.WithContext(auth.WithIdentity(r.Context(), auth.Identity{UserID: 7}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, r)
	if response.Code != http.StatusNoContent {
		t.Fatalf("response status = %d, want %d", response.Code, http.StatusNoContent)
	}
}
