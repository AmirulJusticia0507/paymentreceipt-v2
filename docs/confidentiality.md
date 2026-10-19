# Kebijakan Kerahasiaan Internal

Berlaku untuk seluruh pengguna **Payment Receipt v2** — sistem pengajuan &
verifikasi pembelian General Affairs PT. Asi Pudjiastuti Aviation (Susi Air).

> Bagian yang ditandai **`[ISI-PERUSAHAAN]`** membutuhkan keputusan atau
> restensi dari perusahaan sebelum dokumen ini dianggap final. Seluruh klaim
> di luar penanda itu sudah diperiksa langsung terhadap perilaku sistem.

---

## 1. Mengapa kebijakan ini ada

Aplikasi ini memuat harga pembelian, rekening toko, dan data pengajuan internal
perusahaan. Kebocoran data ini bisa merugikan posisi negosiasi harga, mengganggu
privasi karyawan, atau dipakai untuk penipuan transfer. Dokumen ini menetapkan
apa yang wajib dirahasiakan, siapa yang boleh mengaksesnya, dan bagaimana data
boleh dibagikan.

## 2. Klasifikasi data

| Klasifikasi | Contoh | Di sistem | Aturan |
|---|---|---|---|
| **Terbatas — teknis** | Password (hanya hash argon2id), `AUTH_JWT_SECRET`, `AI_API_KEY`, kredensial database | Tabel `users` (hash), file environment | Tidak pernah dibagikan, tidak pernah muncul di log atau UI |
| **Internal — transaksi** | Harga pembelian, harga rencana, rekening & no. telp toko, link pembelian | `payment_receipt`, `pr_archive`, `sources` | Hanya untuk role terautentikasi yang berwenang (§3) |
| **Internal — operasional** | Master data, status pengajuan, angka dashboard | Tabel master, `payment_receipt.status` | Terbatas di dalam aplikasi |
| **Pribadi karyawan** | Username, email, nama lengkap, riwayat login | `users`, `refresh_tokens`, log | Wajib dilindungi — rujukan lanjutan di `docs/privacy-policy.md` |
| **Dokumen hasil** | Kode PR & isi Purchase Requisition (PDF) | Endpoint cetak `/pr-archives/{id}/pdf` | `[ISI-PERUSAHAAN]` apakah PDF boleh dibagikan via WhatsApp/email dan ke pihak mana |

## 3. Siapa boleh mengakses apa (RBAC)

Aturan ini ditegakkan **server, pada setiap request** — bukan sekadar menyembunyikan
tombol di UI. Melanggar dari sisi klien tidak akan berhasil.

| Modul | admin | ga_staff | approver | viewer |
|---|---|---|---|---|
| Master data (Department, Satuan, Kategori, Base, Request Base, Sumber Beli) — baca | ✓ | ✓ | ✓ | ✓ |
| Master data — tulis / hapus | ✓ | — | — | — |
| Request — baca | ✓ | ✓ | ✓ | ✓ |
| Request — tulis / hapus | ✓ | — | — | — |
| Payment receipt — baca | ✓ | ✓ | ✓ | ✓ |
| Payment receipt — buat / ubah | ✓ | ✓ | — | — |
| Payment receipt — ubah status (termasuk massal) | ✓ | ✓ | ✓ | — |
| Payment receipt — hapus | ✓ | — | — | — |
| Arsip PR — baca & unduh PDF | ✓ | ✓ | ✓ | ✓ |
| Arsip PR — buat (termasuk massal), ubah, hapus | ✓ | — | — | — |
| Dashboard | ✓ | ✓ | ✓ | ✓ |

## 4. Aturan perilaku pengguna

1. **Jangan berbagi akun.** Setiap tanda tangan & persetujuan melekat pada akun
   pemiliknya — berbagi akun merusak akuntabilitas dokumen.
2. **Jangan catat password** di berkas, chat, atau tempel monitor. Lupa password
   → hubungi admin sistem (`[ISI-PERUSAHAAN]`: kanal kontak admin).
