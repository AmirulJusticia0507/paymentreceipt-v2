package router

import (
	"github.com/go-chi/chi/v5"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/dbgen"
	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/masterdata"
)

// DTO bentuk JSON yang dikirim ke klien.
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

// satuanDTO mengosongkan Keterangan karena tabel satuan tidak punya kolom itu.
//
// Bentuk JSON tetap sama dengan tabel master lain supaya satu komponen tabel
// bisa dipakai ulang tanpa percabangan di template.
func satuanDTO(d dbgen.Satuan) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama}
}

func requestDTO(d dbgen.Request) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama, Keterangan: d.Keterangan}
}

func requestbaseDTO(d dbgen.RequestBase) any {
	return masterDTO{ID: d.ID, Kode: d.Kode, Nama: d.Nama, Keterangan: d.Keterangan}
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
) {
	svc := masterdata.NewService(store, resource, noun)
	h := masterdata.NewHandler(svc, toDTO)

	admin := authn.RequireRole(auth.RoleAdmin)

	r.Route(path, func(r chi.Router) {
		// Bacanya terbuka untuk semua user yang sudah login - user perlu melihat
		// daftar department untuk mengisi form payment receipt.
		r.Get("/", h.List)
		r.Get("/{id}", h.GetByID)

		r.With(admin).Post("/", h.Create)
		r.With(admin).Put("/{id}", h.Update)
		r.With(admin).Delete("/{id}", h.Delete)
	})
}
