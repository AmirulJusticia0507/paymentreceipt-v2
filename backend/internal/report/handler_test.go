package report

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	date, err := parseDate("2026-11-09", "date_from")
	if err != nil {
		t.Fatalf("parseDate() error = %v", err)
	}
	if want := time.Date(2026, time.November, 9, 0, 0, 0, 0, time.UTC); !date.Equal(want) {
		t.Errorf("parseDate() = %v, want %v", date, want)
	}

	if _, err := parseDate("09-11-2026", "date_to"); err == nil {
		t.Fatal("parseDate() error = nil, want invalid format error")
	}
}

func TestMount_OperationalEndpointValidatesDateRange(t *testing.T) {
	router := chi.NewRouter()
	router.Route("/api/v1", func(r chi.Router) { Mount(r, nil) })

	tests := []struct {
		name string
		path string
		want int
	}{
		{name: "missing dates", path: "/api/v1/reports/operational", want: http.StatusUnprocessableEntity},
		{name: "invalid dates", path: "/api/v1/reports/operational?date_from=2026-11-10&date_to=2026-11-09", want: http.StatusUnprocessableEntity},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d; body=%s", rec.Code, tt.want, rec.Body.String())
			}
		})
	}
}
