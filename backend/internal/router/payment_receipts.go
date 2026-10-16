package router

import (
	"net/http"
	"strings"
	"time"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/paymentreceipt"
	"github.com/go-chi/chi/v5"
)

type paymentReceiptHandler struct{ service *paymentreceipt.Service }
type createPaymentReceiptBody struct {
	RequestID        int64   `json:"request_id"`
	DepartmentID     int64   `json:"department_id"`
	SourceID         *int64  `json:"source_id"`
	CategoryID       *int64  `json:"category_id"`
	UnitID           *int64  `json:"unit_id"`
	TanggalPembelian string  `json:"tanggal_pembelian"`
	Usage            string  `json:"usage"`
	DeskripsiItems   string  `json:"deskripsi_items"`
	JumlahItems      int32   `json:"jumlah_items"`
	HargaItems       string  `json:"harga_items"`
	PriceEstimate    *string `json:"price_estimate"`
	Keterangan       string  `json:"keterangan"`
}

func parseDateField(value, field string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return nil, apperror.FieldValidation(field, "Tanggal harus berformat YYYY-MM-DD", "date")
	}
	return &parsed, nil
}
func (h *paymentReceiptHandler) list(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)
	from, err := parseDateField(r.URL.Query().Get("date_from"), "date_from")
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	to, err := parseDateField(r.URL.Query().Get("date_to"), "date_to")
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	if from != nil && to != nil && from.After(*to) {
		httputil.Problemf(w, r, apperror.FieldValidation("date_from", "Tanggal awal tidak boleh setelah tanggal akhir", "range"))
		return
	}
	statusValue := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("status")))
	if statusValue != "" {
		allowedStatus := map[string]bool{"pending": true, "checked": true, "processed": true, "finished": true, "cancelled": true}
		if !allowedStatus[statusValue] {
			httputil.Problemf(w, r, apperror.FieldValidation("status", "Status tidak valid", "enum"))
			return
		}
	}
	sort := r.URL.Query().Get("sort")
	allowed := map[string]bool{"tanggal_asc": true, "tanggal_desc": true, "kode_asc": true, "kode_desc": true, "total_asc": true, "total_desc": true}
	if !allowed[sort] {
		sort = "tanggal_desc"
	}
	items, total, err := h.service.List(r.Context(), paymentreceipt.Filters{Status: statusValue, DepartmentID: queryInt64(r, "department_id"), RequestID: queryInt64(r, "request_id"), DateFrom: from, DateTo: to, Query: strings.TrimSpace(r.URL.Query().Get("q")), Sort: sort, Limit: int32(p.Limit()), Offset: int32(p.Offset())})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"data": items, "meta": httputil.NewPagination(p.Page, p.PerPage, total)})
}
func (h *paymentReceiptHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	item, err := h.service.Get(r.Context(), id)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	httputil.JSONData(w, http.StatusOK, item)
}
func (h *paymentReceiptHandler) create(w http.ResponseWriter, r *http.Request) {
	var body createPaymentReceiptBody
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	purchaseDate, err := parseDateField(body.TanggalPembelian, "tanggal_pembelian")
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	var date time.Time
	if purchaseDate != nil {
		date = *purchaseDate
	}
	item, err := h.service.Create(r.Context(), paymentreceipt.CreateInput{RequestID: body.RequestID, DepartmentID: body.DepartmentID, SourceID: body.SourceID, CategoryID: body.CategoryID, UnitID: body.UnitID, TanggalPembelian: date, Usage: body.Usage, DeskripsiItems: body.DeskripsiItems, JumlahItems: body.JumlahItems, HargaItems: body.HargaItems, PriceEstimate: body.PriceEstimate, Keterangan: body.Keterangan})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	httputil.JSONData(w, http.StatusCreated, item)
}
func mountPaymentReceipts(r chi.Router, a *auth.Authenticator, service *paymentreceipt.Service) {
	h := &paymentReceiptHandler{service: service}
	r.Route("/payment-receipts", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
		r.With(a.RequireRole(auth.RoleAdmin, auth.RoleGAStaff)).Post("/", h.create)
	})
}
