package handler

import (
	"encoding/json"
	"net/http"

	"metrix/api-backend/internal/db"
)

type SummaryResponse struct {
	TotalReach     int     `json:"total_reach"`
	ReachDelta     float64 `json:"reach_delta"`
	AvgEngagement  float64 `json:"avg_engagement"`
	EngageDelta    float64 `json:"engage_delta"`
	FollowerGrowth int     `json:"follower_growth"`
	GrowthDelta    float64 `json:"growth_delta"`
}

type TimeSeriesResponse struct {
	Platform string               `json:"platform"`
	Data     []db.TimeSeriesPoint `json:"data"`
}

type ContentItem struct {
	Title      string  `json:"title"`
	Platform   string  `json:"platform"`
	Engagement float64 `json:"engagement"`
	Reach      int     `json:"reach"`
}

func GetSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	ms, err := db.GetSummaryForUser(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch metrics summary"}`, http.StatusInternalServerError)
		return
	}

	res := SummaryResponse{
		TotalReach:     ms.TotalReach,
		ReachDelta:     12.5,
		AvgEngagement:  ms.AvgEngagement,
		EngageDelta:    2.4,
		FollowerGrowth: ms.FollowerGrowth,
		GrowthDelta:    5.1,
	}

	_ = json.NewEncoder(w).Encode(res)
}

func GetTimeSeries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	platform := r.URL.Query().Get("platform")

	points, err := db.GetTimeSeriesForUser(r.Context(), userID, platform)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch timeseries data"}`, http.StatusInternalServerError)
		return
	}

	res := TimeSeriesResponse{
		Platform: platform,
		Data:     points,
	}

	_ = json.NewEncoder(w).Encode(res)
}

func GetTopContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	res := []ContentItem{
		{Title: "How to Grow on YT in 2026", Platform: "youtube", Engagement: 8.5, Reach: 45000},
		{Title: "Day in the Life of a Creator", Platform: "instagram", Engagement: 5.2, Reach: 12000},
		{Title: "Metrix Alpha Reveal!", Platform: "tiktok", Engagement: 12.1, Reach: 85000},
	}

	_ = json.NewEncoder(w).Encode(res)
}
