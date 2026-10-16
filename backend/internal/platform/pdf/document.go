package pdf

import "github.com/jung-kurt/gofpdf"

// pageWidth adalah lebar halaman A4 dalam mm.
const pageWidth = 210.0
const pageHeight = 297.0

// margins konsisten.
const (
	marginLeft  = 15.0
	marginRight = 15.0
	marginTop   = 15.0
	marginBot   = 15.0
)

// buildPDF membuat instance gofpdf dengan header/footer yang sama untuk semua
// dokumen Payment Receipt / PR Archive.
//
// Semua dokumen memakai A4 portrait, margin seragam, font Helvetica (core font,
// tidak butuh embed TTF + cgo).
func buildPDF() *gofpdf.Fpdf {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(marginLeft, marginTop, marginRight)
	pdf.SetAutoPageBreak(true, marginBot)
	pdf.SetCompression(false)
	pdf.AddPage()
	pdf.SetFont("Helvetica", "", 10)
	return pdf
}

// centerText menulis teks di tengah area halaman.
func centerText(pdf *gofpdf.Fpdf, y float64, text string, size float64, bold bool) {
	style := ""
	if bold {
		style = "B"
	}
	pdf.SetFont("Helvetica", style, size)
	pdf.SetXY(marginLeft, y)
	pdf.CellFormat(pageWidth-marginLeft-marginRight, 6, text, "", 0, "C", false, 0, "")
}
