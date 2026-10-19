# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Dua binary dari satu image:
#   - paymentreceipt-api    : server aplikasi (entrypoint default)
#   - paymentreceipt-migrate: menjalankan migrasi skema sebelum aplikasi start
#     (lihat internal/migrator - migrasi di-embed, jadi tidak perlu menyalin
#     berkas SQL terpisah ke image).
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/paymentreceipt-api \
    ./cmd/api \
 && CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/paymentreceipt-migrate \
    ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot

WORKDIR /app
COPY --from=build --chown=nonroot:nonroot /out/paymentreceipt-api /app/paymentreceipt-api
COPY --from=build --chown=nonroot:nonroot /out/paymentreceipt-migrate /app/paymentreceipt-migrate

USER nonroot:nonroot
EXPOSE 8080

ENTRYPOINT ["/app/paymentreceipt-api"]
