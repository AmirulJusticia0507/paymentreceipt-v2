package pdf

import (
	"bytes"
	_ "embed"
	"time"

	"github.com/jung-kurt/gofpdf"
)

//go:embed assets/susi-air-logo.png
var susiAirLogo []byte

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

	// Penandatangan
	UserNama     string // Nama user pemohon
	GANama       string // Nama staf GA yang memeriksa
	ApproverNama string // Geoffrey Meyssonier (Direktur Operasional)
}

// RenderRequisition menghasilkan PDF Purchase Requisition ke writer fpdf.
// Layout: header resmi (Logo Susi Air + PT. ASI PUDJIASTUTI AVIATION + garis ganda pembatas),
// judul PURCHASE REQUISITION terpisah jelas di bawah garis pembatas dengan jeda 3 spasi yang rapi,
// blok info pemohon dua kolom rapi (No, Department, Request, To, Date, Received by),
// tabel item tunggal, total, dan blok tanda tangan tiga kolom (5-space break).
func RenderRequisition(pdf *gofpdf.Fpdf, data ReceiptData) {
	// 1. Header perusahaan resmi (Logo Susi Air + Kop surat + Garis pembatas ganda)
	drawHeader(pdf)

	// 2. Judul dokumen jelas di bawah garis pembatas header dengan jarak lega
	titleY := pdf.GetY() + 8.0
	centerText(pdf, titleY, "PURCHASE REQUISITION", 14, true)

	// 3. Jeda 3 space line (~16mm) setelah judul agar dokumen terlihat lega dan rapi
	infoY := titleY + 16.0
	pdf.SetY(infoY)

	// 4. Blok info pemohon (dua kolom rapi dan sejajar)
	colMid := marginLeft + 95.0

	// Kolom kiri: No, Department, Request
	infoPair(pdf, "No", data.Kode, marginLeft, infoY)
	deptText := data.DepartmentKode
	if data.DepartmentNama != "" {
		deptText += " - " + data.DepartmentNama
	}
	infoPair(pdf, "Department", deptText, marginLeft, infoY+6)
	reqText := data.RequestKode
	if data.RequestNama != "" {
		reqText += " - " + data.RequestNama
	}
	infoPair(pdf, "Request", reqText, marginLeft, infoY+12)

	// Kolom kanan: To, Date, Received by
	toDept := data.ToDepartemenGA
	if toDept == "" {
		toDept = "General Affair Dept."
	}
	infoPair(pdf, "To", toDept, colMid, infoY)
	infoPair(pdf, "Date", formatDateID(data.TanggalPembelian), colMid, infoY+6)
	receivedBy := data.GANama
	if receivedBy == "" {
		receivedBy = "................................"
	}
	infoPair(pdf, "Received by", receivedBy, colMid, infoY+12)

	pdf.SetY(infoY + 22.0)

	// 5. Tabel item
	drawItemTable(pdf, data)

	// 6. Usage / Keterangan catatan tambahan
	if data.Usage != "" || data.Keterangan != "" {
		noteY := pdf.GetY() + 3
		pdf.SetFont("Helvetica", "I", 8.5)
		pdf.SetXY(marginLeft, noteY)
		note := ""
		if data.Usage != "" {
			note += "Usage: " + data.Usage
		}
		if data.Keterangan != "" {
			if note != "" {
				note += " | "
			}
			note += "Keterangan: " + data.Keterangan
		}
		pdf.CellFormat(pageWidth-marginLeft-marginRight, 5, note, "", 1, "L", false, 0, "")
	}

	// 7. Blok tanda tangan 3 kolom dengan jeda 5-space dan nama penandatangan
	drawSignaturesWithData(pdf, data)
}

func drawHeader(pdf *gofpdf.Fpdf) {
	y := marginTop

	// Logo Susi Air di kiri atas
	if len(susiAirLogo) > 0 {
		opt := gofpdf.ImageOptions{ImageType: "PNG"}
		_ = pdf.RegisterImageOptionsReader("susi-air-logo", opt, bytes.NewReader(susiAirLogo))
		logoW := 46.0
		logoH := logoW / 5.27
		pdf.ImageOptions("susi-air-logo", marginLeft, y-1.5, logoW, logoH, false, opt, 0, "")
	}

	// Teks kop surat resmi perusahaan
	centerText(pdf, y, "PT. ASI PUDJIASTUTI AVIATION", 13, true)
	centerText(pdf, y+6, "Head Office: Jl Merdeka 312 Pangandaran, West Java Indonesia 46396", 8, false)
	centerText(pdf, y+11, "General Affairs Department", 9.5, true)

	// Garis ganda pembatas header resmi
	yLine := y + 19.0
	pdf.SetDrawColor(30, 30, 30)
	pdf.SetLineWidth(0.6)
	pdf.Line(marginLeft, yLine, pageWidth-marginRight, yLine)
	pdf.SetLineWidth(0.2)
	pdf.Line(marginLeft, yLine+1.0, pageWidth-marginRight, yLine+1.0)
	pdf.SetLineWidth(0.2)
	pdf.SetDrawColor(0, 0, 0)

	// Titik Y tepat di bawah garis pembatas kop surat
	pdf.SetY(yLine + 2.0)
}

func infoPair(pdf *gofpdf.Fpdf, label, value string, x, y float64) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(x, y)
	pdf.Write(5, label)
	pdf.SetXY(x+25, y)
	pdf.Write(5, ": ")
	pdf.SetFont("Helvetica", "", 9)
	pdf.Write(5, value)
}

