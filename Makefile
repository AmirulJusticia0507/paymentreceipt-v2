# Makefile untuk Payment Receipt v2.
#
# Makefile ini memakai syntax GNU Make dan kompatibel dengan Windows
# (dijalankan dari Git Bash, MSYS2, atau WSL). Target dibuat cross-platform:
# path memakai forward slash, command memakai sh (bukan cmd.exe).
#
# Contoh pemakaian:
#   make build        # build backend + frontend
#   make test         # test backend + frontend lint
#   make run          # jalankan backend (muat backend/.env) + frontend dev server
#   make clean        # bersihkan artefak build

# --- Konfigurasi ----------------------------------------------------------
BACKEND_DIR  := backend
FRONTEND_DIR := frontend
DIST_DIR     := $(FRONTEND_DIR)/dist

# --- Target ---------------------------------------------------------------

.PHONY: all build test run lint clean dev-db

all: build

## Build backend (Go binary) dan frontend (Vite production).
build: build-backend build-frontend

build-backend:
	cd $(BACKEND_DIR) && go build ./...

build-frontend:
	cd $(FRONTEND_DIR) && npm run build

## Jalankan test backend + frontend lint.
test: test-backend lint-frontend

test-backend:
	cd $(BACKEND_DIR) && go test ./...

lint-frontend:
	cd $(FRONTEND_DIR) && npm run lint

## Jalankan backend (Go) dan frontend dev server (Vite) untuk development.
## Dua proses berjalan paralel; tekan Ctrl+C untuk menghentikan keduanya.
##
## backend/.env dimuat ke environment SEBELUM go run (set -a mengekspor semua
## variabel hasil source). Backend tidak membaca .env sendiri - tanpa langkah
## ini go run ./cmd/api gagal validasi config (DB_URL, AUTH_JWT_SECRET, dst.).
run:
	set -a; [ ! -f $(BACKEND_DIR)/.env ] || . $(BACKEND_DIR)/.env; set +a; \
		(cd $(BACKEND_DIR) && go run ./cmd/api) & \
		(cd $(FRONTEND_DIR) && npm run dev)

## Format Go code.
fmt:
	cd $(BACKEND_DIR) && gofmt -w .

## Bersihkan artefak build.
clean:
	if exist $(BACKEND_DIR)\bin rmdir /s /q $(BACKEND_DIR)\bin
	if exist $(DIST_DIR) rmdir /s /q $(DIST_DIR)
	if exist $(BACKEND_DIR)\tmp_ai_test rmdir /s /q $(BACKEND_DIR)\tmp_ai_test

## Setup database lokal (migrasi + seed).
## Migrasi dijalankan lewat cmd/migrate (jalur yang sama dengan deploy).
dev-db:
	cd $(BACKEND_DIR) && DB_URL="postgres://paymentreceipt:paymentreceipt_local@localhost:5432/paymentreceipt_v2?sslmode=disable" go run ./cmd/migrate up
	psql -U postgres -d paymentreceipt_v2 -f $(BACKEND_DIR)/db/seeds/development.sql
