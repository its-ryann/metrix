package handler

import (
	"encoding/json"
	"net/http"
	"time"
)

type SummaryResponse struct {
	TotalReach     int     `json:"total_reach"`
	ReachDelta     float64 `json:"reach_delta"`
	AvgEngagement  float64 `json:"avg_engagement"`
	EngageDelta    float64 `json:"engage_delta"`
	FollowerGrowth int     `json:"follower_growth"`
	GrowthDelta    float64 `json:"growth_delta"`
}

type TimeSeriesPoint struct {
	Date  string `json:"date"`
	Value int    `json:"value"`
}

type TimeSeriesResponse struct {
	Platform string            `json:"platform"`
	Data     []TimeSeriesPoint `json:"data"`
}

type ContentItem struct {
	Title      string `json:"title"`
	Platform   string `json:"platform"`
	Engagement float64 `json:"engagement"`
	Reach      int    `json:"reach"`
}

func GetSummary(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	workspaceID := r.URL.Query().Get("workspace_id")

	res := SummaryResponse{
		TotalReach:     125400,
		ReachDelta:     12.5,
		AvgEngagement:  4.2,
		EngageDelta:    -0.8,
		FollowerGrowth: 850,
		GrowthDelta:    5.2,
	}

	// Vary mock data for "Agency" workspace to show multi-tenancy
	if workspaceID == "workspace-agency" {
		res.TotalReach = 2840500
		res.FollowerGrowth = 15200
		res.ReachDelta = 18.2
	}

	json.NewEncoder(w).Encode(res)
}

func GetTimeSeries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	now := time.Now()
	data := make([]TimeSeriesPoint, 7)
	for i := 0; i < 7; i++ {
		date := now.AddDate(0, 0, -6+i).Format("2006-01-02")
		data[i] = TimeSeriesPoint{
			Date:  date,
			Value: 1000 + (i * 150) + (time.Now().Nanosecond() % 100),
		}
	}

	res := TimeSeriesResponse{
		Platform: "all",
		Data:     data,
	}

	json.NewEncoder(w).Encode(res)
}

func GetTopContent(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	res := []ContentItem{
		{Title: "How to Grow on YT in 2026", Platform: "youtube", Engagement: 8.5, Reach: 45000},
		{Title: "Day in the Life of a Creator", Platform: "instagram", Engagement: 5.2, Reach: 12000},
		{Title: "Metrix Alpha Reveal!", Platform: "tiktok", Engagement: 12.1, Reach: 85000},
	}

	json.NewEncoder(w).Encode(res)
}
