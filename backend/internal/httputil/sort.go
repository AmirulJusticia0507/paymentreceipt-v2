package httputil

import (
	"regexp"
	"strings"
)

// identPattern mencocokkan nama kolom SQL yang sah: huruf atau underscore di awal,
// lalu huruf, angka, atau underscore.
//
// Ini adalah pagar kedua. Allowlist di ParseSort sudah menyaring, tapi SortClause
// tidak bergantung pada itu - pola ini membuat fungsi ini aman dipanggil
// tanpa ParseSort.
var identPattern = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// SortClause mengubah daftar "-col,col" menjadi potongan ORDER BY.
//
// Nama kolom SQL tidak bisa diparameterkan dengan placeholder ($1 hanya berlaku
// untuk nilai), jadi nama kolom adalah satu-satunya bagian SQL yang harus
// dirangkai sebagai teks. Karena itu identPattern wajib ada di sini: kolom yang
// tidak cocok dibuang, bukan diteruskan.
//
// Kolom yang dibuang tidak menghasilkan error karena nilai ini datang dari query
// string yang bisa saja salah ketik - membuang diam-diam lebih baik daripada
// membalas 500.
func SortClause(sort string) string {
	parts := splitComma(sort)
	out := make([]string, 0, len(parts))

	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}

		desc := strings.HasPrefix(p, "-")
		col := strings.TrimSpace(strings.TrimPrefix(p, "-"))

		if !identPattern.MatchString(col) {
			continue
		}

		if desc {
			out = append(out, col+" DESC")
		} else {
			out = append(out, col+" ASC")
		}
	}

	if len(out) == 0 {
		return ""
	}
	return " ORDER BY " + strings.Join(out, ", ")
}
