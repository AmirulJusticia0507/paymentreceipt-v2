// Package masterdata berisi logika CRUD yang dipakai bersama oleh tabel-tabel
// master: department, satuan, dan categories.
//
// Ketiganya punya bentuk yang hampir sama (id, kode, nama, keterangan, is_active)
// dan aturan yang sama: soft delete, kode unik, validasi pola kode. Menulis
// tiga salinan logika yang sama berarti tiga tempat untuk memperbaiki bug yang
// sama, jadi logikanya ditulis sekali di sini.
//
// Yang TIDAK digenerik: handler HTTP dan query SQL. Itu tetap per tabel, karena
// nama fungsi sqlc berbeda-beda (CreateDepartment vs CreateSatuan) dan tidak
// ada cara yang bersih untuk membungkusnya tanpa kehilangan type safety.
package masterdata

import (
	"context"
	"regexp"
	"strings"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

// kodePattern harus sama dengan CHECK constraint di database.
//
// Pola yang sama ditulis dua kali dengan sengaja: yang di sini memberi pesan
// error yang berguna, yang di database menjadi jaring pengaman terakhir kalau
// ada jalur tulis yang somehow melewati service layer. Kalau keduanya tidak
// sinkron, constraint database yang mulai menolak lebih dulu - dan itu terlihat
// sebagai error 500 yang membingungkan.
var kodePattern = regexp.MustCompile(`^[A-Z0-9-]{2,30}$`)

// namaMaxLength sesuai dengan kolom VARCHAR(100) di database.
const namaMaxLength = 100

// keteranganMaxLength sesuai dengan kolom VARCHAR(255).
const keteranganMaxLength = 255

// ResourceDescribe mendeskripsikan sebuah tabel master untuk pesan error.
//
// Dipakai sebagai nilai parameter supaya pesan bisa menyebut nama yang Dimakai
// user ("Department tidak ditemukan"), bukan nama internal tabel.
type ResourceDescribe struct {
	// Singular dipakai di pesan tunggal: "Department tidak ditemukan".
	Singular string
	// Plural dipakai di pesan jamak: "Department tidak boleh kosong".
	Plural string
}

// ListInput adalah parameter list yang sudah tervalidasi.
type ListInput struct {
	Query  string
	Limit  int32
	Offset int32

	// DepartmentID, saat non-nol, membatasi hasil ke data milik departemen
	// tersebut. Dipakai tabel requests.
	DepartmentID int64

	// BaseID, saat non-nol, membatasi hasil ke request base milik base tersebut.
	// Dipakai tabel request_base.
	BaseID int64
}

// Normalize membersihkan input list: query kosong menjadi NULL, dan whitespace
// dibuang dari query pencarian.
func (in *ListInput) Normalize() {
	in.Query = strings.TrimSpace(in.Query)
	// Batas panjang pencarian diambil dari sisi server juga, bukan hanya dari
	// maxlength di input HTML - API bisa dipanggil langsung.
	if len(in.Query) > 100 {
		in.Query = in.Query[:100]
	}
}

// WriteInput adalah input create/update yang sudah tervalidasi.
type WriteInput struct {
	Kode         string
	Nama         string
	Keterangan   string
	Jabatan      string
	DepartmentID int64
	BaseID       int64
}

// Normalize dan memvalidasi input tulis.
//
// Validasi di sini TIDAK menggantikan CHECK constraint database. Constrained
// PostgreSQL yang menjaga integritas data; validasi di sini yang menghasilkan
// pesan error yang bisa dibaca orang.
func (in *WriteInput) Normalize(res ResourceDescribe) error {
	in.Kode = strings.ToUpper(strings.TrimSpace(in.Kode))
	in.Nama = strings.TrimSpace(in.Nama)
	in.Keterangan = strings.TrimSpace(in.Keterangan)

	if in.Kode == "" {
		return apperror.FieldValidation("kode", res.Plural+" wajib diisi", "required")
	}
	if !kodePattern.MatchString(in.Kode) {
		return apperror.FieldValidation("kode",
			"Kode harus 2-30 karakter huruf besar, angka, atau tanda hubung", "pattern")
	}
	if in.Nama == "" {
		return apperror.FieldValidation("nama", "Nama wajib diisi", "required")
	}
	if len([]rune(in.Nama)) > namaMaxLength {
		return apperror.FieldValidation("nama",
			"Nama maksimal 100 karakter", "maxlen")
	}
	if len([]rune(in.Keterangan)) > keteranganMaxLength {
		return apperror.FieldValidation("keterangan",
			"Keterangan maksimal 255 karakter", "maxlen")
	}

	return nil
}

// ErrorTranslate menerjemahkan error constraint database menjadi error domain
// yang berguna.
//
// Tanpa ini, attempt membuat kode duplikat akan menghasilkan 500 dengan pesan
// "duplicate key value violates unique constraint" - teknis dan tidak berguna
// bagi user.
func ErrorTranslate(err error, res ResourceDescribe) error {
	switch {
	case err == nil:
		return nil

	case apperror.IsNotFound(err):
		return apperror.NotFound(res.Singular)

	case IsNotFoundError(err):
		// Query :one mengembalikan pgx.ErrNoRows kalau baris tidak ada.
		//
		// Kasus ini nyata, bukan teoretis: tanpa filter is_active, ID yang
		// sudah di-soft-delete tetap terbaca, jadi ErrNoRows nyaris tak pernah
		// muncul dan cabang ini terlihat tak perlu. Begitu GetByID diperbaiki
		// ikut memfilter is_active, ErrNoRows mulai muncul - dan karena tidak
		// dipetakan, jatuh ke default dan membalas 500. ID yang memang tidak
		// ada sudah salah jadi 500 sejak awal, cuma belum pernah diuji.
		return apperror.NotFound(res.Singular)

	case apperror.IsUniqueViolation(err):
		return apperror.Conflictf(
			"%s dengan kode tersebut sudah ada. Gunakan kode lain.", res.Plural)

	case apperror.IsCheckViolation(err):
		// CHECK constraint gagal padahal validasi service sudah lolos - berarti
		// keduanya tidak sinkron. Ini bug, bukan kesalahan user.
		return apperror.Internalf("masterdata.check_violation", err)

	case apperror.IsForeignKeyViolation(err):
		return apperror.Conflictf(
			"%s masih digunakan oleh data lain dan tidak bisa diubah.", res.Plural)

	default:
		return apperror.Internalf("masterdata.unexpected", err)
	}
}

// InUseMessage membuat pesan yang menjelaskan kenapa sebuah data tidak bisa
// dihapus.
//
// Pesan "tidak bisa dihapus karena masih dipakai" jauh lebih membantu daripada
// "violates foreign key constraint" - yang pertama memberi tahu user apa yang
// harus dilakukan, yang kedua hanya membuat mereka menebak.
func InUseMessage(res ResourceDescribe, noun string, count int64) string {
	if count == 0 {
		return res.Singular + " tidak sedang dipakai"
	}
	return res.Plural + " tidak bisa dihapus karena masih dipakai oleh " +
		pluralID(noun, count)
}

// pluralID membentuk kalimat "3 payment receipt" atau "1 request".
// Catatan: bentuk jamak Bahasa Indonesia tidak selaluadding "s" - "request" dan
// "archieved" tetap singular. Karena itu kata bantu bilangan dipakai sebagai
// penanda ("1 request", "3 payment receipt"), bukan perubahan bentuk kata.
func pluralID(noun string, count int64) string {
	if count == 1 {
		return "1 " + noun
	}
	return itoa(count) + " " + noun
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// Store adalah operasi yang harus disediakan tiap tabel master.
//
// Sengaja berupa interface, bukan generic constraint pada struct sqlc: tipe yang
// dikembalikan setiap query berbeda (Department, Satuan, Category) dan memaksa
// kita lewat interface kosong akan kehilangan type safety.
type Store[T any] interface {
	// List mengembalikan satu halaman data dan total seluruh data yang cocok
	// dengan filter.
	List(ctx context.Context, in ListInput) ([]T, int64, error)
	// GetByID mengambil satu data. Mengembalikan apperror 404 kalau tidak ada.
	GetByID(ctx context.Context, id int64) (T, error)
	// Create membuat data baru.
	Create(ctx context.Context, in WriteInput) (T, error)
	// Update mengubah data yang ada. Mengembalikan 404 kalau tidak ada.
	Update(ctx context.Context, id int64, in WriteInput) (T, error)
	// SoftDelete menonaktifkan data tanpa menghapusnya.
	SoftDelete(ctx context.Context, id int64) (T, error)
	// CountUsage menghitung berapa kali data ini masih direferensikan.
	CountUsage(ctx context.Context, id int64) (int64, error)
}

// Service mengorkestrasi aturan bisnis di atas sebuah Store.
type Service[T any] struct {
	store    Store[T]
	resource ResourceDescribe
	// noun untuk pesan "masih dipakai oleh N <noun>".
	noun string
}

// NewService membuat Service.
func NewService[T any](store Store[T], resource ResourceDescribe, noun string) *Service[T] {
	return &Service[T]{store: store, resource: resource, noun: noun}
}

// List mengembalikan satu halaman data.
func (s *Service[T]) List(ctx context.Context, in ListInput) ([]T, int64, error) {
	in.Normalize()

	items, total, err := s.store.List(ctx, in)
	if err != nil {
		return nil, 0, apperror.Internalf("masterdata.list_failed", err)
	}
	return items, total, nil
}

// GetByID mengembalikan satu data.
func (s *Service[T]) GetByID(ctx context.Context, id int64) (T, error) {
	var zero T

	if id <= 0 {
		return zero, apperror.FieldValidation("id", "ID tidak valid", "invalid")
	}

	item, err := s.store.GetByID(ctx, id)
	if err != nil {
		return zero, ErrorTranslate(err, s.resource)
	}
	return item, nil
}

// Create membuat data baru.
func (s *Service[T]) Create(ctx context.Context, in WriteInput) (T, error) {
	var zero T

	if err := in.Normalize(s.resource); err != nil {
		return zero, err
	}

	item, err := s.store.Create(ctx, in)
	if err != nil {
		return zero, ErrorTranslate(err, s.resource)
	}
	return item, nil
}

// Update mengubah data yang ada.
func (s *Service[T]) Update(ctx context.Context, id int64, in WriteInput) (T, error) {
	var zero T

	if err := in.Normalize(s.resource); err != nil {
		return zero, err
	}

	item, err := s.store.Update(ctx, id, in)
	if err != nil {
		return zero, ErrorTranslate(err, s.resource)
	}
	return item, nil
}

// Delete menonaktifkan data.
//
// Soft delete, bukan DELETE keras. Data master yang sudah dipakai transaksi tidak
// boleh hilang: PDF dan laporan lama masih merujuknya.
func (s *Service[T]) Delete(ctx context.Context, id int64) (T, error) {
	var zero T

	if id <= 0 {
		return zero, apperror.FieldValidation("id", "ID tidak valid", "invalid")
	}

	// Dicek lebih dulu supaya pesan errornya spesifik. Tanpa ini user hanya
	// melihat "duplicate key value violates foreign key constraint".
	usage, err := s.store.CountUsage(ctx, id)
	if err != nil {
		return zero, apperror.Internalf("masterdata.count_usage_failed", err)
	}
	if usage > 0 {
		return zero, apperror.Conflict(InUseMessage(s.resource, s.noun, usage))
	}

	item, err := s.store.SoftDelete(ctx, id)
	if err != nil {
		return zero, ErrorTranslate(err, s.resource)
	}
	return item, nil
}
