// Package config memuat konfigurasi aplikasi dari environment variable.
//
// Prinsip: fail-fast. Aplikasi harus menolak start kalau konfigurasi tidak lengkap
// atau tidak masuk akal, bukan start lalu panic saat request pertama datang.
// Semua kesalahan dikumpulkan dulu agar developer melihat semua masalah sekaligus,
// bukan satu-p satu.
package config

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config adalah seluruh konfigurasi aplikasi.
type Config struct {
	App    AppConfig
	Server ServerConfig
	DB     DBConfig
	Auth   AuthConfig
	AI     AIConfig
	Log    LogConfig
}

// AppConfig konfigurasi umum aplikasi.
type AppConfig struct {
	// Environment: development | test | production. Menentukan beberapa default.
	Env string
	// Name dipakai pada log dan header respons.
	Name string
	// BaseURL untuk membangun tautan absolut (mis. ke dokumen PDF).
	BaseURL string
	// TrustedProxies berisi IP/CIDR yang boleh mengatur X-Forwarded-For.
	// Kosong berarti tidak ada proxy yang dipercaya.
	TrustedProxies []string
}

// IsProduction bernilai true bila Env adalah "production".
func (a AppConfig) IsProduction() bool { return a.Env == "production" }

// IsDevelopment bernilai true bila Env adalah "development".
func (a AppConfig) IsDevelopment() bool { return a.Env == "development" }

// ServerConfig konfigurasi HTTP server.
type ServerConfig struct {
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
	// MaxRequestBody batas ukuran body request dalam byte. Mencegah request
	// raksasa menghabiskan memori.
	MaxRequestBody int64
}

// Addr mengembalikan address dalam format "host:port" untuk di-listen.
func (s ServerConfig) Addr() string { return fmt.Sprintf("%s:%d", s.Host, s.Port) }

// DBConfig konfigurasi koneksi PostgreSQL.
type DBConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
	// StatementCacheMode: "describe" (default) | "prepare".
	// "describe" lebih tahan terhadap perubahan skema tanpa restart, dan juga
	// aman untuk lewat PgBouncer mode transaction.
	StatementCacheMode string
}

// AuthConfig konfigurasi autentikasi.
type AuthConfig struct {
	JWTSecret    []byte
	Issuer       string
	AccessTTL    time.Duration
	RefreshTTL   time.Duration
	CookieDomain string
	// CookieSecure harus true di production. Cookie tanpa Secure bisa dikirim
	// lewat HTTP biasa, jadi token bocor kalau ada yang memaksa downgrade ke HTTP.
	CookieSecure   bool
	CookieSameSite string
	// CookieName untuk access token, RefreshCookieName untuk refresh token.
	CookieName        string
	RefreshCookieName string
	// Argon2id parameters. Nilai ini harus sama dengan yang dipakai saat membuat
	// hash - kalau berbeda, verifikasi password akan gagal.
	Argon2Memory  uint32
	Argon2Time    uint32
	Argon2Threads uint8
	Argon2KeyLen  uint32
	Argon2SaltLen uint32
}

// LogConfig konfigurasi logging.
type LogConfig struct {
	Level  string
	Format string // "json" | "text"
}

// AIConfig konfigurasi penyedia AI untuk ekstraksi dokumen (struk/faktur).
//
// Semua nilai bisa di-override lewat environment variable. APIKey wajib diisi
// saat fitur AI dipakai, tapi aplikasi tetap bisa start tanpa agar fitur AI
// boleh di-aktifkan nanti tanpa merusak deployment yang tidak memakainya.
type AIConfig struct {
	// BaseURL endpoint OpenAI-compatible (contoh: https://smartsproxy.clouad/v1).
	BaseURL string
	// APIKey untuk autentikasi ke penyedia AI. Tidak ada default - kosong
	// berarti fitur AI tidak tersedia sampai diisi.
	APIKey string
	// Model default untuk ekstraksi struk/faktur.
	Model string
	// Timeout per permintaan ke penyedia AI.
	Timeout time.Duration
	// MaxRequestBody batas ukuran file unggahan dalam byte.
	MaxRequestBody int64
}

