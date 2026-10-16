// Package logging menyiapkan logger terstruktur untuk aplikasi.
//
// Format dipilih lewat environment:
//   - text : manusia, enak dibaca saat development
//   - json : mesin, untuk agregator log di production
//
// Level dikontrol lewat LOG_LEVEL supaya bisa dinaikkan tanpa build ulang.
package logging

import (
	"log/slog"
	"os"
	"strings"
	"time"
)

// New membuat logger sesuai level dan format yang diminta.
func New(level, format string) *slog.Logger {
	opts := &slog.HandlerOptions{
		Level: parseLevel(level),
		// AddSource hanya berguna saat debugging - di production menambah
		// source ke setiap baris log jauh lebih mahal tanpa manfaat.
		AddSource:   false,
		ReplaceAttr: replaceAttr,
	}

	var handler slog.Handler
	if strings.EqualFold(format, "json") {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler)

	return logger
}

// Nop mengembalikan logger yang membuang semua output.
//
// Dipakai di test supaya output test tidak tercemar log aplikasi.
func Nop() *slog.Logger {
	return slog.New(slog.NewTextHandler(discard{}, &slog.HandlerOptions{Level: slog.LevelError + 1}))
}

type discard struct{}

func (discard) Write(p []byte) (int, error) { return len(p), nil }

// parseLevel mengubah string level menjadi slog.Level.
func parseLevel(level string) slog.Level {
	switch strings.ToLower(level) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// replaceAttr merapikan beberapa field agar lebih enak dibaca.
func replaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.TimeKey:
		// Waktu dalam UTC dengan presisi detik. Tanpa ini setiap entri bisa
		// berbeda beberapa milidetik hanya karena collecteTime.
		if t, ok := a.Value.Any().(time.Time); ok {
			return slog.String(a.Key, t.UTC().Format(time.RFC3339))
		}
	case slog.LevelKey:
		// Level dalam huruf kecil: lebih ringkas di json dan enak di-grep.
		if l, ok := a.Value.Any().(slog.Level); ok {
			return slog.String(a.Key, strings.ToLower(l.String()))
		}
	}
	return a
}
