package router

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// requestBaseWriteBody adalah body create/update untuk request base.
//
// BaseID wajib: setiap request base harus punya induk. Jabatan diisi sebagai
// penanggung jawab pengajuan dan boleh kosong.
type requestBaseWriteBody struct {
	Kode       string `json:"kode"`
	Nama       string `json:"nama"`
	Jabatan    string `json:"jabatan"`
	Keterangan string `json:"keterangan,omitempty"`
	BaseID     int64  `json:"base_id"`
}

// requestBaseHandler melayani CRUD request base.
//
// .Handler generik tidak bisa dipakai karena dua hal: base_id wajib ada dan
// harus diverifikasi sebelum menulis, serta ada kolom jabatan yang tidak ada di
// tabel master lain.
type requestBaseHandler struct {
	svc     *masterdata.Service[dbgen.RequestBase]
	queries *dbgen.Queries
}

// List menangani GET /request-bases.
func (h *requestBaseHandler) List(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)

	input := masterdata.ListInput{
		Query:  r.URL.Query().Get("q"),
		Limit:  int32(p.Limit()),
		Offset: int32(p.Offset()),
		// base_id tidak valid berarti ""tampilkan semua"", bukan error.
		BaseID: queryInt64(r, "base_id"),
	}

	items, total, err := h.svc.List(r.Context(), input)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	responses := make([]any, 0, len(items))
	for _, item := range items {
		responses = append(responses, requestBaseDTO(item))
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": responses,
		"meta": httputil.NewPagination(p.Page, p.PerPage, total),
	})
}

// GetByID menangani GET /request-bases/{id}.
func (h *requestBaseHandler) GetByID(w http.ResponseWriter, r *http.Request) {
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

	httputil.JSONData(w, http.StatusOK, requestBaseDTO(item))
}

// Create menangani POST /request-bases.
func (h *requestBaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	body, ok := h.decodeAndValidate(w, r)
	if !ok {
		return
	}

	item, err := h.svc.Create(r.Context(), body)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusCreated, requestBaseDTO(item))
}

// Update menangani PUT /request-bases/{id}.
func (h *requestBaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	body, ok := h.decodeAndValidate(w, r)
	if !ok {
		return
	}

	item, err := h.svc.Update(r.Context(), id, body)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, requestBaseDTO(item))
}

// Delete menangani DELETE /request-bases/{id}.
func (h *requestBaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
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

// decodeAndValidate membaca body lalu memastikan base_id ada dan department induknya
// benar-benar ada. Mengembalikan false kalau respons sudah ditulis.
func (h *requestBaseHandler) decodeAndValidate(w http.ResponseWriter, r *http.Request) (masterdata.WriteInput, bool) {
	var body requestBaseWriteBody
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return masterdata.WriteInput{}, false
	}

	if body.BaseID <= 0 {
		httputil.Problemf(w, r, apperror.FieldValidation("base_id", "Base wajib dipilih", "required"))
		return masterdata.WriteInput{}, false
	}

	// Base dicek lebih dulu supaya pesannya "Base tidak ditemukan", bukan
	// foreign key violation dari database yang tidakEPOGRAMkebalkan user.
	if _, err := h.queries.GetBaseByID(r.Context(), body.BaseID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httputil.Problemf(w, r, apperror.FieldValidation("base_id", "Base tidak ditemukan", "not_found"))
			return masterdata.WriteInput{}, false
		}
		httputil.Problemf(w, r, err)
		return masterdata.WriteInput{}, false
	}

	return masterdata.WriteInput{
		Kode:       body.Kode,
		Nama:       body.Nama,
		Jabatan:    body.Jabatan,
		Keterangan: body.Keterangan,
		BaseID:     body.BaseID,
	}, true
}

// requestBaseDTO memetakan baris ke bentuk JSON yang dikirim ke klien.
func requestBaseDTO(d dbgen.RequestBase) any {
	return map[string]any{
		"id":         d.ID,
		"kode":       d.Kode,
		"nama":       d.Nama,
		"jabatan":    d.Jabatan,
		"keterangan": d.Keterangan,
		"base_id":    d.BaseID,
	}
}

// mountRequestBases memasang route CRUD untuk request base.
//
// Route ini harus dipasang SEBELUM /bases/{id} dalam router, kalau tidak pola
// /{id} akan menelan /search lebih dulu. Chi mencocokkan dalam urutan
// pendaftaran.
func mountRequestBases(
	r chi.Router,
	authn *auth.Authenticator,
	queries *dbgen.Queries,
	store masterdata.Store[dbgen.RequestBase],
	path string,
) {
	svc := masterdata.NewService(store,
		masterdata.ResourceDescribe{Singular: "Request Base", Plural: "Request Base"},
		"pengajuan")
	h := &requestBaseHandler{svc: svc, queries: queries}

	admin := authn.RequireRole(auth.RoleAdmin)

	r.Route(path, func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)

		r.With(admin).Post("/", h.Create)
		r.With(admin).Put("/{id}", h.Update)
		r.With(admin).Delete("/{id}", h.Delete)
	})
}
