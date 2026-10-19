# Runbook Deploy & Rollback

Runbook operasional untuk **Payment Receipt v2**. Dokumen ini menyasar operator
yang menjalankan deploy, rollback, dan penanganan insiden - bukan developer yang
mengerjakan kode.

Dokumen terkait:

- [docs/logging.md](logging.md) — skema log & korelasi `request_id`
- [deploy/production.env.example](production.env.example) — daftar env production
- [TODO.md § Phase 12](../TODO.md#phase-12--deploy) — keputusan di balik setup ini

---

## Arsitektur deploy

```
                    ┌──────────────────────────────────────────────┐
   user ──HTTPS──▶  │ web (nginx) ──proxy──▶ api (Go) ──▶ PostgreSQL │
                    │      ▲                    ▲                   │
                    └──────┼────────────────────┼───────────────────┘
                           │                    │
                    healthcheck /healthz  healthcheck /healthz
                                          (+ /readyz cek DB)
```

Urutan start di compose:

1. **`migrate`** — one-shot container menjalankan `paymentreceipt-migrate up`.
   Compose menunggu **exit code 0** sebelum lanjut (ini yang menjamin aplikasi
   tidak pernah start dengan skema yang belum sesuai).
2. **`api`** — server aplikasi. Sehat bila `/healthz` menjawab 200.
3. **`web`** — nginx; start setelah `api` dilaporkan `service_healthy`.

Image produksi distroless (tanpa shell). Binary yang tersedia di dalam image:

| Binary | Fungsi |
|---|---|
| `/app/paymentreceipt-api` | Server API. Juga mendukung `-healthcheck` (probe `/healthz`, exit 0/1) dan `-version`. |
| `/app/paymentreceipt-migrate` | Migrator: `up`, `down -steps N`, `version`, `force VERSION`. |

---

## Prasyarat deploy

- Akses ke registry image (atau mesin build).
- `DB_URL` produksi dengan `sslmode=require|verify-ca|verify-full`
  (config aplikasi menolak start tanpa ini di `APP_ENV=production`).
- `AUTH_JWT_SECRET` minimal 32 byte dari secret manager.
- Backup database terbaru (lihat [Backup](#backup) sebelum deploy besar).

---

## Deploy normal

### 1. Build & push image

```bash
# tag dengan commit SHA supaya bisa ditelusuri
SHA=$(git rev-parse --short HEAD)
docker build -t registry.example.com/paymentreceipt-api:$SHA   ./backend
docker build -t registry.example.com/paymentreceipt-web:$SHA   ./frontend
docker push registry.example.com/paymentreceipt-api:$SHA
docker push registry.example.com/paymentreceipt-web:$SHA
```

### 2. Siapkan environment

Salin `deploy/production.env.example` ke environment orchestrator, lalu
injeksikan secret (jangan pernah menyimpan secret di image/repo):

```bash
# contoh; sesuaikan dengan tooling (systemd, k8s secret, vault, dsb.)
export DB_URL='postgres://USER:PASSWORD@HOST:5432/paymentreceipt_v2?sslmode=verify-full'
export AUTH_JWT_SECRET='<dari secret manager>'
```

Daftar lengkap env: [deploy/production.env.example](production.env.example).
Env yang **wajib** ada (tanpa default):

| Env | Nilai |
|---|---|
| `DB_URL` | URL PostgreSQL produksi (sslmode wajib) |
| `AUTH_JWT_SECRET` | ≥ 32 byte acak |

### 3. Jalankan migrasi dulu

Dengan compose, langkah ini otomatis (service `migrate` jalan sebelum `api`).
Secara manual pun bisa - binary migrate dan image API yang sama:

```bash
docker run --rm \
  -e DB_URL="$DB_URL" \
  registry.example.com/paymentreceipt-api:$SHA \
  /app/paymentreceipt-migrate up
```

Kalau tidak ada migrasi tertunda, perintah keluar 0 tanpa melakukan apa pun
(idempoten) - aman dijalankan berulang.

### 4. Start / restart stack

```bash
docker compose pull
docker compose up -d
```

### 5. Verifikasi

```bash
# status container - semua harus "healthy" (bukan cuma "running")
docker compose ps

# health end-to-end (lewati nginx, menguji rantai penuh)
curl -fsS https://receipt.example.com/healthz

# readiness (memastikan backend bisa mencapai database)
curl -fsS https://receipt.example.com/readyz

# pastikan versi binary = tag yang dideploy
docker compose exec api /app/paymentreceipt-api -version
```

Kalau `/readyz` menjawab 503, lihat field `checks` pada response - komponen
`database` yang `down` berarti masalah koneksi, bukan aplikasi.

---

## Rollback

Kapan rollback perlu? Umumnya karena: (a) bug perilaku di rilis baru, atau
(b) migrasi gagal. **Backup adalah syarat** untuk aman rollback skema.

### Kasus A — Bug aplikasi, skema TIDAK berubah

Paling sederhana: jalankan ulang image versi sebelumnya. Migrasi idempoten -
tidak ada yang diterapkan ulang.

```bash
PREV=<SHA sebelumnya>
docker compose stop api web

docker run --rm -e DB_URL="$DB_URL" \
  registry.example.com/paymentreceipt-api:$PREV \
  /app/paymentreceipt-migrate up    # no-op kalau skema sudah cocok

# sesuaikan tag image ke $PREV, lalu start lagi
docker compose up -d
```

### Kasus B — Migrasi bermasalah (dirty state)

Kalau sebuah migrasi gagal di tengah jalan, tabel `schema_migrations` menandai
database **dirty**, dan semua migrasi berikutnya akan DITOLAK sampai penyebab
diperbaiki.

Gejala:

```
migrate gagal: database dalam kondisi dirty state pada versi N: ...
```

Penanganan:

1. **Jangan** menjalankan `force` sampai penyebab kegagalan diketahui dan skema
   di database dipastikan kondisinya (biasanya: baca pesan error migrasi,
   periksa apakah statement terakhir berhasil).
2. Setelah tahu kondisi skema sebenarnya, setel penanda versi. `force` HANYA
   mengubah penanda - tidak menambah/menghapus objek:

   ```bash
   docker run --rm -e DB_URL="$DB_URL" \
     registry.example.com/paymentreceipt-api:$SHA \
     /app/paymentreceipt-migrate force <N>
   ```

3. Lanjutkan dengan `up`. Kalau ternyata migrasi `N` memang belum/tidak
   selesai, sesuaikan skema manual (atau pulihkan dari backup) dulu.

### Kasus C — Rollback skema (darurat)

Setiap migrasi punya pasangan `.down.sql`. Untuk mengembalikan beberapa versi:

```bash
docker run --rm -e DB_URL="$DB_URL" \
  registry.example.com/paymentreceipt-api:$PREV \
  /app/paymentreceipt-migrate down -steps 1
```

> **Peringatan:** `down` MENGHAPUS objek (tabel, kolom, tipe) beserta datanya.
> Rollback skema hanya untuk darurat - jalankan hanya setelah backup, dan
> pastikan aplikasi versi lama benar-benar cocok dengan skema hasil rollback.

Kalau sebuah versi tidak punya berkas `.down.sql` (jarang; dicegah oleh test
`TestEmbeddedFS_Lengkap`), `down` berhenti dengan error eksplisit - jangan
memaksa; pulihkan dari backup.

---

## Backup

Backup rutin dengan `pg_dump` (jadwalkan via cron/systemd timer):

```bash
pg_dump --format=custom --no-owner \
  --file="paymentreceipt_$(date +%Y%m%d_%H%M%S).dump" \
  "$DB_URL"
```

Pulihkan:

```bash
pg_restore --clean --if-exists --no-owner \
  --dbname="$DB_URL" paymentreceipt_YYYYMMDD_HHMMSS.dump
```

Simpan dump di luar mesin database. **Sebelum deploy dengan migrasi skema,
buat backup manual tambahan** (lihat langkah 2 di setiap skenario rollback).

---

## Monitoring

| Sinyal | Ambang | Tindakan |
|---|---|---|
| `/readyz` != 200 | 1 menit | Cek `checks` di response; cek koneksi DB |
| Container `unhealthy` | 3 kali berturut | `docker inspect` + log container |
| Log `level=error` melonjak | korelasi waktu | Cari `request_id` di log edge & api |
| Request `status>=500` | per menit | Filter log nginx `status` >= 500 |
| `request lambat` (warn) | sering | Cek query & koneksi DB |

Field lengkap per entri log: [docs/logging.md](logging.md). Contoh investigasi
laporan user (dia menyebut `X-Request-ID`):

```bash
RID=0af3c2e1d4b5a6978c1f2e3d4b5a6978
docker compose logs api web | grep "$RID"
```

---

## Troubleshooting cepat

| Gejala | Penyebab paling mungkin | Tindakan |
|---|---|---|
| `migrate` exit 1, "DB_URL wajib diisi" | env `DB_URL` tidak terbaca | Cek injeksi env / file `.env` compose |
| `migrate` exit 1, "dirty state" | migrasi gagal di tengah jalan | Lihat [Kasus B](#kasus-b--migrasi-bermasalah-dirty-state) |
| `api` exit 1, pesan `konfigurasi tidak valid` | env kurang/salah (lihat pesan per item) | Lengkapin env sesuai `production.env.example` |
| `api` sehat tapi `/readyz` 503 | DB tidak terjangkau / sslmode | Cek `DB_URL`, firewall, `sslmode` |
| `web` `unhealthy`, `api` healthy | nginx tidak bisa resolve `api` | Cek name resolution / network compose |
| 502/504 dari nginx | api down atau lambat | `docker compose logs api`; cek `request lambat` |
| Login gagal setelah deploy | `AUTH_JWT_SECRET` berubah | Token lama tidak berlaku - user perlu login ulang (normal) |

---

## Catatan keamanan operasional

- Image production **distroless**: tidak ada shell. Untuk masuk container hanya
  lewat binary yang tersedia (mis. `/app/paymentreceipt-api -version`).
- Secret tidak pernah ditaruh di image, compose file, atau argumen command yang
  muncul di `ps` - injeksikan lewat environment/side-channel secret manager.
- Setiap perubahan password seed (`admin123` dsb.) wajib dilakukan sebelum
  produksi; seed `backend/db/seeds/development.sql` hanya untuk lokal.
