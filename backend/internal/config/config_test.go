package config

import (
	"os"
	"strings"
	"testing"
	"time"
)

// setEnv menyetel sekumpulan environment variable untuk satu test.
// t.SetEnv otomatis mengembalikan nilai semula saat test selesai.
func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

// baseEnv konfigurasi minimum yang valid.
func baseEnv() map[string]string {
	return map[string]string{
		"DB_URL":          "postgres://user:pass@localhost:5432/testdb",
		"AUTH_JWT_SECRET": strings.Repeat("a", 32),
	}
}

func TestLoad_Success(t *testing.T) {
	setEnv(t, baseEnv())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil", err)
	}

	if cfg.App.Env != "development" {
		t.Errorf("App.Env = %q, want %q", cfg.App.Env, "development")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Auth.AccessTTL != 15*time.Minute {
		t.Errorf("Auth.AccessTTL = %s, want 15m", cfg.Auth.AccessTTL)
	}
	// Default argon2 harus sesuai rekomendasi OWASP.
	if cfg.Auth.Argon2Memory != 19*1024 {
		t.Errorf("Auth.Argon2Memory = %d, want %d (19 MiB)", cfg.Auth.Argon2Memory, 19*1024)
	}
	if cfg.Auth.Argon2Time != 2 {
		t.Errorf("Auth.Argon2Time = %d, want 2", cfg.Auth.Argon2Time)
	}
}

