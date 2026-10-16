# ADR 0001 — Greenfield rewrite, bukan refactor PHP

- **Status:** Accepted
- **Tanggal:** 2026-10-16
- **Related:** TODO.md § Mengapa Rewrite

## Konteks

Aplikasi legacy (`AmirulJusticia0507/paymentreceipt`) adalah aplikasi PHP prosedural +
MySQL yang sudah berjalan lama di PT. Asi Pudjiastuti Aviation. Analisis kode menemukan:

**Kel fatally blocking (aplikasi tidak bisa dijalankan apa adanya):**

- `koneksidb.php` — di-`include` oleh hampir semua file, tidak ada di repo
- `template/header_admin.php` & `template/footer.php` — di-`include` oleh semua halaman
- `index.php`, `login.php`, `logout.php` — tidak ada
- `paymentreceipt/paymentreceipt.css` — di-`include` oleh `data_paymentreceipt.php`
- Tidak ada file `.sql` / dump skema database

Artinya skema database **tidak dapat diverifikasi** — harus direkonstruksi dari query yang
ada. Ini mengunci kita pada tebakan.

**Kerentanan:**

- SQL injection di seluruh `function_*.php` — `$_POST` diinterpolasi langsung ke string SQL
- Tidak ada auth guard — endpoint hapus bisa dipanggil langsung via URL
- Delete dilakukan via GET, tanpa CSRF token
- Password di-hash dengan MD5 (jika tabel `user` pernah ada — tidak ada di kode)

**Ketidaklengkapan:**

- Tabel `satuan` dan `categories` di-`JOIN` di banyak query, tapi tidak punya form/handler
  CRUD. User tidak bisa menambah satuan atau kategori baru.
- Tidak ada validasi server-side. Validasi hanya `required` di HTML + JS (yang bahkan
  contains bug — `validasi()` memakai variabel yang tidak dideklarasikan).
- Total harga dikirim dari client (`$_POST['total_harga']`), bukan dihitung di server.

**Isu teknis:**

- 54 MB / 7.290 file aset statis (AdminLTE, bower, jQuery) di-commit ke git
- `sources/data_basename.php` duplikat dari `basename/data_basename.php`
- `fpdf/` dan `google-api-php-client-main/` tidak pernah dipakai — 143 file mati
- Git repo: hanya 1 commit, tanpa `.gitignore`

## Keputusan

Tulis ulang aplikasi dari nol di repo baru (`paymentreceipt-v2`), memakai stack Go + Vue 3 +
PostgreSQL. Legacy **tidak** di-refactor dan **tidak** di-port.

Legacy tetap dibaca sebagai sumber kebenaran untuk:
1. **Aturan bisnis** — field apa yang wajib, alur status, relasi antar tabel
2. **Bentuk dokumen** — template Purchase Requisition (referensi:
   `document/PurchaseReceipt_blank.htm`)
3. **UX yang sudah dikenal pengguna** — pengguna tidak boleh perlu belajar ulang

Legacy **tidak** dipakai untuk:
- Menyalin kode
- Menyalin skema apa adanya (rekonstruksi schema, lalu perbaiki)

## Alternatif yang dipertimbangkan

| Opsi | Kenapa ditolak |
|---|---|
| Refactor PHP bertahap | File inti hilang, jadi tidak ada baseline yang bisa diverifikasi. Setiap perubahan tetap berhadapan dengan SQL injection di 5 file handler. Risiko tinggi tanpa test. |
| Pakai framework PHP baru (Laravel) di atas kode lama | Legacy bukan kode idiomatis Laravel, dan tidak ada test dasar yang bisa dijadikan pegangan. Migrasi tetap setara banyak kerja dengan rewrite, tapi menambah kurva belajar framework yang tidak dibutuhkan tim. |
| Đặt cache/proxy di depan aplikasi lama | Tidak memperbaiki kerentanan. SQL injection tetap ada. |

## Konsekuensi

**Positif:**

- Skema database didesain ulang dengan benar: snake_case, FK eksplisit, constraint, generated
  column untuk `total`, enum untuk status
- Tidak ada satu pun SQL injection — semua query parameterized via sqlc
- Type safety end-to-end: Go struct → JSON → TypeScript type
- Test bisa ditulis (legacy tidak punya test sama sekali)
- Bundle frontend kecil (~200 KB vs 54 MB aset legacy)
- Tidak ada file mati di repo

**Negatif / Risiko:**

- **Skema database harus direkonstruksi.** Risiko data hilang atau tidak termigrasi. Mitigasi:
  dumping data legacy ke staging dan validasi kolom per kolom sebelum production.
- **Tidak ada kompatibilitas backward.** User harus beradaptasi ke UI baru. Mitigasi: analisis
  legacy jadi checklist fitur, dan smoke test manual sebelum release.
- **Estimasi waktu lebih panjang** daripada stabilisasi legacy. Legacy mungkin masih jalan untuk
  sebagian kebutuhan. Mitigasi: sistem baru dibangun bertahap, legacy tetap hidup selama masa
  transisi.
- Outcome butuh konfirmasi eksplisit sebelum production cutover.

## Verifikasi

Keberhasilan keputusan ini dievaluasi dengan:
- Semua query parameterized — dicek via `T11.10` (regression test SQL injection)
- Semua endpoint tertutup punya auth guard — dicek via `T3.9` (matriks RBAC)
- Skema DB terverifikasi — dicek via `T1.20` (migrasi dari nol, 0 error)