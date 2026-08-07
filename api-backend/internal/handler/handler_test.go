package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHealthCheck(t *testing.T) {
	req, err := http.NewRequest("GET", "/health", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(HealthCheck)

	handler.ServeHTTP(rr, req)

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

func TestLogin(t *testing.T) {
	body := `{"email": "test@metrix.com", "password": "password123"}`
	req, err := http.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(Login)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res AuthResponse
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}

	if res.User.Email != "test@metrix.com" {
		t.Errorf("expected email test@metrix.com, got %v", res.User.Email)
	}
}

func TestGetSummary(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/metrics/summary?workspace_id=workspace-agency", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetSummary)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var res SummaryResponse
	if err := json.NewDecoder(rr.Body).Decode(&res); err != nil {
		t.Fatal(err)
	}

	// Verify dynamic workspace simulation
	if res.TotalReach != 2840500 {
		t.Errorf("expected TotalReach for agency to be 2840500, got %v", res.TotalReach)
	}
}

func TestGetPlatformAccounts(t *testing.T) {
	req, err := http.NewRequest("GET", "/api/v1/platform-accounts", nil)
	if err != nil {
		t.Fatal(err)
	}

	rr := httptest.NewRecorder()
	handler := http.HandlerFunc(GetPlatformAccounts)

	handler.ServeHTTP(rr, req)

	if status := rr.Code; status != http.StatusOK {
		t.Errorf("handler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var accounts []PlatformAccount
	if err := json.NewDecoder(rr.Body).Decode(&accounts); err != nil {
		t.Fatal(err)
	}

	if len(accounts) == 0 {
		t.Error("expected non-empty platform accounts list")
	}
}
