package pdf

import (
	"bytes"
	"time"
)

// ArchiveData adalah input untuk dokumen PDF arsip PR.
type ArchiveData struct {
	Kode             string
	PaymentReceiptID int64
	ReceiptKode      string
	TanggalSelesai   time.Time
	NamaRequest      string
	ManagerApproved  string
	Description      string
	Quantity         int32
	HargaRencana     string
	HargaAktual      string
}

// RenderArchive menghasilkan PDF satu arsip PR.
func RenderArchive(data ArchiveData) ([]byte, error) {
	pdf := buildPDF()
	drawHeader(pdf)
	pdf.Ln(2)
	centerText(pdf, pdf.GetY(), "PURCHASE REQUISITION - ARCHIVE", 14, true)
	pdf.Ln(6)

	infoY := pdf.GetY()
	infoLeft(pdf, "No", data.Kode, infoY, 0)
	infoLeft(pdf, "Receipt Asal", data.ReceiptKode, infoY+6, 0)
	infoLeft(pdf, "Nama Request", data.NamaRequest, infoY+12, 0)
	infoLeft(pdf, "Manager Approved", data.ManagerApproved, infoY+18, 0)
	infoLeft(pdf, "Tanggal Selesai", formatDateID(data.TanggalSelesai), infoY+24, 0)
	pdf.SetY(infoY + 32)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(pageWidth-marginLeft-marginRight, 6, "Description", "1", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.MultiCell(pageWidth-marginLeft-marginRight, 6, data.Description, "1", "L", false)
	pdf.Ln(2)

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(90, 6, "Quantity", "1", 0, "L", false, 0, "")
	pdf.CellFormat(pageWidth-marginLeft-marginRight-90, 6, formatNumber(data.Quantity), "1", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(90, 6, "Harga Rencana", "1", 0, "L", false, 0, "")
	pdf.CellFormat(pageWidth-marginLeft-marginRight-90, 6, formatRupiah(data.HargaRencana), "1", 1, "R", false, 0, "")

	pdf.SetFont("Helvetica", "B", 9)
	pdf.CellFormat(90, 6, "Harga Aktual", "1", 0, "L", false, 0, "")
	pdf.CellFormat(pageWidth-marginLeft-marginRight-90, 6, formatRupiah(data.HargaAktual), "1", 1, "R", false, 0, "")

	pdf.Ln(8)
	drawSignatures(pdf)

	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