func infoLeft(pdf *gofpdf.Fpdf, label, value string, y, _ float64) {
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(marginLeft, y)
	pdf.Write(5, label)
	pdf.SetXY(marginLeft+25, y)
	pdf.Write(5, ": ")
	pdf.SetFont("Helvetica", "", 9)
	pdf.Write(5, value)
}

func drawItemTable(pdf *gofpdf.Fpdf, data ReceiptData) {
	cols := []float64{10, 80, 14, 24, 24, 28} // No, Description, Qty, Satuan, Harga, Total
	headers := []string{"No", "Description", "Qty", "Satuan", "Harga", "Total"}
	y := pdf.GetY()

	// Header tabel
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetFillColor(235, 238, 242)
	x := marginLeft
	for i, h := range headers {
		pdf.SetXY(x, y)
		pdf.CellFormat(cols[i], 8, h, "1", 0, "C", true, 0, "")
		x += cols[i]
	}
	pdf.SetY(y + 8)

	// Item baris data
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
		if i == 0 || i == 2 || i == 3 {
			align = "C"
		}
		if i >= 4 {
			align = "R"
		}
		pdf.CellFormat(cols[i], 8, v, "1", 0, align, false, 0, "")
		x += cols[i]
	}
	pdf.SetY(y + 8)

	// Total baris
	y = pdf.GetY()
	pdf.SetFont("Helvetica", "B", 9)
	pdf.SetXY(marginLeft, y)
	subTotalWidth := cols[0] + cols[1] + cols[2] + cols[3] + cols[4]
	pdf.CellFormat(subTotalWidth, 8, "TOTAL", "1", 0, "R", false, 0, "")
	pdf.SetXY(marginLeft+subTotalWidth, y)
	pdf.CellFormat(cols[5], 8, formatRupiah(total), "1", 0, "R", false, 0, "")
	pdf.SetY(y + 8)
}

// drawSignatures menjaga kompatibilitas pemanggil lama (seperti arsip & test).
func drawSignatures(pdf *gofpdf.Fpdf) {
	drawSignaturesWithData(pdf, ReceiptData{ApproverNama: "Geoffrey Meyssonier"})
}

// drawSignaturesWithData merender 3 kolom tanda tangan:
//  1. Kolom User (Pemohon): Nama user & departemen
//  2. Kolom GA (Pemeriksa): Nama staf GA
//  3. Kolom Direktur Operasional: Geoffrey Meyssonier di garis tanda tangan
//
// Dilengkapi jeda ruang tanda tangan ~5 spasi (~24mm) sebelum garis tanda tangan.
func drawSignaturesWithData(pdf *gofpdf.Fpdf, data ReceiptData) {
	sigY := pdf.GetY() + 8
	if sigY < 185 {
		sigY = 185 // Menjaga proporsi seimbang pada kertas A4
	}

	blockW := (pageWidth - marginLeft - marginRight) / 3

	// Nama approver
	approverName := data.ApproverNama
	if approverName == "" {
		approverName = "Geoffrey Meyssonier"
	}

	// Nama GA (diisi nama staf GA jika sudah diproses, atau titik-titik jika belum)
	gaName := data.GANama
	if gaName == "" {
		gaName = "( .................................... )"
	} else {
		gaName = "( " + gaName + " )"
	}

	// Nama User pemohon
	userName := data.UserNama
	if userName == "" {
		userName = "( .................................... )"
	} else {
		userName = "( " + userName + " )"
	}

	userDept := data.DepartmentNama
	if userDept == "" {
		userDept = "User Department"
	}

	type colData struct {
		title string
		role  string
		name  string
		sub   string
	}

	cols := []colData{
		{
			title: "Diajukan oleh:",
			role:  "User / Pemohon",
			name:  userName,
			sub:   userDept,
		},
		{
			title: "Diperiksa oleh:",
			role:  "General Affairs",
			name:  gaName,
			sub:   "GA Department",
		},
		{
			title: "Disetujui oleh:",
			role:  "Direktur Operasional",
			name:  "( " + approverName + " )",
			sub:   "Direktur Operasional",
		},
	}

	for i, c := range cols {
		x := marginLeft + float64(i)*blockW

		// 1. Header peran (Diajukan oleh / Diperiksa oleh / Disetujui oleh)
		pdf.SetFont("Helvetica", "B", 9)
		pdf.SetXY(x, sigY)
		pdf.CellFormat(blockW, 4.5, c.title, "", 1, "C", false, 0, "")

		pdf.SetFont("Helvetica", "", 8.5)
		pdf.SetXY(x, sigY+4.5)
		pdf.CellFormat(blockW, 4, c.role, "", 1, "C", false, 0, "")

		// 2. Jeda break untuk tanda tangan (5 space line = ~24mm ruang tanda tangan)
		lineY := sigY + 28.0

		// 3. Garis tanda tangan
		lineWidth := blockW - 16.0
		lineX := x + 8.0
		pdf.SetDrawColor(40, 40, 40)
		pdf.SetLineWidth(0.3)
		pdf.Line(lineX, lineY, lineX+lineWidth, lineY)

		// 4. Nama di bawah garis tanda tangan (Geoffrey Meyssonier / Nama GA / Nama User)
		pdf.SetFont("Helvetica", "B", 8.5)
		pdf.SetXY(x, lineY+1.5)
		pdf.CellFormat(blockW, 4.5, c.name, "", 1, "C", false, 0, "")

		// 5. Jabatan / departemen di bawah nama
		pdf.SetFont("Helvetica", "", 7.5)
		pdf.SetXY(x, lineY+6.0)
		pdf.CellFormat(blockW, 4, c.sub, "", 1, "C", false, 0, "")
	}
}
