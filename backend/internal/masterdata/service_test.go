package masterdata

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"
)

// stubItem adalah tipe data uji yang tidak butuh database.
type stubItem struct {
	ID    int64
	Kode  string
	Nama  string
	Usage int64
}

// stubStore adalah Store[T] palsu yang mencatat panggilan.
type stubStore struct {
	items     []stubItem
	listErr   error
	getErr    error
	createErr error
	updateErr error
	deleteErr error
	usageErr  error

	// lastCreateInput dan lastUpdateInput dipakai untuk memeriksa input yang
	// diteruskan ke store setelah normalisasi.
	lastCreateInput WriteInput
	lastUpdateInput WriteInput
	lastUpdateID    int64
}

func (s *stubStore) List(_ context.Context, in ListInput) ([]stubItem, int64, error) {
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.items, int64(len(s.items)), nil
}

func (s *stubStore) GetByID(_ context.Context, id int64) (stubItem, error) {
	if s.getErr != nil {
		return stubItem{}, s.getErr
	}
	for _, it := range s.items {
		if it.ID == id {
			return it, nil
		}
	}
	return stubItem{}, apperror.NotFound("Stub")
}

func (s *stubStore) Create(_ context.Context, in WriteInput) (stubItem, error) {
	if s.createErr != nil {
		return stubItem{}, s.createErr
	}
	s.lastCreateInput = in
	return stubItem{ID: 99, Kode: in.Kode, Nama: in.Nama}, nil
}

func (s *stubStore) Update(_ context.Context, id int64, in WriteInput) (stubItem, error) {
	if s.updateErr != nil {
		return stubItem{}, s.updateErr
	}
	s.lastUpdateInput = in
	s.lastUpdateID = id
	return stubItem{ID: id, Kode: in.Kode, Nama: in.Nama}, nil
}

func (s *stubStore) SoftDelete(_ context.Context, id int64) (stubItem, error) {
	if s.deleteErr != nil {
		return stubItem{}, s.deleteErr
	}
	return stubItem{ID: id}, nil
}

func (s *stubStore) CountUsage(_ context.Context, id int64) (int64, error) {
	if s.usageErr != nil {
		return 0, s.usageErr
	}
	for _, it := range s.items {
		if it.ID == id {
			return it.Usage, nil
		}
	}
	return 0, nil
}

func newTestService(store Store[stubItem]) *Service[stubItem] {
	return NewService(store,
		ResourceDescribe{Singular: "Department", Plural: "Department"},
		"request")
}

