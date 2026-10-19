# API Reference — Payment Receipt v2

Referensi HTTP API untuk frontend dan integrator eksternal. Semua endpoint ada di
bawah **`/api/v1`** (kecuali healthcheck root, lihat [Health](#healthcheck)).

Kontrak di dokumen ini mengikuti kode di `backend/internal/router/` — kalau
berbeda, kode yang menang, dan dokumen wajib diperbarui di commit yang sama.

Dokumen terkait:

- [docs/logging.md](logging.md) — skema log & korelasi `X-Request-ID`
- [docs/runbook-deploy.md](runbook-deploy.md) — operasi server

---

## Konvensi

### Envelope respons

Sukses selalu memakai envelope:

```json
// Satu objek (GET by id, create, update)
{ "data": { ... } }

// Daftar dengan paginasi
{ "data": [ ... ], "meta": { "page": 1, "per_page": 20, "total": 137, "total_pages": 7 } }
```

Beberapa endpoint agregat (dashboard, autocomplete) hanya punya `data` tanpa
`meta` — bentuk masing-masing disebut di bawah.

### Paginasi

Parameter query untuk semua endpoint daftar:

| Param | Default | Nilai |
|---|---|---|
| `page` | `1` | ≥ 1. Nilai tidak valid jatuh ke default (tidak error). |
| `per_page` | `20` | 1–100. Di atas 100 dipangkas ke 100. |
| `q` | — | Pencarian teks (kebanyakan cocok sebagian di kode/nama). |

`meta.total` adalah jumlah baris yang cocok dengan filter (bukan jumlah baris di
halaman).

### Autentikasi

Dua cara, dipilih otomatis oleh server:

1. **Cookie HttpOnly** (`pr_access_token` untuk access, `pr_refresh_token` untuk
   refresh) — dipakai frontend di origin yang sama. `Secure` + `SameSite` sesuai
   konfigurasi.
2. **`Authorization: Bearer <access token>`** — untuk integrator/klien non-browser.

Refresh token **hanya** lewat cookie (`POST /auth/refresh` tidak menerima body
token) — ini disengaja supaya refresh token tidak pernah berada di
`localStorage`.

Endpoint yang butuh autentikasi mengembalikan `401` bila token tidak ada/absen,
dan `403` bila sudah login tapi role tidak cukup.

### Role

| Role | Nilai | Hak umum |
|---|---|---|
| Admin | `admin` | Semua, termasuk hapus & manajemen master data. |
| GA Staff | `ga_staff` | Buat/ubah payment receipt, ubah status. |
| Approver | `approver` | Ubah status payment receipt (termasuk menandai finished/cancelled). |
| Viewer | `viewer` | Baca saja. |

Matriks per-endpoint ada di tiap tabel di bawah (`Admin only` = hanya `admin`).

### Error (RFC 9457 Problem Details)

Semua error memakai bentuk yang sama:

```json
{
  "type": "https://httpstatuses.io/422",
  "title": "Unprocessable Entity",
  "status": 422,
  "detail": "Data payment receipt belum valid",
  "instance": "0af3c2e1d4b5a6978c1f2e3d4b5a6978",
  "code": "payment_receipt.validation",
  "errors": [
    { "field": "jumlah_items", "message": "Jumlah harus lebih dari 0", "rule": "positive" }
  ]
}
```

| Field | Kapan ada | Keterangan |
|---|---|---|
| `code` | hampir selalu | Kode stabil untuk logika frontend (mis. `payment_receipt.validation`, `unauthorized`). |
| `errors` | error validasi (422) | Detail per-field; `rule` membantu pemilihan pesan UI. |
| `instance` | selalu | `request ID` — identik dengan header `X-Request-ID`; kunci pencarian di log. |
| `retry_after` | 429 | Detik sampai boleh coba lagi. |

Pemetaan `Kind` → status:

| Status | Arti |
|---|---|
| `401` | Belum login / token kedaluwarsa. |
| `403` | Login tapi role kurang. |
| `404` | Resource/route tidak ada. |
| `409` | Konflik state (mis. transisi status ilegal). |
| `422` | Validasi gagal — perbaiki input. |
| `429` | Rate limit. |
| `500` | Bug server (detail tidak pernah bocor ke klien). |
| `501` | Endpoint belum diimplementasikan. |
| `502` | Dependensi luar (database) bermasalah. |

### Header

- **`X-Request-ID`** — dikirim di request (opsional; harus
  `[A-Za-z0-9_-]`, panjang 8–64) dan selalu dibalas di response. Dipakai untuk
  korelasi dengan log (lihat `docs/logging.md`).
- `Content-Type: application/json; charset=utf-8` untuk semua body JSON.
- Batas ukuran body global: **2 MiB** (`SERVER_MAX_REQUEST_BODY`, default) untuk
  semua endpoint — request yang melebihi ditolak saat dibaca (`422`).
  Pengecualian: endpoint unggahan AI memakai batas khusus
  `AI_MAX_REQUEST_BODY` (default 10 MiB), lihat
  [AI extract struk](#ai-extract-struk).

### Transisi status payment receipt

```
pending ──▶ checked ──▶ processed ──▶ finished   (terminal)
                │            │
                └────┬───────┘
                     ▼
               cancelled (terminal)
```

Transisi di luar diagram menghasilkan `409`. `finished` dan `cancelled` bersifat
terminal.

---

## Healthcheck

| Method | Path | Auth | Keterangan |
|---|---|---|---|
| GET | `/healthz` | — | Liveness. Tidak mengecek database. Juga tersedia di `/api/v1/healthz`. |
| GET | `/readyz` | — | Readiness. Memeriksa tiap komponen (mis. `database`); `200` bila semua `ok`, `503` bila ada `down`. |

Response `/healthz` — `{ "status": "ok", "uptime": "1h30m0s", "time": "..." }`.

Response `/readyz` (`200` bila semua komponen `ok`, `503` kalau ada `down`):

```json
{
  "status": "ok",
  "checks": [
    { "component": "database", "status": "ok", "duration_ms": 3 }
  ],
  "time": "2026-10-19T08:00:00Z"
}
```

Saat database tidak terjangkau, `status` menjadi `"down"`, `checks[0].status`
menjadi `"down"` dengan `error` berisi pesan koneksi, dan status HTTP `503`.

---

## Auth

### POST `/api/v1/auth/login` — publik

```json
{ "username": "admin", "password": "rahasia" }
```

`200` + menyetel cookie access & refresh (HttpOnly), body:

```json
{
  "data": {
    "user": { "id": 1, "username": "admin", "email": "...", "full_name": "...", "role": "admin", "last_login_at": "..." },
    "expires_at": "2026-10-19T09:00:00Z"
  }
}
```

Token **tidak pernah** ada di body — hanya cookie. Pesan error login sengaja
seragam untuk "username tidak ada" dan "password salah" (tanpa field-level hint)
agar tidak membocorkan daftar username. Rate limit aktif pada endpoint ini.

### POST `/api/v1/auth/refresh` — publik

Tanpa body; server membaca cookie refresh. Mengembalikan bentuk yang sama
seperti login (cookie diperbarui). Gagal → cookies dibersihkan + `401`.

### POST `/api/v1/auth/logout`

Butuh autentikasi. Selalu `200 { "data": { "message": "Berhasil keluar" } }`
meski token sudah tidak ada (idempoten); kedua cookie dibersihkan.

### GET `/api/v1/auth/me`

Butuh autentikasi. Mengembalikan `data` = profil user (tanpa data sensitif).

---

## Master data

Semua endpoint daftar mendukung `q`, `page`, `per_page`. **Tulis (POST/PUT/DELETE)
hanya admin.** Baca butuh login apa pun role-nya.

Response objek memakai bentuk umum:

```json
{ "data": { "id": 1, "kode": "GA", "nama": "General Affairs", "keterangan": "..." } }
```

`satuan` tidak punya field `keterangan` di response (tabelnya tidak punya kolom
itu) — bentuk JSON sengaja disamakan supaya satu komponen UI bisa dipakai semua.

| Method & path | Catatan |
|---|---|
| GET/POST `/departments`, PUT/DELETE `/departments/{id}` | Unit kerja pemohon. |
| GET/POST `/satuan`, PUT/DELETE `/satuan/{id}` | Satuan barang. |
| GET/POST `/categories`, PUT/DELETE `/categories/{id}` | Kategori pembelian. |
| GET/POST `/bases`, PUT/DELETE `/bases/{id}` | Base pengajuan. |

### GET `/api/v1/bases/search?q=...` — autocomplete

Untuk form request base. Query kosong → `data: []` (bukan seluruh tabel). Hasil
maksimal 10 baris:

```json
{ "data": [ { "value": 3, "label": "Operasional Kantor", "kode": "OPS-01" } ] }
```

---

## Requests (pengajuan department)

| Method & path | Role |
|---|---|
| GET `/requests` | Login |
| GET `/requests/{id}` | Login |
| POST `/requests` | Admin |
| PUT `/requests/{id}` | Admin |
| DELETE `/requests/{id}` | Admin |

Query list: `q`, `department_id`, `page`, `per_page` — filter per departemen
dipakai frontend untuk daftar pengajuan milik satu department.

Body create/update:

```json
{ "kode": "REQ-2026-001", "nama": "Pembelian ATK", "keterangan": "...", "department_id": 1 }
```

`department_id` wajib > 0 (`422` bila tidak).

---

## Request bases

| Method & path | Role |
|---|---|
| GET `/request-bases` | Login |
| GET `/request-bases/{id}` | Login |
| POST `/request-bases` | Admin |
| PUT `/request-bases/{id}` | Admin |
| DELETE `/request-bases/{id}` | Admin |

Body create/update:

```json
{ "kode": "RB-001", "nama": "...", "jabatan": "Kepala GA", "keterangan": "...", "base_id": 3 }
```

`base_id` wajib (setiap request base punya induk base) dan diverifikasi ada
sebelum disimpan. `jabatan` boleh kosong. Response objek memuat `base_id` +
`jabatan` selain field umum.

---

## Sources (sumber pembelian)

| Method & path | Role |
|---|---|
| GET `/sources` | Login |
| GET `/sources/{id}` | Login |
| POST `/sources` | Admin |
| PUT `/sources/{id}` | Admin |
| DELETE `/sources/{id}` | Admin |

Query list: `q`, `source` (`online` \| `offline`), `page`, `per_page`.

Body:

```json
{
  "source": "online",
  "nama_barang": "Printer",
  "nama_toko": "", "no_telp": "", "no_rekening": "",
  "link_pembelian": "https://toko.example/printer",
  "keterangan": ""
}
```

Validasi conditional — ini aturan bisnisnya:

- `source` wajib `online` atau `offline` (enum).
- `nama_barang` wajib, maks 200 karakter.
- **`online`**: `link_pembelian` wajib berupa URL `http/https` — field toko,
  telp, rekening **dikosongkan otomatis**.
- **`offline`**: `nama_toko`, `no_telp`, `no_rekening` wajib — `link_pembelian`
  **dikosongkan otomatis**.

Response objek: `id`, `source`, `nama_barang`, `nama_toko`, `no_telp`,
`no_rekening`, `link_pembelian`, `keterangan`.

---

## Users (admin)

`/api/v1/users` — **khusus admin untuk seluruh operasinya**: daftar user
memuat email & riwayat login, bukan data untuk semua role.

### GET `/api/v1/users/`

Daftar semua user (aktif & nonaktif), tanpa paginasi.

`data`: objek `id`, `username`, `email`, `full_name`, `role`, `is_active`,
`last_login_at`, `created_at`, `updated_at`. `password_hash` tidak pernah
dikirim — sudah disaring di query.

### POST `/api/v1/users/{id}/reset-password`

Menetapkan password baru dan **mencabut semua refresh token** user tersebut —
ia keluar dari seluruh perangkat dan harus login ulang. Access token yang
sudah terbit tetap berlaku sampai kedaluwarsa (maksimal `AUTH_ACCESS_TTL`).

| Field body | Aturan |
|---|---|
| `password` | wajib, minimal **8 karakter**; tidak di-trim (spasi adalah karakter valid) |

`200` → `data`: `id`, `username`, `reset: true`.
`404` bila user tidak ada · `422` bila password kurang dari 8 karakter ·
`403` untuk non-admin.

---

## Payment receipts

Modul inti. Transaksi pembelian: item, qty, satuan, harga, usage.

| Method & path | Role |
|---|---|
| GET `/payment-receipts` | Login |
| GET `/payment-receipts/{id}` | Login |
| GET `/payment-receipts/{id}/pdf` | Login |
| GET `/payment-receipts/{id}/status-history` | Login |
| GET `/payment-receipts/export` | Login |
| POST `/payment-receipts` | Admin, GA Staff |
| PUT `/payment-receipts/{id}` | Admin, GA Staff |
| DELETE `/payment-receipts/{id}` | Admin |
| PATCH `/payment-receipts/{id}/status` | Admin, GA Staff, Approver |
| PATCH `/payment-receipts/bulk-status` | Admin, GA Staff, Approver |

### GET `/payment-receipts` — list

Query:

| Param | Keterangan |
|---|---|
| `status` | `pending` \| `checked` \| `processed` \| `finished` \| `cancelled` (case-insensitive; nilai lain → `422`). |
| `department_id`, `request_id` | Filter numerik. |
| `date_from`, `date_to` | `YYYY-MM-DD`. `date_from > date_to` → `422`. |
| `q` | Pencarian teks. |
| `sort` | `tanggal_asc`, `tanggal_desc` (default), `kode_asc`, `kode_desc`, `total_asc`, `total_desc`. Nilai asing jatuh ke default. |
| `page`, `per_page` | Paginasi. |

Response `data` berupa array `Receipt` (lihat bawah).

### Objek Receipt

```json
{
  "id": 1,
  "kode": "PR-2026-0001",
  "request_id": 3, "request_kode": "REQ-001", "request_nama": "Pembelian ATK",
  "department_id": 1, "department_kode": "GA", "department_nama": "General Affairs",
  "source_id": 2, "source_mode": "online", "source_nama_barang": "Printer",
  "category_id": 1, "category_nama": "ATK",
  "unit_id": 4, "unit_nama": "pcs",
  "tanggal_pembelian": "2026-10-15T00:00:00Z",
  "usage": "Operasional",
  "deskripsi_items": "Printer Epson L3210",
  "jumlah_items": 2,
  "harga_items": "2500000",
  "total_harga": "5000000",
  "price_estimate": "5500000",
  "keterangan": "",
  "status": "pending",
  "input_date": "...",
  "created_at": "...", "updated_at": "..."
}
```

- `total_harga` = `jumlah_items` × `harga_items`, **dihitung di database**
  (generated column) — klien tidak bisa mengirimnya.
- Uang (`harga_items`, `total_harga`, `price_estimate`) diserialisasi sebagai
  **string desimal**, bukan number — menghindari presisi hilang di JavaScript.
- Field `*_nama` / `*_kode` adalah JOIN bawaan, bukan field terpisah yang perlu
  diisi.

### POST `/payment-receipts` & PUT `/{id}`

```json
{
  "request_id": 3,
  "department_id": 1,
  "source_id": 2,
  "category_id": 1,
  "unit_id": 4,
  "tanggal_pembelian": "2026-10-15",
  "usage": "Operasional",
  "deskripsi_items": "Printer Epson L3210",
  "jumlah_items": 2,
  "harga_items": "2500000",
  "price_estimate": "5500000",
  "keterangan": ""
}
```

`source_id`, `category_id`, `unit_id`, `price_estimate` boleh `null`.

Validasi server (semua menghasilkan `422` + `errors[]` per-field):

| Field | Aturan |
|---|---|
| `request_id`, `department_id` | Wajib > 0. |
| `tanggal_pembelian` | Wajib `YYYY-MM-DD`; tidak boleh di masa depan; tidak boleh > 90 hari ke belakang. |
| `deskripsi_items` | Wajib, maks 255 karakter. |
| `jumlah_items` | Wajib > 0. |
| `harga_items` | Wajib, angka > 0. |
| `usage` | Opsional, maks 150 karakter. |
| konsistensi | `request_id` **harus milik** `department_id` yang dikirim (dicek via DB, bukan tebakan klien). |

### PATCH `/payment-receipts/{id}/status`

```json
{ "status": "checked", "note": "Dicek GA" }
```

Transisi harus legal (diagram di atas); selain itu `409`. `note` opsional,
tersimpan di status history.

### PATCH `/payment-receipts/bulk-status`

```json
{ "ids": [1, 2, 3], "status": "processed", "note": "..." }
```

`200` → `data: { "changed": 3 }` — jumlah receipt yang statusnya benar-benar
diubah. Perilaku **all-or-nothing pada tahap validasi**: semua `ids` dicek dulu
(ada semua + transisi legal) sebelum satu pun diubah — satu id tidak valid
membatalkan seluruh permintaan (`404` bila id tidak ada, `409` bila transisi
ilegal, `422` bila `ids` kosong atau `status` tidak dikenal). `changed` bisa
kurang dari `len(ids)` hanya kalau terjadi error database di tengah eksekusi
(partial failure).

### GET `/payment-receipts/{id}/status-history`

Array `StatusHistory` di `data`:

```json
{
  "id": 10, "payment_receipt_id": 1,
  "from_status": "pending", "to_status": "checked",
  "changed_by": 1, "changed_by_nama": "Admin",
  "note": "Dicek GA", "created_at": "..."
}
```

`from_status` bisa `null` bila tidak ada status sebelumnya (kolom nullable;
dalam alur transisi normal, `from_status` selalu terisi status awal receipt).
Entri history hanya dibuat saat **perubahan status** — pembuatan receipt tidak
menghasilkan entri.

### GET `/payment-receipts/{id}/pdf`

Response `application/pdf` — dokumen *Purchase Requisition* berlogo PT. Asi
Pudjiastuti Aviation, `Content-Disposition: inline; filename="Purchase-Requisition-<kode>.pdf"`.

### GET `/payment-receipts/export`

CSV (`text/csv`, `attachment`) dengan kolom:
`Kode, Request Kode, Request Nama, Department Kode, Department Nama, Deskripsi
Items, Jumlah Items, Harga Items, Total Harga, Tanggal Pembelian, Status, Usage,
Kategori, Satuan, Source, Keterangan, Price Estimate`.

Filter sama dengan list (`status`, `department_id`, `request_id`, `date_from`,
`date_to`, `q`), selalu urut `tanggal_desc`. `page`/`per_page` diabaikan —
export selalu menarik hingga **5000 baris** pertama (batas atas ekspor).

---

## AI extract struk

### POST `/api/v1/ai/extract-receipt`

`multipart/form-data` dengan satu field **`file`**:

- Tipe: JPG, PNG, WEBP (ditolak selain itu → `422`).
- Ukuran: batas khusus endpoint ini `AI_MAX_REQUEST_BODY` (default **10 MiB**)
  — endpoint lain memakai `SERVER_MAX_REQUEST_BODY` (default 2 MiB). Melebihi
  batas membuat request ditolak middleware saat dibaca; handler menjawab `422`
  "Berkas terlalu besar".

Respons:

```json
{
  "data": { "...field terstruktur hasil ekstraksi..." },
  "meta": { "fields_missing": ["harga"] }
}
```

`meta.fields_missing` berisi field yang gagal diekstrak supaya frontend bisa
menandainya untuk diisi manual. `AI_API_KEY` belum diatur → `422` ("Fitur AI
belum dikonfigurasi"); provider tidak terpasang sama sekali (bug konfigurasi) →
`500` dengan code `ai.provider_not_set`.

---

## PR Archive

Purchase requisition yang sudah disetujui, siap cetak.

| Method & path | Role |
|---|---|
| GET `/pr-archives` | Login |
| GET `/pr-archives/{id}` | Login |
| GET `/pr-archives/{id}/pdf` | Login |
| POST `/pr-archives` | Admin |
| PUT `/pr-archives/{id}` | Admin |
| DELETE `/pr-archives/{id}` | Admin |
| POST `/pr-archives/bulk-archive` | Admin |

Query list: `q`, `date_from`, `date_to` (format **`YYYY-MM-DD`** — nilai yang
tidak valid diabaikan diam-diam, bukan error), `approver` (id user), `page`,
`per_page`.

Objek response (penting):

```json
{
  "id": 1, "kode": "PR-2026-0001",
  "payment_receipt_id": 1, "receipt_kode": "PR-2026-0001",
  "tanggal_selesai": "...", "nama_request": "...", "manager_approved": "...",
  "approved_by": 2, "approved_by_nama": "Manager",
  "description": "...", "quantity": 2,
  "harga_rencana": "5000000", "harga_aktual": "4800000",
  "created_at": "...", "updated_at": "..."
}
```

### POST `/pr-archives/bulk-archive`

Arsipkan banyak payment receipt sekaligus dalam satu PR:

```json
{
  "kode": "PR-2026-0009",
  "tanggal_selesai": "2026-10-19T00:00:00Z",
  "nama_request": "Pembelian ATK",
  "manager_approved": "Budi",
  "description": "...",
  "harga_rencana": "5000000",
  "payment_receipt_ids": [1, 2, 3]
}
```

Aturan server — seluruh batch divalidasi dalam satu transaksi (`422` bila
melanggar):

- `payment_receipt_ids` wajib berisi minimal satu id.
- `tanggal_selesai` wajib format **RFC3339**.
- **Semua** receipt yang disebut harus `finished` DAN belum pernah diarsipkan
  (`archived_at IS NULL`) — satu saja yang belum (tidak ada, sudah diarsipkan,
  atau statusnya belum `finished`) membatalkan seluruh permintaan.
- `kode` opsional: kosong = digenerate otomatis di database
  (`PR-<tahun>-<n>`, sekuens).
- `approved_by` diisi otomatis dari user yang login (bukan dari body).
- `harga_aktual` dihitung sebagai jumlah `total_harga` semua receipt terpilih;
  `quantity` = jumlah receipt.
- `description` kosong → diisi default `"Arsip bulk dari <N> payment receipt"`.
- Receipt pertama (`payment_receipt_ids[0]`) disimpan sebagai
  `payment_receipt_id` referensi PR; setelah sukses semua receipt ditandai
  `archived_at` (inilah yang membuat receipt kedua kalinya ditolak).

Respons sukses `201`:

```json
{ "data": { "kode": "PR-2026-0009", "status": "archived", "received": 3 } }
```

> Catatan: saat `kode` di-request kosong, field `kode` di respons ini juga
> ikut kosong (kode hasil generate tersimpan di database) — baca ulang lewat
> `GET /pr-archives` untuk mendapatkan kode final.

PDF arsip: `GET /pr-archives/{id}/pdf` (bentuk sama dengan PDF payment
receipt).

---

## Dashboard

Semua butuh login (tanpa pembatasan role khusus).

| Endpoint | Respons `data` |
|---|---|
| GET `/dashboard/summary` | `{ total_requests, total_payment_receipts, total_bases, total_departments }` |
| GET `/dashboard/recent` | Array Receipt — 10 terbaru (`tanggal_desc`). |
| GET `/dashboard/status-breakdown` | Array `{ status, count }`. |
| GET `/dashboard/monthly-spend` | Array 12 bulan terakhir `{ month: "2026-10", label: "Oct 2026", spend: "1500000" }` — bulan kosong diisi `"0"`. |
| GET `/dashboard/top-departments?limit=` | Array `{ id, nama, total }`. `limit` 1–20 (default 5); nilai di luar rentang jatuh ke default. |

---

## Contoh alur lengkap

```bash
BASE=https://receipt.example.com/api/v1

# 1. Login (cookie disimpan ke cookie jar)
curl -c jar.txt -X POST "$BASE/auth/login" \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"rahasia"}'

# 2. List payment receipt pending, halaman 2
curl -b jar.txt "$BASE/payment-receipts?status=pending&page=2&per_page=10&sort=total_desc"

# 3. Buat payment receipt baru
curl -b jar.txt -X POST "$BASE/payment-receipts" \
  -H 'Content-Type: application/json' -H 'X-Request-ID: my-trace-0001' \
  -d '{"request_id":3,"department_id":1,"tanggal_pembelian":"2026-10-19",
       "deskripsi_items":"Kertas A4","jumlah_items":10,"harga_items":"55000"}'

# 4. Cek status → lanjutkan alur
curl -b jar.txt -X PATCH "$BASE/payment-receipts/42/status" \
  -H 'Content-Type: application/json' \
  -d '{"status":"checked","note":"dicek"}'

# 5. Ambil PDF Purchase Requisition
curl -b jar.txt -o pr.pdf "$BASE/payment-receipts/42/pdf"

# 6. Investigasi error: pakai X-Request-ID dari response header
curl -b jar.txt -D - -o /dev/null "$BASE/payment-receipts/42" | grep -i x-request-id
```

---

## Aturan untuk kontributor

- Menambah endpoint baru → update tabel role + contoh di dokumen ini **di
  commit yang sama**.
- Mengubah field response / aturan validasi → perbarui bagian terkait.
- Error selalu lewat `apperror` + `httputil.Problemf` — jangan menulis status
  atau pesan error manual di handler.
- Semua query list wajib lewat `ParsePagination` (jangan menerima `per_page`
  tanpa batas).
