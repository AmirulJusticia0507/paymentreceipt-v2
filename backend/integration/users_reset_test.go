//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/AmirulJusticia0507/paymentreceipt-v2/backend/internal/auth"
)

// insertUserWithPassword membuat user dengan hash argon2id ASLI dari password
// yang diberikan - berbeda dari placeholder di harness - supaya test bisa
// login sungguhan ke endpoint /auth/login.
func (e *apiTestEnv) insertUserWithPassword(t *testing.T, username, password, role string) int64 {
	t.Helper()
	hash, err := auth.Hash(password, auth.ParamsFrom(e.cfg.Auth))
	if err != nil {
		t.Fatalf("hash password fixture: %v", err)
	}
	var id int64
	if err := e.conn.QueryRow(context.Background(), `
		INSERT INTO users (username, email, password_hash, role)
		VALUES ($1, $2, $3, $4::user_role) RETURNING id`,
		username, username+"@test.example", hash, role,
	).Scan(&id); err != nil {
		t.Fatalf("insert user fixture: %v", err)
	}
	return id
}

// login memanggil /auth/login tanpa token dan mengembalikan hasilnya beserta
// cookie refresh yang diterbitkan (nil kalau login gagal).
func (e *apiTestEnv) login(t *testing.T, username, password string) (*httptest.ResponseRecorder, *http.Cookie) {
	t.Helper()
	rec := e.request(t, http.MethodPost, "/api/v1/auth/login",
		`{"username":"`+username+`","password":"`+password+`"}`, "")
	var refresh *http.Cookie
	for _, c := range rec.Result().Cookies() {
		if c.Name == e.cfg.Auth.RefreshCookieName {
			refresh = c
		}
	}
	return rec, refresh
}

// TestUsers_ListAdminOnly memastikan daftar user hanya menjangkau admin dan
// tidak pernah membocorkan password_hash.
func TestUsers_ListAdminOnly(t *testing.T) {
	e := newAPITestEnv(t)

	rec := e.request(t, http.MethodGet, "/api/v1/users/", "", e.adminToken)
	if rec.Code != http.StatusOK {
		t.Fatalf("admin GET /users = %d, ingin 200 (body: %s)", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "admin-test") {
		t.Fatal("daftar harus memuat user admin fixture")
	}
	if strings.Contains(rec.Body.String(), "password_hash") {
		t.Fatal("daftar user tidak boleh memuat password_hash")
	}

	// Endpoint ini memuat email & riwayat login - viewer harus ditolak.
	viewerToken, _, err := auth.NewTokenManager(e.cfg.Auth).IssueAccess(9999, "viewer-test", auth.RoleViewer)
	if err != nil {
		t.Fatalf("issue viewer token: %v", err)
	}
	rec = e.request(t, http.MethodGet, "/api/v1/users/", "", viewerToken)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("viewer GET /users = %d, ingin 403", rec.Code)
	}
}

// TestUsers_ResetPassword_PasswordBaruBerlakuDanSesiLamaDicabut membuktikan
// alur lengkap reset admin: password baru langsung berlaku untuk login,
// sementara refresh token lama mati sehingga sesi perangkat lain terputus.
func TestUsers_ResetPassword_PasswordBaruBerlakuDanSesiLamaDicabut(t *testing.T) {
	e := newAPITestEnv(t)
	userID := e.insertUserWithPassword(t, "resetuser", "password-lama-123", "ga_staff")

	// Login dengan password lama - bukti akun hidup, sekaligus ambil refresh
	// token yang nanti harus sudah mati.
	loginRec, refreshCookie := e.login(t, "resetuser", "password-lama-123")
	if loginRec.Code != http.StatusOK {
		t.Fatalf("login awal = %d, ingin 200 (body: %s)", loginRec.Code, loginRec.Body.String())
	}
	if refreshCookie == nil {
		t.Fatal("login harus menerbitkan cookie refresh")
	}

	rec := e.postJSON(t, "/api/v1/users/"+strconv.FormatInt(userID, 10)+"/reset-password",
		`{"password":"password-baru-456"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("reset = %d, ingin 200 (body: %s)", rec.Code, rec.Body.String())
	}

	// Refresh token lama sudah dicabut - perangkat lain harus login ulang.
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(refreshCookie)
	refreshRec := httptest.NewRecorder()
	e.handler.ServeHTTP(refreshRec, req)
	if refreshRec.Code != http.StatusUnauthorized {
		t.Fatalf("refresh cookie lama = %d, ingin 401 - sesi lama harus dicabut", refreshRec.Code)
	}

	// Login password lama gagal...
	rec, _ = e.login(t, "resetuser", "password-lama-123")
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("login password lama = %d, ingin 401", rec.Code)
	}
	// ...dan password baru berhasil.
	rec, _ = e.login(t, "resetuser", "password-baru-456")
	if rec.Code != http.StatusOK {
		t.Fatalf("login password baru = %d, ingin 200 (body: %s)", rec.Code, rec.Body.String())
	}
}

// TestUsers_ResetPassword_Validasi memastikan aturan input: password terlalu
// pendek ditolak 422 dengan pesan per-field, user yang tidak ada 404.
func TestUsers_ResetPassword_Validasi(t *testing.T) {
	e := newAPITestEnv(t)
	userID := e.insertUserWithPassword(t, "validuser", "password-lama-123", "viewer")

	rec := e.postJSON(t, "/api/v1/users/"+strconv.FormatInt(userID, 10)+"/reset-password",
		`{"password":"pendek"}`)
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("password pendek = %d, ingin 422 (body: %s)", rec.Code, rec.Body.String())
	}

	rec = e.postJSON(t, "/api/v1/users/999999/reset-password", `{"password":"cukuppanjang123"}`)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("user tidak ada = %d, ingin 404", rec.Code)
	}
}
