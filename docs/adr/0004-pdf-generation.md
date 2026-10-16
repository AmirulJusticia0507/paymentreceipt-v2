# ADR 0004 — PDF: render server-side dengan gofpdf

- **Status:** Accepted
- **Tanggal:** 2026-10-16
- **Related:** TODO.md § Phase 8

## Konteks

Aplikasi legacy menghasilkan dokumen **Purchase Requisition** sebagai PDF lewat TCPDF
(`paymentreceipt/printdocuments.php`). Dokumen ini adalah output utama sistem — ia yang
ditandatangani dan diarsipkan.

Ada **dua** jalur output di legacy, dan keduanya bermasalah:

**1. `printdocuments.php` (TCPDF) — jalur utama**

```php
$pdf = new TCPDF(PDF_PAGE_ORIENTATION, PDF_UNIT, PDF_PAGE_FORMAT, true, 'UTF-8', false);
$pdf->AddPage();
$pdf->Cell(15, 7, 'No', 1, 0, 'C', 0);
$pdf->Cell(60, 7, 'Description', 1, 0, 'L', 0);
// ... per item: Cell() langsung tanpa reset X → kolom ter-NEXT
```

Bug yang ditemukan:

- **`?GApayment_id=` diterima tapi tidak pernah dipakai di query.** PDF selalu mencetak
  *seluruh* data, bukan hanya transaksi yang dipilih. Tombol "Cetak" di setiap baris
  menghasilkan dokumen identik.
- **`Cell()` dipakai untuk isi tabel tanpa wrapping.** Setiap item ditambahkan secara
  horizontal, bukan menjadi baris tabel. Kolom akan keluar dari halaman.
- **`AddPage()` dipanggil berulang** sehingga satu item = satu halaman.
- `GROUP BY keterangandepartment` menggabungkan banyak baris jadi satu → data yang ditampilkan
  tidak konsisten dengan `GApayment_id` yang diklik.
- `MultiCell`/`Cell` dicampur, menghasilkan layout yang tidak stabil.

**2. `cetaklaporanpayment.php` — jalur kedua (broken)**

File ini meng-`include` `../koneksidb.php` di baris 3, artinya dirancang sebagai **partial**
yang di-`include` halaman lain, bukan entrypoint mandiri. Tidak ada yang meng-include-nya —
tombol cetak di `data_paymentreceipt.php` sudah dikomentari-out.

Legacy juga punya `document/PurchaseReceipt_blank.htm` — konversi PDF ke HTML dengan
gambar JPG dan posisi absolut. Ini **artefak**, bukan template yang dipakai runtime.

## Keputusan

Render PDF **server-side** di Go dengan library **`gofpdf`**, dan **secara paralel** sediakan
**print CSS** di frontend untuk kebutuhan *quick print*.

Output PDF dikirim sebagai binary stream dengan header:

```
Content-Type: application/pdf
Content-Disposition: inline; filename="Purchase-Requisition-{id}.pdf"
```

Endpoint:

```
GET /api/v1/payment-receipts/:id/pdf   → PDF satu transaksi
GET /api/v1/reports/pr-archive/:id/pdf → PDF satu arsip PR
```

Query untuk PDF **wajib** filter by `:id` — ini langsung menutup bug legacy.

Struktur kode:

```
internal/platform/pdf/
├── document.go     # konfigurasi umum (font, margin, page size)
├── requisition.go  # template Purchase Requisition
└── *_test.go       # test: generate → assert teks & total
```

## Why server-side, bukan print CSS saja

Print CSS (`@media print`) lebih murah dan cepat untuk kasus sederhana. Tapi tidak
memenuhi kebutuhan:

| Kebutuhan | Print CSS | Server-side PDF |
|---|---|---|
| Dokumen yang bisa diarsipkan & dikirim via email | Gagal | Berhasil |
| Output identik di semua browser | Gagal (bergantung browser/OS/font) | Berhasil |
| Bisa disimpan ke sistem dokumen | Gagal | Berhasil |
| Tidak perlu print dialog tiap kali | Gagal | Berhasil |
| Header/footer, nomor halaman, nomor dokumen | Terbatas | Penuh |

