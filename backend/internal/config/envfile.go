package config

import (
	"bufio"
	"os"
	"strings"
)

// loadDotEnv membaca file .env lokal (jika ada) dan mengisi os.Setenv untuk variabel
// yang belum diset di environment. Memudahkan eksekusi "go run cmd/api/main.go" langsung.
func loadDotEnv() {
	paths := []string{".env", "../.env", "backend/.env", "../backend/.env"}
	for _, p := range paths {
		f, err := os.Open(p)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			if (strings.HasPrefix(val, "\"") && strings.HasSuffix(val, "\"")) ||
				(strings.HasPrefix(val, "'") && strings.HasSuffix(val, "'")) {
				val = val[1 : len(val)-1]
			}
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
		_ = f.Close()
		break
	}
}
