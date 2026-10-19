package router

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/database"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/platform/pdf"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/prarchive"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/prarchivestore"
)

// prArchiveHandler melayani CRUD dan bulk archive untuk tabel pr_archive.
type prArchiveHandler struct {
	svc *prarchive.Service
	db  *database.DB
	q   *dbgen.Queries
}

// bulkArchiveInput adalah body POST /pr-archives/bulk-archive.
type bulkArchiveInput struct {
	Kode              string  `json:"kode"`
	TanggalSelesai    string  `json:"tanggal_selesai"` // RFC3339
	NamaRequest       string  `json:"nama_request"`
	ManagerApproved   string  `json:"manager_approved"`
	Description       string  `json:"description"`
	HargaRencana      string  `json:"harga_rencana"`
	PaymentReceiptIDs []int64 `json:"payment_receipt_ids"`
}

// List menangani GET /pr-archives.
func (h *prArchiveHandler) List(w http.ResponseWriter, r *http.Request) {
	p := httputil.ParsePagination(r)

	dateFrom := parseDate(r.URL.Query().Get("date_from"))
	dateTo := parseDate(r.URL.Query().Get("date_to"))
	approverID, _ := strconv.ParseInt(r.URL.Query().Get("approver"), 10, 64)

	items, total, err := h.svc.List(r.Context(), prarchive.Filters{
		DateFrom:   dateFrom,
		DateTo:     dateTo,
		Query:      r.URL.Query().Get("q"),
		ApproverID: approverID,
		Limit:      int32(p.Limit()),
		Offset:     int32(p.Offset()),
	})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	page := httputil.NewPagination(p.Page, p.PerPage, total)
	httputil.JSONList(w, items, page)
}

// GetByID menangani GET /pr-archives/{id}.
func (h *prArchiveHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, item)
}

// Create menangani POST /pr-archives.
func (h *prArchiveHandler) Create(w http.ResponseWriter, r *http.Request) {
	var body prarchive.CreateInput
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		httputil.Problemf(w, r, apperror.Internal(errors.New("identity tidak ada di context")))
		return
	}
	body.ApprovedBy = &identity.UserID

	item, err := h.svc.Create(r.Context(), body)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusCreated, item)
}

// Update menangani PUT /pr-archives/{id}.
func (h *prArchiveHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	var body prarchive.CreateInput
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	item, err := h.svc.Update(r.Context(), id, body)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusOK, item)
}

// pdf menangani GET /pr-archives/{id}/pdf.
func (h *prArchiveHandler) pdf(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	item, err := h.svc.Get(r.Context(), id)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	data := pdf.ArchiveData{
		Kode:             item.Kode,
		PaymentReceiptID: item.PaymentReceiptID,
		ReceiptKode:      item.ReceiptKode,
		TanggalSelesai:   item.TanggalSelesai,
		NamaRequest:      item.NamaRequest,
		ManagerApproved:  item.ManagerApproved,
		Description:      item.Description,
		Quantity:         item.Quantity,
		HargaRencana:     item.HargaRencana,
		HargaAktual:      item.HargaAktual,
	}
	buf, err := pdf.RenderArchive(data)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", "inline; filename=\"PR-Archive-"+item.Kode+".pdf\"")
	_, _ = w.Write(buf)
}

