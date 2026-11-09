// Package report menyediakan laporan operasional.
package report

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

type handler struct{ q *dbgen.Queries }

func parseDate(value, field string) (time.Time, error) {
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return time.Time{}, apperror.FieldValidation(field, "Tanggal harus berformat YYYY-MM-DD", "date")
	}
	return date, nil
}

func (h *handler) operational(w http.ResponseWriter, r *http.Request) {
	from, err := parseDate(r.URL.Query().Get("date_from"), "date_from")
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	to, err := parseDate(r.URL.Query().Get("date_to"), "date_to")
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	if to.Before(from) {
		httputil.Problemf(w, r, apperror.FieldValidation("date_to", "Tanggal akhir harus sama atau setelah tanggal awal", "range"))
		return
	}

	var departmentID pgtype.Int8
	if raw := r.URL.Query().Get("department_id"); raw != "" {
		id, parseErr := strconv.ParseInt(raw, 10, 64)
		if parseErr != nil || id <= 0 {
			httputil.Problemf(w, r, apperror.FieldValidation("department_id", "Department tidak valid", "number"))
			return
		}
		departmentID = pgtype.Int8{Int64: id, Valid: true}
	}

	rows, err := h.q.OperationalReport(r.Context(), dbgen.OperationalReportParams{
		DateFrom:     pgtype.Date{Time: from, Valid: true},
		DateTo:       pgtype.Date{Time: to, Valid: true},
		DepartmentID: departmentID,
	})
	if err != nil {
		httputil.Problemf(w, r, apperror.Wrap(apperror.KindInternal, "report.operational_failed", "Gagal memuat laporan operasional", err))
		return
	}

	data := make([]map[string]any, 0, len(rows))
	for _, row := range rows {
		data = append(data, map[string]any{
			"department_id":   row.DepartmentID,
			"department_kode": row.DepartmentKode,
			"department_nama": row.DepartmentNama,
			"status":          string(row.Status),
			"receipt_count":   row.ReceiptCount,
			"total_amount":    row.TotalAmount,
		})
	}
	httputil.JSONData(w, http.StatusOK, data)
}

// Mount mendaftarkan endpoint laporan di dalam grup autentikasi.
func Mount(r chi.Router, q *dbgen.Queries) {
	h := &handler{q: q}
	r.Route("/reports", func(r chi.Router) {
		r.Get("/operational", h.operational)
	})
}
