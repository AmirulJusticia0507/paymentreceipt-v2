# AGENTS.md

Instruksi untuk AI agent (dan manusia) yang mengerjakan repo ini.
**Baca dokumen ini sebelum mengubah kode apa pun.**

Repo ini adalah rewrite (greenfield) dari aplikasi Payment Receipt / Purchase Requisition
PT. Asi Pudjiastuti Aviation (Susi Air). Repo legacy PHP ada di
`AmirulJusticia0507/paymentreceipt` — dipakai **hanya sebagai referensi aturan bisnis & bentuk
dokumen**, bukan untuk di-refactor atau di-port langsung.

---

## 1. Aturan Emas

1. **Satu task = satu commit = satu push.** Selesai satu task → `git commit` → `git push`
   **langsung**. Jangan kumpulkan work di akhir sesi.
2. **Tidak boleh ada perubahan lokal yang tidak di-commit** saat sesi berakhir.
3. **Task hanya tandai `[x]` di `TODO.md` kalau Definition of Done terpenuhi.**
4. **Jangan pernah** membuat commit langsung ke `main` dengan `--amend` pada commit yang
   sudah di-push. Kalau salah, tambah commit baru atau `revert`.
5. **Jangan pernah** commit kredensial, `.env`, atau file build.

---

## 2. Workflow

### Awal sesi

```bash
cd C:\laragon\www\paymentreceipt-v2
git status              # pastikan bersih
git pull --rebase       # ambil perubahan terbaru
cat TODO.md             # cari task berikutnya
```

### Selesaikan satu task

```bash
# 1. kerjakan (kode + test + docs)
# 2. verifikasi
cd backend && go build ./... && go test ./... && golangci-lint run
cd frontend && npm run lint && npm run type-check && npm run build

# 3. update TODO.md (centang [x])
# 4. commit
git add -A
git commit -m "feat(payment-receipt): filter by status and date range"

# 5. push
git push origin main
```

### Kalau task terlalu besar

Pecah jadi sub-task bernomor di `TODO.md` lebih dulu, **commit dokumennya**, baru kerjakan
satu per satu. Jangan ditunda — push per sub-task.

### Kalau ada blocker

- Tandai `[!]` di `TODO.md` dengan catatan blocker.
- **Tetap push** dokumenasi blocker-nya.
- Jangan diam-diam meninggalkan task setengah jadi tanpa dicatat.

---

## 3. Format Commit Message

