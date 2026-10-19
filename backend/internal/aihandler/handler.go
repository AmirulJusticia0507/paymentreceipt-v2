// Package aihandler melayani endpoint HTTP untuk ekstraksi data struk/faktur
// dengan AI.
package aihandler

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/ai"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// Handler melayani permintaan ekstraksi struk/faktur.
type Handler struct {
	provider ai.Provider
}

// NewHandler membuat Handler.
func NewHandler(provider ai.Provider) *Handler {
	return &Handler{provider: provider}
}

// allowedMimeType daftar tipe berkas yang boleh diproses AI.
//
// Hingga PDF didukung: model multimodal (Gemini, Qwen Omni) menerima gambar
// langsung. PDF perlu konversi ke gambar terlebih dahulu, yang menjadi kerja
// untuk fase berikutnya.
var allowedMimeType = map[string]bool{
	"image/jpeg": true,
	"image/jpg":  true,
	"image/png":  true,
	"image/webp": true,
}

// ExtractReceipt menangani POST /api/v1/ai/extract-receipt.
//
// Body adalah multipart/form-data dengan satu field berkas bernama "file".
func (h *Handler) ExtractReceipt(w http.ResponseWriter, r *http.Request) {
	if h.provider == nil {
		httputil.Problemf(w, r, apperror.Internalf("ai.provider_not_set", nil))
		return
	}

	// Batas memori 10 MiB untuk parsing multipart. File itu sendiri juga akan
	// dicek ukurannya oleh konfigurasi, tapi menceburi diri lebih awal
	// mencegah upload raksasa menjamu memori.
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		httputil.Problemf(w, r, apperror.Validationf("Berkas terlalu besar. Maksimal 10 MB."))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httputil.Problemf(w, r, apperror.FieldValidation("file", "Berkas struk/faktur wajib diunggah", "required"))
		return
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		httputil.Problemf(w, r, apperror.Internal(fmt.Errorf("baca berkas: %w", err)))
		return
	}

	if len(data) == 0 {
		httputil.Problemf(w, r, apperror.FieldValidation("file", "Berkas kosong", "required"))
		return
	}

	// Tentukan MIME type: prioritas header dari klien, kemudian deteksi dari
	// isi berkas, dan terakhir dari ekstensi nama file.
	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = http.DetectContentType(data)
	}
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = mime.TypeByExtension(filepath.Ext(header.Filename))
	}

	if !allowedMimeType[mimeType] {
		httputil.Problemf(w, r, apperror.FieldValidation("file",
			"Tipe berkas tidak didukung. Gunakan JPG, PNG, atau WEBP.", "unsupported_type"))
		return
	}

	// Ekstraksi melalui AI.
	ctx := r.Context()
	extracted, err := h.provider.ExtractReceipt(ctx, header.Filename, mimeType, data)
	if err != nil {
		if errors.Is(err, ai.ErrProviderNotConfigured) {
			httputil.Problemf(w, r, apperror.Unprocessable("Fitur AI belum dikonfigurasi. Atur AI_API_KEY di server."))
			return
		}
		httputil.Problemf(w, r, apperror.Internal(fmt.Errorf("ekstraksi AI: %w", err)))
		return
	}

	httputil.JSONData(w, http.StatusOK, map[string]any{
		"data": extracted,
		"meta": map[string]any{
			"fields_missing": extracted.Validate(),
		},
	})
}
