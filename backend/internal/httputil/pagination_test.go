package httputil

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParsePagination_Defaults(t *testing.T) {
	r := httptest.NewRequest("GET", "/api/v1/x", nil)
	p := ParsePagination(r)

	if p.Page != 1 {
		t.Errorf("Page = %d, want 1", p.Page)
	}
	if p.PerPage != DefaultPerPage {
		t.Errorf("PerPage = %d, want %d", p.PerPage, p.PerPage)
	}
	if p.Offset() != 0 {
		t.Errorf("Offset() = %d, want 0", p.Offset())
	}
}

func TestParsePagination_PerPageIsCapped(t *testing.T) {
	// Tanpa batas, ?per_page=1000000 akan menarik seluruh tabel ke memori.
	r := httptest.NewRequest("GET", "/api/v1/x?per_page=1000000", nil)
	p := ParsePagination(r)

	if p.PerPage != MaxPerPage {
		t.Errorf("PerPage = %d, want %d (harus di-cap)", p.PerPage, MaxPerPage)
	}
}

func TestParsePagination_RejectsNonPositive(t *testing.T) {
	tests := []struct {
		query  string
		wantPg int
		wantPP int
	}{
		{"?page=0", 1, DefaultPerPage},
		{"?page=-5", 1, DefaultPerPage},
		{"?per_page=0", 1, DefaultPerPage},
		{"?per_page=-1", 1, DefaultPerPage},
		{"?page=abc&per_page=xyz", 1, DefaultPerPage},
		{"?page=3&per_page=25", 3, 25},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/x"+tt.query, nil)
			p := ParsePagination(r)
			if p.Page != tt.wantPg {
				t.Errorf("Page = %d, want %d", p.Page, tt.wantPg)
			}
			if p.PerPage != tt.wantPP {
				t.Errorf("PerPage = %d, want %d", p.PerPage, tt.wantPP)
			}
		})
	}
}

func TestPaginationOffsetAndLimit(t *testing.T) {
	p := NewPagination(3, 20, 100)

	if p.Offset() != 40 {
		t.Errorf("Offset() = %d, want 40 (halaman 3 mulai baris 41)", p.Offset())
	}
	if p.Limit() != 20 {
		t.Errorf("Limit() = %d, want 20", p.Limit())
	}
	if p.TotalPages != 5 {
		t.Errorf("TotalPages = %d, want 5", p.TotalPages)
	}
	if !p.HasNext() {
		t.Error("HasNext() = false, want true")
	}
	if !p.HasPrev() {
		t.Error("HasPrev() = false, want true")
	}
}

func TestPaginationEmptyResult(t *testing.T) {
	// Total 0 harus menghasilkan TotalPages 0, bukan 1 - supaya frontend tidak
	// menampilkan "Halaman 1 dari 1" untuk data kosong.
	p := NewPagination(1, 20, 0)

	if p.TotalPages != 0 {
		t.Errorf("TotalPages = %d, want 0", p.TotalPages)
	}
	if p.HasNext() {
		t.Error("HasNext() = true, want false")
	}
	if p.HasPrev() {
		t.Error("HasPrev() = true, want false")
	}
}

func TestPaginationLastPageHasNoNext(t *testing.T) {
	p := NewPagination(5, 20, 100)
	if p.HasNext() {
		t.Error("HasNext() = true on last page, want false")
	}
}