Gunakan [Conventional Commits](https://www.conventionalcommits.org/).

```
<type>(<scope>): <subject>

[body jika perlu]
[footer: BREAKING CHANGE / Refs: #task-id]
```

**Type:**

| Type | Kapan |
|---|---|
| `feat` | Fitur baru |
| `fix` | Perbaikan bug |
| `refactor` | Ubah struktur tanpa ubah perilaku |
| `perf` | Performa |
| `test` | Menambah/memperbaiki test |
| `docs` | Hanya dokumentasi |
| `build` | Build system, dependency |
| `ci` | GitHub Actions |
| `chore` | Misc (gitignore, dll) |
| `style` | Formatting saja |

**Scope:** `config`, `database`, `auth`, `department`, `satuan`, `categories`, `base`,
`requests`, `requestbase`, `sources`, `payment-receipt`, `pr-archive`, `pdf`, `dashboard`,
`ui`, `docs`, `deps`.

**Subject:** imperative mood (perintah), huruf kecil, tanpa titik di akhir, ≤ 72 karakter.

Contoh yang benar:

```
feat(auth): add JWT refresh token rotation
fix(payment-receipt): total harga now uses generated column
docs(todo): mark Phase 0 foundation tasks complete
chore(deps): add chi router and pgx driver
```

Contoh yang salah:

```
update code          # terlalu vague
feat                 # tanpa scope & subject
Fixed the bug.      # bukan imperative, ada titik
WIP                 # tidak jelas
```

---

## 4. Struktur Repo

```
backend/
  cmd/api/main.go           # entrypoint — harus tetap tipis
  internal/
    config/                 # load & validasi env
    database/               # pool, WithTx helper
    domain/                 # entitas + aturan bisnis — PURE Go, tanpa I/O
    repository/             # akses data (sqlc generated)
    service/                # orkestrasi use case
    handler/                # HTTP layer (chi)
    middleware/
    dto/                    # request/response structs
    platform/               # pdf, dsb
  db/
    migrations/             # .sql (golang-migrate)
    queries/                # .sql (sqlc)
  sqlc.yaml

frontend/
  src/
    components/             # generik & reusable
    composables/
    layouts/
    pages/
    features/               # per-domain: components + api + types
    lib/                    # http client, query client, utils
    router/
    stores/                 # Pinia
    styles/                 # tailwind entry + tokens
docs/
TODO.md
AGENTS.md
```

**Aturan import:**

- `handler` boleh import `service`, `dto`, `domain`. **Tidak boleh** import `database`/
  `pgx` langsung.
- `service` boleh import `repository`, `domain`. **Tidak boleh** import `handler`/`chi`.
- `domain` **tidak boleh** import apa pun dari `internal/` lain. Murni.
- `frontend/features/X` boleh import `lib`, `components`, `stores`. **Tidak boleh** import
  `features/Y` (cross-feature import = smell).

---

## 5. Konvensi Go

### Gaya umum

- `gofumpt` formatting. Jangan manual atur whitespace — jalankan formatter.
- Error handling: **bukan** `panic` di request path. Pakai `error` + `errors.Is/As`.
- Semua context pada fungsi yang melakukan I/O harus menerima `context.Context`.
- Komentar exported item **wajib** (`// Fungsi ini...`).
- Nama receiver: 1–2 huruf, konsisten (`func (s *PaymentReceiptService) Get(...)`).

### Aturan database — **KRITIS**

> Legacy project diretas via SQL injection. **Tidak boleh adainterpolasi string ke SQL.**

```go
// ✅ BENAR — pakai sqlc (parameterized)
q := queries.GetByID(ctx, id)

// ✅ BENAR — kalau memang butuh raw query,WAJIB placeholder
row := db.QueryRow(ctx, "SELECT * FROM payment_receipt WHERE id = $1", id)

// ❌ SALAH — string formatting ke SQL
row := db.QueryRow(ctx, fmt.Sprintf("SELECT * FROM payment_receipt WHERE id = '%s'", id))
```

Semua query **harus** lewat `db/queries/*.sql` + sqlc generated. Raw query hanya dengan
alasan tertulis di komentar.

### Naming

| Jenis | Konvensi | Contoh |
|---|---|---|
| File | `snake_case` | `payment_receipt.go` |
| Tipe exported | `PascalCase` | `PaymentReceipt` |
| Fungsi/method | `camelCase` | `GetByStatus` |
| Variabel lokal | `camelCase` | `receiptCount` |
| Const | `SCREAMING_SNAKE` | `MaxPerPage = 100` |
| Kolom DB | `snake_case` | `tanggal_pembelian` |

### Error

```go
var ErrNotFound = errors.New("resource not found")

// wrap dengan konteks, jangan kehilangan error asli
if errors.Is(err, sql.ErrNoRows) {
    return nil, fmt.Errorf("get receipt %d: %w", id, ErrNotFound)
}
```

---

## 6. Konvensi Vue 3 + TypeScript

### Component

- Selalu **Composition API** dengan `<script setup lang="ts">`. Jangan pakai Options API.
- `defineProps` / `defineEmits` dengan **type eksplisit** (interface atau `type`).
- Props yang punya default → `withDefaults`.
- Emit dengan **event declare** (`defineEmits<{...}>()`), bukan `$emit` string bebas.

```vue
<script setup lang="ts">
interface Props {
  receipt: PaymentReceipt
  canEdit?: boolean
}

const props = withDefaults(defineProps<Props>(), { canEdit: false })

const emit = defineEmits<{ delete: [id: string] }>()

const formattedTotal = computed(() => formatRupiah(props.receipt.total))
</script>
```

### Penamaan

| Jenis | Konvensi | Contoh |
|---|---|---|
| Component | `PascalCase` | `DataTable.vue` |
| Composable | `useXxx` | `usePaymentReceipts` |
| Store | `useXxxStore` | `useAuthStore` |
| Type/interface | `PascalCase` | `PaymentReceipt` |
| Type alias nilai | `camelCase` | `type ReceiptId = string` |
| Fungsi util | `camelCase` | `formatRupiah` |
| File (selain component) | `kebab-case` | `http-client.ts` |

### Data fetching

- **Server state → TanStack Query.** Jangan simpan data server di Pinia.
- **UI state (sidebar, tema, modal) → Pinia.**
- `queryKey` harus **stabil & konsisten** dan di-export dari feature:

```ts
export const paymentReceiptKeys = {
  all: ['payment-receipts'] as const,
  list: (filters: Filters) => [...paymentReceiptKeys.all, 'list', filters] as const,
  detail: (id: string) => [...paymentReceiptKeys.all, 'detail', id] as const,
}
```

- Setelah mutate → `invalidateQueries`. Jangan memakai optimistic update kecuali UX benar-benar butuh.

### Tailwind

- **Tidak boleh** menulis warna hex/arbitrary value kalau sudah ada token:

```html
<!-- ❌ -->
<div class="bg-[#1a56db] text-white rounded-[7px]">

<!-- ✅ -->
<div class="bg-brand-600 text-white rounded-lg">
```

- Token warna & spacing didefinisikan sekali di `src/styles/` via `@theme`.
- Layout pakai utility yang sudah ada — hindari CSS custom kalau tidak perlu.
- **Setiap** komponen interaktif wajib punya state `hover:`, `focus-visible:`, `disabled:`.
- Responsive: mobile-first, tambahkan `sm:`, `md:`, `lg:` seperlunya (jangan semua breakpoint).

---

## 7. Keamanan — NON-NEGOTIABLE

> Legacy project direct SQL injection. Proyek ini **tidak boleh** mengulanginya.

- [ ] **Semua query parameterized.** Tidak ada `fmt.Sprintf` ke SQL.
- [ ] **Validasi di server.** Validasi client hanya UX, bukansecurity.
- [ ] **Escape semua output** yang berisi data user di HTML.
- [ ] **Otorisasi per-request.** Cek kepemilikan resource, bukan hanya role.
- [ ] Endpoint yang mengubah data wajib punya auth middleware.
- [ ] Sanitize input, batasi panjang string, whitelist nilai enum.
- [ ] Rate limit pada endpoint sensitif (login, reset password).
- [ ] Jangan pernah log password, token, atau data sensitif.
- [ ] Cookie auth: `HttpOnly` + `Secure` + `SameSite`.
- [ ] Password: argon2id. **Tidak pernah** MD5/SHA1 unsalted (legacy pakai MD5 — jangan).

Kalau ragu soal sebuah input: **bukan aman**. Validasi lagi.

---

## 8. Testing

- Test diletakkan **sejajar** dengan kode: `foo.go` → `foo_test.go`.
- Nama test: `TestNamaFungsi_SkenarioYangDiuji`.
  - Contoh: `TestPaymentReceiptService_UpdateStatus_RejectsIllegalTransition`
- Tabel-driven test untuk banyak kasus dengan input/output serupa.
- **Test wajib ditulis** untuk: setiap aturan bisnis, setiap transisi status, setiap endpoint
  (happy path + error), setiap query yang diubah.
- Bug yang ditemukan → tulis test yang gagal dulu, baru perbaiki (red → green).

```bash
cd backend
go test ./...
go test -race ./...
go test -cover ./...
```

---

## 9. Documentation

- **Code > komentar.** Komentar menjelaskan *kenapa*, bukan *apa*.
- Fungsi yang non-obvious butuh penjelasan, bukan kode yang diulang.
- Semua keputusan arsitektur besar → ADR di `docs/adr/`.
- Perubahan aturan bisnis → update `TODO.md` **di commit yang sama**.

---

## 10. Anti-Pattern — Jangan Dilakukan

| Jangan | Lakukan |
|---|---|
| Commit semua perubahan sekaligus di akhir | Commit per task, push langsung |
| `git commit --amend` pada commit yang sudah pushed | Commit baru atau `revert` |
| Hardcode URL, kredensial, atau API key | `.env` + config |
| `fmt.Sprintf` untuk SQL | Parameterized query |
| `panic` di request handler | Return error |
| Format manual,RESULT skip formatter | `gofmt`/`gofumpt`, Prettier |
| Langsung commit ke `main` tanpa test | Test dulu, lalu commit |
| `console.log` untuk debug | Structured logging / hapus sebelum commit |
| Menyalin kode legacy apa adanya | Tulis ulang sesuai konvensi repo ini |
| Menambah dependency tanpa alasan | Cek apakah stdlib cukup dulu |
| Perbarui TODO.md tanpa task-nya selesai | Centang hanya saat DoD terpenuhi |
| Menyesuaikan kode agar test lulus dishonor | Perbaiki kodenya, bukan testnya |

---

## 11. Ringkasan Command

```bash
# Backend
cd backend
go build ./...
go test ./...
go test -race ./...
golangci-lint run
gofumpt -w .

# Frontend
cd frontend
npm run dev
npm run lint
npm run type-check
npm run build

# Database
psql -U postgres -d paymentreceipt_v2 -f db/migrations/0001_xxx.up.sql
```

---

## 12. Definisi Selesai

Sebuah task selesai bila **SEMUA** ini terpenuhi:

- [ ] Kode berjalan — build lolos, tidak ada error
- [ ] Lint + type-check bersih
- [ ] Test ada & hijau (termasuk `-race`)
- [ ] Validasi server-side untuk semua input baru
- [ ] Loading / empty / error state ditangani di UI
- [ ] Tidak ada kredensial atau debug code tersisa
- [ ] TODO.md diperbarui bila relevan
- [ ] **Commit** dengan pesan Conventional Commit
- [ ] **Push** ke `origin/main`
- [ ] Working tree bersih

---

## 13. Konteks Proyek

**Produk:** sistem pengajuan & verifikasi pembelian untuk General Affairs.

**Domain:**

- **Department** — unit kerja pemohon
- **Request** — pengajuan milik satu department
- **Base / Request Base** — kategori pengajuan umum & turunannya
- **Source** — toko/sumber pembelian, mode `online` (link) atau `offline` (nama toko, telp, rekening)
- **Payment Receipt** — transaksi pembelian: item, qty, satuan, harga, usage
- **PR Archive** — purchase requisition yang sudah disetujui, ready cetak

**Alur status Payment Receipt:**

```
pending → checked → processed → finished
             ↓          ↓
          cancelled (terminal)
```

Legal transisi harus ditegakkan di `internal/domain` dan punya test.

**Dokumen output:** PDF *Purchase Requisition* berlogo PT. Asi Pudjiastuti Aviation,
dituju "General Affair Department", dengan blok tanda tangan Department / Manager-Director /
General Affairs Department. Template asli ada di legacy `document/PurchaseReceipt_blank.htm`.

**Aturan bisnis yang sudah ketemu saat analisis legacy:**

- Harga & qty tidak boleh nol/negatif
- Tanggal pembelian tidak boleh terlalu jauh ke belakang (legacy membatasi "tidak boleh < hari ini" — cek ulang apakah ini masih relevan untuk bisnis)
- Total = qty × harga, dihitung di database (generated column), bukan di client
- Satu payment receipt harus konsisten: request & department harus nyambung

> Kalau ada aturan bisnis baru atau perubahan, **tanyakan dulu** — jangan diasumsikan.