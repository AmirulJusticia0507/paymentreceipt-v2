package router

import (
	"net/http"
	"strings"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/httputil"
)

// searchLimit adalah batas atas hasil autocomplete.
//
// Tanpa batas, pengguna yang mengetik satu huruf akan menerima seluruh daftar
// base. Pada tabel kecil ini tidak besar masalah, tapi batas tetap dipasang agar
// perilaku tidak berubah diam-diam kalau datanya tumbuh.
const searchLimit = 10

// baseSuggestion adalah satu hasil autocomplete.
//
// Bentuknya sengaja terpisah dari baseDTO: frontend hanya perlu dua field ini
// untuk mengisi kontrol, dan mengirim keterangan yang tidak terpakai hanya
// menambah ukuran response.
type baseSuggestion struct {
	Value int64  `json:"value"`
	Label string `json:"label"`
	// Kode ditampilkan sebagai teks kecil di bawah label supaya user bisa
	// membedakan dua base yang namanya mirip.
	Kode string `json:"kode"`
}

// SearchBases menangani GET /bases/search.
//
// Menggantikan searchbase.php legacy yang menerima ?term. Perbedaannya: nama
// parameternya sekarang q (konsisten dengan endpoint list lain), batas hasil
// ditegakkan di server, dan query kosong mengembalikan array kosong - bukan
// seluruh tabel.
//
// Route ini harus dipasang SEBELUM /bases/{id}; kalau tidak, pola {id} akan
// menelan "/search" dan nilainya tidak akan sampai ke sini.
func SearchBases(queries *dbgen.Queries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := strings.TrimSpace(r.URL.Query().Get("q"))

		if q == "" {
			// Respons kosong, bukan seluruh daftar: pengguna baru mulai mengetik
			// dan belum bermaksud melihat semua base.
			httputil.JSON(w, http.StatusOK, map[string]any{"data": []baseSuggestion{}})
			return
		}

		rows, err := queries.SearchBases(r.Context(), dbgen.SearchBasesParams{
			Q:     q,
			Limit: searchLimit,
		})
		if err != nil {
			httputil.Problemf(w, r, err)
			return
		}

		suggestions := make([]baseSuggestion, 0, len(rows))
		for _, row := range rows {
			suggestions = append(suggestions, baseSuggestion{
				Value: row.ID,
				Label: row.Nama,
				Kode:  row.Kode,
			})
		}

		httputil.JSON(w, http.StatusOK, map[string]any{"data": suggestions})
	}
}
