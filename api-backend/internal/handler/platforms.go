package handler

import (
	"encoding/json"
	"net/http"
)

type PlatformAccount struct {
	ID          string `json:"id"`
	Platform    string `json:"platform"`
	DisplayName string `json:"display_name"`
	Status      string `json:"status"`
}

func GetPlatformAccounts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// Mock accounts based on workspace (demo purposes)
	workspaceID := r.URL.Query().Get("workspace_id")
	
	var accounts []PlatformAccount
	if workspaceID == "workspace-agency" {
		accounts = []PlatformAccount{
			{ID: "pa-1", Platform: "youtube", DisplayName: "Agency Main YT", Status: "connected"},
			{ID: "pa-2", Platform: "instagram", DisplayName: "@agency_creators", Status: "connected"},
			{ID: "pa-3", Platform: "tiktok", DisplayName: "Agency.Official", Status: "reconnect_required"},
		}
	} else {
		accounts = []PlatformAccount{
			{ID: "pa-4", Platform: "youtube", DisplayName: "Solo Creator YT", Status: "connected"},
			{ID: "pa-5", Platform: "instagram", DisplayName: "@solo_traveler", Status: "connected"},
		}
	}

	json.NewEncoder(w).Encode(accounts)
}
