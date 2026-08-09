package handler

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"

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

var contentTitlePool = map[string][]string{
	"youtube": {
		"How to Grow on YT in 2026",
		"Behind the Scenes: My Filming Setup",
		"3 Mistakes Killing Your CTR",
		"I Tested 1000 Creators' Hooks",
		"Reacting to My First Video Ever",
	},
	"instagram": {
		"Day in the Life of a Creator",
		"Carousel: 10 Reel Ideas That Work",
		"Golden Hour Photo Dump",
		"Story: Q&A - Your Questions Answered",
		"Reel: One Year of Consistency",
	},
	"tiktok": {
		"Metrix Alpha Reveal!",
		"POV: The Algorithm Finally Found You",
		"Trending Sound Challenge",
		"How I Edit in Under 10 Minutes",
		"Ghosting the Haters: Week 3",
	},
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

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	res := make([]ContentItem, 0, len(connected)*3)
	for _, platform := range connected {
		titles := contentTitlePool[platform]
		if len(titles) == 0 {
			continue
		}
		perm := rng.Perm(len(titles))
		count := 3
		if len(perm) < count {
			count = len(perm)
		}
		for i := 0; i < count; i++ {
			reach := 8000 + rng.Intn(80000)
			engagement := float64(2 + rng.Intn(1100)) / 100
			res = append(res, ContentItem{
				Title:      titles[perm[i]],
				Platform:   platform,
				Engagement: engagement,
				Reach:      reach,
			})
		}
	}

	if len(res) == 0 {
		res = []ContentItem{
			{Title: "Connect a platform to see your top content", Platform: "", Engagement: 0, Reach: 0},
		}
	}

	_ = json.NewEncoder(w).Encode(res)
}