3. **Jangan teruskan data harga/rekening toko ke luar aplikasi** (chat pribadi,
   email pribadi) kecuali untuk keperluan resmi yang sudah disetujui
   `[ISI-PERUSAHAAN]`.
4. **Buka aplikasi hanya dari perangkat & jaringan resmi**; jangan login dari
   komputer publik. Kalau terlanjur, Keluar dari sesi saat selesai.
5. **Laporkan insiden** (akun tercuri, data bocor, anomali) segera — alur
   pelaporan di §7.

## 5. Kontrol teknis yang sudah berlaku (fakta sistem)

- **Password**: argon2id (format PHC). Plaintext tidak pernah disimpan, di-log,
  maupun ditampilkan — bahkan ke admin.
- **Sesi**: access token 15 menit, cookie `HttpOnly` + `Secure` + `SameSite=Lax`
  + `Path=/`. Refresh token 7 hari, disimpan sebagai **hash** dan **dirotasi
  setiap dipakai** — refresh token lama yang terekspos tidak lagi berlaku.
- **Rate limit login**: 5 percobaan gagal per 15 menit per kombinasi
  username + IP.
- **Pesan error login seragam**: "user tidak ada" dan "password salah"
  membalas identik — penyerang tidak bisa memvalidasi daftar username.
- **Akun nonaktif** (`is_active = false`) ditolak: pada login, perpanjangan sesi,
  dan verifikasi tiap request.
- **Semua query database parameterized** (sqlc) — tanpa konkatenasi string SQL.
- **Otorisasi per request**: middleware role + validasi ulang di service layer.
- **Log akses** memuat `request_id`, method, path, status, durasi, dan
  `remote_addr` — **tanpa** body request, password, maupun token
  (`docs/logging.md`).
- **Batas ukuran body** 2 MiB (10 MiB untuk unggahan gambar AI) — request raksasa
  ditolak sebelum membebani memori.
- `[ISI-PERUSAHAAN]` HTTPS wajib di production: pastikan TLS aktif di reverse
  proxy sebelum data nyata masuk (config sudah menandai cookie `Secure`).

## 6. Retensi & penghapusan data

Fakta sistem saat ini:

- `payment_receipt` dan `pr_archive` **tidak pernah dihapus otomatis** — arsip
  jangka panjang adalah tujuan aplikasi.
- Log aplikasi berputar otomatis di deployment Docker (maks 10 MB × 5 file per
  container).
- Menonaktifkan akun (`is_active = false`) memblokir akses **tanpa** menghapus
  riwayat — ada endpoint hapus akun, tapi penghapusan permanen berarti
  kehilangan referensi persetujuan.

`[ISI-PERUSAHAAN]` Tentukan: masa simpan tiap klasifikasi data, siapa yang
berwenang memutuskan penghapusan, dan prosedur penghapusan akun karyawan yang
berhenti.

## 7. Insiden kebocoran

`[ISI-PERUSAHAAN]` Tetapkan: ke siapa insiden dilaporkan, SLA respons, dan siapa
yang berwenang mengambil tindakan.

Tindakan peredaman yang tersedia pada sistem saat ini:

- **Ganti `AUTH_JWT_SECRET`** → seluruh sesi yang ada langsung tidak berlaku
  (semua pengguna diminta login ulang).
- **Set `is_active = false`** pada akun terdampak → login & sesi berjalan ditolak
  tanpa menghapus riwayat.
- **Ganti password** akun terdampak → hash baru, hash lama tidak berlaku lagi.

## 8. Penegakan & sanksi

`[ISI-PERUSAHAAN]` Hubungkan pelanggaran kebijakan ini ke sanksi disiplin sesuai
peraturan perusahaan (mis. teguran tertulis, pembekuan akses, proses lebih
lanjut).

## 9. Revisi dokumen

| Versi | Tanggal | Perubahan |
|---|---|---|
| 0.1 | 2026-10-19 | Draf awal — bagian `[ISI-PERUSAHAAN]` belum diisi |

Perubahan aturan akses atau klasifikasi data wajib memperbarui dokumen ini di
commit yang sama dengan perubahan kodenya.
