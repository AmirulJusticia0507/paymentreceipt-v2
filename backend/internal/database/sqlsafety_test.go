// Package database_test memuat regresi keamanan yang menyangkut seluruh
// codebase, bukan hanya package database.
//
// Aturan yang ditegakkan: legacy project ini pernah ditembus lewat SQL
// injection karena query dibentuk dengan interpolasi string. Karena itu
// setiap pemanggilan yang membawa SQL harus memakai string literal (yang
// tidak mungkin berisi input user), lalu meneruskan nilai sebagai parameter
// terpisah.
package database_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// methodSQL menamai method pgx yang menerima SQL sebagai argumen kedua.
//
// Bentuknya sama untuk semuanya: Method(ctx, sql, args...). Artinya SQL selalu
// berada di posisi argumen pertama setelah ctx, yaitu indeks 1.
var methodSQL = map[string]bool{
	"Query":           true,
	"QueryRow":        true,
	"Exec":            true,
	"QueryContext":    true,
	"QueryRowContext": true,
	"ExecContext":     true,
	"SendBatch":       true,
	"Prepare":         true,
}

// allowDynamicSQL adalah pengecualian yang disetujui dengan alasan tertulis.
//
// Tiap entri WAJIB punya alasan. Daftar ini bukan tempat menyimpan temuan yang
// belum dikerjakan - itu berarti menutupi masalah, bukan memutuskannya.
var allowDynamicSQL = map[string]string{}

// TestNoSQLStringInterpolation memeriksa bahwa tidak ada query yang dibentuk
// dari nilai dinamis.
//
// Cara kerjanya: seluruh berkas .go diparse dengan go/ast. Setiap pemanggilan
// method yang membawa SQL diperiksa argumen SQL-nya. Kalau argumen itu string
// literal, aman - literal tidak pernah berisi data user. Kalau dia berupa
// apa pun selain literal (fmt.Sprintf, penjumlahan string, variabel), berarti
// SQL-nya dibentuk di runtime dan layak diperiksa manusia.
//
// Test ini menangkap pola yang paling berbahaya. Ia tidak bisa menangkap
// kasus di mana string literal dulu diisi nilai user lewat cara lain, jadi
// ini lapisan pertahanan, bukan satu-satunya.
func TestNoSQLStringInterpolation(t *testing.T) {
	backendDir := filepath.Join("..", "..")
	fset := token.NewFileSet()

	var problems []string

	err := filepath.WalkDir(backendDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			// Vendor dan direktori hasil generate sqlc. sqlc sendiri selalu
			// mengeluarkan query terparameter, jadi tidak perlu diperiksa lagi.
			switch d.Name() {
			case "vendor", "dbgen", "node_modules", "dist":
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := parser.ParseFile(fset, path, nil, parser.AllErrors)
		if err != nil {
			return err
		}

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) < 2 {
				return true
			}

			// Hanya memanggil method dengan nama yang dikenali, bukan fungsi
			// bebas dengan nama sama.
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !methodSQL[sel.Sel.Name] {
				return true
			}

			sqlArg := call.Args[1]
			lit, isLit := sqlArg.(*ast.BasicLit)
			if isLit && lit.Kind == token.STRING {
				return true // SQL literal, aman.
			}

			rel, relErr := filepath.Rel(backendDir, path)
			if relErr != nil {
				rel = path
			}
			pos := fset.Position(call.Pos())

			if why, allowed := allowDynamicSQL[rel]; allowed {
				t.Logf("diizinkan %s:%d (%s): %s", rel, pos.Line, sel.Sel.Name, why)
				return true
			}

			problems = append(problems,
				rel+":"+itoa(pos.Line)+": SQL dinamis di "+sel.Sel.Name+
					" (argumen kedua bukan string literal)")
			return true
		})

		return nil
	})

	if err != nil {
		t.Fatalf("gagal memindai berkas: %v", err)
	}

	for _, p := range problems {
		t.Errorf("SQL injection risk: %s", p)
	}

	if len(problems) > 0 {
		t.Errorf("%d temuan. Perbaiki dengan parameter terpisah ($1, $2, ...), "+
			"bukan dengan menyusun string SQL. Kalau pola dinamis memang "+
			"dibutuhkan dan sudah aman, tambahkan ke allowDynamicSQL dengan "+
			"alasan tertulis.", len(problems))
	}
}

// TestNoSprintfFeedingSQL sengaja dihapus. Versi awalnya menandai SEMUA
// pemanggilan fmt.Sprintf di seluruh codebase, termasuk yang tidak
// berhubungan sama sekali dengan SQL. Test yang selalu gagal akan diabaikan
// orang, dan itu lebih buruk daripada tidak ada test. Pemeriksaan yang benar
// ada di TestNoSQLStringInterpolation: ia memeriksa argumen SQL pada method
// query, yang lebih presisi dan tidak menghasilkan false positive.

// itoa menghindari import strconv hanya untuk satu pemakaian di pesan test.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	negatif := n < 0
	if negatif {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if negatif {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
