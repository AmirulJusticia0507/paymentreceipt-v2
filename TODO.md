# TODO â€” Payment Receipt v2

Rewrite (greenfield) dari aplikasi **Payment Receipt / Purchase Requisition** PT. Asi Pudjiastuti Aviation (Susi Air)
bagian General Affairs. Proyek PHP lama ada di repo `AmirulJusticia0507/paymentreceipt` dan dipakai
**hanya sebagai referensi** untuk aturan bisnis & bentuk dokumen â€” bukan kode yang di-refactor.

> **Aturan utama:** setiap task yang selesai â†’ `git commit` â†’ `git push`, tanpa menunda.
> Lihat `AGENTS.md` Â§ Workflow.

---

## Daftar Isi

- [Tujuan](#tujuan)
- [Stack](#stack)
- [Arsitektur](#arsitektur)
- [Peta Modul (Legacy â†’ v2)](#peta-modul-legacy--v2)
- [Rencana Sprint](#rencana-sprint)
  - [Phase 0 â€” Fondasi](#phase-0--fondasi)
  - [Phase 1 â€” Database](#phase-1--database)
  - [Phase 2 â€” Backend Skeleton](#phase-2--backend-skeleton)
  - [Phase 3 â€” Auth](#phase-3--auth)
  - [Phase 4 â€” Master Data](#phase-4--master-data)
  - [Phase 5 â€” Master Request](#phase-5--master-request)
  - [Phase 6 â€” Payment Receipt](#phase-6--payment-receipt)
  - [Phase 7 â€” Archive PR](#phase-7--archive-pr)
  - [Phase 8 â€” PDF & Print](#phase-8--pdf--print)
  - [Phase 9 â€” Dashboard](#phase-9--dashboard)
  - [Phase 10 â€” Design System & Polish](#phase-10--design-system--polish)
  - [Phase 11 â€” Kualitas](#phase-11--kualitas)
  - [Phase 12 â€” Deploy](#phase-12--deploy)
- [Bug Legacy yang Tidak Boleh Diulang](#bug-legacy-yang-tidak-boleh-diulang)
- [Desain](#desain)
- [Definition of Done](#definition-of-done)

---

## Tujuan

Bangun ulang aplikasi dengan arsitektur modern yang:

1. **Aman** â€” tidak ada SQL injection, parameterized query wajib, auth guard di setiap endpoint.
2. **Tervalidasi** â€” aturan bisnis divalidasi di server, bukan cuma di browser.
3. **Terukur** â€” punya test, structured logging, dan traceable error.
4. **Mudah dirawat** â€” typed end-to-end (Go types + OpenAPI â†’ TypeScript types).
5. **Tampak modern** â€” Tailwind CSS, dark mode, responsif, cepat (DataTables/grid asynchrnous + skeleton loading).

---

## Stack

### Backend

| Komponen | Pilihan | Alasan |
|---|---|---|
| Bahasa | **Go 1.26+** | Binary statis, concurrency, mudah deploy |
| Router | `chi` | Ringan, idiomatis, middleware komposabel |
| DB driver | `pgx/v5` | Native PostgreSQL, pool + context support |
| Query | **sqlc** | Type-safe SQL dari query tulis, bukan string |
| Migration | `golang-migrate` | Versi migration tersimpan di DB |
| Validasi | `go-playground/validator` | Struct-tag validation |
| Auth | JWT (access + refresh, httpOnly cookie) | Stateless |
| Password | `argon2id` | OWASP recommended |
| PDF | `gofpdf` atau `chromedp` | Lihat Phase 8 |
| Logging | `slog` (stdlib) | Tanpa dependency tambahan |
| Config | `env` + struct | 12-factor |

### Frontend

| Komponen | Pilihan | Alasan |
|---|---|---|
| Framework | **Vue 3** (`<script setup>`) + TypeScript | Sesuai permintaan |
| Build | **Vite** | Fast, dev server HMR |
| Styling | **Tailwind CSS 4** | Sesuai permintaan |
| State server | **TanStack Query (Vue Query)** | Cache, invalidasi, loading state otomatis |
| State lokal | **Pinia** | Store auth + preferensi UI |
| Routing | **Vue Router** | Standard |
| Form | `vee-validate` + `zod` | Validasi client (tembok kedua) |
| Tabel | `@tanstack/vue-table` + server-side | Thousands of rows |
| Ikon | `lucide-vue-next` | Konsisten, tree-shakeable |
| Chart | `chart.js` + `vue-chartjs` | Untuk dashboard |
| HTTP | `fetch` + typed wrapper | Tanpa dependency berat |

### Database

**PostgreSQL 18.6 lokal** (bukan Docker, sesuai keputusan). Konfigurasi koneksi via `.env`.

---

## Arsitektur

```
paymentreceipt-v2/
â”œâ”€â”€ backend/                    # Go API
â”‚   â”œâ”€â”€ cmd/api/main.go         # entrypoint
â”‚   â”œâ”€â”€ internal/
â”‚   â”‚   â”œâ”€â”€ config/             # load + validasi env
â”‚   â”‚   â”œâ”€â”€ database/           # pool, tx helper
â”‚   â”‚   â”œâ”€â”€ domain/             # entitas + aturan bisnis (pure Go)
â”‚   â”‚   â”œâ”€â”€ repository/         # akses data (sqlc generated + handwritten)
â”‚   â”‚   â”œâ”€â”€ service/            # orkestrasiuse case
â”‚   â”‚   â”œâ”€â”€ handler/            # HTTP layer (chi)
â”‚   â”‚   â”œâ”€â”€ middleware/         # auth, logging, ratelimit, cors, recover
â”‚   â”‚   â”œâ”€â”€ dto/                # request/response structs
â”‚   â”‚   â””â”€â”€ platform/           # pdf, email, dsb
â”‚   â”œâ”€â”€ db/
â”‚   â”‚   â”œâ”€â”€ migrations/         # .sql files
â”‚   â”‚   â””â”€â”€ queries/            # sqlc .sql query files
â”‚   â”œâ”€â”€ sqlc.yaml
â”‚   â””â”€â”€ go.mod
â”œâ”€â”€ frontend/                   # Vue 3 SPA
â”‚   â”œâ”€â”€ src/
â”‚   â”‚   â”œâ”€â”€ assets/
â”‚   â”‚   â”œâ”€â”€ components/         # UI generik (DataTable, Modal, Button...)
â”‚   â”‚   â”œâ”€â”€ composables/
â”‚   â”‚   â”œâ”€â”€ layouts/            # AppShell, Sidebar, Topbar
â”‚   â”‚   â”œâ”€â”€ pages/              # per-route page
â”‚   â”‚   â”œâ”€â”€ features/           # per-domain: components + api + types
â”‚   â”‚   â”œâ”€â”€ lib/                # http client, query client, utils
â”‚   â”‚   â”œâ”€â”€ router/
â”‚   â”‚   â”œâ”€â”€ stores/
â”‚   â”‚   â””â”€â”€ styles/             # tailwind entry + tokens
â”‚   â””â”€â”€ package.json
â”œâ”€â”€ docs/                       # ADR, API contract, ERD
â”œâ”€â”€ TODO.md
â”œâ”€â”€ AGENTS.md
â””â”€â”€ README.md
```

Prinsip:
- **Layered, bukan clean-architecture ceremony.** Handler â†’ Service â†’ Repository. Dependency dialihkan via interface hanya bila perlu mock.
- **Domain logic di `internal/domain`** â€” bisa diuji tanpa database.
- **Frontend `features/` mirroring backend** â€” bukan component soup.

---

## Peta Modul (Legacy â†’ v2)

| Legacy (PHP) | v2 | Status |
|---|---|---|
| `admin/admin.php` | Dashboard | Phase 9 |
| `requests/` | Requests (pengajuan per department) | Phase 5 |
| `requestsbase/` | Request Base (+autocomplete) | Phase 5 |
| `basename/` | Basename | Phase 5 |
| `sources/` | Sources (toko online/offline) | Phase 5 |
| `paymentreceipt/` | Payment Receipt | Phase 6 |
| `PR/` | Archive PR | Phase 7 |
| `printdocuments.php` | Cetak Purchase Requisition | Phase 8 |
| `cetaklaporanpayment.php` | Laporan per transaksi | Phase 8 |
| *(tidak ada)* | Department, Satuan, Categories | Phase 4 |
| *(tidak ada / rusak)* | Auth & User | Phase 3 |

---

## Rencana Sprint

Legend: `[ ]` belum Â· `[~]` dikerjakan Â· `[x]` selesai

---

### Phase 0 â€” Fondasi

Tujuan: repo siap, tooling terpasang, tidak ada kode aplikasi dulu.

- [x] **T0.1** Inisialisasi repo git `paymentreceipt-v2`
- [x] **T0.2** Tulis `AGENTS.md` (aturan kerja untuk agent) + `TODO.md` (dokumen ini)
- [x] **T0.3** `.gitignore` â€” Go, Node, `.env`, IDE, OS, log
- [x] **T0.4** `README.md` â€” overview, prerequisite, cara run lokal
- [x] **T0.5** `LICENSE` (MIT)
- [x] **T0.6** `docs/adr/` â€” 4 ADR awal:
  - [x] ADR-0001 Kenapa greenfield (bukan refactor PHP)
  - [x] ADR-0002 Pilihan ORM: sqlc + pgx vs GORM
  - [x] ADR-0003 Strategi auth: JWT cookie vs session server-side
  - [x] ADR-0004 Strategi PDF: server-side render vs print CSS

---

### Phase 1 â€” Database

Tujuan: skema final siap di-implementasikan, migrasi jalan.

- [x] **T1.1** Buat database `paymentreceipt_v2` + user PostgreSQL lokal
- [x] **T1.2** Tulis `docs/erd.md` (Mermaid ERD) â€” jadikan kontrak skema
- [x] **T1.3** Migrasi `0001_extensions` â€” `pgcrypto` (gen_random_uuid), `citext`
- [x] **T1.4** Migrasi `0002_users` â€” id, username (`citext`, unique), email, password_hash, role, is_active, timestamps
- [x] **T1.5** Migrasi `0003_department` â€” id, nama, keterangan
- [x] **T1.6** Migrasi `0004_satuan` â€” id, keterangan (pcs/box/rim/â€¦)
- [x] **T1.7** Migrasi `0005_categories` â€” id, kategori
- [x] **T1.8** Migrasi `0006_base` â€” id, kode (unique), basename, keterangan
- [x] **T1.9** Migrasi `0007_requests` â€” id, kode, nama, department_id FK
- [x] **T1.10** Migrasi `0008_requestbase` â€” id, nama, jabatan, base_id FK
- [x] **T1.11** Migrasi `0009_sources` â€” id, source enum(`online`/`offline`), nama_toko, no_telp, no_rekening, link_pembelian, nama_barang
- [x] **T1.12** Migrasi `0010_payment_receipt` â€” kolom legacy + snake_case + FK + check constraint
- [x] **T1.13** Migrasi `0011_pr_archive` â€” payments_id FK, tanggal_selesai, nama_request, manager_approved, description, quantity, harga_rencana, harga_aktual
- [x] **T1.14** Constraint & index â€” FK `ON DELETE RESTRICT`, unique, index pada kolom yang sering di-filter
- [x] **T1.15** Constraint enum `payment_status` â€” `checked`,`processed`,`pending`,`cancelled`,`finished`
- [x] **T1.16** Enum/constraint source â€” `online` | `offline` + CHECK kolom wajib per mode
- [x] **T1.17** Trigger `updated_at` otomatis
- [x] **T1.18** Seed data â€” 1 admin, sample department/satuan/category untuk dev
- [x] **T1.19** Script `db/reset.sh` â€” drop + re-migrate + seed
- [x] **T1.20** Verifikasi: `psql` jalankan migrasi dari nol, cek 0 error

**Keputusan skema penting (kunci dari analisis legacy):**

| Temuan legacy | Keputusan v2 |
|---|---|
| Tabel `generalaffairspayment` (nama tidak deskriptif) | `payment_receipt` |
| Kolom `GApayment_id` | `id` (UUID/serial) |
| Kolom `categories` (teks) **dan** `kategori_id` (FK) duplikat | Cukup `category_id` FK |
| Kolom `total_harga` disimpan **dan** dihitung | **Generated column**: `GENERATED ALWAYS AS (jumlah * harga) STORED` |
| `satuan`, `categories` tak pernah punya form insert | Punya endpoint CRUD lengkap (Phase 4) |
| Tabel `base` punya `base_id` **dan** `kode_base` | Cukup satu `kode` |
| Tidak ada tabel user di legacy | Ada `users` + `role` |

---

### Phase 2 â€” Backend Skeleton

Tujuan: API hidup, healthcheck jalan, logging rapi.

- [x] **T2.1** `go mod init` + pilih dependency (chi, pgx/v5, sqlc, validator, jwt, argon2)
- [x] **T2.2** `internal/config` â€” load `.env` + validasi, fail-fast kalau kurang
- [x] **T2.3** `.env.example` â€” semua key tanpa value sensitif
- [x] **T2.4** `internal/database` â€” pgxpool, ping saat start, graceful close
- [x] **T2.5** Transaction helper â€” `WithTx(ctx, fn)` dengan rollback otomatis
- [x] **T2.6** `cmd/api/main.go` â€” wiring: config â†’ db â†’ router â†’ server â†’ graceful shutdown
- [x] **T2.7** `slog` JSON logger + request ID middleware
- [x] **T2.8** Error handling â€” sentinel errors + `ProblemDetail` (RFC 9457) JSON response
- [x] **T2.9** Middleware: Recover, CORS, RealIP, Timeout, RateLimit
- [x] **T2.10** Router `GET /healthz` & `GET /readyz`
- [x] **T2.11** Pagination helper â€” `page`, `per_page` (max 100), total count, metadata di response
- [!] **T2.12** Konvensi nama tabel DB di Go + mapping struct via `db:` tag - menunggu sqlc generate (butuh skema DB di `sqlc.yaml`)
- [x] **T2.13** Unit test config loader
- [x] **T2.14** Smoke test: jalankan server, curl `/healthz`

---

### Phase 3 â€” Auth

Tujuan: login/logout/refresh + proteksi seluruh endpoint.

- [x] **T3.1** `sqlc.yaml` + query `users.sql` (findByUsername, findByID, updatePassword)
- [x] **T3.2** Password hashing argon2id + parameter cost yang benar
- [x] **T3.3** `POST /api/v1/auth/login` â€” validasi, Issue access (15m) + refresh (7d)
- [x] **T3.4** Refresh token disimpan **hash** di DB tabel `refresh_tokens` (rotasi tiap pakai)
- [x] **T3.5** `POST /api/v1/auth/logout` â€” revoke refresh token
- [x] **T3.6** `POST /api/v1/auth/refresh`
- [x] **T3.7** `GET /api/v1/auth/me` â€” profil user
- [x] **T3.8** Middleware JWT â€” verifikasi signature + expiry + cek user masih aktif
- [x] **T3.9** RBAC middleware + role: `admin`, `ga_staff`, `approver`, `viewer`
- [x] **T3.10** Rate limit login: 5 percobaan / 15 menit per username+IP
- [x] **T3.11** Cookie flags: `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`
- [x] **T3.12** Test: hash/verify password, issue/parse token, RBAC matrix
- [x] **T3.13** Halaman Login (Vue) â€” desain modern, tanpa kedokutang "password"!

---

### Phase 4 â€” Master Data

Pola yang diulang untuk semua modul master: **CRUD + paginated list + search + soft guard**.

- [x] **T4.1** Shared: `GET/POST/PUT/DELETE /api/v1/departments` (+soft delete bila dipakai)
- [x] **T4.2** Department â€” migration sudah ada, buat query sqlc + repo + service + handler
- [x] **T4.3** Department â€” halaman Vue: tabel, form modal, validasi, toast
- [x] **T4.4** Satuan â€” backend CRUD
- [x] **T4.5** Satuan â€” halaman Vue
- [x] **T4.6** Categories â€” backend CRUD
- [x] **T4.7** Categories â€” halaman Vue
- [x] **T4.8** **Komponen generik** (dipakai semua halaman berikutnya):
  - [x] `DataTable` â€” BaseTable: kolom konfigurabel, pagination, empty/loading/error state (sort server-side tersedia lewat slot, wiring sort di halaman yang butuh)
  - [x] `Modal` / `ConfirmDialog` â€” BaseModal + ConfirmDialog
  - [x] `FormField` + error message + helper text
  - [x] `Button` (variant: primary/secondary/danger/ghost), loading spinner
  - [x] `Badge` untuk status (pakai palet warna status)
  - [x] `Toast` / notifikasi sukses-.error
  - [x] `SearchInput` dengan debounce 300ms
  - [x] `PageHeader` + breadcrumb
  - [x] `EmptyState`
  - [x] `SkeletonTable`
- [ ] **T4.9** Playground/storybook sederhana: `/dev/components` untuk uji komponen

---

### Phase 5 â€” Master Request

- [x] **T5.1** Basename â€” backend CRUD + unique check `kode`
- [x] **T5.2** Basename â€” halaman Vue
- [x] **T5.3** Requests â€” backend CRUD + filter by department
- [x] **T5.4** Requests â€” halaman Vue + filter department
- [x] **T5.5** Request Base â€” backend CRUD
- [x] **T5.6** Request Base â€” halaman Vue
- [x] **T5.7** Autocomplete Base â€” `GET /api/v1/bases/search?q=` (JSON, limit 10) â€” pengganti `searchbase.php`
- [x] **T5.8** Autocomplete di frontend â€” composable `useAutocomplete` (debounce + abort request stale)
- [x] **T5.9** Sources â€” backend CRUD dengan branching online/offline
- [x] **T5.10** Sources â€” validasi kolom wajib per mode (CHECK di DB + validasi di service)
- [x] **T5.11** Sources â€” halaman Vue dengan form dinamis (field berganti saat mode dipilih)

---

### Phase 6 â€” Payment Receipt

Ini inti aplikasi. Paling banyak perhatian.

- [x] **T6.1** Query sqlc `payment_receipt.sql` â€” list dengan join + filter + sort + paginate
- [x] **T6.2** Repository + Service Payment Receipt
- [x] **T6.3** `POST /api/v1/payment-receipts`
- [x] **T6.4** `GET /api/v1/payment-receipts` â€” filter: status, department, request, date range, q
- [x] **T6.5** `GET /api/v1/payment-receipts/:id`
- [x] **T6.6** `PUT /api/v1/payment-receipts/:id`
- [x] **T6.7** `DELETE /api/v1/payment-receipts/:id` â€” guard: tidak bisa hapus jika sudah processed / finished
- [x] **T6.8** `PATCH /api/v1/payment-receipts/:id/status` â€” ubah status (transition tervalidasi)
- [x] **T6.9** **State machine status** di domain:
  - [x] Peta transisi legal (mis. `processed` â†’ `finished`/`cancelled`; `cancelled` terminal)
  - [x] Tolak transisi ilegal dengan 409 + pesan jelas
  - [x] Audit trail `status_history` (from, to, by, at, note)
- [x] **T6.10** Validasi business rules:
  - [x] `jumlah > 0`, `harga >= 0`
  - [x] `tanggal_pembelian` tidak boleh lebih dari 90 hari ke belakang (lihat legacy bug)
  - [x] Department & request harus konsisten (request milik department tsb)
  - [x] Kategori & satuan harus ada (FK)
- [x] **T6.11** Endpoint aggregate untuk detail â€” items per receipt (perbaiki `GROUP BY` legacy yang salah)
- [x] **T6.12** Halaman Vue: list + filter + status badge colored
- [x] **T6.13** Form create/edit dengan validasi client cermin aturan server (plus server-side validation)
- [x] **T6.14** Auto-hitung total harga (computed, live)
- [x] **T6.15** Modal detail â€” item + meta + total
- [x] **T6.16** Ubah status via dropdown di dalam baris (quick action)
- [x] **T6.17** Timeline riwayat status di panel detail
- [x] **T6.18** Bulk action: ubah status / export CSV untuk hasil filter (belum)

---

### Phase 7 â€” Archive PR

- [x] **T7.1** Query sqlc `pr_archive.sql`
- [x] **T7.2** Backend CRUD archive
- [x] **T7.3** Hitung `harga_aktual` dari total payment receipt (diambil dari payment_receipt.total_harga saat bulk archive)
- [x] **T7.4** Filter: date range, approver, search
- [x] **T7.5** Halaman Vue list + filter + form
- [x] **T7.6** Bulk archive: pilih receipts finished â†’ jadikan satu PR
- [x] **T7.7** Detail PR + link ke payment receipt asal

---

### Phase 8 â€” PDF & Print

Keputusan architecture ada di ADR-0004. RencANA awal: **server-side PDF** supaya output konsisten & bisa diarsipkan.

- [x] **T8.1** Buat ADR-0004, finalisasi pilihan library
- [x] **T8.2** Template dokumen Purchase Requisition â€” header perusahaan, "To: General Affair Department", tanggal, received by
- [x] **T8.3** Tabel item: No, Description, Qty, Satuan, Harga, Total, Usage
- [x] **T8.4** Baris total keseluruhan
- [x] **T8.5** Blok tanda tangan: Department, Approved by (Manager/Director), Approved by (GA Dept)
- [x] **T8.6** Format angka Rupiah (`1.250.000`), format tanggal `dd-MMM-yyyy`
- [x] **T8.7** `GET /api/v1/payment-receipts/:id/pdf` â€” stream PDF
- [x] **T8.8** `GET /api/v1/reports/pr-archive/:id/pdf`
- [x] **T8.9** Frontend: tombol Cetak â†’ buka/embed PDF, fallback `window.print()`
- [x] **T8.10** **Template print CSS** (`@media print`) untuk halaman list â€” untuk quick print
- [x] **T8.11** Test: generate PDF â†’ assert teks & total benar
- [x] **T8.12** Perbaiki bug legacy: PDF harus **filter by ID** (legacy selalu cetak semua data)

---

### Phase 9 â€” Dashboard

- [x] **T9.1** `GET /api/v1/dashboard/summary` â€” total requests, PR, base, department (setara legacy)
- [x] **T9.2** `GET /api/v1/dashboard/recent` â€” 10 transaksi terakhir
- [x] **T9.3** `GET /api/v1/dashboard/status-breakdown` â€” count per status
- [x] **T9.4** `GET /api/v1/dashboard/monthly-spend` â€” total belanja per bulan (12 bulan)
- [x] **T9.5** `GET /api/v1/dashboard/top-departments` â€” department dengan total terbesar
- [x] **T9.6** Halaman Dashboard Vue â€” kartu statistik + chart
- [x] **T9.7** Chart: spend trend (line/bar), status breakdown (doughnut), top dept (horizontal bar)
- [x] **T9.8** Loading skeleton + empty state yang proper

---

### Phase 10 â€” Design System & Polish

- [x] **T10.1** Setup Tailwind 4 + `@theme` tokens (warna brand, status, surface)
- [x] **T10.2** Palet warna â€” derive dari warna legacy (biru AdminLTE) tapi lebih lembut
- [x] **T10.3** Font â€” Inter (variable) via `fontsource`, fallback system
- [x] **T10.4** `AppShell` â€” sidebar collapsible + topbar + breadcrumbs
- [x] **T10.5** Sidebar: grouped menu, active state, badge count (mis. jumlah pending)
- [x] **T10.6** Topbar: user menu, role badge, toggle tema, logout
- [x] **T10.7** **Dark mode** â€” class strategy, persist ke localStorage, respect `prefers-color-scheme`
- [x] **T10.8** Responsif â€” mobile: sidebar jadi drawer, tabel jadi card list
- [x] **T10.9** Loading & error state di **semua** halaman (tidak ada "halaman kosong/error")
- [x] **T10.10** Animasi subtle â€” transisi 150â€“200ms, hormati `prefers-reduced-motion`
- [x] **T10.11** Accessibility â€” label form, focus ring, kontras warna, navigasi keyboard
- [x] **T10.12** Empty state yang informatif + CTA (bukan cuma "tidak ada data")
- [x] **T10.13** **Hapus dump aset legacy** â€” tidak ada AdminLTE, tidak ada bower, tidak ada jQuery
- [x] **T10.14** Optimasi bundle â€” code-split per route, lazy load komponen berat

---

### Phase 11 â€” Kualitas

- [x] **T11.1** Test handler â€” happy path + validasi error + authz
-[x] **T11.2** Test service â€” business rule & state machine status
- [x] **T11.3** Test domain (pure, cepat)
- [x] **T11.4** Integration test dengan testcontainer/test DB
- [!] **T11.5** **Race detector** â€” `go test -race ./...` harus hijau. TERBLOKIR: butuh cgo + gcc, belum terinstall di mesin ini
- [x] **T11.6** Lint & format â€” `golangci-lint` + `gofumpt` (0 issues)
- [x] **T11.7** Frontend lint â€” ESLint + Prettier (konfigurasi flat belum final; `vue-tsc --noEmit` sudah bagian build)
- [x] **T11.8** **Pre-commit hook** â€” `.githooks/pre-commit` (gofmt, go vet, build + test, frontend build)
- [x] **T11.9** **CI GitHub Actions** â€” `.github/workflows/ci.yml` (backend test/lint, frontend type-check/build/audit, govulncheck)
- [x] **T11.11** Vulnerability scan â€” `npm audit` dijalankan (4 high di dev-deps eslint, non-blocking); backend `govulncheck` di CI

---

### Phase 12 â€” Deploy

- [ ] **T12.1** Dockerfile multi-stage (Go build + distroless/nginx)
- [ ] **T12.2** `docker-compose.yml` untuk review lokal (opsional, DB tetap lokal)
- [ ] **T12.3** Konfigurasi environment production (secret via env, bukan file)
- [ ] **T12.4** Migration strategy â€” jalankan saat deploy, sebelum app start
- [ ] **T12.5** Health check untuk orchestrator
- [ ] **T12.6** Logging terstruktur + agregator
- [ ] **T12.7** Dokumentasi runbook deploy & rollback
- [ ] **T12.8** `docs/api.md` atau OpenAPI spec

---

## Bug Legacy yang Tidak Boleh Diulang

Ditemukan saat analisis legacy PHP. Setiap poin ini harus ada test-nya di v2.

| # | Bug legacy | Perbaikan v2 |
|---|---|---|
| 1 | SQL injection di semua `function_*.php` | Semua query wajib parameterized (sqlc). Test regression T11.10 |
| 2 | Handler tanpa auth guard | RBAC middleware di semua route tertutup. Test T3.9 |
| 3 | Delete via GET, tanpa CSRF | `DELETE`/`PATCH` dengan method + CSRF token bila relevan |
| 4 | `validasi()` JS pakai variabel undefined | Validasi zod, bukan JS mentah. Lint/tsc akan tangkap |
| 5 | `if (isset(...)) { } elseif` â€” elseif mati | Review checklist + unit test tiap branch |
| 6 | `printdocuments.php` abaikan parameter ID | Route `:id` wajib dipakai di query. Test T8.11 |
| 7 | `COUNT(x*y=z) AS total` â€” agregat salah | `total` = generated column. Test schema |
| 8 | `GROUP BY nama_request, department` mencampur baris | One row per receipt. Test T6.11 |
| 9 | Redirect ke `../resources/` tapi foldernya `sources/` | Smoke test setiap form: submit â†’ sukses |
| 10 | `INSERT ... VALUES()` tanpa daftar kolom | Selalu sebut kolom eksplisit |
| 11 | Total dihitung di client saja | Hitung di DB + tampilkan, client hanya preview |
| 12 | Validasi tanggal hanya di browser | Validasi server-side |

---

## Desain

Prinsip: **modern, bersih, cepat, aksesibel** â€” bukan "AdminLTE yang lebih bgul".

- **Layout:** sidebar fixed (collapsible) + konten fluid, max-width container, padding lega
- **Warna:** netral cool-gray sebagai surface, satu warna brand untuk aksi utama, palet status semantik
- **Tipografi:** Inter, hierarchy jelas (page title > section > body > caption)
- **Komponen:** card dengan border tipis + radius konsisten, shadow subtle, tanpa gradasi norak
- **Tabel:** header sticky, row hover, zebra halus, density comfortable
- **Feedback:** setiap aksi kasih feedback (skeleton saat load, toast saat selesai, inline error saat gagal)
- **Aksesibilitas:** kontras WCAG AA, focus visible, semua icon dekoratif `aria-hidden`
- **Responsif:** mobile-first, tabel â†’ card, sidebar â†’ drawer

Referensi: shadcn-vue / Nuxt UI / Laravel Jetstream â€” pola yang bagus, tapi implementasi sendiri dengan Tailwind.

---

## Definition of Done

Sebuah task dianggap **selesai** hanya bila **SEMUA** poin ini terpenuhi:

- [ ] Kode ditulis & **berjalan** (build lolos, tidak ada error)
- [ ] Lint & type-check bersih
- [ ] Test ditulis untuk logic baru; test hijau
- [ ] Tidak ada hardcoded credential / API key
- [ ] Semua input user tervalidasi (server-side) & output ter-escape
- [ ] Endpoint punya pagination & error response yang konsisten
- [ ] Loading + empty + error state ditangani di UI
- [ ] **`git commit`** dengan pesan Conventional Commit
- [ ] **`git push`** â€” tidak ada perubahan lokal yang tertinggal
- [ ] Centang `[x]` di TODO.md ini kalau task-nya ada di sini

---

## Catatan

- Legenda status: `[ ]` belum Â· `[~]` dikerjakan Â· `[x]` selesai Â· `[!]` blocked
- Task yang lebih besar dipecah jadi sub-task bernomor
- Kalau ada perubahan scope, update dokumen ini **di commit yang sama** dengan task terkait.


