// Package pdf merender dokumen resmi server-side.
package pdf

import (
	"time"

	"github.com/jung-kurt/gofpdf"
)

// ReceiptData adalah data yang dibutuhkan to render dokumen Purchase Requisition
// dari satu payment receipt.
type ReceiptData struct {
	// Pengajuan
	Kode             string
	DepartmentKode   string
	DepartmentNama   string
	RequestKode      string
	RequestNama      string
	NamaBarang       string // deskripsi item
	DeskripsiItems   string
	KategoriNama     string
	SatuanNama       string
	JumlahItems      int32
	HargaItems       string
	TotalHarga       string
	Usage            string
	Keterangan       string
	TanggalPembelian time.Time

	// Perusahaan / pihak
	ToDepartemenGA string // "General Affair Department"
}

// RenderRequisition menghasilkan PDF Purchase Requisition ke writer fpdf.
// Layout: header perusahaan, judul, info pemohon, tabel item tunggal,
// total, blok tanda tangan.
func RenderRequisition(pdf *gofpdf.Fpdf, data ReceiptData) {
	// Header perusahaan
	drawHeader(pdf)
	pdf.Ln(2)

	centerText(pdf, pdf.GetY(), "PURCHASE REQUISITION", 14, true)
	pdf.Ln(6)

	// Blok info pemohon (duas kolom)
	infoY := pdf.GetY()
	infoLeft(pdf, "No", data.Kode, infoY, 0)
	infoLeft(pdf, "Department", data.DepartmentKode+" - "+data.DepartmentNama, infoY+6, 0)
	infoLeft(pdf, "Request", data.RequestKode+" - "+data.RequestNama, infoY+12, 0)
	infoLeft(pdf, "To", "General Affair Dept.", infoY+18, 0)
	infoLeft(pdf, "Date", formatDateID(data.TanggalPembelian), infoY+24, 0)
	infoLeft(pdf, "Received by", "", infoY+30, 0)
	pdf.SetY(infoY + 38)

	// Tabel item
	drawItemTable(pdf, data)

	// Blok tanda tangan
	pdf.Ln(12)
	drawSignatures(pdf)
}

func drawHeader(pdf *gofpdf.Fpdf) {
	centerText(pdf, marginTop, "PT. ASI PUDJIASTUTI AVIATION", 14, true)
	centerText(pdf, marginTop+6, "Head Office: Jl Merdeka 312 Pangandaran, West Java Indonesia 46396", 8, false)
	centerText(pdf, marginTop+11, "General Affairs Department", 10, false)
	// garis bawah
	pdf.SetDrawColor(60, 60, 60)
	pdf.Line(marginLeft, marginTop+17, pageWidth-marginRight, marginTop+17)
	pdf.SetDrawColor(0, 0, 0)
}

func infoLeft(pdf *gofpdf.Fpdf, label, value string, y, _ float64) {
	pdf.SetFont("Helvetica", "B", 10)
	pdf.SetXY(marginLeft, y)
	pdf.Write(5, label+": ")
	pdf.SetFont("Helvetica", "", 10)
	pdf.Write(5, value)
}

func drawItemTable(pdf *gofpdf.Fpdf, data ReceiptData) {
	cols := []float64{10, 80, 14, 24, 24, 26} // No, Description, Qty, Satuan, Harga, Total
	headers := []string{"No", "Description", "Qty", "Satuan", "Harga", "Total"}
	y := pdf.GetY()
	// header
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(230, 230, 230)
	x := marginLeft
	for i, h := range headers {
		pdf.SetXY(x, y)
		pdf.CellFormat(cols[i], 8, h, "1", 0, "C", true, 0, "")
		x += cols[i]
	}
	pdf.SetY(y + 8)

	// item row
	pdf.SetFont("Helvetica", "", 9)
	desc := data.NamaBarang
	if desc == "" {
		desc = data.DeskripsiItems
	}
	if len(desc) > 50 {
		desc = desc[:50]
	}
	total := data.TotalHarga
	if total == "" {
		total = "0"
	}
	values := []string{"1", desc, formatNumber(data.JumlahItems), data.SatuanNama, formatRupiah(data.HargaItems), formatRupiah(total)}
	x = marginLeft
	y = pdf.GetY()
	for i, v := range values {
		pdf.SetXY(x, y)
		align := "L"
		if i >= 2 {
			align = "C"
		}
		if i >= 4 {
			align = "R"
		}
		pdf.CellFormat(cols[i], 8, v, "1", 0, align, false, 0, "")
		x += cols[i]
	}
	pdf.SetY(y + 8)

	// total baris
	y = pdf.GetY()
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(marginLeft, y)
	pdf.CellFormat(cols[0]+cols[1]+cols[2]+cols[3]+cols[4], 8, "TOTAL", "1", 0, "R", false, 0, "")
	pdf.SetXY(marginLeft+cols[0]+cols[1]+cols[2]+cols[3]+cols[4], y)
	pdf.CellFormat(cols[5], 8, formatRupiah(total), "1", 0, "R", false, 0, "")
	pdf.SetY(y + 8)
}

func drawSignatures(pdf *gofpdf.Fpdf) {
	sigY := pdf.GetY()
	pdf.SetFont("Helvetica", "", 9)
	// tiga kolom blok tanda tangan: User, GA, Direktur Operasional
	blockW := (pageWidth - marginLeft - marginRight) / 3
	for i, label := range []string{"User", "GA", "Direktur Operasional"} {
		x := marginLeft + float64(i)*blockW
		pdf.SetXY(x, sigY)
		pdf.MultiCell(blockW, 4, label, "", "C", false)
		pdf.SetXY(x, sigY+14)
		pdf.Line(x+5, sigY+14, x+blockW-5, sigY+14)
	}
}
