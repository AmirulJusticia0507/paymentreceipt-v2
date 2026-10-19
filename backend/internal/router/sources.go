package router

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

type sourceBody struct {
	Source        string `json:"source"`
	NamaBarang    string `json:"nama_barang"`
	NamaToko      string `json:"nama_toko,omitempty"`
	NoTelp        string `json:"no_telp,omitempty"`
	NoRekening    string `json:"no_rekening,omitempty"`
	LinkPembelian string `json:"link_pembelian,omitempty"`
	Keterangan    string `json:"keterangan,omitempty"`
}

func validateSource(body *sourceBody) error {
	body.Source = strings.ToLower(strings.TrimSpace(body.Source))
	body.NamaBarang = strings.TrimSpace(body.NamaBarang)
	body.NamaToko, body.NoTelp, body.NoRekening = strings.TrimSpace(body.NamaToko), strings.TrimSpace(body.NoTelp), strings.TrimSpace(body.NoRekening)
	body.LinkPembelian, body.Keterangan = strings.TrimSpace(body.LinkPembelian), strings.TrimSpace(body.Keterangan)
	err := apperror.New(apperror.KindValidation, "source.validation", "Data sumber pembelian belum valid")
	if body.NamaBarang == "" {
		err.WithField("nama_barang", "Nama barang wajib diisi", "required")
	}
	if len(body.NamaBarang) > 200 {
		err.WithField("nama_barang", "Nama barang maksimal 200 karakter", "maxlen")
	}
	switch body.Source {
	case "online":
		if body.LinkPembelian == "" {
			err.WithField("link_pembelian", "Link pembelian wajib diisi", "required")
		} else if parsed, parseErr := url.ParseRequestURI(body.LinkPembelian); parseErr != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			err.WithField("link_pembelian", "Link pembelian harus berupa URL HTTP/HTTPS", "url")
		}
		body.NamaToko, body.NoTelp, body.NoRekening = "", "", ""
	case "offline":
		if body.NamaToko == "" {
			err.WithField("nama_toko", "Nama toko wajib diisi", "required")
		}
		if body.NoTelp == "" {
			err.WithField("no_telp", "Nomor telepon wajib diisi", "required")
		}
		if body.NoRekening == "" {
			err.WithField("no_rekening", "Nomor rekening wajib diisi", "required")
		}
		body.LinkPembelian = ""
	default:
		err.WithField("source", "Mode harus online atau offline", "enum")
	}
	if len(err.Fields) > 0 {
		return err
	}
	return nil
}

type sourceHandler struct{ q *dbgen.Queries }

func nullable(value string) pgtype.Text { return pgtype.Text{String: value, Valid: value != ""} }
func sourceDTO(s dbgen.Source) any {
	return map[string]any{"id": s.ID, "source": s.Source, "nama_barang": s.NamaBarang, "nama_toko": s.NamaToko.String, "no_telp": s.NoTelp.String, "no_rekening": s.NoRekening.String, "link_pembelian": s.LinkPembelian.String, "keterangan": s.Keterangan}
}

func (h *sourceHandler) list(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)
	mode := strings.ToLower(r.URL.Query().Get("source"))
	filter := dbgen.NullSourceType{}
	if mode == "online" || mode == "offline" {
		filter = dbgen.NullSourceType{SourceType: dbgen.SourceType(mode), Valid: true}
	}
	query := nullable(strings.TrimSpace(r.URL.Query().Get("q")))
	items, err := h.q.ListSources(r.Context(), dbgen.ListSourcesParams{Source: filter, Q: query, Limit: int32(p.Limit()), Offset: int32(p.Offset())})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	total, err := h.q.CountSources(r.Context(), dbgen.CountSourcesParams{Source: filter, Q: query})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	data := make([]any, 0, len(items))
	for _, item := range items {
		data = append(data, sourceDTO(item))
	}
	httputil.JSON(w, http.StatusOK, map[string]any{"data": data, "meta": httputil.NewPagination(p.Page, p.PerPage, total)})
}

func (h *sourceHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	item, err := h.q.GetSourceByID(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = apperror.New(apperror.KindNotFound, "source.not_found", "Sumber pembelian tidak ditemukan")
	}
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	httputil.JSONData(w, http.StatusOK, sourceDTO(item))
}

func (h *sourceHandler) decode(w http.ResponseWriter, r *http.Request) (sourceBody, bool) {
	var body sourceBody
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return body, false
	}
	if err := validateSource(&body); err != nil {
		httputil.Problemf(w, r, err)
		return body, false
	}
	return body, true
}

func params(body sourceBody) dbgen.CreateSourceParams {
	return dbgen.CreateSourceParams{Source: dbgen.SourceType(body.Source), NamaBarang: body.NamaBarang, NamaToko: nullable(body.NamaToko), NoTelp: nullable(body.NoTelp), NoRekening: nullable(body.NoRekening), LinkPembelian: nullable(body.LinkPembelian), Keterangan: body.Keterangan}
}

func (h *sourceHandler) create(w http.ResponseWriter, r *http.Request) {
	body, ok := h.decode(w, r)
	if !ok {
		return
	}
	item, err := h.q.CreateSource(r.Context(), params(body))
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	httputil.JSONData(w, http.StatusCreated, sourceDTO(item))
}

func (h *sourceHandler) update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	body, ok := h.decode(w, r)
	if !ok {
		return
	}
	p := params(body)
	item, err := h.q.UpdateSource(r.Context(), dbgen.UpdateSourceParams{ID: id, Source: p.Source, NamaBarang: p.NamaBarang, NamaToko: p.NamaToko, NoTelp: p.NoTelp, NoRekening: p.NoRekening, LinkPembelian: p.LinkPembelian, Keterangan: p.Keterangan})
	if errors.Is(err, pgx.ErrNoRows) {
		err = apperror.New(apperror.KindNotFound, "source.not_found", "Sumber pembelian tidak ditemukan")
	}
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	httputil.JSONData(w, http.StatusOK, sourceDTO(item))
}

func (h *sourceHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	affected, err := h.q.SoftDeleteSource(r.Context(), id)
	if err == nil && affected == 0 {
		err = apperror.New(apperror.KindNotFound, "source.not_found", "Sumber pembelian tidak ditemukan")
	}
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func mountSources(r chi.Router, a *auth.Authenticator, q *dbgen.Queries) {
	h := &sourceHandler{q: q}
	r.Route("/sources", func(r chi.Router) {
		r.Get("/", h.list)
		r.Get("/{id}", h.get)
		r.Group(func(r chi.Router) {
			r.Use(a.RequireRole(auth.RoleAdmin))
			r.Post("/", h.create)
			r.Put("/{id}", h.update)
			r.Delete("/{id}", h.delete)
		})
	})
}
