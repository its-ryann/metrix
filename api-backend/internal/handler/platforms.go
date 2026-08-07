package handler

import (
	"encoding/json"
	"net/http"

	"metrix/api-backend/internal/db"
)

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

	_ = json.NewEncoder(w).Encode(accounts)
}
