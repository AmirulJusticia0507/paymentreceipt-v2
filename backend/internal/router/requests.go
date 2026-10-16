package router

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/jackc/pgx/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// requestWriteBody adalah body create/update untuk request.
// DepartmentID wajib karena sebuah request selalu miliki satu departemen.
type requestWriteBody struct {
	Kode         string `json:"kode"`
	Nama         string `json:"nama"`
	Keterangan   string `json:"keterangan,omitempty"`
	DepartmentID int64  `json:"department_id"`
}

// requestHandler melayani CRUD requests. Bedanya dengan generic Handler[T]:
// create/update membutuhkan department_id yang divalidasi sebelum disimpan
// ke store, dan list bisa difilter per departemen.
type requestHandler struct {
	svc   *masterdata.Service[dbgen.Request]
	q     *dbgen.Queries
	toDTO func(dbgen.Request) any
}

// List menangani GET /requests.
func (h *requestHandler) List(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)

	departmentID, _ := strconv.ParseInt(r.URL.Query().Get("department_id"), 10, 64)

	input := masterdata.ListInput{
		Query:        r.URL.Query().Get("q"),
		Limit:        int32(p.Limit()),
		Offset:       int32(p.Offset()),
		DepartmentID: departmentID,
	}

	items, total, err := h.svc.List(r.Context(), input)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	page := httputil.NewPagination(p.Page, p.PerPage, total)

	responses := make([]any, 0, len(items))
	for _, item := range items {
		responses = append(responses, h.toDTO(item))
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": responses,
		"meta": page,
	})
}

// GetByID menangani GET /requests/{id}.
func (h *requestHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, h.toDTO(item))
}

// Create menangani POST /requests.
func (h *requestHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body requestWriteBody
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	if body.DepartmentID <= 0 {
		httputil.Problemf(w, r, apperror.FieldValidation("department_id", "Department wajib dipilih", "required"))
		return
	}

	// Departemen tujuan harus ada agar pesan kesalahan spesifik, bukan foreign
	// key violation yang membingungkan dari database.
	if _, err := h.q.GetDepartmentByID(r.Context(), body.DepartmentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.Problemf(w, r, apperror.FieldValidation("department_id", "Department tidak ditemukan", "not_found"))
			return
		}
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.svc.Create(r.Context(), masterdata.WriteInput{
		Kode:         body.Kode,
		Nama:         body.Nama,
		Keterangan:   body.Keterangan,
		DepartmentID: body.DepartmentID,
	})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusCreated, h.toDTO(item))
}

// Update menangani PUT /requests/{id}.
func (h *requestHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	var body requestWriteBody
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	if body.DepartmentID <= 0 {
		httputil.Problemf(w, r, apperror.FieldValidation("department_id", "Department wajib dipilih", "required"))
		return
	}

	if _, err := h.q.GetDepartmentByID(r.Context(), body.DepartmentID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.Problemf(w, r, apperror.FieldValidation("department_id", "Department tidak ditemukan", "not_found"))
			return
		}
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.svc.Update(r.Context(), id, masterdata.WriteInput{
		Kode:         body.Kode,
		Nama:         body.Nama,
		Keterangan:   body.Keterangan,
		DepartmentID: body.DepartmentID,
	})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, h.toDTO(item))
}

// Delete menangani DELETE /requests/{id}.
func (h *requestHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	if _, err := h.svc.Delete(r.Context(), id); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// mountRequests memasang route CRUD untuk tabel requests. Request membutuhkan
// department_id yang wajib, jadi pakai handler khusus alih-alih generic
// Handler[T].
func mountRequests(
	r chi.Router,
	authn *auth.Authenticator,
	queries *dbgen.Queries,
	store masterdata.Store[dbgen.Request],
	path string,
) {
	svc := masterdata.NewService(
		store,
		masterdata.ResourceDescribe{Singular: "Request", Plural: "Request"},
		"payment request",
	)
	h := &requestHandler{svc: svc, q: queries, toDTO: requestDTO}

	admin := authn.RequireRole(auth.RoleAdmin)

	r.Route(path, func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)

		r.With(admin).Post("/", h.Create)
		r.With(admin).Put("/{id}", h.Update)
		r.With(admin).Delete("/{id}", h.Delete)
	})
}
