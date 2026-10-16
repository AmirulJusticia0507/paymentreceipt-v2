package router

import "github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/apperror"

// notFound menghasilkan error 404 dengan pesan yang menyebut path yang Dicari.
func notFound(path string) error {
	return apperror.Newf(apperror.KindNotFound, "route.not_found",
		"Endpoint %s tidak ditemukan", path)
}

// methodNotAllowed menghasilkan error 405.
func methodNotAllowed(method string) error {
	return apperror.Newf(apperror.KindValidation, "route.method_not_allowed",
		"Metode %s tidak didukung pada endpoint ini", method)
}

// notImplemented menghasilkan error 501 untuk route yang belum diimplementasikan.
//
// Dipakai sebagai placeholder sampai modul bisnis selesai. 501 lebih jujur
// daripada 404: frontend bisa membedakan "endpoint belum ada" dari "salah alamat".
func notImplemented(path string) error {
	return apperror.NotImplementedf("Endpoint %s belum diimplementasikan", path)
}
