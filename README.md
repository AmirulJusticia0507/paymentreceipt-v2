# Payment Receipt v2

Sistem pengajuan dan verifikasi pembelian (*Purchase Requisition* / *Payment Receipt*)
untuk bagian **General Affairs** — rewrite dari aplikasi PHP legacy
([`paymentreceipt`](../paymentreceipt)) dengan arsitektur modern.

**Stack:** Go 1.26 · PostgreSQL 18 · Vue 3 · Tailwind CSS 4

---

## Mengapa Rewrite?

Aplikasi legacy (PHP + MySQL) rewritten, bukan di-refactor, karena:

- **Tidak aman** — SQL injection di seluruh handler, tidak ada auth guard, delete via GET
- **Tidak lengkap** — file inti (`koneksidb.php`, `template/`) hilang, tidak bisa dijalankan
- **Tidak terukur** — tidak ada test, tidak ada validasi server-side, tidak ada CI
- **Sulit dirawat** — 54 MB aset bower/AdminLTE, query bertumpuk, logika bercabang di view

Detail lengkap temuan ada di [TODO.md § Bug Legacy yang Tidak Boleh Diulang](TODO.md#bug-legacy-yang-tidak-boleh-diulang).

Repo legacy tetap dipakai **hanya sebagai referensi aturan bisnis dan bentuk dokumen** — bukan
kode yang di-port.

---

## Prasyarat

| Kebutuhan | Versi | Cara cek |
|---|---|---|
| Go | 1.26+ | `go version` |
| Node.js | 20+ | `node --version` |
| PostgreSQL | 16+ (18 direkomendasikan) | `psql --version` |
| Git | 2.40+ | `git --version` |

Belum punya? — [go.dev](https://go.dev/dl/) · [nodejs.org](https://nodejs.org) · [postgresql.org](https://www.postgresql.org/download/)

---

## Menjalankan Secara Lokal

### 1. Clone

```bash
git clone https://github.com/AmirulJusticia0507/paymentreceipt-v2.git
cd paymentreceipt-v2
```

### 2. Setup database

```bash
# buat database
createdb -U postgres paymentreceipt_v2

# jalankan migrasi (dari root repo; memakai migrator bawaan repo)
cd backend
DB_URL="postgres://postgres@localhost:5432/paymentreceipt_v2?sslmode=disable" go run ./cmd/migrate up
cd ..

# isi data awal
psql -U postgres -d paymentreceipt_v2 -f backend/db/seeds/development.sql
```

Alternatif: `./scripts/db-reset.sh` untuk drop + migrate + seed dari nol.

### 3. Backend

```bash
cd backend
cp .env.example .env        # sesuaikan kredensial DB
go run ./cmd/api
```

Server listen di `http://localhost:8080`. Cek `http://localhost:8080/healthz`.

### 4. Frontend

```bash
cd frontend
npm install
npm run dev
```

Dev server di `http://localhost:5173` dengan proxy ke backend.

### 5. Login

```bash
# user seed dari development.sql
username: admin
password: admin123
```

> **Ganti password ini sebelum deploy ke mana pun.** User seed hanya untuk lokal.

---

## Struktur Repo

```
paymentreceipt-v2/
├── backend/              # Go API
│   ├── cmd/api/          # entrypoint
│   ├── internal/
│   │   ├── config/       #   load & validasi env
│   │   ├── database/     #   pool, transaction helper
│   │   ├── domain/       #   entitas + aturan bisnis (pure Go)
│   │   ├── repository/   #   akses data (sqlc)
│   │   ├── service/      #   orkestrasi use case
│   │   ├── handler/      #   HTTP layer (chi)
│   │   ├── middleware/   #   auth, logging, ratelimit
│   │   ├── dto/          #   request/response structs
│   │   └── platform/     #   pdf, dsb
│   └── db/
│       ├── migrations/   #   .sql
│       ├── queries/      #   .sql (sqlc)
│       └── seeds/
├── frontend/             # Vue 3 SPA
│   └── src/
│       ├── components/   #   generik & reusable
│       ├── features/     #   per-domain
│       ├── layouts/      #   AppShell, Sidebar, Topbar
│       ├── pages/
│       └── styles/       #   tailwind entry + tokens
├── docs/                 # ADR, ERD
├── scripts/              # db-reset, seed
├── TODO.md               # roadmap
└── AGENTS.md             # konvensi & workflow
```

Arsitektur lengkap: [TODO.md § Arsitektur](TODO.md#arsitektur)

---

## Modul

| Modul | Deskripsi |
|---|---|
| **Dashboard** | Statistik, trend belanja, breakdown status |
| **Department** | Unit kerja pemohon |
| **Request** | Pengajuan milik satu department |
| **Base / Request Base** | Kategori pengajuan umum dan turunannya |
| **Sources** | Toko/sumber pembelian — mode `online` atau `offline` |
| **Payment Receipt** | Transaksi pembelian + alur status |
| **PR Archive** | Purchase requisition yang sudah disetujui |
| **Cetak PDF** | Dokumen Purchase Requisition |

---

## Alur Status Payment Receipt

```
pending ──→ checked ──→ processed ──→ finished
               │            │
               └──→ cancelled (terminal)
```

Transisi ilegal ditolak dengan `409 Conflict`. Riwayat perubahan status disimpan di
`status_history`.

---

## Endpoint Dasar

Semua endpoint memakai prefix `/api/v1`. Endpoint terlindungi memakai JWT pada cookie
`HttpOnly`.

| Method | Path | Auth |
|---|---|---|
| `GET` | `/healthz` | — |
| `GET` | `/readyz` | — |
| `POST` | `/api/v1/auth/login` | — |
| `POST` | `/api/v1/auth/refresh` | refresh token |
| `POST` | `/api/v1/auth/logout` | ya |
| `GET` | `/api/v1/auth/me` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/departments` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/satuan` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/categories` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/bases` | ya |
| `GET` | `/api/v1/bases/search?q=` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/requests` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/request-bases` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/sources` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/payment-receipts` | ya |
| `PATCH` | `/api/v1/payment-receipts/:id/status` | ya |
| `GET` | `/api/v1/payment-receipts/:id/pdf` | ya |
| `GET/POST/PUT/DELETE` | `/api/v1/pr-archive` | ya |

Error response mengikuti [RFC 9457 (Problem Details for HTTP APIs)](https://www.rfc-editor.org/rfc/rfc9457).

---

## Command Umum

```bash
# Dua-duanya sekaligus (butuh GNU make + sh: Git Bash/WSL;
# backend/.env dimuat otomatis oleh target ini)
make run

# Backend
cd backend
go build ./...          # build
go test ./...           # test
go test -race ./...     # test + race detector
golangci-lint run       # lint
sqlc generate           # regenerate kode dari db/queries

# Frontend
cd frontend
npm run dev             # dev server
npm run lint            # eslint
npm run type-check      # vue-tsc
npm run build           # production build

# Database
./scripts/db-reset.sh                                   # drop + migrate + seed
cd backend && DB_URL=... go run ./cmd/migrate up        # migrasi saja
cd backend && DB_URL=... go run ./cmd/migrate version   # cek versi skema
cd backend && DB_URL=... go run ./cmd/migrate down -steps 1   # rollback 1 versi
```

Deploy, rollback, dan troubleshooting produksi: [docs/runbook-deploy.md](docs/runbook-deploy.md).
Skema log & agregator: [docs/logging.md](docs/logging.md).
Referensi endpoint API: [docs/api.md](docs/api.md).
Petunjuk pengguna: [docs/user-guide.md](docs/user-guide.md).
Kebijakan internal: [docs/confidentiality.md](docs/confidentiality.md) — privasi: [docs/privacy-policy.md](docs/privacy-policy.md).

---

## Kontribusi

Baca **[AGENTS.md](AGENTS.md)** terlebih dahulu. Aturan utamanya:

> Satu task = satu commit = satu push. Jangan kumpulkan perubahan di akhir sesi.

Semua perubahan harus lewat Conventional Commits, query wajib parameterized, dan input
wajib divalidasi di server.

---

## Lisensi

[MIT](LICENSE)

---

## Credits

- **Legacy project** - [`AmirulJusticia0507/paymentreceipt`](https://github.com/AmirulJusticia0507/paymentreceipt) - referensi aturan bisnis & dokumen
- **TCPDF** (legacy, LGPLv3) — sebagai referensi bentuk dokumen PDF