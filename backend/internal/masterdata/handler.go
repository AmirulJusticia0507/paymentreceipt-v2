package masterdata

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// Handler melayani endpoint CRUD untuk satu tabel master.
//
// Generic terhadap tipe data, tapi http.Handler dari cara kerja yang sama untuk
// semua tabel master. Response memakai bentuk yang sama supaya frontend bisa
// memakai satu komponen DataTable untuk semuanya.
type Handler[T any] struct {
	service *Service[T]
	// toResponse mengubah tipe sqlc menjadi bentuk JSON yang dikirim ke klien.
	// Field password_hash dan internal lain tidak pernah ikut karena pemetaan
	// ini ditulis eksplisit, bukan dengan marshal seluruh struct.
	toResponse func(T) any
}

// NewHandler membuat Handler.
func NewHandler[T any](
	service *Service[T],
	toResponse func(T) any,
) *Handler[T] {
	return &Handler[T]{service: service, toResponse: toResponse}
}

// ListBody adalah body POST/PUT untuk tabel master.
//
// Keterangan opsional supaya satuan - yang tidak punya kolom keterangan - tetap
// bisa memakai struct ini tanpa perubahan.
type ListBody struct {
	Query string `json:"q"`
}

// WriteBody adalah body create/update.
type WriteBody struct {
	Kode       string `json:"kode"`
	Nama       string `json:"nama"`
	Keterangan string `json:"keterangan,omitempty"`
}

// List menangani GET /<resource>.
func (h *Handler[T]) List(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)

	input := ListInput{
		Query:  r.URL.Query().Get("q"),
		Limit:  int32(p.Limit()),
		Offset: int32(p.Offset()),
	}

	items, total, err := h.service.List(r.Context(), input)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	// Ditempelkan di response, bukan ditulis ke database: meta selalu berasal
	// dari hasil perhitungan saat request berjalan.
	page := httputil.NewPagination(p.Page, p.PerPage, total)

	responses := make([]any, 0, len(items))
	for _, item := range items {
		responses = append(responses, h.toResponse(item))
	}

	httputil.JSON(w, http.StatusOK, map[string]any{
		"data": responses,
		"meta": page,
	})
}

// GetByID menangani GET /<resource>/{id}.
func (h *Handler[T]) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.service.GetByID(r.Context(), id)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, h.toResponse(item))
}

// Create menangani POST /<resource>.
func (h *Handler[T]) Create(w http.ResponseWriter, r *http.Request) {
	var body WriteBody
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.service.Create(r.Context(), WriteInput{
		Kode:       body.Kode,
		Nama:       body.Nama,
		Keterangan: body.Keterangan,
	})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusCreated, h.toResponse(item))
}

// Update menangani PUT /<resource>/{id}.
func (h *Handler[T]) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	var body WriteBody
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.service.Update(r.Context(), id, WriteInput{
		Kode:       body.Kode,
		Nama:       body.Nama,
		Keterangan: body.Keterangan,
	})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, h.toResponse(item))
}

// Delete menangani DELETE /<resource>/{id}.
func (h *Handler[T]) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	if _, err := h.service.Delete(r.Context(), id); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	// 204 tanpa body: request berhasil dan tidak ada lagi yang perlu dikirim.
	w.WriteHeader(http.StatusNoContent)
}

// pathID membaca dan memvalidasi parameter {id} dari path.
func pathID(r *http.Request) (int64, error) {
	// r.PathValue tersedia sejak Go 1.22 lewat pola route {id} milik chi,
	// jadi tidak perlu parse URL manual.
	raw := r.PathValue("id")
	if raw == "" {
		return 0, apperror.FieldValidation("id", "ID wajib diisi", "required")
	}

	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		// Pesannya menyebut format yang diharapkan, bukan "invalid syntax".
		return 0, apperror.FieldValidation("id", "ID harus berupa angka", "numeric")
	}
	if id <= 0 {
		return 0, apperror.FieldValidation("id", "ID harus lebih besar dari 0", "range")
	}

	return id, nil
}

// IsNotFoundError melaporkan apakah err adalah error "tidak ditemukan".
//
// Menghemat boilerplate di adapter per tabel, yang semuanya menghadapi hal yang sama:
// query mengembalikan pgx.ErrNoRows dan harus diterjemahkan jadi 404.
func IsNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) || apperror.IsNotFound(err)
}