func TestParseSort_AllowlistIsEnforced(t *testing.T) {
	// INI ADALAH TES KEAMANAN.
	//
	// Nama kolom SQL tidak bisa diparameterkan, jadi allowlist wajib ada.
	// Kalau kolom di luar daftar diteruskan, klien bisa menyisipkan SQL lewat
	// parameter sort.
	allowed := []string{"nama", "kode", "created_at"}

	tests := []struct {
		name  string
		query string
		want  string
	}{
		{
			name:  "kolom valid ascending",
			query: "?sort=nama",
			want:  "nama",
		},
		{
			name:  "kolom valid descending",
			query: "?sort=-created_at",
			want:  "-created_at",
		},
		{
			name:  "beberapa kolom",
			query: "?sort=-created_at,nama",
			want:  "-created_at, nama",
		},
		{
			name:  "kolom di luar allowlist diabaikan",
			query: "?sort=password_hash",
			want:  "kode, nama", // jatuh ke defaultSort
		},
		{
			name: "upaya SQL injection diabaikan",
			// Query harus di-encode: spasi mentah membuat httptest.NewRequest panic.
			// Pola serangan sebenarnya yang sampai ke server tetap terenkode.
			query: "?sort=nama%3B%20DROP%20TABLE%20users--",
			want:  "kode, nama", // semua kolom ditolak -> default
		},
		{
			name:  "upaya injection dengan spasi terenkode",
			query: "?sort=nama%3B%20DROP%20TABLE%20users--%20--",
			want:  "kode, nama",
		},
		{
			name:  "subquery digabung dengan kolom valid",
			query: "?sort=nama,(SELECT+1)",
			want:  "nama",
		},
		{
			name:  "subquery terenkode",
			query: "?sort=nama%2C(SELECT%201)",
			want:  "nama",
		},
		{
			name:  "sort kosong pakai default",
			query: "?sort=",
			want:  "kode, nama",
		},
		{
			name:  "sebagian kolom di luar allowlist, sebagian valid",
			query: "?sort=password_hash,nama",
			want:  "nama",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v1/x"+tt.query, nil)
			got := ParseSort(r, "kode, nama", allowed)
			if got != tt.want {
				t.Errorf("ParseSort() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSortClauseProducesSafeSQL(t *testing.T) {
	tests := []struct {
		sort string
		want string
	}{
		{"nama", " ORDER BY nama ASC"},
		{"-created_at", " ORDER BY created_at DESC"},
		{"-created_at, nama", " ORDER BY created_at DESC, nama ASC"},
		{"", ""},
	}

	for _, tt := range tests {
		if got := SortClause(tt.sort); got != tt.want {
			t.Errorf("SortClause(%q) = %q, want %q", tt.sort, got, tt.want)
		}
	}
}

func TestSortClauseNeverEmitsSemicolon(t *testing.T) {
	// Lapisan kedua: SortClause harus aman walau dipanggil dengan input yang
	// BELUM melewati ParseSort. Nama kolom SQL tidak bisa diparameterkan, jadi
	// pola ident di dalam SortClause yang menjadi penjaganya.
	//
	// Catatan: "nama; DROP TABLE users" gagal sebagai satu token ident karena
	// mengandung spasi dan titik koma, sehingga SELURUH input dibuang dan
	// hasilnya string kosong - bukan hanya "nama" yang lolos.
	dangerous := []string{
		"nama; DROP TABLE users",
		"nama UNION SELECT password_hash FROM users",
		"1; DELETE FROM payment_receipt",
		"nama)--",
		"nama'",
		"(SELECT 1)",
		"nama/*comment*/",
	}

	for _, d := range dangerous {
		clause := SortClause(d)
		upper := strings.ToUpper(clause)
		for _, bad := range []string{";", "DROP", "UNION", "DELETE", "SELECT", "--", "/*"} {
			if strings.Contains(upper, bad) {
				t.Errorf("SortClause(%q) = %q, mengandung %q", d, clause, bad)
			}
		}
	}
}

func TestSortClauseKeepsValidIdentifiers(t *testing.T) {
	// Nama kolom yang sah tetap harus diteruskan - SortClause bukan black hole.
	// Kolom seperti password_hash memang lolos di sini - membatasi kolom mana yang
	// boleh dipakai adalah tugas ParseSort (allowlist per-endpoint).
	valid := []string{"nama", "kode", "created_at", "total_harga", "_private", "col2"}
	for _, v := range valid {
		if clause := SortClause(v); clause == "" {
			t.Errorf("SortClause(%q) kosong, want kolom valid diteruskan", v)
		}
	}
}