// Load memuat konfigurasi dari environment variable dan memvalidasinya.
//
// Nilai dibaca dari OS environment. Untuk pengembangan lokal, gunakan godotenv
// style loader di cmd/api atau set manual lewat `set` di shell.
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Env:            getEnv("APP_ENV", "development"),
			Name:           getEnv("APP_NAME", "paymentreceipt-v2"),
			BaseURL:        strings.TrimRight(getEnv("APP_BASE_URL", "http://localhost:8080"), "/"),
			TrustedProxies: getEnvList("APP_TRUSTED_PROXIES"),
		},
		Server: ServerConfig{
			Host:            getEnv("SERVER_HOST", "0.0.0.0"),
			Port:            getEnvInt("SERVER_PORT", 8080),
			ReadTimeout:     getEnvDuration("SERVER_READ_TIMEOUT", 15*time.Second),
			WriteTimeout:    getEnvDuration("SERVER_WRITE_TIMEOUT", 30*time.Second),
			IdleTimeout:     getEnvDuration("SERVER_IDLE_TIMEOUT", 60*time.Second),
			ShutdownTimeout: getEnvDuration("SERVER_SHUTDOWN_TIMEOUT", 15*time.Second),
			MaxRequestBody:  getEnvInt64("SERVER_MAX_REQUEST_BODY", 2<<20), // 2 MiB
		},
		DB: DBConfig{
			URL:                os.Getenv("DB_URL"),
			MaxConns:           int32(getEnvInt("DB_MAX_CONNS", 10)),
			MinConns:           int32(getEnvInt("DB_MIN_CONNS", 2)),
			MaxConnLifetime:    getEnvDuration("DB_MAX_CONN_LIFETIME", time.Hour),
			MaxConnIdleTime:    getEnvDuration("DB_MAX_CONN_IDLE_TIME", 30*time.Minute),
			ConnectTimeout:     getEnvDuration("DB_CONNECT_TIMEOUT", 10*time.Second),
			StatementCacheMode: getEnv("DB_STATEMENT_CACHE_MODE", "describe"),
		},
		Auth: AuthConfig{
			JWTSecret:         []byte(os.Getenv("AUTH_JWT_SECRET")),
			Issuer:            getEnv("AUTH_ISSUER", "paymentreceipt-v2"),
			AccessTTL:         getEnvDuration("AUTH_ACCESS_TTL", 15*time.Minute),
			RefreshTTL:        getEnvDuration("AUTH_REFRESH_TTL", 7*24*time.Hour),
			CookieDomain:      getEnv("AUTH_COOKIE_DOMAIN", ""),
			CookieSecure:      getEnvBool("AUTH_COOKIE_SECURE", false),
			CookieSameSite:    getEnv("AUTH_COOKIE_SAMESITE", "Lax"),
			CookieName:        getEnv("AUTH_COOKIE_NAME", "pr_access_token"),
			RefreshCookieName: getEnv("AUTH_REFRESH_COOKIE_NAME", "pr_refresh_token"),
			Argon2Memory:      uint32(getEnvInt("AUTH_ARGON2_MEMORY", 19*1024)), // 19 MiB
			Argon2Time:        uint32(getEnvInt("AUTH_ARGON2_TIME", 2)),
			Argon2Threads:     uint8(getEnvInt("AUTH_ARGON2_THREADS", 1)),
			Argon2KeyLen:      uint32(getEnvInt("AUTH_ARGON2_KEY_LEN", 32)),
			Argon2SaltLen:     uint32(getEnvInt("AUTH_ARGON2_SALT_LEN", 16)),
		},
		Log: LogConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "text"),
		},
		AI: AIConfig{
			BaseURL:        strings.TrimRight(getEnv("AI_BASE_URL", "https://smartsproxy.clouad/v1"), "/"),
			APIKey:         os.Getenv("AI_API_KEY"),
			Model:          getEnv("AI_MODEL_RECEIPT", "gemini-3.8-flash-high"),
			Timeout:        getEnvDuration("AI_TIMEOUT", 60*time.Second),
			MaxRequestBody: getEnvInt64("AI_MAX_REQUEST_BODY", 10<<20), // 10 MiB
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// validate mengecek konfigurasi. Semua error dikumpulkan supaya developer
// bisa memperbaiki semuanya sekaligus.
func (c *Config) validate() error {
	var errs []error

	switch c.App.Env {
	case "development", "test", "production":
	default:
		errs = append(errs, fmt.Errorf("APP_ENV: %q tidak valid (pilihan: development, test, production)", c.App.Env))
	}

	if c.DB.URL == "" {
		errs = append(errs, errors.New("DB_URL wajib diisi"))
	}

	if c.Server.Port < 1 || c.Server.Port > 65535 {
		errs = append(errs, fmt.Errorf("SERVER_PORT: %d di luar rentang 1-65535", c.Server.Port))
	}

	if c.DB.MaxConns < 1 {
		errs = append(errs, fmt.Errorf("DB_MAX_CONNS: harus >= 1, dapat %d", c.DB.MaxConns))
	}
	if c.DB.MinConns < 0 {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS: harus >= 0, dapat %d", c.DB.MinConns))
	}
	if c.DB.MinConns > c.DB.MaxConns {
		errs = append(errs, fmt.Errorf("DB_MIN_CONNS (%d) tidak boleh lebih besar dari DB_MAX_CONNS (%d)",
			c.DB.MinConns, c.DB.MaxConns))
	}

	if len(c.Auth.JWTSecret) == 0 {
		errs = append(errs, errors.New("AUTH_JWT_SECRET wajib diisi"))
	} else if c.App.IsProduction() && len(c.Auth.JWTSecret) < 32 {
		// Secret pendek di produksi berarti token bisa ditebak. 32 byte minimum
		// memberi 256 bit entropi yang jauh melampaui brute force yang masuk akal.
		errs = append(errs, fmt.Errorf("AUTH_JWT_SECRET: minimal 32 byte di production, sekarang %d byte",
			len(c.Auth.JWTSecret)))
	}

	if c.App.IsProduction() && !c.Auth.CookieSecure {
		errs = append(errs, errors.New("AUTH_COOKIE_SECURE: harus true di production, kalau tidak token bisa bocor lewat HTTP"))
	}

	switch strings.ToLower(c.Auth.CookieSameSite) {
	case "lax", "strict", "none":
	default:
		errs = append(errs, fmt.Errorf("AUTH_COOKIE_SAMESITE: %q tidak valid (pilihan: Lax, Strict, None)", c.Auth.CookieSameSite))
	}
	// SameSite=None tanpa Secure ditolak browser modern, jadi cookie akan diam-diam
	// tidak terkirim. Lebih baik gagal saat start daripada debugging nanti.
	if strings.EqualFold(c.Auth.CookieSameSite, "none") && !c.Auth.CookieSecure {
		errs = append(errs, errors.New("AUTH_COOKIE_SAMESITE=None memerlukan AUTH_COOKIE_SECURE=true"))
	}

	if c.Auth.AccessTTL <= 0 {
		errs = append(errs, fmt.Errorf("AUTH_ACCESS_TTL: harus > 0, dapat %s", c.Auth.AccessTTL))
	}
	if c.Auth.RefreshTTL <= c.Auth.AccessTTL {
		errs = append(errs, fmt.Errorf("AUTH_REFRESH_TTL (%s) harus lebih besar dari AUTH_ACCESS_TTL (%s)",
			c.Auth.RefreshTTL, c.Auth.AccessTTL))
	}

	// Argon2 memory dalam KiB. Angkanya dibatasi bawah supaya konfigurasi tidak
	// sengaja diperlemah (mis. 8 KiB, yang mudah dipecahkan GPU).
	if c.Auth.Argon2Memory < 19*1024 {
		errs = append(errs, fmt.Errorf("AUTH_ARGON2_MEMORY: minimal 19456 (19 MiB) sesuai rekomendasi OWASP, dapat %d", c.Auth.Argon2Memory))
	}
	if c.Auth.Argon2Time < 2 {
		errs = append(errs, fmt.Errorf("AUTH_ARGON2_TIME: minimal 2, dapat %d", c.Auth.Argon2Time))
	}
	if c.Auth.Argon2Threads < 1 {
		errs = append(errs, fmt.Errorf("AUTH_ARGON2_THREADS: minimal 1, dapat %d", c.Auth.Argon2Threads))
	}
	if c.Auth.Argon2KeyLen < 16 {
		errs = append(errs, fmt.Errorf("AUTH_ARGON2_KEY_LEN: minimal 16, dapat %d", c.Auth.Argon2KeyLen))
	}
	if c.Auth.Argon2SaltLen < 16 {
		errs = append(errs, fmt.Errorf("AUTH_ARGON2_SALT_LEN: minimal 16, dapat %d", c.Auth.Argon2SaltLen))
	}

	// AI config: timeout harus positif, BaseURL harus valid kalau diset.
	if c.AI.Timeout <= 0 {
		errs = append(errs, fmt.Errorf("AI_TIMEOUT: harus > 0, dapat %s", c.AI.Timeout))
	}
	if c.AI.BaseURL != "" {
		if _, err := url.ParseRequestURI(c.AI.BaseURL); err != nil {
			errs = append(errs, fmt.Errorf("AI_BASE_URL: %q bukan URL yang valid", c.AI.BaseURL))
		}
	}

	switch strings.ToLower(c.Log.Format) {
	case "json", "text":
	default:
		errs = append(errs, fmt.Errorf("LOG_FORMAT: %q tidak valid (pilihan: json, text)", c.Log.Format))
	}

	switch strings.ToLower(c.Log.Level) {
	case "debug", "info", "warn", "error":
	default:
		errs = append(errs, fmt.Errorf("LOG_LEVEL: %q tidak valid (pilihan: debug, info, warn, error)", c.Log.Level))
	}

	if len(errs) > 0 {
		return fmt.Errorf("konfigurasi tidak valid:\n  - %w", errors.Join(errs...))
	}
	return nil
}

// --- helper ---------------------------------------------------------------

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getEnvList(key string) []string {
	raw, ok := os.LookupEnv(key)
	if !ok || raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getEnvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvInt64(key string, fallback int64) int64 {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return fallback
}

func getEnvDuration(key string, fallback time.Duration) time.Duration {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
