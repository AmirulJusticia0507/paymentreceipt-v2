package pdf

import (
	"bytes"
)

// ReceiptPDFData is the public alias for the input data.
type ReceiptPDFData = ReceiptData

// RenderReceipt builds the Purchase Requisition PDF for a single payment receipt
// and returns the raw bytes suitable for HTTP streaming.
func RenderReceipt(data ReceiptData) ([]byte, error) {
	pdf := buildPDF()
	RenderRequisition(pdf, data)
	var buf bytes.Buffer
	if err := pdf.Output(&buf); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
