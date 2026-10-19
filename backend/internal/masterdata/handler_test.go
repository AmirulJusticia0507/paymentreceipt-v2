package masterdata

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
)

func newTestHandler(store *stubStore) *Handler[stubItem] {
	return NewHandler(newTestService(store), func(item stubItem) any {
		return map[string]any{
			"id":   item.ID,
			"kode": item.Kode,
			"nama": item.Nama,
		}
	})
}

func decodeResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v; body=%q", err, recorder.Body.String())
	}
	return body
}

func TestHandlerCreate_HappyPath(t *testing.T) {
	store := &stubStore{}
	handler := newTestHandler(store)
	request := httptest.NewRequest(http.MethodPost, "/departments", strings.NewReader(
		`{"kode":"ga","nama":"General Affairs"}`,
	))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	handler.Create(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusCreated, response.Body.String())
	}
	body := decodeResponse(t, response)
	data, ok := body["data"].(map[string]any)
	if !ok {
		t.Fatalf("data = %#v, want object", body["data"])
	}
	if data["kode"] != "GA" || data["nama"] != "General Affairs" {
		t.Errorf("data = %#v, want normalized created item", data)
	}
	if store.lastCreateInput.Kode != "GA" {
		t.Errorf("store kode = %q, want GA", store.lastCreateInput.Kode)
	}
}

func TestHandlerCreate_RejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        string
	}{
		{name: "malformed JSON", contentType: "application/json", body: `{"kode":`},
		{name: "unknown field", contentType: "application/json", body: `{"kode":"GA","nama":"X","rahasia":true}`},
		{name: "wrong content type", contentType: "text/plain", body: `{"kode":"GA","nama":"X"}`},
		{name: "business validation", contentType: "application/json", body: `{"kode":"!","nama":"X"}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &stubStore{}
			handler := newTestHandler(store)
			request := httptest.NewRequest(http.MethodPost, "/departments", strings.NewReader(test.body))
			request.Header.Set("Content-Type", test.contentType)
			response := httptest.NewRecorder()

			handler.Create(response, request)

			if response.Code != http.StatusUnprocessableEntity {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
			}
			problem := decodeResponse(t, response)
			if problem["status"] != float64(http.StatusUnprocessableEntity) {
				t.Errorf("problem status = %#v, want %d", problem["status"], http.StatusUnprocessableEntity)
			}
			if store.lastCreateInput.Kode != "" {
				t.Error("store dipanggil walaupun request tidak valid")
			}
		})
	}
}

func TestHandlerGetByID_RejectsInvalidPathID(t *testing.T) {
	handler := newTestHandler(&stubStore{})
	request := httptest.NewRequest(http.MethodGet, "/departments/abc", nil)
	request.SetPathValue("id", "abc")
	response := httptest.NewRecorder()

	handler.GetByID(response, request)

	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want %d; body=%s", response.Code, http.StatusUnprocessableEntity, response.Body.String())
	}
}

func TestHandlerWriteAuthorization(t *testing.T) {
	authenticator := &auth.Authenticator{}
	called := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusCreated)
	})
	protected := authenticator.RequireRole(auth.RoleAdmin)(next)

	tests := []struct {
		name       string
		role       string
		wantStatus int
		wantCalled bool
	}{
		{name: "viewer forbidden", role: auth.RoleViewer, wantStatus: http.StatusForbidden},
		{name: "GA staff forbidden", role: auth.RoleGAStaff, wantStatus: http.StatusForbidden},
		{name: "admin allowed", role: auth.RoleAdmin, wantStatus: http.StatusCreated, wantCalled: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called = false
			request := httptest.NewRequest(http.MethodPost, "/departments", nil)
			identity := auth.Identity{UserID: 1, Username: "tester", Role: test.role}
			request = request.WithContext(auth.WithIdentity(context.Background(), identity))
			response := httptest.NewRecorder()

			protected.ServeHTTP(response, request)

			if response.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d; body=%s", response.Code, test.wantStatus, response.Body.String())
			}
			if called != test.wantCalled {
				t.Errorf("next called = %v, want %v", called, test.wantCalled)
			}
		})
	}
}
