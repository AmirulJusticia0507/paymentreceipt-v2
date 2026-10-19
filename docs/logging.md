# Logging & Aduk Log (Observability)

Dokumen ini menjelaskan bagaimana log aplikasi disusun, field apa yang selalu
ada, dan bagaimana menghubungkannya ke agregator log (Loki, ELK, CloudWatch,
dsb.).

---

## Prinsip

1. **Terstruktur, bukan teks bebas.** Seluruh log keluaran dalam format JSON
   satu baris per entri (`LOG_FORMAT=json`). Tidak ada log multi-baris yang
   harus di-grok.
2. **Satu request = satu `request_id`.** ID yang sama dipakai di log nginx
   (edge) dan log aplikasi (backend), sehingga satu request bisa dicari lintas
   service tanpa mencocokkan timestamp.
3. **Log ke stdout/stderr, bukan file.** Container tidak menulis file log
   sendiri; docker/orchestrator yang mengumpulkan. Ini 12-factor dan membuat
   `docker logs` selalu menampilkan apa yang aplikasi tampilkan.
4. **Tidak ada data sensitif di log.** Password, token, hash, dan isi body
   upload tidak pernah ditulis ke log (lihat `AGENTS.md` § Keamanan).

---

## Skema entri log backend

Semua entri memakai `slog` (Go stdlib) dengan handler JSON. Field bawaan
ditambah field request:

| Field | Tipe | Keterangan |
|---|---|---|
| `time` | string (RFC 3339, UTC) | Waktu entri. |
| `level` | string | `debug` \| `info` \| `warn` \| `error` (huruf kecil). |
| `msg` | string | Ringkasan peristiwa (`"request"`, `"request lambat"`, `"panic"`, ...). |
| `request_id` | string | ID request; identik dengan `X-Request-ID` di response dan `$request_id` nginx. |
| `method` | string | HTTP method (hanya di entri request). |
| `path` | string | Path request tanpa query string. |
| `status` | int | Status HTTP response. |
| `bytes` | int | Ukuran body response. |
| `duration` | string | Durasi pemrosesan (contoh: `12.5ms`). |
| `remote_addr` | string | IP client asli (sudah melewati `RealIP` bila ada trusted proxy). |

Contoh satu baris:

```json
{"time":"2026-10-19T01:23:45Z","level":"info","msg":"request","request_id":"0af3c2e1d4b5a6978c1f2e3d4b5a6978","method":"POST","path":"/api/v1/payment-receipts","status":201,"bytes":512,"duration":"8.2ms","remote_addr":"10.0.0.5:53422"}
```

Request yang lambat (> 1 detik) dinaikkan otomatis ke level `warn` dengan `msg`
`"request lambat"` - tidak perlu query duration untuk menemukannya.

## Skema entri log nginx (edge)

nginx dikonfigurasi (`frontend/nginx.conf`) menulis akses log sebagai JSON ke
stdout dengan field:

| Field | Keterangan |
|---|---|
| `time` | Waktu ISO 8601. |
| `request_id` | Dihasilkan nginx (`$request_id`), diteruskan ke backend sebagai `X-Request-ID`. |
| `remote_addr`, `method`, `host`, `path`, `query` | Detail request. |
| `status`, `bytes_sent`, `duration_s` | Detail response. |
| `upstream_status` | Status dari backend (proxy). |
| `user_agent`, `referer` | Header standar. |

---

## Request ID: korelasi edge <-> aplikasi

Alur satu request:

1. Client mengirim request ke nginx.
2. nginx membuat `$request_id` (32 karakter hex) dan mencatatnya di akses log.
3. nginx meneruskan ID itu ke backend lewat header `X-Request-ID`.
4. Backend **menerima** ID yang sama - hanya bila formatnya aman
   (`[A-Za-z0-9_-]`, panjang 8-64). Nilai tidak valid (mis. ada karakter
   kontrol / upaya log-injection) dibuang dan diganti dengan ID buatan sendiri.
5. Backend memakai ID itu untuk: baris log request, logger di context
   (`LoggerFrom(ctx)`), field `instance` pada respons RFC 9457 (problem
   detail), dan header response `X-Request-ID`.

Konsekuensi praktis: kalau user melapor error dan menyebut `X-Request-ID`
(ada di response header, dan di panel dev browser), cukup cari ID itu:

```bash
docker compose logs api web | grep 0af3c2e1d4b5a6978c1f2e3d4b5a6978
```

atau di agregator (Loki/LogQL):

```
{job=~"api|web"} |= "0af3c2e1d4b5a6978c1f2e3d4b5a6978"
```

---

## Rotasi log container

`docker-compose.yml` memakai driver `json-file` dengan rotasi untuk semua
service (`max-size: 10m`, `max-file: 5` - maksimum ~50 MB per service).
Tanpa batas ini, docker menumpuk stdout tanpa batas dan bisa menghabiskan disk
pada server yang berjalan lama.

Untuk produksi dengan orchestrator lain, setara di Kubernetes adalah
`kubelet` `containerLogMaxSize` / `containerLogMaxFiles`, dan di systemd
adalah `journald` `SystemMaxUse`.

---

## Menghubungkan ke agregator

Aplikasi tidak memilih agregator tertentu - keluarannya JSON ke stdout sehingga
kompatibel dengan semuanya. Beberapa pola umum:

### Docker / Compose (skala kecil)

`docker logs -f` atau `docker compose logs -f api` sudah cukup untuk review.
Untuk pengumpulan lintas host, pasang collector (Loki docker driver,
Elasticsearch logstash driver, atau promtail) sebagai logging driver global.

### Loki (Grafana)

Pasang [promtail](https://grafana.com/docs/loki/latest/clients/promtail/) atau
docker loki-driver, lalu query dengan LogQL. Contoh mencari semua error:

```
{job="paymentreceipt-v2", container="api"} | json | level="error"
```

### ELK (Elasticsearch + Logstash + Kibana)

Logstash `json` codec langsung mem-parse baris; buat index template dengan
`request_id` sebagai `keyword` supaya pencarian dan korelasi cepat.

### CloudWatch / Stackdriver

Container logging bawaan platform sudah mem-parse JSON stdout. Petakan
`request_id` ke field agar bisa dipakai sebagai correlation ID.

---

## Level logging

| Env | Efek |
|---|---|
| `LOG_LEVEL=debug` | Semua, termasuk diagnostik. |
| `LOG_LEVEL=info` (default) | Request & peristiwa normal. |
| `LOG_LEVEL=warn` | Hanya peringatan & di atasnya (request lambat, panic). |
| `LOG_LEVEL=error` | Hanya error. |

Level diatur lewat env tanpa build ulang. `LOG_FORMAT` memilih `json`
(produksi/agregator) atau `text` (development, lebih enak dibaca manusia).

---

## Aturan untuk kontributor

- Gunakan `slog` dari logger di context (`httputil.LoggerFrom(ctx)`), bukan
  `fmt.Println` / `log` bawaan.
- Jangan menulis password, token, hash, atau isi body upload ke log.
- Untuk error yang dialihkan (wrap), sertakan konteks dengan `%w` supaya rantai
  error tetap terlihat di log.
- Setiap perubahan field log harus ikut memperbarui tabel skema di dokumen ini.