func TestLoad_RequiredFields(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		wantErr string
	}{
		{
			name:    "DB_URL kosong",
			env:     map[string]string{"DB_URL": "", "AUTH_JWT_SECRET": strings.Repeat("a", 32)},
			wantErr: "DB_URL wajib diisi",
		},
		{
			name:    "JWT secret kosong",
			env:     map[string]string{"DB_URL": "postgres://x", "AUTH_JWT_SECRET": ""},
			wantErr: "AUTH_JWT_SECRET wajib diisi",
		},
		{
			name: "JWT secret pendek di production",
			env: map[string]string{
				"DB_URL": "postgres://x", "AUTH_JWT_SECRET": "pendek",
				"APP_ENV": "production", "AUTH_COOKIE_SECURE": "true",
			},
			wantErr: "minimal 32 byte di production",
		},
		{
			name: "APP_ENV tidak dikenal",
			env: map[string]string{
				"DB_URL": "postgres://x", "AUTH_JWT_SECRET": strings.Repeat("a", 32),
				"APP_ENV": "staging",
			},
			wantErr: "APP_ENV",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnv(t, tt.env)

			_, err := Load()
			if err == nil {
				t.Fatalf("Load() error = nil, want error containing %q", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Load() error = %q, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_ProductionRequiresSecureCookie(t *testing.T) {
	setEnv(t, map[string]string{
		"DB_URL":          "postgres://x",
		"AUTH_JWT_SECRET": strings.Repeat("a", 32),
		"APP_ENV":         "production",
		// AUTH_COOKIE_SECURE sengaja tidak di-set
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error about AUTH_COOKIE_SECURE")
	}
	if !strings.Contains(err.Error(), "AUTH_COOKIE_SECURE") {
		t.Errorf("Load() error = %q, want it to mention AUTH_COOKIE_SECURE", err)
	}
}

func TestLoad_SameSiteNoneRequiresSecure(t *testing.T) {
	setEnv(t, map[string]string{
		"DB_URL":               "postgres://x",
		"AUTH_JWT_SECRET":      strings.Repeat("a", 32),
		"AUTH_COOKIE_SAMESITE": "None",
		"AUTH_COOKIE_SECURE":   "false",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error linking SameSite=None to Secure")
	}
	if !strings.Contains(err.Error(), "SAMESITE") {
		t.Errorf("Load() error = %q, want it to mention SAMESITE", err)
	}
}

func TestLoad_RejectsWeakArgon2(t *testing.T) {
	// Argon2 memory 8 KiB mudah dipecahkan GPU. Loader harus menolak,
	// bukan diam-diam memakai konfigurasi lemah.
	setEnv(t, map[string]string{
		"DB_URL":             "postgres://x",
		"AUTH_JWT_SECRET":    strings.Repeat("a", 32),
		"AUTH_ARGON2_MEMORY": "8",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error about weak argon2 memory")
	}
	if !strings.Contains(err.Error(), "ARGON2_MEMORY") {
		t.Errorf("Load() error = %q, want it to mention ARGON2_MEMORY", err)
	}
}

func TestLoad_PoolBounds(t *testing.T) {
	setEnv(t, map[string]string{
		"DB_URL":          "postgres://x",
		"AUTH_JWT_SECRET": strings.Repeat("a", 32),
		"DB_MIN_CONNS":    "20",
		"DB_MAX_CONNS":    "10",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error when min > max conns")
	}
	if !strings.Contains(err.Error(), "DB_MIN_CONNS") {
		t.Errorf("Load() error = %q, want it to mention DB_MIN_CONNS", err)
	}
}

func TestLoad_ReportsAllErrorsAtOnce(t *testing.T) {
	// Dua masalah sekaligus harus muncul dua-duanya supaya developer tidak
	// perlu start ulang setelah setiap perbaikan.
	setEnv(t, map[string]string{
		"DB_URL":          "",
		"AUTH_JWT_SECRET": "",
		"APP_ENV":         "ngawur",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}

	msg := err.Error()
	for _, want := range []string{"DB_URL", "AUTH_JWT_SECRET", "APP_ENV"} {
		if !strings.Contains(msg, want) {
			t.Errorf("Load() error = %q, want it to mention %s", msg, want)
		}
	}
}

func TestLoad_EnvHelpers(t *testing.T) {
	setEnv(t, map[string]string{
		"DB_URL":              "postgres://x",
		"AUTH_JWT_SECRET":     strings.Repeat("a", 32),
		"SERVER_PORT":         "9999",
		"SERVER_READ_TIMEOUT": "45s",
		"LOG_LEVEL":           "debug",
		"AUTH_COOKIE_SECURE":  "true",
		"APP_TRUSTED_PROXIES": "10.0.0.1, 10.0.0.2 ,",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Port != 9999 {
		t.Errorf("Server.Port = %d, want 9999", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 45*time.Second {
		t.Errorf("Server.ReadTimeout = %s, want 45s", cfg.Server.ReadTimeout)
	}
	if cfg.Log.Level != "debug" {
		t.Errorf("Log.Level = %q, want debug", cfg.Log.Level)
	}
	if !cfg.Auth.CookieSecure {
		t.Error("Auth.CookieSecure = false, want true")
	}
	// Spasi kosong harus dibuang, entri kosong tidak boleh jadi elemen.
	want := []string{"10.0.0.1", "10.0.0.2"}
	if len(cfg.App.TrustedProxies) != len(want) {
		t.Fatalf("TrustedProxies = %v, want %v", cfg.App.TrustedProxies, want)
	}
	for i := range want {
		if cfg.App.TrustedProxies[i] != want[i] {
			t.Errorf("TrustedProxies[%d] = %q, want %q", i, cfg.App.TrustedProxies[i], want[i])
		}
	}
}

func TestLoad_UnparseableNumberFallsBackToDefault(t *testing.T) {
	// Nilai yang tidak bisa di-parse harus jatuh ke default, bukan jadi 0.
	// Ini penting: SERVER_PORT=abc tidak boleh membuat aplikasi listen di port acak
	// atau crash saat parsing.
	setEnv(t, map[string]string{
		"DB_URL":              "postgres://x",
		"AUTH_JWT_SECRET":     strings.Repeat("a", 32),
		"SERVER_PORT":         "bukan-angka",
		"SERVER_READ_TIMEOUT": "45detik",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v, want nil (nilai buruk harus jatuh ke default)", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080 (fallback)", cfg.Server.Port)
	}
	if cfg.Server.ReadTimeout != 15*time.Second {
		t.Errorf("Server.ReadTimeout = %s, want 15s (fallback)", cfg.Server.ReadTimeout)
	}
}

func TestLoad_InvalidEnumIsRejectedNotSilentlyAccepted(t *testing.T) {
	// Berbeda dari number: nilai enum yang tidak dikenal tidak bisa ditebak defaultnya,
	// jadi harus error. LOG_LEVEL=trace kemungkinan besar typo dari info.
	setEnv(t, map[string]string{
		"DB_URL":          "postgres://x",
		"AUTH_JWT_SECRET": strings.Repeat("a", 32),
		"LOG_LEVEL":       "trace",
	})

	_, err := Load()
	if err == nil {
		t.Fatal("Load() error = nil, want error for unknown LOG_LEVEL")
	}
	if !strings.Contains(err.Error(), "LOG_LEVEL") {
		t.Errorf("Load() error = %q, want it to mention LOG_LEVEL", err)
	}
}

func TestConfigPredicates(t *testing.T) {
	cfg := &Config{App: AppConfig{Env: "production"}}
	if !cfg.App.IsProduction() {
		t.Error("IsProduction() = false, want true")
	}
	if cfg.App.IsDevelopment() {
		t.Error("IsDevelopment() = true, want false")
	}

	cfg.App.Env = "development"
	if !cfg.App.IsDevelopment() {
		t.Error("IsDevelopment() = false, want true")
	}
	if cfg.App.IsProduction() {
		t.Error("IsProduction() = true, want false")
	}
}

func TestServerAddr(t *testing.T) {
	s := ServerConfig{Host: "127.0.0.1", Port: 3000}
	if got := s.Addr(); got != "127.0.0.1:3000" {
		t.Errorf("Addr() = %q, want %q", got, "127.0.0.1:3000")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
