package pdf

import (
	"fmt"
	"strings"
	"time"
)

// formatRupiah memformat angka (string decimal "150000.50") menjadi "150.000".
// Desimal dihilangkan kalau 0, supaya ringkas sesuai dokumen legacy.
func formatRupiah(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "0"
	}
	dot := strings.Index(value, ".")
	if dot >= 0 {
		value = value[:dot]
	}
	neg := ""
	if strings.HasPrefix(value, "-") {
		neg = "-"
		value = strings.TrimPrefix(value, "-")
	}

	// Pisahkan per 3 digit dari kanan.
	var parts []string
	for len(value) > 3 {
		parts = append([]string{value[len(value)-3:]}, parts...)
		value = value[:len(value)-3]
	}
	if value != "" {
		parts = append([]string{value}, parts...)
	}
	return neg + strings.Join(parts, ".")
}

// formatDateID memformat waktu ke "dd-MMM-yyyy" yang dipakai dokumen cetak.
func formatDateID(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	months := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	// time.Time.Month() 1-based; index langsung ke slices 0-based.
	return fmt.Sprintf("%02d-%s-%d", t.Day(), months[int(t.Month())-1], t.Year())
}

// formatNumber menulis angka biasa dengan pemisah ribuan (tanpa desimal).
func formatNumber(n int32) string { return formatRupiah(fmt.Sprintf("%d", n)) }
