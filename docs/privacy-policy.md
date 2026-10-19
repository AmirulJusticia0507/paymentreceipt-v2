# Kebijakan Privasi

Berlaku untuk data pribadi karyawan yang diproses **Payment Receipt v2**
(PT. Asi Pudjiastuti Aviation — Susi Air). Ini aplikasi **internal**: data
dipakai untuk operasional General Affairs — bukan untuk pemasaran, analitik
pihak ketiga, maupun tujuan lain di luar §2.

> Bagian bertanda **`[ISI-PERUSAHAAN]`** butuh keputusan perusahaan sebelum
> final. Klaim lain sudah diperiksa terhadap perilaku sistem.

---

## 1. Data pribadi yang diproses

| Data | Sumber | Dipakai untuk |
|---|---|---|
| Username, email, nama lengkap | Dibuat oleh admin sistem | Otentikasi; identitas pembuat/penanggung jawab dokumen |
| Password | Diketik pengguna | Otentikasi — disimpan sebagai **hash argon2id**, tidak pernah plaintext |
| Riwayat login (waktu, hasil) | Otomatis | Keamanan & audit |
| Alamat IP + user-agent saat login | Otomatis | Rate limit & deteksi penyalahgunaan |
| Jejak aktivitas transaksi (pembuat, pengubah, `approved_by`) | Otomatis | Akuntabilitas alur persetujuan |
| Gambar struk/faktur yang diunggah ke fitur **Ekstraksi AI** | Pengguna | Ekstraksi data terstruktur — **dikirim ke penyedia layanan AI eksternal** (§4) |

Data yang **tidak** diproses: nomor rekening pribadi karyawan, nomor kartu
pembayaran, atau data pribadi apa pun di luar daftar §1 — aplikasi tidak
memintanya.

## 2. Tujuan & dasar pemrosesan

Pemrosesan terbatas pada: (a) mengelola pengajuan pembelian; (b) menerbitkan
dokumen Purchase Requisition yang ditandatangani pejabat berwenang; (c) laporan
pengeluaran per departemen; (d) keamanan sistem (rate limit, audit trail).

`[ISI-PERUSAHAAN]` Dasar formal pemrosesan (kebijakan internal perusahaan /
ketentuan hubungan kerja) dan lingkup organisasi yang tercakup.

## 3. Siapa yang melihat data

Di dalam perusahaan: matriks hak akses di `docs/confidentiality.md` §3. Admin
sistem dapat melihat seluruh data, termasuk email dan riwayat aktivitas, namun
**tidak pernah** melihat password plaintext (hanya hash).

Di luar perusahaan: tidak ada data yang dikirim keluar, **kecuali** gambar struk
pada fitur AI — lihat §4.

## 4. Pihak ketiga — penyedia layanan AI

Fitur **Ekstraksi AI Struk** mengirim gambar yang diunggah (faktur/struk
pembelian) ke endpoint AI eksternal yang dikonfigurasi lewat environment
`AI_BASE_URL` + `AI_API_KEY`. Nilai default saat ini menunjuk layanan proxy
AI pihak ketiga (`smartsproxy.clouad/v1`) — layanan tersebut menerima gambar
beserta isinya (nama toko, daftar barang, harga).

Gambar struk umumnya memuat data transaksi, bukan data pribadi karyawan; namun
struk bisa saja memuat nama orang.

`[ISI-PERUSAHAAN]` Sebelum fitur ini dipakai luas, putuskan dan catat:

- penyedia AI mana yang disetujui perusahaan dan status perjanjiannya;
- apakah gambar dipakai untuk training model penyedia;
- lokasi pemrosesan data.

Bila kebijakan penyedia tidak sesuai, fitur dapat dimatikan tanpa memengaruhi
fungsi lain (cukup kosongkan/ubah `AI_BASE_URL` ke endpoint yang disetujui —
ekstraksi manual selalu tersedia).

## 5. Keamanan

Ringkasan kontrol: hash argon2id, cookie sesi `HttpOnly` + `Secure` +
`SameSite=Lax`, rate limit login 5 gagal / 15 menit, pesan error login seragam,
seluruh query database parameterized, otorisasi per request, log tanpa isi body
maupun kredensial. Detail lengkap: `docs/confidentiality.md` §5.

`[ISI-PERUSAHAAN]` Pastikan HTTPS aktif di production sebelum data nyata masuk.

## 6. Retensi & penghapusan

Fakta sistem: tidak ada penghapusan otomatis untuk data transaksi maupun akun
(arsip adalah tujuan aplikasi); log aplikasi berputar otomatis. Menonaktifkan
akun memblokir akses tanpa menghapus riwayat.

`[ISI-PERUSAHAAN]` Masa simpan data pribadi, prosedur penghapusan akun karyawan
berhenti, dan dasar penyimpanan riwayat persetujuan.

## 7. Hak karyawan atas datanya

Karyawan berhak: mengetahui data apa tentang dirinya yang diproses; meminta
koreksi email/nama lengkap; meminta penonaktifan akun.

`[ISI-PERUSAHAAN]` Kanal & SLA permintaan ini (ke mana mengajukan, berapa hari
respons) — sistem belum punya fitur self-service untuk hak ini.

## 8. Cookie

| Cookie | Isi | Umur (default) | Sifat |
|---|---|---|---|
| Access token (nama: `AUTH_COOKIE_NAME`) | Token sesi singkat | 15 menit | `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/` |
| Refresh token | Token perpanjang sesi | 7 hari | `HttpOnly`, `Secure`, `SameSite=Lax`; disimpan hash & dirotasi tiap dipakai |

Tidak ada cookie analitik, iklan, atau pelacakan pihak ketiga.

## 9. Kontak & perubahan kebijakan

`[ISI-PERUSAHAAN]` Kontak pengelola data (HR / GA / pejabat yang ditunjuk).

| Versi | Tanggal | Perubahan |
|---|---|---|
| 0.1 | 2026-10-19 | Draf awal — bagian `[ISI-PERUSAHAAN]` belum diisi |
