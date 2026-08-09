package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"metrix/api-backend/internal/db"
)

func timeframeDays(r *http.Request) (int, error) {
	value := r.URL.Query().Get("timeframe")
	if value == "" {
		return 28, nil
	}
	if len(value) < 2 || value[len(value)-1] != 'd' {
		return 0, strconv.ErrSyntax
	}
	days, err := strconv.Atoi(value[:len(value)-1])
	if err != nil || (days != 7 && days != 28 && days != 90) {
		return 0, strconv.ErrSyntax
	}
	return days, nil
}

type SummaryResponse struct {
	TotalReach    int     `json:"total_reach"`
	ReachDelta    float64 `json:"reach_delta"`
	AvgEngagement float64 `json:"avg_engagement"`
	EngageDelta   float64 `json:"engage_delta"`
	ReachGrowth   int     `json:"reach_growth"`
	GrowthDelta   float64 `json:"growth_delta"`
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

	days, err := timeframeDays(r)
	if err != nil {
		http.Error(w, `{"error":"timeframe must be one of 7d, 28d, or 90d"}`, http.StatusBadRequest)
		return
	}
	platform := r.URL.Query().Get("platform")
	windowEnd := time.Now().Add(24 * time.Hour)
	windowStart := time.Now().AddDate(0, 0, -days+1)
	previousStart := windowStart.AddDate(0, 0, -days)
	current, err := db.GetMetricWindow(r.Context(), userID, platform, windowStart, windowEnd)
	if err != nil {
		log.Printf("metrics summary current window failed for user %s, platform %q: %v", userID, platform, err)
		http.Error(w, `{"error":"Failed to fetch metrics summary"}`, http.StatusInternalServerError)
		return
	}
	previous, err := db.GetMetricWindow(r.Context(), userID, platform, previousStart, windowStart)
	if err != nil {
		log.Printf("metrics summary comparison window failed for user %s, platform %q: %v", userID, platform, err)
		http.Error(w, `{"error":"Failed to fetch metrics summary"}`, http.StatusInternalServerError)
		return
	}
	res := SummaryResponse{
		TotalReach:    current.TotalReach,
		AvgEngagement: current.AvgEngagement,
		ReachGrowth:   current.TotalReach - previous.TotalReach,
		ReachDelta:    percentDelta(float64(current.TotalReach), float64(previous.TotalReach)),
		EngageDelta:   percentDelta(current.AvgEngagement, previous.AvgEngagement),
	}
	res.GrowthDelta = res.ReachDelta

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
	days, err := timeframeDays(r)
	if err != nil {
		http.Error(w, `{"error":"timeframe must be one of 7d, 28d, or 90d"}`, http.StatusBadRequest)
		return
	}

	points, err := db.GetTimeSeriesForUser(r.Context(), userID, platform, days)
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

	days, err := timeframeDays(r)
	if err != nil {
		http.Error(w, `{"error":"timeframe must be one of 7d, 28d, or 90d"}`, http.StatusBadRequest)
		return
	}
	platform := r.URL.Query().Get("platform")
	connected, err := db.GetConnectedPlatforms(r.Context(), userID)
	if err != nil {
		http.Error(w, `{"error":"Failed to fetch platforms"}`, http.StatusInternalServerError)
		return
	}

	var contentItems []db.ContentItem
	if len(connected) > 0 {
		contentItems, err = db.GetContentItemsForUser(r.Context(), userID, platform, time.Now().AddDate(0, 0, -days), 20)
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
