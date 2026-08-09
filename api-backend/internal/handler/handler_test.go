package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metrix/api-backend/internal/db"
)

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h := http.HandlerFunc(HealthCheck)

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res HealthResponse
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}

	if res.Status != "ok" {
		t.Errorf("expected status ok, got %v", res.Status)
	}
}

func TestProtectedEndpointWithoutAuth(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/metrics/summary", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	h := AuthMiddleware(GetSummary)

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("expected status %v for unauthenticated request, got %v", http.StatusUnauthorized, status)
	}
}

func TestAuthMiddlewareRejectsMissingBearerPrefix(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/v1/metrics/summary", nil)
	req.Header.Set("Authorization", "some-token-without-bearer")

	rr := httptest.NewRecorder()
	h := AuthMiddleware(GetSummary)

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("expected status %v, got %v", http.StatusUnauthorized, status)
	}
}

func TestAuthMiddlewareRejectsInvalidToken(t *testing.T) {
	req, _ := http.NewRequest("GET", "/api/v1/metrics/summary", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-here")

	rr := httptest.NewRecorder()
	h := AuthMiddleware(GetSummary)

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("expected status %v for invalid token, got %v", http.StatusUnauthorized, status)
	}
}

func TestRegisterRequiresEmailAndPassword(t *testing.T) {
	body := bytes.NewBufferString(`{"email":"","password":"","name":"test"}`)
	req, _ := http.NewRequest("POST", "/api/v1/auth/register", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	Register(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestPasswordResetRequestInvalidJSON(t *testing.T) {
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password", bytes.NewBufferString(`{bad json`))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	SendPasswordReset(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestPasswordResetConfirmRequiresFields(t *testing.T) {
	body := bytes.NewBufferString(`{"email":"a@b.com","token":"","new_password":""}`)
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password/confirm", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	ResetPassword(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v, got %v", http.StatusBadRequest, status)
	}
}

func TestPasswordResetConfirmShortPassword(t *testing.T) {
	body := bytes.NewBufferString(`{"email":"a@b.com","token":"sometoken","new_password":"short"}`)
	req, _ := http.NewRequest("POST", "/api/v1/auth/reset-password/confirm", body)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	ResetPassword(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v for short password, got %v", http.StatusBadRequest, status)
	}
}

func TestOAuthConnectRequiresAuth(t *testing.T) {
	body := bytes.NewBufferString(`{"platform":"youtube"}`)
	req, _ := http.NewRequest("POST", "/api/v1/oauth/connect", body)

	rr := httptest.NewRecorder()
	h := AuthMiddleware(GetOAuthConnectURL)

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusUnauthorized {
		t.Errorf("expected status %v for unauthenticated OAuth connect, got %v", http.StatusUnauthorized, status)
	}
}

func TestOAuthConnectRejectsInvalidPlatform(t *testing.T) {
	token := issueTestToken(t)
	body := bytes.NewBufferString(`{"platform":"linkedin"}`)
	req, _ := http.NewRequest("POST", "/api/v1/oauth/connect", body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	h := AuthMiddleware(GetOAuthConnectURL)

	h.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusBadRequest {
		t.Errorf("expected status %v for unsupported platform, got %v", http.StatusBadRequest, status)
	}
}

func issueTestToken(t *testing.T) string {
	t.Helper()
	user := &db.User{
		ID:    "test-user-id",
		Email: "test@example.com",
		Name:  "Test User",
	}
	token, err := generateToken(user)
	if err != nil {
		t.Fatalf("failed to generate test token: %v", err)
	}
	return token
}
