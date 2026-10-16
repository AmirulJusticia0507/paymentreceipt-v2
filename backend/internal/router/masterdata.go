package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// masterDTO adalah bentuk JSON dasar untuk tabel master sederhana.
//
// Ditulis eksplisit, bukan marshal langsung dari struct sqlc. Struct database
// adalah detail implementasi yang boleh berubah; bentuk JSON adalah kontrak
// dengan frontend. Kalau dipetakan eksplisit, menambah kolom di database tidak
// otomatis membocorkannya ke API.
type masterDTO struct {
	ID         int64  `json:"id"`
	Kode       string `json:"kode"`
	Nama       string `json:"nama"`
	Keterangan string `json:"keterangan"`
}

func departmentDTO(d dbgen.Department) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama, Keterangan: d.Keterangan}
}

func categoryDTO(d dbgen.Category) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama, Keterangan: d.Keterangan}
}

func baseDTO(d dbgen.Base) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama, Keterangan: d.Keterangan}
}

func requestDTO(d dbgen.Request) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama, Keterangan: d.Keterangan}
}

// satuanDTO mengosongkan Keterangan karena tabel satuan tidak punya kolom itu.
//
// Bentuk JSON tetap sama dengan tabel master lain supaya satu komponen tabel
// bisa dipakai ulang tanpa percabangan di template.
func satuanDTO(d dbgen.Satuan) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama}
}

// requestBaseDTO memetakan request base ke bentuk JSON.
//
// Bukan masterDTO biasa karena ada dua field tambahan: jabatan (penanggung
// jawab pengajuan) dan base_id (induknya). base_id wajib dikembalikan supaya
// frontend bisa menampilkan base mana yang sedang dipilih tanpa menebak dari
// daftar.
func requestBaseDTO(d dbgen.RequestBase) any {
	return map[string]any{
		"id":         d.ID,
		"kode":       d.Kode,
		"nama":       d.Nama,
		"jabatan":    d.Jabatan,
		"keterangan": d.Keterangan,
		"base_id":    d.BaseID,
	}
}

// mountMasterData memasang route CRUD untuk satu tabel master.
//
// Store sudah dikonstruksi oleh pemanggil dan tipe T ditulis eksplisit
// (mountMasterData[dbgen.Department](...)). Go tidak bisa menyimpulkan T dari
// fungsi factory func(*Queries) Store[T], jadi lebih jelas kalau pemanggil
// menyebut tipenya sendiri.
//
// Hanya admin yang boleh menulis. Pembatasan ini di router, bukan di service:
// service juga dipakai jalur non-HTTP (seeding, impor) yang tidak punya konsep
// session, jadi kalau ditambahkannya di sana setiap pemanggil harus ingat.
func mountMasterData[T any](
	r chi.Router,
	authn *auth.Authenticator,
	store masterdata.Store[T],
	path string,
	resource masterdata.ResourceDescribe,
	noun string,
	toDTO func(T) any,
	// searchHandler opsional - hanya tabel yang punya endpoint autocomplete
	// yang mengisinya.
	searchHandler http.HandlerFunc,
) {
	svc := masterdata.NewService(store, resource, noun)
	h := masterdata.NewHandler(svc, toDTO)

	admin := authn.RequireRole(auth.RoleAdmin)

	r.Route(path, func(r chi.Router) {
		// /search HARUS didaftarkan sebelum /{id}, dan di DALAM subrouter ini.
		//
		// Dua kesalahan bisa terjadi di sini, dan keduanya sudah pernah terjadi:
		//
		//  1. Terdaftar setelah /{id}. Chi mencocokkan dalam urutan pendaftaran,
		//     jadi /{id} akan menelan "/search", mencoba mem-parse "search"
		//     sebagai angka, lalu membalas 422 "ID harus berupa angka". Gejalanya
		//     jauh dari penyebabnya.
		//  2. Terdaftar di luar r.Route(path, ...). Registrasinya jadi "/search"
		//     di root router, bukan "/<path>/search", dan tidak pernah terpanggil.
		if searchHandler != nil {
			r.Get("/search", searchHandler)
		}

		// Bacanya terbuka untuk semua user yang sudah login - user perlu melihat
		// daftar department dan satuan untuk mengisi form payment receipt.
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)

		r.With(admin).Post("/", h.Create)
		r.With(admin).Put("/{id}", h.Update)
		r.With(admin).Delete("/{id}", h.Delete)
	})
}