func TestWriteInputNormalize(t *testing.T) {
	res := ResourceDescribe{Singular: "Department", Plural: "Department"}

	tests := []struct {
		name    string
		in      WriteInput
		wantErr string
	}{
		{"valid", WriteInput{Kode: "GA", Nama: "General Affairs"}, ""},
		{"lowercase kode di-uppercase", WriteInput{Kode: "ga", Nama: "X"}, ""},
		{"spasi dipangkas", WriteInput{Kode: "  GA  ", Nama: "  X  "}, ""},
		{"kode kosong", WriteInput{Kode: "", Nama: "X"}, "Department wajib diisi"},
		{"kode hanya spasi", WriteInput{Kode: "   ", Nama: "X"}, "Department wajib diisi"},
		{"kode terpendek", WriteInput{Kode: "GA", Nama: "X"}, ""},
		{"kode terlalu pendek", WriteInput{Kode: "G", Nama: "X"}, "2-30 karakter"},
		{"kode huruf kecil tidak diterima", WriteInput{Kode: "ga!", Nama: "X"}, "2-30 karakter"},
		{"kode dengan spasi", WriteInput{Kode: "GA 1", Nama: "X"}, "2-30 karakter"},
		{"nama kosong", WriteInput{Kode: "GA", Nama: ""}, "Nama wajib diisi"},
		{"nama terlalu panjang", WriteInput{Kode: "GA", Nama: strings.Repeat("a", 101)}, "maksimal 100"},
		{"keterangan terlalu panjang", WriteInput{Kode: "GA", Nama: "X", Keterangan: strings.Repeat("b", 256)}, "maksimal 255"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := tt.in
			err := in.Normalize(res)

			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Normalize() error = %v, want nil", err)
				}
				return
			}

			if err == nil {
				t.Fatalf("Normalize() error = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Normalize() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestWriteInputNormalizeUnicodeLength(t *testing.T) {
	// Batas panjang dihitung dalam rune, bukan byte: "é" adalah 2 byte tapi
	// hanya 1 karakter bagi pengguna. Menghitung byte akan menolak nama valid
	// yang memakai huruf beraksen.
	res := ResourceDescribe{Singular: "Kategori", Plural: "Kategori"}
	name := strings.Repeat("é", 100) // 200 byte, 100 rune

	in := WriteInput{Kode: "MN", Nama: name}
	if err := in.Normalize(res); err != nil {
		t.Errorf("Normalize() error = %v, want nil (100 rune harus valid)", err)
	}

	in = WriteInput{Kode: "MN", Nama: name + "a"}
	if err := in.Normalize(res); err == nil {
		t.Error("Normalize() error = nil, want error untuk 101 rune")
	}
}

func TestServiceCreateNormalizesBeforeStore(t *testing.T) {
	store := &stubStore{}
	svc := newTestService(store)

	_, err := svc.Create(context.Background(), WriteInput{
		Kode: "  ga  ",
		Nama: "  General Affairs  ",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Store harus menerima input yang sudah bersih.
	if store.lastCreateInput.Kode != "GA" {
		t.Errorf("store menerima kode %q, want GA (uppercase, tanpa spasi)", store.lastCreateInput.Kode)
	}
	if store.lastCreateInput.Nama != "General Affairs" {
		t.Errorf("store menerima nama %q, want tanpa spasi excess", store.lastCreateInput.Nama)
	}
}

func TestServiceCreateRejectsInvalidBeforeStore(t *testing.T) {
	store := &stubStore{}
	svc := newTestService(store)

	if _, err := svc.Create(context.Background(), WriteInput{Kode: "!!", Nama: "X"}); err == nil {
		t.Fatal("Create() error = nil, want validation error")
	}
	// Penting: store TIDAK boleh terpanggil kalau validasi gagal.
	if store.lastCreateInput.Kode != "" {
		t.Error("store dipanggil padahal validasi gagal")
	}
}

func TestServiceCreateTranslatesUniqueViolation(t *testing.T) {
	// Ini yang terjadi tanpa translasi: user dapat 500 dengan pesan
	// "duplicate key value violates unique constraint".
	store := &stubStore{
		createErr: errors.New(`ERROR: duplicate key value violates unique constraint "department_kode_key" (SQLSTATE 23505)`),
	}
	svc := newTestService(store)

	_, err := svc.Create(context.Background(), WriteInput{Kode: "GA", Nama: "X"})
	if err == nil {
		t.Fatal("Create() error = nil, want conflict error")
	}
	if !apperror.IsConflict(err) {
		t.Errorf("kind = %v, want conflict", apperror.KindOf(err))
	}
	if !strings.Contains(err.Error(), "sudah ada") {
		t.Errorf("error = %q, want pesan yang bisa dibaca user", err)
	}
}

func TestServiceCreateTranslatesCheckViolationToInternal(t *testing.T) {
	// CHECK gagal padahal validasi service lolos = keduanya tidak sinkron.
	// Itu bug, bukan kesalahan user, jadi harus jadi 500 dan tercatat.
	store := &stubStore{
		createErr: errors.New(`ERROR: new row violates check constraint "department_kode_format" (SQLSTATE 23514)`),
	}
	svc := newTestService(store)

	_, err := svc.Create(context.Background(), WriteInput{Kode: "GA", Nama: "X"})
	if apperror.KindOf(err) != apperror.KindInternal {
		t.Errorf("kind = %v, want internal (bug konfigurasi, bukan salah user)", apperror.KindOf(err))
	}
}

func TestServiceDeleteRefusesWhenInUse(t *testing.T) {
	store := &stubStore{items: []stubItem{{ID: 1, Usage: 7}}}
	svc := newTestService(store)

	_, err := svc.Delete(context.Background(), 1)
	if err == nil {
		t.Fatal("Delete() error = nil, want conflict")
	}
	if !apperror.IsConflict(err) {
		t.Errorf("kind = %v, want conflict", apperror.KindOf(err))
	}

	msg := err.Error()
	// Pesan harus menyebut jumlah dan nama datanya - "violates foreign key
	// constraint" tidak memberi tahu user harus melakukan apa.
	if !strings.Contains(msg, "7") || !strings.Contains(msg, "request") {
		t.Errorf("error = %q, want jumlah pemakaian dan nama datanya", msg)
	}
}

func TestServiceDeleteAllowsWhenUnused(t *testing.T) {
	store := &stubStore{items: []stubItem{{ID: 1, Usage: 0}}}
	svc := newTestService(store)

	got, err := svc.Delete(context.Background(), 1)
	if err != nil {
		t.Fatalf("Delete() error = %v, want nil", err)
	}
	if got.ID != 1 {
		t.Errorf("Delete() ID = %d, want 1", got.ID)
	}
}

func TestServiceRejectsBadIDs(t *testing.T) {
	svc := newTestService(&stubStore{items: []stubItem{{ID: 1}}})

	for _, id := range []int64{0, -1} {
		if _, err := svc.GetByID(context.Background(), id); err == nil {
			t.Errorf("GetByID(%d) error = nil, want validation error", id)
		}
		if _, err := svc.Delete(context.Background(), id); err == nil {
			t.Errorf("Delete(%d) error = nil, want validation error", id)
		}
	}
}

func TestListInputNormalize(t *testing.T) {
	in := ListInput{Query: "  gaussian  "}
	in.Normalize()

	if in.Query != "gaussian" {
		t.Errorf("Query = %q, want gaussian (spasi dipangkas)", in.Query)
	}

	// Query yang sangat panjang dipangkas supaya tidak membebani database.
	long := ListInput{Query: strings.Repeat("a", 500)}
	long.Normalize()
	if len(long.Query) != 100 {
		t.Errorf("len(Query) = %d, want dipangkas ke 100", len(long.Query))
	}
}

func TestPluralIDHandlesSingularAndPlural(t *testing.T) {
	// Bahasa Indonesia tidak selalu menambahkan -s: "1 request" bukan
	// "1 requests". Kata bantu bilangan dipakai sebagai penanda jumlah.
	if got := pluralID("request", 1); got != "1 request" {
		t.Errorf("pluralID(1) = %q, want %q", got, "1 request")
	}
	if got := pluralID("request", 3); got != "3 request" {
		t.Errorf("pluralID(3) = %q, want %q", got, "3 request")
	}
}

func TestServiceListPassesThrough(t *testing.T) {
	store := &stubStore{items: []stubItem{{ID: 1}, {ID: 2}}}
	svc := newTestService(store)

	items, total, err := svc.List(context.Background(), ListInput{Limit: 20, Offset: 0})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 2 {
		t.Errorf("len(items) = %d, want 2", len(items))
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

func TestServiceListWrapsStoreError(t *testing.T) {
	store := &stubStore{listErr: errors.New("connection refused")}
	svc := newTestService(store)

	_, _, err := svc.List(context.Background(), ListInput{})
	if err == nil {
		t.Fatal("List() error = nil, want error")
	}

	appErr, ok := apperror.AsAPPError(err)
	if !ok {
		t.Fatalf("error bukan domain error: %v", err)
	}

	// Yang dikirim ke klien adalah Message, bukan Error(). Error() sengaja
	// memuat cause supaya berguna di log - yang tidak boleh bocor ke user
	// adalah Message.
	if strings.Contains(appErr.Message, "connection refused") {
		t.Errorf("Message = %q, detail koneksi membocor ke user", appErr.Message)
	}

	// Tapi cause harus tetap bisa ditelusuri untuk debug.
	if appErr.Unwrap() == nil {
		t.Error("Unwrap() = nil, want cause asli tetap tersimpan untuk log")
	}
	if apperror.KindOf(err) != apperror.KindInternal {
		t.Errorf("kind = %v, want internal", apperror.KindOf(err))
	}
}
