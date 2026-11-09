// Package auditlog provides the administrator view of recorded API mutations.
package auditlog

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

type handler struct{ q *dbgen.Queries }

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)
	rows, err := h.q.ListAuditLogs(r.Context(), dbgen.ListAuditLogsParams{
		PageLimit: int32(p.Limit()), PageOffset: int32(p.Offset()),
	})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	total, err := h.q.CountAuditLogs(r.Context())
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	data := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		data = append(data, map[string]any{
			"id": row.ID, "actor_id": row.ActorID, "username": row.Username,
			"full_name": row.FullName, "http_method": row.HttpMethod,
			"request_path": row.RequestPath, "status_code": row.StatusCode,
			"request_id": row.RequestID, "created_at": row.CreatedAt,
		})
	}
	httputil.JSONList(w, data, httputil.NewPagination(p.Page, p.PerPage, total))
}

// Mount mendaftarkan endpoint audit yang hanya dapat dibaca admin.
func Mount(r chi.Router, authn *auth.Authenticator, q *dbgen.Queries) {
	h := &handler{q: q}
	r.Route("/audit-logs", func(r chi.Router) {
		r.Use(authn.RequireRole(auth.RoleAdmin))
		r.Get("/", h.list)
	})
}
