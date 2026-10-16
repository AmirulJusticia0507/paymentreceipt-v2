// Package httputil berisi helper HTTP yang dipakai lintas handler.
package httputil

import (
	"math"
	"net/http"
	"strconv"
	"strings"
)

// MaxPerPage adalah batas atas jumlah item per halaman.
//
// Tanpa batas, ?per_page=1000000 akan membuat server menarik seluruh tabel ke
// memori dan mengirimnya sekaligus. 100 cukup untuk penggunaan realistis dan
// menjaga respons tetap wajar ukurannya.
const MaxPerPage = 100

// DefaultPerPage dipakai kalau klien tidak mengirim per_page.
const DefaultPerPage = 20

// Pagination adalah parameter paginasi yang sudah divalidasi.
type Pagination struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
	// Total jumlah baris yang cocok dengan filter (bukan jumlah baris di halaman).
	Total int64 `json:"total"`
	// TotalPages jumlah halaman yang tersedia.
	TotalPages int `json:"total_pages"`
}

// Offset mengembalikan jumlah baris yang harus dilewati.
//
// Perhatikan: dari Page=1, PerPage=20 -> offset 0 (baris 1..20).
func (p Pagination) Offset() int { return (p.Page - 1) * p.PerPage }

// Limit mengembalikan batas baris untuk LIMIT di SQL.
func (p Pagination) Limit() int { return p.PerPage }

// HasNext melaporkan apakah ada halaman berikutnya.
func (p Pagination) HasNext() bool { return p.Page < p.TotalPages }

// HasPrev melaporkan apakah ada halaman sebelumnya.
func (p Pagination) HasPrev() bool { return p.Page > 1 }

// NewPagination menyusun Pagination lengkap dari parameter yang sudah dibatasi
// dan jumlah total baris.
func NewPagination(page, perPage int, total int64) Pagination {
	// Halaman kosong sebelum data ada: TotalPages 0 supaya frontend tidak
	// menampilkan "Halaman 1 dari 0".
	totalPages := 0
	if total > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(perPage)))
	}

	return Pagination{
		Page:       page,
		PerPage:    perPage,
		Total:      total,
		TotalPages: totalPages,
	}
}

// ParsePagination membaca dan memvalidasi parameter paginasi dari query string.
//
// Nilai di luar batas TIDAK menghasilkan error - dilewati ke default. Alasannya
// input paginasi datang dari URL yang bisa diklik atau diedit user; menolak
// halaman dengan ?page=999999 tidak menambah keamanan, hanya memunculkan pesan
// error yang membingungkan.
func ParsePagination(r *http.Request) Pagination {
	q := r.URL.Query()

	page := parseIntDefault(q.Get("page"), 1)
	if page < 1 {
		page = 1
	}

	perPage := parseIntDefault(q.Get("per_page"), DefaultPerPage)
	if perPage < 1 {
		perPage = DefaultPerPage
	}
	if perPage > MaxPerPage {
		perPage = MaxPerPage
	}

	return NewPagination(page, perPage, 0)
}

func parseIntDefault(s string, fallback int) int {
	if s == "" {
		return fallback
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return fallback
	}
	return n
}

// ParseSort membaca parameter sort dari query string, misalnya
// ?sort=-tanggal_pembelian,id
//
// Format: nama_kolom atau -nama_kolom (tanda minus = descending).
// Kolom yang diizinkan diambil dari allowlist supaya klien tidak bisa menyuruh
// server mengurutkan berdasarkan kolom sembarang.
func ParseSort(r *http.Request, defaultSort string, allowed []string) string {
	raw := r.URL.Query().Get("sort")
	if raw == "" {
		raw = defaultSort
	}

	allow := make(map[string]bool, len(allowed))
	for _, a := range allowed {
		allow[a] = true
	}

	out := make([]string, 0, 2)
	for _, part := range splitComma(raw) {
		part = strings.TrimSpace(part)
		desc := false
		col := part
		if len(col) > 0 && col[0] == '-' {
			desc = true
			col = strings.TrimSpace(col[1:])
		}
		if col == "" {
			continue
		}
		// allowlist: nama kolom di luar daftar diabaikan, bukan error.
		// Trim wajib: defaultSort kita sendiri berformat "a, b", jadi tanpa trim
		// kolom kedua terbaca sebagai " b" lalu diam-diam dibuang.
		if !allow[col] {
			continue
		}
		if desc {
			out = append(out, "-"+col)
		} else {
			out = append(out, col)
		}
	}

	if len(out) == 0 {
		return defaultSort
	}
	return joinComma(out)
}

func splitComma(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	out = append(out, s[start:])
	return out
}

func joinComma(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += ", "
		}
		out += p
	}
	return out
}