// Delete menangani DELETE /pr-archives/{id}.
func (h *prArchiveHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r)
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// BulkArchive menangani POST /pr-archives/bulk-archive.
//
// Dalam satu transaksi:
//  1. Validasi semua payment_receipt_ids ada, sudah finished, dan belum diarsipkan.
//  2. Hitung harga_aktual = jumlah total_harga dari receipt yang dipilih.
//  3. Buat satu pr_archive.
//  4. Tandai semua receipt sebagai diarsipkan (archived_at = NOW()).
func (h *prArchiveHandler) BulkArchive(w http.ResponseWriter, r *http.Request) {
	var body bulkArchiveInput
	if err := httputil.DecodeJSON(w, r, &body); err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	if len(body.PaymentReceiptIDs) == 0 {
		httputil.Problemf(w, r, apperror.FieldValidation("payment_receipt_ids", "Minimal satu payment receipt harus dipilih", "required"))
		return
	}

	tanggalSelesai, err := time.Parse(time.RFC3339, body.TanggalSelesai)
	if err != nil {
		httputil.Problemf(w, r, apperror.FieldValidation("tanggal_selesai", "Format tanggal tidak valid (RFC3339)", "format"))
		return
	}

	identity, ok := auth.IdentityFrom(r.Context())
	if !ok {
		httputil.Problemf(w, r, apperror.Internal(errors.New("identity tidak ada di context")))
		return
	}

	// kode diisi di dalam transaksi: dari body kalau disediakan, atau
	// digenerate oleh sekuens database. Diangkat keluar closure supaya respons
	// bisa mengembalikan kode hasil generate - bukan string kosong dari body.
	var kode string
	err = database.WithTx(r.Context(), h.db, func(tx database.Tx) error {
		// 1. Validasi dan hitung agregasi dalam satu query.
		var qty int
		var totalHarga pgtype.Numeric
		err := tx.QueryRow(r.Context(), `
			SELECT count(*), coalesce(sum(p.total_harga), 0)
			FROM payment_receipt p
			WHERE p.id = ANY($1::bigint[])
			  AND p.status = 'finished'::payment_status
			  AND p.archived_at IS NULL
		`, body.PaymentReceiptIDs).Scan(&qty, &totalHarga)
		if err != nil {
			return err
		}
		if qty != len(body.PaymentReceiptIDs) {
			return apperror.FieldValidation("payment_receipt_ids",
				"Beberapa receipt tidak ditemukan, sudah diarsipkan, atau belum berstatus finished", "required")
		}

		// 2. Generate kode otomatis kalau tidak disediakan.
		if body.Kode == "" {
			err = tx.QueryRow(r.Context(),
				"SELECT 'PR-' || to_char(CURRENT_DATE, 'YYYY') || '-' || lpad(nextval('seq_pr_archive_kode')::text, 6, '0')").
				Scan(&kode)
			if err != nil {
				return err
			}
		} else {
			kode = body.Kode
		}

		// 3. Buat pr_archive. payment_receipt_id menyimpan receipt pertama sebagai
		// referensi primer; detail lengkap ada di deskripsi gabungan.
		var description string
		if body.Description != "" {
			description = body.Description
		} else {
			description = "Arsip bulk dari " + strconv.Itoa(qty) + " payment receipt"
		}

		_, err = tx.Exec(r.Context(), `
			INSERT INTO pr_archive (kode, payment_receipt_id, tanggal_selesai, nama_request,
			                        manager_approved, approved_by, description, quantity,
			                        harga_rencana, harga_aktual)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		`,
			kode, body.PaymentReceiptIDs[0],
			pgtype.Timestamptz{Time: tanggalSelesai, Valid: true},
			body.NamaRequest, body.ManagerApproved, int64Ptr(&identity.UserID),
			description, int32(qty),
			numericPtr(&body.HargaRencana), totalHarga,
		)
		if err != nil {
			return err
		}

		// 4. Tandai semua receipt sebagai diarsipkan.
		_, err = tx.Exec(r.Context(), `
			UPDATE payment_receipt
			SET archived_at = NOW()
			WHERE id = ANY($1::bigint[]) AND archived_at IS NULL
		`, body.PaymentReceiptIDs)
		return err
	})
	if err != nil {
		httputil.Problemf(w, r, err)
		return
	}

	httputil.JSONData(w, http.StatusCreated, map[string]any{
		"kode":     kode,
		"status":   "archived",
		"received": len(body.PaymentReceiptIDs),
	})
}

func parseDate(s string) *time.Time {
	if s == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return nil
	}
	return &t
}

func int64Ptr(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}
	return pgtype.Int8{Int64: *v, Valid: true}
}

func numericPtr(v *string) pgtype.Numeric {
	if v == nil || *v == "" {
		return pgtype.Numeric{}
	}
	var n pgtype.Numeric
	if err := n.Scan(*v); err != nil {
		return pgtype.Numeric{}
	}
	return n
}

// mountPrArchive memasang route CRUD dan bulk archive untuk pr_archive.
//
// Hanya admin yang boleh membuat/mengubah/menghapus arsip. Semua user yang
// sudah login bisa melihat daftar dan detail.
func mountPrArchive(
	r chi.Router,
	authn *auth.Authenticator,
	db *database.DB,
	q *dbgen.Queries,
) {
	store := prarchivestore.New(q)
	svc := prarchive.NewService(store)
	h := &prArchiveHandler{svc: svc, db: db, q: q}

	admin := authn.RequireRole(auth.RoleAdmin)

	r.Route("/pr-archives", func(r chi.Router) {
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)
		r.Get("/{id}/pdf", h.pdf)

		r.With(admin).Post("/", h.Create)
		r.With(admin).Put("/{id}", h.Update)
		r.With(admin).Delete("/{id}", h.Delete)
		r.With(admin).Post("/bulk-archive", h.BulkArchive)
	})
}