Kebutuhan dokumen resmiatisfaction ini yang menentukan. Print CSS tetap disediakan
(T8.10) sebagai jalur cepat untuk daftar yang sedang dikerjakan.

## Why gofpdf

- **Pure Go, tanpa cgo** — binary tetap bisa di-build static, tidak butuh C toolchain
- API-nya **sederhana dan stabil** — `Cell`, `MultiCell`, `Output` — persis pola yang
  familiar dari TCPDF/FPDF, jadi migrasi template lebih mudah
- Font TTF embedding mendukung **UTF-8** — penting karena dokumen memakai nama barang
  Bahasa Indonesia
- Lisensi permissive (MIT) — berbeda dari TCPDF (LGPLv3)

## Alternatif yang dipertimbangkan

| Opsi | Pro | Kontra |
|---|---|---|
| **`gofpdf`** | Pure Go, tanpa cgo, API mirip FPDF/TCPDF, MIT | ❌ Agak "legacy" API; ❌ Tidak punya layout engine sophisticated (grid/flexible); ❌ Proyek relatif lama, activity-nya rendah — perlu fork bila sudah tidak dirawat |
| **`gofpdf` fork (`phpg/go-pdf-fork`)** | Go 1.7+ compatible, maintained | Lebih banyak dependensi |
| **`unipdf`** | FitUR lengkap, aktif | **AGPL / proprietary license** — tidak cocok untuk aplikasi korporat |
| **`chromedp` (headless Chrome → PDF)** | Output **sempurna** — bisa pakai HTML+CSS+Tailwind, identik dengan web | ❌ Butuh Chrome di server (image ~500MB) ❌ Lambat (~2-5 detik/dokumen) ❌ Berat di resource ❌ Cocok untuk batch, tidak cocok untuk request sinkron |
| **`wkhtmltopdf`** | Bagus | Proyek **sudah mati** (2020), tidak ada maintenance |
| **Print CSS saja** | Murah, cepat | Tidak bisa diarsipkan, output tidak konsisten |
| **Client-side `jsPDF` / `pdfmake`** | Tanpa round-trip server | Output-an.cz authenticity bermasalah — PDF bisa diedit; burden di browser |

### Catatan tentang `chromedp`

Opsi ini/RSHELL sangat menarik secara kualitas output: kita bisa menulis template HTML dengan
Tailwind, lalu Chrome merendernya jadi PDF. Cocok kalau dokumen butuh layout kompleks.

Ditolak untuk **request sinkron** karena alasan performa dan ukuran. Tapi ini **bukan
ditutup permanen** — kalau nanti dokumen tumbuh kompleks (multi-halaman dengan header
berbeda per bagian, grafik, watermark), **`chromedp` sebagai worker async adalah jalur
migrasi yang wajar**. Ini dicatat sebagai opsi, bukan penolakan permanen.

## Konsekuensi

**Positif:**

- Bug "PDF tidak filter by ID" langsung tertutup — endpoint pakai `:id` yang wajib
- Output konsisten & bisa diarsipkan
- Binary tetap pure Go, tanpa cgo
- Layout memakai **satu template HTML terpisah** (bukan template TCPDF) → mudah direview
  dan di-debug
- Test otomatis: generate PDF lalu assert isi & total

**Negatif / Risiko:**

- `gofpdf` API basic — tabel harus dibangun manual dengan perhitungan lebar kolom
- Font harus di-embed sebagai file `.ttf` (repo membesar ± 300 KB per font, masih wajar)
- Tidak mendukung browser-based layout — perubahan layout butuh regenerate manual
- Server harus punya font subset yang dipakai dokumen

**Mitigasi:** Widar kolom dihitung di satu tempat (`columnWidths()`), jadi perubahan layout
cuma ubah angka, bukan logika. Test `T8.11` mengunci tampilan.

## Verifikasi

- `T8.11` — generate PDF dari fixture data, assert teks (nama perusahaan, nama item, nomor)
  dan **total** muncul. Ini langsung menangkap bug legacy.
- `T8.12` — konfirmasi query PDF filter by `:id`: generate PDF untuk 2 transaksi berbeda,
  pastikan isi keduanya **tidak sama**.
