# ADR 0002 — sqlc + pgx, bukan GORM dan bukan SQL string

- **Status:** Accepted
- **Tanggal:** 2026-10-16
- **Related:** TODO.md § Phase 2, AGENTS.md § Konvensi Go

## Konteks

Aplikasi legacy memakai `mysqli_query()` dengan **interpolasi string langsung**:

```php
$querytambah = mysqli_query($koneksi,
    "INSERT INTO generalaffairspayment VALUES('$GApayment_id', '$request_id', ...)");
```

Ini adalah sumber SQL injection — setiap `$_POST` dan `$_GET` masuk ke query tanpa
parameterisasi. Apps yang bisa menerima input `'` dari user bisa dihancurkan atau diekploitasi.

Untuk rewrite, kita butuh lapisan akses data yang membuat SQL injection **tidak mungkin
terjadi secara struktural**, bukan sekadar "dapat admonisi di code review".

Ada satu pengecualian di legacy (`PR/function_PR.php`) yang benar —
`mysqli_prepare` + `mysqli_stmt_bind_param` — membuktikan prepared statement adalah jalur
yang sudah pernah dicoba.

Kebutuhan:

1. Parameterized query sebagai **default**, bukan pilihan
2. Type safety antara database dan Go
3. Ringan — tanpa overhead runtime signifikan
4. Support penuh fitur PostgreSQL (enum, generated column, `jsonb`, CTE)
5. Ecosystem modern, aktif maintained

## Keputusan

Gunakan **sqlc** untuk generate kode Go dari query SQL, dengan **pgx/v5** sebagai driver
PostgreSQL.

- Query ditulis di `backend/db/queries/*.sql` sebagai SQL asli (bukan string Go)
- sqlc membaca skema database dan generate typed Go structs + query functions
- pgx/v5 menangani koneksi, pool, dan context

**Tidak ada `fmt.Sprintf` ke SQL. Tidak ada string yang bisa disisipi input user.**

Router: **chi** (bukan gin/echo) — stdlib `net/http`-compatible, composable middleware,
overhead rendah.

## Alternatif yang dipertimbangkan

| Opsi | Pro | Kontra |
|---|---|---|
| **sqlc + pgx** | Query tetap SQL asli (bukan DSL); type dijamin saat compile; nol reflection; generated code bisa dibaca & di-review | Perlu `sqlc generate` setiap query berubah; learning curve sqlc |
| **GORM** | Fast to prototype, auto-migrate | ❌ Query di-generate runtime → tidak bisa di-review. ❌ SQL injection tetap mungkin lewat raw query. ❌ Overhead tinggi. ❌ Auto-migrate berisiko merusak data. |
| **Bun** | Lebih modern dari GORM, type-safe | Layer abstraksi lebih tebal; kurva belajar lebih tinggi; belum sedewasa |
| **pgx + query string manual** | Paling simple | ❌ Raw string SQL → kembali ke risiko yang sama |
| **Ent (Facebook)** | Schema-as-code, bagus untuk graph | Overkill untuk CRUD relational; menambah konsep baru |
| **sqlx** | Ringan, `?` placeholder | Tidak ada type safety; kolom typo baru ketahet saat runtime |

### Kenapa bukan GORM (alasan utama)

GORMvsqlc adalah **pilihan keamanan, bukan soal selera**. Dengan GORM, developer bisa menulis:

```go
db.Where("nama = '" + input + "'").Find(&users)   // tetap vulnerable
```

Tidak ada mekanisme yang mencegah itu. Dengan sqlc, query hanya bisa ditulis di file `.sql`
dan akan di-parse — tidak ada jalur kode untuk menyisipkan string ke SQL.

## Konsekuensi

**Positif:**

- SQL injection mustahil secara struktural, bukan karena disiplin
- Type mismatch (mis. string di kolom int) jadi **compile error**, bukan runtime error
- Query tetap SQL yang bisa dibaca DBA — tidak ada DSL misterius
- Generated code di-commit ke repo sehingga review bisa dilakukan
- N-fold reduction in boilerplate (tidak tulis scan/tidak tulis bind manual)

**Negatif / Cost:**

- Semua query harus ditulis ulang di `.sql` — ada upfront cost
- Perubahan skema → `sqlc generate` ulang
- Internal queries yang sangat dinamis (dynamic WHERE) lebih sulit — harus tulis beberapa
  query dengan `sqlc` `narg` + `sqlc.slice`
-sqlc perlu akses ke skema DB saat generate — workflow sedikit lebih banyak langkah

**Mitigasi biaya:** Query dinamis yang ekstrem (filter bebas) ditangani dengan
`sqlc.arg()` + `sqlc.slice()`; kasus yang benar-benar dinamis boleh pakai raw query
**dengan placeholder** (`$1`, `$2`) dan **wajib** ada komentar alasan.

## Verifikasi

- `T11.10` — regression test yang inject payload SQL ke semua endpoint input, pastikan tidak
  ada efek samping
- `T2.12` — naming convention & mapping terverifikasi
- Code review: cari `Sprintf` di package `repository` — harus nol