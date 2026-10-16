package pdf

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func fixtureReceipt() ReceiptData {
	return ReceiptData{
		Kode:             "PR-2026-000001",
		DepartmentKode:   "GA",
		DepartmentNama:   "General Affairs",
		RequestKode:      "REQ-GA",
		RequestNama:      "Pengajuan ATK",
		NamaBarang:       "Kertas A4",
		KategoriNama:     "ATK",
		SatuanNama:       "rim",
		JumlahItems:      2,
		HargaItems:       "50000",
		TotalHarga:       "100000",
		Usage:            "Penggunaan kantor",
		Keterangan:       "Urgent",
		TanggalPembelian: time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC),
	}
}

func TestRenderReceipt_ContainsCompanyNameAndTotal(t *testing.T) {
	data := fixtureReceipt()
	buf, err := RenderReceipt(data)
	if err != nil {
		t.Fatalf("RenderReceipt() error = %v", err)
	}
	if len(buf) < 100 {
		t.Fatalf("PDF terlalu kecil: %d bytes", len(buf))
	}
	text := string(buf)
	for _, want := range []string{"PT. ASI PUDJIASTUTI AVIATION", "PURCHASE REQUISITION", data.Kode, "Kertas A4", "100.000"} {
		if !strings.Contains(text, want) {
			t.Errorf("PDF tidak mengandung %q", want)
		}
	}
}

func TestRenderReceipt_FiltersByIDDifferentiates(t *testing.T) {
	a := fixtureReceipt()
	b := fixtureReceipt()
	b.Kode = "PR-2026-000002"
	b.TotalHarga = "200.000"
	b.HargaItems = "100000"
	b.JumlahItems = 2

	outA, err := RenderReceipt(a)
	if err != nil {
		t.Fatal(err)
	}
	outB, err := RenderReceipt(b)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(outA, outB) {
		t.Error("PDF dari dua transaksi berbeda seharusnya tidak identik")
	}
	if !strings.Contains(string(outB), b.Kode) {
		t.Error("PDF kedua tidak mengandung kode-nya sendiri")
	}
}

// formatRupiah sanity checks.
func TestFormatRupiah(t *testing.T) {
	cases := []struct{ in, want string }{
		{"100000", "100.000"},
		{"1250000.50", "1.250.000"},
		{"0", "0"},
		{"", "0"},
	}
	for _, c := range cases {
		got := formatRupiah(c.in)
		if got != c.want {
			t.Errorf("formatRupiah(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}
