#!/usr/bin/env bash
#
# db-reset.sh — drop database, jalankan semua migrasi, lalu isi seed.
#
# PERINGATAN: script ini MENGHAPUS SELURUH DATA di database target.
# Jangan pernah mengarahkannya ke database production.
#
# Pakai:
#   ./scripts/db-reset.sh                  # migrate + seed (default)
#   ./scripts/db-reset.sh --no-seed        # hanya migrasi
#   ./scripts/db-reset.sh --db mydb_test   # database lain
#
# Env yang dibaca (bisa di-override langsung):
#   PGHOST      default localhost
#   PGPORT      default 5432
#   PGUSER      default postgres
#   PGPASSWORD  default (kosong)
#   PGDATABASE  default paymentreceipt_v2

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MIGRATIONS_DIR="$REPO_ROOT/backend/db/migrations"
SEED_FILE="$REPO_ROOT/backend/db/seeds/development.sql"

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${PGUSER:-postgres}"
PGDATABASE="${PGDATABASE:-paymentreceipt_v2}"
# User aplikasi yang memegang hak akses tabel. Database harus dibuat dengan OWNER
# ini, kalau tidak `db-reset.sh` akan membuat tabel milik superuser dan aplikasi
# akan gagal dengan "permission denied for table ...".
PGAPPUSER="${PGAPPUSER:-paymentreceipt}"
RUN_SEED=1

# Catatan: pakai while + shift, BUKAN `for arg in "$@"`. Di dalam for-loop, daftar
# Argumen sudah di-expand sebelum loop berjalan, jadi `shift` tidak memengaruhi iterasi berikutnya.
while [ $# -gt 0 ]; do
  case "$1" in
    --no-seed)   RUN_SEED=0; shift ;;
    --with-seed) RUN_SEED=1; shift ;;
    --db)        [ $# -ge 2 ] || { echo "--db butuh nama database." >&2; exit 2; }
                 PGDATABASE="$2"; shift 2 ;;
    -h|--help)   sed -n '2,16p' "${BASH_SOURCE[0]}" | sed 's/^#\{1,\} \{0,1\}//'; exit 0 ;;
    *) echo "Argumen tidak dikenal: $1 (lihat --help)" >&2; exit 2 ;;
  esac
done

export PGHOST PGPORT PGUSER PGDATABASE PGAPPUSER

echo "==> Target: $PGUSER@$PGHOST:$PGPORT/$PGDATABASE"

# --- Guard ----------------------------------------------------------------
# Cegah drop database production karena salah ketik nama.
case "$PGDATABASE" in
  *prod*|*production*)
    echo "DITOLAK: nama database '$PGDATABASE' mengandung 'prod'." >&2
    echo "Script ini tidak boleh dipakai untuk database production." >&2
    exit 1
    ;;
  paymentreceipt*) : ;;   # nama yang diharapkan, lanjut
  *)
    echo "DITOLAK: nama database '$PGDATABASE' tidak dikenali." >&2
    echo "Hanya database berawalan 'paymentreceipt' yang boleh dipakai." >&2
    exit 1
    ;;
esac

command -v psql >/dev/null 2>&1 || { echo "psql tidak ditemukan di PATH." >&2; exit 1; }

# --- 1. Drop & recreate ---------------------------------------------------
# Superuser hanya dipakai untuk DROP/CREATE DATABASE. Migrasi dan seed dijalankan
# sebagai user aplikasi supaya tabelnya DIMILIKI user aplikasi - kalau migrasi
# dijalankan sebagai superuser, tabel jadi milik superuser dan aplikasi gagal
# dengan "permission denied for table ...".
echo "==> Drop database (jika ada)"
psql -U "$PGUSER" -d postgres -v ON_ERROR_STOP=1 -q <<SQL
DROP DATABASE IF EXISTS ${PGDATABASE} WITH (FORCE);
CREATE DATABASE ${PGDATABASE} OWNER ${PGAPPUSER} ENCODING 'UTF8';
SQL
echo "    Database dibuat (owner: $PGAPPUSER)."

# --- 2. Migrasi -----------------------------------------------------------
# Dijalankan sebagai PGAPPUSER. Kalau role belum ada, gagal dengan pesan jelas
# alih-alih membuat tabel yang tidak bisa diakses aplikasi.
if ! psql -U "$PGAPPUSER" -d "$PGDATABASE" -q -c "SELECT 1;" >/dev/null 2>&1; then
  echo "    GAGAL: role '$PGAPPUSER' tidak ada atau tidak bisa login." >&2
  echo "    Buat dulu dengan:" >&2
  echo "      CREATE ROLE $PGAPPUSER LOGIN PASSWORD '...';" >&2
  echo "    atau set PGAPPUSER=<role yang sudah ada>." >&2
  exit 1
fi

echo "==> Menjalankan migrasi (sebagai $PGAPPUSER)"
count=0
while IFS= read -r f; do
  name="$(basename "$f")"
  printf '    %-40s ' "$name"
  if psql -U "$PGAPPUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -q -f "$f"; then
    echo "OK"
    count=$((count + 1))
  else
    echo "GAGAL"
    echo "Migrasi '$name' gagal. Database dalam kondisi sebagian." >&2
    exit 1
  fi
done < <(find "$MIGRATIONS_DIR" -name '*.up.sql' | sort)
echo "    $count migrasi diterapkan."

# --- 3. Seed --------------------------------------------------------------
if [ "$RUN_SEED" -eq 1 ]; then
  echo "==> Menjalankan seed development"
  psql -U "$PGAPPUSER" -d "$PGDATABASE" -v ON_ERROR_STOP=1 -q -f "$SEED_FILE"
  echo "    Seed selesai."
else
  echo "==> Seed dilewati (--no-seed)"
fi

# --- 4. Verifikasi --------------------------------------------------------
echo "==> Verifikasi"
tables=$(psql -U "$PGAPPUSER" -d "$PGDATABASE" -t -A -c "
  SELECT count(*) FROM information_schema.tables
  WHERE table_schema = 'public' AND table_type = 'BASE TABLE';")

if [ "${tables:-0}" -eq 0 ]; then
  echo "    GAGAL: tidak ada tabel ditemukan." >&2
  exit 1
fi
echo "    Tabel dibuat: $tables"

# Cek user aplikasi benar-benar bisa membaca TABEL (bukan cuma connect).
if ! psql -U "$PGAPPUSER" -d "$PGDATABASE" -q -c "SELECT 1 FROM users LIMIT 1;" >/dev/null 2>&1; then
  echo "    GAGAL: user '$PGAPPUSER' tidak bisa membaca tabel hasil migrasi." >&2
  exit 1
fi
echo "    Akses tabel sebagai '$PGAPPUSER': OK"

echo
echo "Selesai. Kredensial seed (LOCAL SAJA):"
echo "    admin    / admin123"
echo "    ga_staff / ga_staff123"
echo "    approver / approver123"
echo "    viewer   / approver123"
