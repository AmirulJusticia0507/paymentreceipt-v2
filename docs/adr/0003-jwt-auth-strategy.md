# ADR 0003 — JWT di httpOnly cookie, bukan server-side session

- **Status:** Accepted
- **Tanggal:** 2026-10-16
- **Related:** TODO.md § Phase 3, AGENTS.md § Keamanan

## Konteks

Aplikasi legacy tidak punya sistem autentikasi sama sekali. Yang ditemukan:

- `$_SESSION['username']` dan `$_SESSION['user_role']` dibaca di `admin/admin.php` dan
  `admin/admin--.php`
- `session_start()` dipanggil di dalam `admin/admin--.php` — yaitu di **setiap halaman** yang
  meng-include file itu, bukan hanya di entry point
- `sources/data_basename.php` punya timeout sesi manual:

```php
$timeout = 10 * 60;
if (isset($_SESSION['start_time'])) {
    if (time() - $_SESSION['start_time'] >= $timeout) {
        session_destroy();
        echo "<script>alert('Session Anda Telah Habis!'); ...</script>";
    }
}
$_SESSION['start_time'] = time();
```

Masalahnya:

1. Timeout hanya ada di 1 dari 9 halaman list → **tidak konsisten**
2. `session_start()` per halaman menghasilkan warning "session already active"
3. **Tidak ada proteksi akses sama sekali.** Tidak ada halaman yang mengecek apakah user
   login. `function_paymentreceipt.php?act=deletepayments&GApayment_id=1` bisa dipanggil
   siapa pun yang tahu URL-nya.
4. Tidak ada proteksi brute-force pada login
5. Tidak ada role enforcement — `$_SESSION['user_role']` hanya ditampilkan, tidak dipakai
   untuk otorisasi

Untuk rewrite, kita butuh mekanisme yang **benar-benar menegakkan** autentikasi dan otorisasi
di setiap endpoint.

## Keputusan

**JWT stateless** dengan hybrid access/refresh token, disimpan di cookie `HttpOnly`:

| Token | Masa berlaku | Disimpan | Contents |
|---|---|---|---|
| Access | 15 menit | Cookie `HttpOnly` | sub, role, username, exp, iat |
| Refresh | 7 hari | Cookie `HttpOnly` + **hash** di DB | opaque random string |

Aturan:

- Refresh token **rotates setiap kali dipakai** — token lama langsung di-revoke. Jika ada
  token lama yang dipakai ulang, seluruh chain dicabut (deteksi token theft).
- Refresh token disimpan di database sebagai **hash**, bukan plaintext — dump database
  tidak langsung memberi akses.
- `logout` menghapus refresh token dari DB.
- Password di-hash dengan **argon2id** (OWASP recommendation), bukan MD5/SHA1.
- Rate limit login: 5 percobaan per 15 menit per kombinasi username+IP.
- Role dicek **di setiap request** lewat middleware, bukan hanya di UI.

**Role:**

| Role | Akses |
|---|---|
| `admin` | Semua, termasuk master data & user management |
| `ga_staff` | Payment Receipt & arsip PR (create/edit/delete) |
| `approver` | Lihat semua, ubah status menjadi `checked`/`finished` |
| `viewer` | Read-only |

## Alternatif yang dipertimbangkan

| Opsi | Pro | Kontra |
|---|---|---|
| **JWT httpOnly cookie** | Token tidak bisa dicuri via XSS (`HttpOnly`); tetap stateless untuk access token; rotatable | Butuh endpoint refresh; tidak bisa langsung di-invalidate access token sebelum expired |
| **Server-side session (cookie session ID)** | Revocation instan; sederhana secara konsep | Butuh shared store (Redis/memcached) untuk horizontal scale; setiap request butuh lookup |
| **Token di `localStorage`** | Sederhana, tidak ada CORS issue saat pakai header | ❌ **Rentan XSS** — token bisa dicuri oleh skrip rogue. Ini pola yang paling sering direkomendasikan secara keliru |
| **Token di `Authorization` header** | Standar API | ❌ BerGdengan cookie HTTP-only: XSS tetap bisa akses kalau token disimpan di memory, tapi banyak implementasi menyimpannya di `localStorage` untuk persistensi |
| **Session cookie bawaan PHP-style, di Go** | Mirip legacy | Sama dengan server-side session |

### Kenapa httpOnly cookie, bukan localStorage

Ini keputusan keamanan utama. Dengan token di `localStorage`:

```js
// Satu XSS saja sudah cukup
localStorage.getItem('token')
```

Dengan cookie `HttpOnly`, token **tidak bisa diakses JavaScript sama sekali**. Cookie tetap
dikirim browser ke request kita, tapi `document.cookie` tidak membocorkannya. Iniymsnfactor
risiko XSS yang paling umum pada aplikasi SPA.

Because token tidak perlu dibaca JS, backend tetap bisa menerima token via cookie — dan
request tetap same-origin, jadi CORS tidak menjadi masalah.

## Konsekuensi

**Positif:**

- Otorisasi ditegakkan di server — tidak bisa dilewati dengan memanggil URL langsung
- XSS tidak bisa mencuri token
- Password storage aman (argon2id + hash refresh token)
- Brute-force login dibatasi rate limit
- Role-based access konsisten di seluruh aplikasi

**Negatif / Risiko:**

- Access token tidak bisa di-revoke sebelum expired (15 menit) → **jika akun dibobol,
  maksimal paparan 15 menit**. Untuk kasus ekstrem (admin), pertimbangkan access token
  pendek (5 menit).
- Klien harus handle 401 →alkali refresh → retry. Butuh interceptor di HTTP client.
- Tabel `refresh_tokens` perlu di-maintenance (cleanup token kedaluwarsa).
- Rate limit butuh store. Untuk single instance cukup in-memory map + TTL; kalau perlu
  horizontal scale, pindah ke Redis.

**Mitigasi refresh loop:** Kalau refresh juga gagal dengan 401, langsung logout — jangan
loop.

## Verifikasi

- `T3.12` — test hash/verify password, issue/parse token, dan **matriks RBIK** (setiap role ×
  setiap endpoint tertutup)
- `T3.10` — test rate limit login: percobaan ke-6 harus `429`
- `T3.11` — test cookie flags: `HttpOnly`, `Secure`, `SameSite` harus diset
- Test refresh rotation: memakai refresh token lama setelah rotasi harus **gagal**