package ai

import (
	"encoding/json"
	"testing"
)

// TestNormalizeReceipt memastikan map hasil parse JSON AI diubah dengan benar
// menjadi ExtractedReceipt, termasuk angka yang datang sebagai string dan
// null yang diubah jadi 0.
func TestNormalizeReceipt(t *testing.T) {
	raw := map[string]any{
		"merchant_name":    "Toko Makmur Jaya",
		"merchant_address": "Jl. Sudirman No. 10",
		"merchant_phone":   "08123456789",
		"invoice_number":   "INV-2026-001",
		"invoice_date":     "2026-01-15",
		"purchase_date":    "2026-01-15",
		"payment_method":   "tunai",
		"currency":         "IDR",
		"subtotal":         "100000", // string -> float
		"tax":              10000,    // float64
		"discount":         nil,      // null -> 0
		"total":            110000,   // float64
		"raw_text":         "Toko Makmur...",
		"confidence":       0.92, // float64
		"items": []any{
			map[string]any{
				"name":        "Pulpen",
				"quantity":    5,
				"unit_price":  2000,
				"total_price": 10000,
				"unit":        "pcs",
			},
			map[string]any{
				"name":        "Kertas HVS",
				"quantity":    "2",     // string -> float
				"unit_price":  "50000", // string -> float
				"total_price": 100000,
				"unit":        "rim",
			},
		},
	}

	r := normalizeReceipt(raw)

	if r.MerchantName != "Toko Makmur Jaya" {
		t.Errorf("MerchantName = %q, want %q", r.MerchantName, "Toko Makmur Jaya")
	}
	if r.MerchantAddress != "Jl. Sudirman No. 10" {
		t.Errorf("MerchantAddress = %q, want %q", r.MerchantAddress, "Jl. Sudirman No. 10")
	}
	if r.InvoiceNumber != "INV-2026-001" {
		t.Errorf("InvoiceNumber = %q, want %q", r.InvoiceNumber, "INV-2026-001")
	}
	if r.Subtotal != 100000 {
		t.Errorf("Subtotal = %v, want 100000", r.Subtotal)
	}
	if r.Tax != 10000 {
		t.Errorf("Tax = %v, want 10000", r.Tax)
	}
	if r.Discount != 0 {
		t.Errorf("Discount = %v, want 0 (null diubah ke 0)", r.Discount)
	}
	if r.Total != 110000 {
		t.Errorf("Total = %v, want 110000", r.Total)
	}
	if r.Confidence != 0.92 {
		t.Errorf("Confidence = %v, want 0.92", r.Confidence)
	}
	if len(r.Items) != 2 {
		t.Fatalf("len(Items) = %d, want 2", len(r.Items))
	}
	if r.Items[0].Name != "Pulpen" || r.Items[0].Quantity != 5 || r.Items[0].UnitPrice != 2000 {
		t.Errorf("Items[0] = %+v, salah", r.Items[0])
	}
	if r.Items[1].Quantity != 2 || r.Items[1].UnitPrice != 50000 {
		t.Errorf("Items[1] = %+v, quantity/unit_price harus di-parse dari string", r.Items[1])
	}
}

// TestNormalizeReceipt_Empty memastikan map kosong tidak panic dan hasilnya nol.
func TestNormalizeReceipt_Empty(t *testing.T) {
	r := normalizeReceipt(map[string]any{})
	if r.MerchantName != "" || r.Total != 0 || len(r.Items) != 0 {
		t.Errorf("hasil kosong tidak nol: %+v", r)
	}
}

// TestParseJSONFromText memastikan objek JSON diekstrak dari teks yang
// berisi awalan/akapan penjelasan (perilaku umum model AI).
func TestParseJSONFromText(t *testing.T) {
	tests := []struct {
		name    string
		text    string
		want    string
		wantErr bool
	}{
		{
			name: "JSON murni",
			text: `{"merchant_name":"Toko A","total":1000}`,
			want: `{"merchant_name":"Toko A","total":1000}`,
		},
		{
			name: "JSON dalam teks penjelasan",
			text: `Berdasarkan gambar ini, berikut hasil ekstraksi:

{"merchant_name":"Toko B","total":2000}

Semoga membantu.`,
			want: `{"merchant_name":"Toko B","total":2000}`,
		},
		{
			name:    "Tanpa JSON",
			text:    "Hanya teks biasa tanpa kurung kurawal.",
			wantErr: true,
		},
		{
			name:    "Kurung kurawal tidak terurut",
			text:    "}",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseJSONFromText(tt.text)
			if tt.wantErr {
				if err == nil {
					t.Error("parseJSONFromText error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseJSONFromText error = %v, want nil", err)
			}
			if got != tt.want {
				t.Errorf("parseJSONFromText = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestExtractedReceipt_Validate memastikan daftar field yang hilang terdeteksi.
func TestExtractedReceipt_Validate(t *testing.T) {
	tests := []struct {
		name string
		r    ExtractedReceipt
		want []string
	}{
		{
			name: "lengkap",
			r: ExtractedReceipt{
				MerchantName: "Toko A",
				Total:        50000,
				Items:        []ExtractedItem{{Name: "Pulpen"}},
			},
			want: nil,
		},
		{
			name: "semua hilang",
			r:    ExtractedReceipt{},
			want: []string{"nama toko/merchant", "total harga", "daftar barang/item"},
		},
		{
			name: "total nol",
			r: ExtractedReceipt{
				MerchantName: "Toko A",
				Items:        []ExtractedItem{{Name: "Pulpen"}},
			},
			want: []string{"total harga"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.r.Validate()
			if len(got) != len(tt.want) {
				t.Errorf("Validate() = %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("Validate() = %v, want %v", got, tt.want)
					return
				}
			}
		})
	}
}

// TestExtractedReceipt_JSONRoundTrip memastikan struct bisa di-marshal dan
// di-unmarshal kembali tanpa kehilangan data.
func TestExtractedReceipt_JSONRoundTrip(t *testing.T) {
	r := ExtractedReceipt{
		MerchantName: "Toko ABC",
		Total:        75000,
		Items: []ExtractedItem{
			{Name: "Buku", Quantity: 2, UnitPrice: 15000, TotalPrice: 30000, Unit: "pcs"},
		},
	}

	data, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("json.Marshal error = %v, want nil", err)
	}

	var r2 ExtractedReceipt
	if err := json.Unmarshal(data, &r2); err != nil {
		t.Fatalf("json.Unmarshal error = %v, want nil", err)
	}

	if r2.MerchantName != r.MerchantName || r2.Total != r.Total {
		t.Errorf("round-trip gagal: %+v vs %+v", r2, r)
	}
	if len(r2.Items) != 1 || r2.Items[0].Name != "Buku" {
		t.Errorf("items round-trip gagal: %+v", r2.Items)
	}
}
