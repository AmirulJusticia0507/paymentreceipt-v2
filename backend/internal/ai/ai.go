// Package ai berisi domain dan layanan ekstraksi data dari struk/faktur
// menggunakan model AI melalui API OpenAI-compatible.
//
// Hasil ekstraksi bersifat struktural (JSON) agar bisa langsung dipakai
// untuk mengisi form payment receipt, bukan hanya teks mentah.
package ai

import (
	"context"
	"fmt"
	"strings"
)

// ExtractedReceipt adalah hasil ekstraksi struk/faktur yang sudah terstruktur.
//
// Semua field boleh kosong/null kalau tidak terlihat di dokumen. Penanggung
// jawab melengkapi/menyaring hasilnya adalah user melalui UI, bukan AI secara
// mandiri - ekstraksi AI adalah saran, bukan kebenaran mutlak.
type ExtractedReceipt struct {
	MerchantName    string          `json:"merchant_name"`
	MerchantAddress string          `json:"merchant_address"`
	MerchantPhone   string          `json:"merchant_phone"`
	InvoiceNumber   string          `json:"invoice_number"`
	InvoiceDate     string          `json:"invoice_date"`  // YYYY-MM-DD
	PurchaseDate    string          `json:"purchase_date"` // YYYY-MM-DD
	PaymentMethod   string          `json:"payment_method"`
	Currency        string          `json:"currency"` // IDR, USD, dst.
	Subtotal        float64         `json:"subtotal"`
	Tax             float64         `json:"tax"`
	Discount        float64         `json:"discount"`
	Total           float64         `json:"total"`
	Items           []ExtractedItem `json:"items"`
	RawText         string          `json:"raw_text"`
	Confidence      float64         `json:"confidence"` // 0.0 - 1.0
}

// ExtractedItem adalah satu barang pada struk/faktur.
type ExtractedItem struct {
	Name       string  `json:"name"`
	Quantity   float64 `json:"quantity"`
	UnitPrice  float64 `json:"unit_price"`
	TotalPrice float64 `json:"total_price"`
	Unit       string  `json:"unit"` // pcs, box, rim, botol, dst.
}

// Provider adalah antarmuka untuk memanggil model AI.
//
// Diekspose sebagai interface supaya mudah diuji dengan stub/mock tanpa
// harus melakukan jaringan nyata.
type Provider interface {
	// ExtractReceipt mengirim gambar struk/faktur dan mengembalikan hasil
	// ekstraksi yang sudah di-parse menjadi ExtractedReceipt.
	ExtractReceipt(ctx context.Context, fileName string, mimeType string, data []byte) (*ExtractedReceipt, error)
}

// ErrProviderNotConfigured dikembalikan kalau API key belum diisi.
var ErrProviderNotConfigured = fmt.Errorf("AI API key belum diisi (AI_API_KEY)")

// parseJSONFromText mengeluarkan objek JSON pertama yang ditemukan dari teks.
//
// Beberapa model AI menambahkan penjelasan sebelum atau sesudah blok JSON,
// jadi kita cari kurung kurawal paling luar dan coba parse isinya.
func parseJSONFromText(text string) (string, error) {
	start := strings.Index(text, "{")
	end := strings.LastIndex(text, "}")
	if start < 0 || end < 0 || end <= start {
		return "", fmt.Errorf("tidak ada objek JSON dalam respons AI")
	}
	return text[start : end+1], nil
}

// cleanNumberMembers memastikan field numerik yang null diubah jadi 0,
// sehingga UnmarshalJSON tidak gagal saat AI mengirim null untuk angka.
func normalizeReceipt(in map[string]any) *ExtractedReceipt {
	r := &ExtractedReceipt{}

	if v, ok := in["merchant_name"].(string); ok {
		r.MerchantName = v
	}
	if v, ok := in["merchant_address"].(string); ok {
		r.MerchantAddress = v
	}
	if v, ok := in["merchant_phone"].(string); ok {
		r.MerchantPhone = v
	}
	if v, ok := in["invoice_number"].(string); ok {
		r.InvoiceNumber = v
	}
	if v, ok := in["invoice_date"].(string); ok {
		r.InvoiceDate = v
	}
	if v, ok := in["purchase_date"].(string); ok {
		r.PurchaseDate = v
	}
	if v, ok := in["payment_method"].(string); ok {
		r.PaymentMethod = v
	}
	if v, ok := in["currency"].(string); ok {
		r.Currency = v
	}
	r.Subtotal = toFloat(in["subtotal"])
	r.Tax = toFloat(in["tax"])
	r.Discount = toFloat(in["discount"])
	r.Total = toFloat(in["total"])
	r.RawText = toString(in["raw_text"])
	r.Confidence = toFloat(in["confidence"])

	if items, ok := in["items"].([]any); ok {
		r.Items = make([]ExtractedItem, 0, len(items))
		for _, raw := range items {
			if m, ok := raw.(map[string]any); ok {
				r.Items = append(r.Items, ExtractedItem{
					Name:       toString(m["name"]),
					Quantity:   toFloat(m["quantity"]),
					UnitPrice:  toFloat(m["unit_price"]),
					TotalPrice: toFloat(m["total_price"]),
					Unit:       toString(m["unit"]),
				})
			}
		}
	}

	return r
}

func toFloat(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case string:
		// Angka mungkin datang sebagai string dari AI, coba parse.
		if n == "" {
			return 0
		}
		var f float64
		fmt.Sscanf(n, "%f", &f)
		return f
	case nil:
		return 0
	}
	return 0
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

// Validate melakukan validasi dasar pada hasil ekstraksi.
//
// Tidak memblokir: hanya menyoroti field yang terpenting dan belum terisi.
func (r *ExtractedReceipt) Validate() []string {
	var missing []string
	if strings.TrimSpace(r.MerchantName) == "" {
		missing = append(missing, "nama toko/merchant")
	}
	if r.Total <= 0 {
		missing = append(missing, "total harga")
	}
	if len(r.Items) == 0 {
		missing = append(missing, "daftar barang/item")
	}
	return missing
}
