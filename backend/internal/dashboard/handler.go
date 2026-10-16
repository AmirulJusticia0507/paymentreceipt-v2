// Package dashboard menangani endpoint ringkasan untuk halaman dashboard.
package dashboard

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/paymentreceipt"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/paymentreceiptstore"
)

type handler struct {
	q *dbgen.Queries
}

// summary menangani GET /api/v1/dashboard/summary.
func (h *handler) summary(w http.ResponseWriter, r *http.Request) {
	counts, err := h.q.DashboardCounts(r.Context())
	if err != nil {
		httputil.Problemf(w, r, apperror.Wrap(apperror.KindInternal, "dashboard.summary_failed", "Gagal memuat statistik", err))
		return
	}
	httputil.JSONData(w, http.StatusOK, map[string]any{
		"total_requests":         counts.TotalRequests,
		"total_payment_receipts": counts.TotalPaymentReceipts,
		"total_bases":            counts.TotalBases,
		"total_departments":      counts.TotalDepartments,
	})
}

// recent menangani GET /api/v1/dashboard/recent.
func (h *handler) recent(w http.ResponseWriter, r *http.Request) {
	store := paymentreceiptstore.New(h.q)
	items, _, err := store.List(r.Context(), paymentreceipt.Filters{
		Limit:  10,
		Offset: 0,
		Sort:   "tanggal_desc",
	})
	if err != nil {
		httputil.Problemf(w, r, apperror.Wrap(apperror.KindInternal, "dashboard.recent_failed", "Gagal memuat transaksi terakhir", err))
		return
	}
	httputil.JSONData(w, http.StatusOK, items)
}

// statusBreakdown menangani GET /api/v1/dashboard/status-breakdown.
func (h *handler) statusBreakdown(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.DashboardStatusBreakdown(r.Context())
	if err != nil {
		httputil.Problemf(w, r, apperror.Wrap(apperror.KindInternal, "dashboard.status_failed", "Gagal memuat breakdown status", err))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{"status": string(r.Status), "count": r.Count})
	}
	httputil.JSONData(w, http.StatusOK, out)
}

// monthlySpend menangani GET /api/v1/dashboard/monthly-spend.
func (h *handler) monthlySpend(w http.ResponseWriter, r *http.Request) {
	rows, err := h.q.DashboardMonthlySpend(r.Context())
	if err != nil {
		httputil.Problemf(w, r, apperror.Wrap(apperror.KindInternal, "dashboard.monthly_spend_failed", "Gagal memuat total belanja bulanan", err))
		return
	}
	// Isi bulan yang kosong dengan 0.
	spendMap := map[string]string{}
	for _, r := range rows {
		spendMap[r.Bulan.Time.Format("2006-01")] = r.Spend
	}

	now := time.Now()
	out := make([]map[string]any, 0, 12)
	for i := 11; i >= 0; i-- {
		t := now.AddDate(0, -i, 0)
		key := t.Format("2006-01")
		spend := "0"
		if v, ok := spendMap[key]; ok {
			spend = v
		}
		out = append(out, map[string]any{
			"month": key,
			"label": t.Format("Jan 2006"),
			"spend": spend,
		})
	}
	httputil.JSONData(w, http.StatusOK, out)
}

// topDepartments menangani GET /api/v1/dashboard/top-departments.
func (h *handler) topDepartments(w http.ResponseWriter, r *http.Request) {
	limit := int32(5)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 && n <= 20 {
			limit = int32(n)
		}
	}

	rows, err := h.q.DashboardTopDepartments(r.Context(), limit)
	if err != nil {
		httputil.Problemf(w, r, apperror.Wrap(apperror.KindInternal, "dashboard.top_departments_failed", "Gagal memuat top department", err))
		return
	}
	out := make([]map[string]any, 0, len(rows))
	for _, r := range rows {
		out = append(out, map[string]any{
			"id":    r.ID,
			"nama":  r.Nama,
			"total": r.Total,
		})
	}
	httputil.JSONData(w, http.StatusOK, out)
}

// Mount memasang route dashboard di bawah auth guard.
func Mount(r chi.Router, authn *auth.Authenticator, q *dbgen.Queries) {
	h := &handler{q: q}
	r.Route("/dashboard", func(r chi.Router) {
		r.Get("/summary", h.summary)
		r.Get("/recent", h.recent)
		r.Get("/status-breakdown", h.statusBreakdown)
		r.Get("/monthly-spend", h.monthlySpend)
		r.Get("/top-departments", h.topDepartments)
	})
}

// (chi digunakan melalui tipe chi.Router pada mountDashboard).
