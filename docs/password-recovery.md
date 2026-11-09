# Pemulihan password

Pemulihan password mandiri melalui email atau SMS **belum diaktifkan**. Repo ini
belum memiliki provider email/SMS, verifikasi kepemilikan kanal, maupun mekanisme
token pemulihan sekali pakai. Karena itu aplikasi tidak menampilkan alur yang
seolah-olah mengirim kode dan tidak mengubah password berdasarkan nomor telepon
yang belum diverifikasi.

Saat provider tersedia, implementasi wajib mencakup token acak sekali pakai yang
disimpan dalam bentuk hash, masa berlaku singkat, rate limit, respons publik yang
tidak membocorkan keberadaan akun, serta audit log. Untuk saat ini, reset password
hanya dapat dilakukan admin melalui halaman Pengguna setelah verifikasi manual.
