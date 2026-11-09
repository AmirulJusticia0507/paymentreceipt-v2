package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

type receiptNotificationDTO struct {
	ID             int64  `json:"id"`
	Kode           string `json:"kode"`
	Status         string `json:"status"`
	DepartmentNama string `json:"department_nama"`
	DaysWaiting    int32  `json:"days_waiting"`
}

func mountReceiptNotifications(r chi.Router, a *auth.Authenticator, q *dbgen.Queries) {
	r.With(a.RequireRole(auth.RoleAdmin, auth.RoleGAStaff, auth.RoleApprover, auth.RoleViewer, auth.RoleRequester)).Get("/notifications/receipts", func(w http.ResponseWriter, req *http.Request) {
		identity, _ := auth.IdentityFrom(req.Context())
		var dept pgtype.Int8
		if identity.Role == auth.RoleRequester {
			if identity.DepartmentID <= 0 {
				httputil.Problemf(w, req, apperror.Forbidden("Akun requester belum ditetapkan ke department"))
				return
			}
			dept = pgtype.Int8{Int64: identity.DepartmentID, Valid: true}
		}
		items, err := q.ListReceiptNotifications(req.Context(), dept)
		if err != nil {
			httputil.Problemf(w, req, err)
			return
		}
		data := make([]receiptNotificationDTO, 0, len(items))
		for _, item := range items {
			data = append(data, receiptNotificationDTO{ID: item.ID, Kode: item.Kode, Status: string(item.Status), DepartmentNama: item.DepartmentNama, DaysWaiting: item.DaysWaiting})
		}
		httputil.JSON(w, http.StatusOK, map[string]any{"data": data})
	})
}
