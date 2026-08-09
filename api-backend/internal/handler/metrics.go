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

func percentDelta(current, previous float64) float64 {
	if previous == 0 {
		return 0
	}
	return ((current - previous) / previous) * 100
}

func GetSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	history, err := db.GetSummaryHistory(r.Context(), userID, 2)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch metrics summary"}`, http.StatusInternalServerError)
		return
	}
	if len(history) == 0 {
		res := SummaryResponse{}
		_ = json.NewEncoder(w).Encode(res)
		return
	}

	latest := history[0]
	res := SummaryResponse{
		TotalReach:     latest.TotalReach,
		AvgEngagement:  latest.AvgEngagement,
		FollowerGrowth: latest.FollowerGrowth,
	}

	if len(history) > 1 {
		prev := history[1]
		res.ReachDelta = percentDelta(float64(latest.TotalReach), float64(prev.TotalReach))
		res.EngageDelta = percentDelta(latest.AvgEngagement, prev.AvgEngagement)
		res.GrowthDelta = percentDelta(float64(latest.FollowerGrowth), float64(prev.FollowerGrowth))
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

	connected, err := db.GetConnectedPlatforms(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch platforms"}`, http.StatusInternalServerError)
		return
	}

	var contentItems []db.ContentItem
	if len(connected) > 0 {
		contentItems, err = db.GetContentItemsForUser(r.Context(), userID, 20)
		if err != nil {
			http.Error(w, `{"error":"Failed to fetch content items"}`, http.StatusInternalServerError)
			return
		}
	}

	if len(contentItems) == 0 {
		contentItems = []db.ContentItem{
			{Title: "Connect a platform to see your top content", Platform: "", Engagement: 0, Reach: 0},
		}
	}

	_ = json.NewEncoder(w).Encode(contentItems)
}
