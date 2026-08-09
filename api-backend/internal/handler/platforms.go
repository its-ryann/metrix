package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"metrix/api-backend/internal/db"
)

type PlatformAccountResponse struct {
	ID          string  `json:"id"`
	Platform    string  `json:"platform"`
	DisplayName string  `json:"display_name"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	Followers   int     `json:"followers"`
	LastSynced  *string `json:"last_synced"`
}

type ConnectPlatformRequest struct {
	Platform    string `json:"platform"`
	DisplayName string `json:"display_name"`
}

var validPlatforms = map[string]bool{
	"youtube":   true,
	"instagram": true,
	"tiktok":    true,
}

// PlatformAccountsRouter dispatches /api/v1/platform-accounts by HTTP method:
// GET lists a user's connected accounts, POST connects a new one.
func PlatformAccountsRouter(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		GetPlatformAccounts(w, r)
	case http.MethodPost:
		ConnectPlatformAccount(w, r)
	default:
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}

func GetPlatformAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	accounts, err := db.GetPlatformAccountsForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch platform accounts"}`, http.StatusInternalServerError)
		return
	}

	res := make([]PlatformAccountResponse, 0, len(accounts))
	for _, acc := range accounts {
		stats, err := db.GetPlatformStats(r.Context(), userID, acc.Platform)
		if err != nil {
			continue
		}
		item := PlatformAccountResponse{
			ID:          acc.ID,
			Platform:    acc.Platform,
			DisplayName: acc.DisplayName,
			Status:      acc.Status,
			CreatedAt:   acc.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
			Followers:   stats.Followers,
		}
		if stats.LastSyncedAt != nil {
			last := stats.LastSyncedAt.UTC().Format("2006-01-02T15:04:05Z")
			item.LastSynced = &last
		}
		res = append(res, item)
	}

	_ = json.NewEncoder(w).Encode(res)
}

func ConnectPlatformAccount(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req ConnectPlatformRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	req.Platform = strings.ToLower(strings.TrimSpace(req.Platform))
	if !validPlatforms[req.Platform] {
		http.Error(w, `{"error":"Unsupported platform"}`, http.StatusBadRequest)
		return
	}

	existing, err := db.GetPlatformAccountForUser(r.Context(), userID, req.Platform)
	if err != nil {
		http.Error(w, `{"error":"Database query failed"}`, http.StatusInternalServerError)
		return
	}
	if existing != nil && existing.Status == "connected" {
		http.Error(w, `{"error":"This platform is already connected"}`, http.StatusConflict)
		return
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = platformDefaultName(req.Platform)
	}

	// Reconnecting a previously disconnected account just flips it back.
	if existing != nil {
		if _, err := db.UpdatePlatformAccountStatus(r.Context(), existing.ID, userID, "connected"); err != nil {
			http.Error(w, `{"error":"Failed to reconnect platform"}`, http.StatusInternalServerError)
			return
		}
		if err := db.SeedMetricsForPlatform(r.Context(), userID, req.Platform); err != nil {
			http.Error(w, `{"error":"Failed to sync platform data"}`, http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "connected", "reconnected": "true"})
		return
	}

	acc, err := db.CreatePlatformAccount(r.Context(), userID, req.Platform, displayName)
	if err != nil {
		http.Error(w, `{"error":"Failed to connect platform"}`, http.StatusInternalServerError)
		return
	}

	if err := db.SeedMetricsForPlatform(r.Context(), userID, req.Platform); err != nil {
		http.Error(w, `{"error":"Failed to sync platform data"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(acc)
}

func platformDefaultName(platform string) string {
	switch platform {
	case "youtube":
		return "My Channel"
	default:
		return "@my_profile"
	}
}

// HandlePlatformAccountAction routes /api/v1/platform-accounts/{id} and
// /api/v1/platform-accounts/{id}/reconnect to disconnect/reconnect.
func HandlePlatformAccountAction(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/platform-accounts/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		http.Error(w, `{"error":"Account id required"}`, http.StatusBadRequest)
		return
	}
	accountID := parts[0]

	switch {
	case r.Method == http.MethodDelete && len(parts) == 1:
		ok, err := db.UpdatePlatformAccountStatus(r.Context(), accountID, userID, "disconnected")
		if err != nil || !ok {
			http.Error(w, `{"error":"Account not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "disconnected"})

	case r.Method == http.MethodPost && len(parts) == 2 && parts[1] == "reconnect":
		ok, err := db.UpdatePlatformAccountStatus(r.Context(), accountID, userID, "connected")
		if err != nil || !ok {
			http.Error(w, `{"error":"Account not found"}`, http.StatusNotFound)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "connected"})

	default:
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
	}
}
