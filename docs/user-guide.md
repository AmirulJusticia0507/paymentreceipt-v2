# Petunjuk Pengguna — Payment Receipt v2

Panduan pemakaian sehari-hari untuk pengguna aplikasi (GA Staff, Approver,
Admin, Viewer) di PT. Asi Pudjiastuti Aviation — Susi Air.

Aturan keamanan & kerahasiaan saat memakai aplikasi: `docs/confidentiality.md`.

> Tanda **`[ISI-PERUSAHAAN]`** = baris yang perlu diisi oleh perusahaan
> (URL, kontak admin, dst.) sebelum panduan ini dibagikan.

---

## 1. Masuk ke aplikasi

1. Buka alamat aplikasi `[ISI-PERUSAHAAN: URL production]`.
2. Masuk dengan **username** dan **password** yang diberikan admin sistem.
   - Belum ada fitur "lupa password" — lupa password → hubungi admin
     `[ISI-PERUSAHAAN: kanal kontak]`.
   - Setelah 5 kali salah password, akun terkunci sementara **15 menit**.
3. Sesi aktif ±15 menit dan diperpanjang selama Anda terus memakai aplikasi.
   Setelah sesi berakhir, cukup masuk ulang.
4. **Keluar** dari menu akun (kanan bawah) saat selesai — wajib pada komputer
   bersama.

## 2. Peran Anda

| Peran | Bisa apa |
|---|---|
| **Admin** | Semua: master data, request, kelola receipt (buat, ubah, hapus), ubah status, arsip PR |
| **GA Staff** | Buat & ubah payment receipt, ubah status, lihat semua data |
| **Approver** | Memeriksa & menyelesaikan receipt (ubah status), lihat semua data |
| **Viewer** | Hanya melihat — dashboard, receipt, arsip, master data |

Yang tidak tercantum di tabel berarti tidak berwenang — tombol terkait tidak
muncul, dan akses langsung ke URL akan ditolak server.

## 3. Dashboard

Halaman pertama setelah masuk: ringkasan jumlah receipt per status, pengeluaran
bulanan, departemen dengan pengeluaran terbesar, dan aktivitas terbaru.

## 4. Master Data (Admin)

Sidebar → **Master Data**: Department, Satuan, Kategori, Base, **Sumber Beli**.

- **Sumber Beli** punya dua mode: *online* (tautan link pembelian) atau
  *offline* (nama toko, no. telp, no. rekening — rekening ini dipakai pada kolom
  transfer di dokumen PR).
- Hanya admin yang menambah/mengubah/menghapus. Pengguna lain mengajukan
  perubahan ke admin `[ISI-PERUSAHAAN: kanal]`.

Sidebar → **Pengajuan**: Request & Request Base (daftar pengajuan per
departemen) — kelola oleh admin; setiap payment receipt merujuk satu request.

## 5. Payment Receipt — mencatat pembelian

Buka sidebar → **Payment Receipt**.

### Membuat receipt (Admin / GA Staff)

1. Klik **Tambah**. Isi tanggal pembelian (tidak boleh melewati hari ini),
   request & department (keduanya harus nyambung), sumber beli, lalu item:
   deskripsi, qty, satuan, harga.
2. **Total dihitung sistem** (qty × harga) — tidak diketik manual; qty atau
   harga tidak boleh nol/negatif.
3. Opsional: gunakan **Ekstraksi AI** (§6) untuk mengisi item dari foto struk.
4. Simpan.

### Mengubah status (Admin / GA Staff / Approver)

Alurnya: `pending → checked → processed → finished`, atau `cancelled` dari
status apa pun sebelum finished. Setelah **finished**, receipt tidak bisa
diubah lagi.

- Approval biasanya: Approver memeriksa → **checked**; GA memproses →
  **processed**; selesai → **finished**.
- Ubah banyak sekaligus: centang beberapa baris → **Ubah Status Massal**.
  Semua receipt harus lolos validasi — bila satu gagal, **tidak ada** yang
  berubah.

### Filter, pencarian, ekspor

Filter status & tanggal, pencarian teks, pengurutan kolom, dan **Export CSV**
(maksimal 5.000 baris per ekspor — saring dulu bila data lebih banyak).

### Menghapus (Admin saja)

Hapus permanen dari halaman daftar/detail. Gunakan hanya untuk data keliru —
riwayat transaksi sebaiknya diarsipkan, bukan dihapus.

## 6. Ekstraksi AI dari foto struk

Halaman **Ekstraksi AI** (`/ai/extract-receipt`) mengunggah gambar struk, lalu
AI membaca dan mengisikan data item secara otomatis.

- Tipe file: JPG, PNG, WEBP — maksimal 10 MiB.
- **Periksa selalu hasilnya** sebelum dipakai: AI bisa salah baca angka.
- Gambar dikirim ke layanan AI eksternal — jangan unggah struk berisi data yang
  tidak boleh keluar perusahaan (`docs/privacy-policy.md` §4).

## 7. Arsip PR & cetak Purchase Requisition

Sidebar → **Arsip PR**.

1. Receipt berstatus **finished** diarsipkan oleh **Admin**: tombol arsip pada
   receipt, atau **Arsipkan Massal** untuk banyak receipt sekaligus. Kode PR
   dibuat otomatis (atau isi kode sendiri bila perlu).
2. Buka arsip untuk melihat detail, atau unduh **PDF Purchase Requisition** —
   dokumen berlogo, ditujukan ke General Affair Department, dengan blok tanda
   tangan Department / Manager–Director / General Affairs Department.
3. `[ISI-PERUSAHAAN]` Prosedur tanda tangan basah & penyebaran PDF hasil cetak.

Admin juga dapat mengubah data arsip dan menghapusnya bila terjadi kesalahan.

## 8. Masalah umum

| Gejala | Penyebab | Yang dilakukan |
|---|---|---|
| "Anda harus masuk terlebih dahulu" | Sesi berakhir | Masuk ulang |
| "Username atau password salah" | Salah ketik, atau akun terkunci | Tunggu 15 menit (setelah 5 gagal) atau hubungi admin |
| "Anda tidak berwenang" (403) | Peran Anda tidak mencakup akses itu | Hubungi admin |
| "Berkas terlalu besar" | Unggahan melewati batas (10 MiB AI / 2 MiB lainnya) | Kecilkan file |
| "Tanggal pembelian tidak boleh melewati hari ini" | Tanggal di masa depan | Pilih hari ini atau sebelumnya |
| "Total harus > 0" | Qty atau harga 0/negatif | Periksa kolom item |
| Upload AI ditolak (tipe file) | Bukan JPG/PNG/WEBP | Konversi gambar dulu |

## 9. Yang belum tersedia (bukan bug)

- Reset/lupa password mandiri — hubungi admin.
- Manajemen user via aplikasi — akun dibuat admin sistem di sisi server.
- Notifikasi email/WhatsApp otomatis.
- Halaman Ekstraksi AI belum ditautkan ke menu sidebar — akses langsung dengan
  URL `/ai/extract-receipt`.

---

Terakhir diperbarui: 2026-10-19.
