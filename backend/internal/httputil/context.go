package httputil

import (
	"context"
	"log/slog"
	"net/http"
)

// ctxKeys milik package ini - TIDAK diekspor sebagai tipe.
//
// Penting: middleware dan httputil harus memakai key yang sama, kalau tidak
// request ID yang dipasang middleware tidak akan terbaca di sini dan field
// "instance" pada Problem Details selalu kosong - bug yang sulit dilacak.
type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyLogger
)

// ContextWithLogger menyimpan logger ke context.
func ContextWithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKeyLogger, l)
}

// LoggerFromContext mengambil logger dari context, atau logger default.
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if v, ok := ctx.Value(ctxKeyLogger).(*slog.Logger); ok {
		return v
	}
	return slog.Default()
}

// ContextWithRequestID menyimpan request ID ke context.
func ContextWithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// RequestIDFromContext membaca request ID dari context.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return v
	}
	return ""
}

// loggerFrom adalah pintasan untuk loggerFrom(r.Context()).
func loggerFrom(r *http.Request) *slog.Logger { return LoggerFromContext(r.Context()) }
