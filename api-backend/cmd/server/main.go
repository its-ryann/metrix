package main

import (
	"log"
	"net/http"
	"os"

	"metrix/api-backend/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HealthCheck)

	// Auth Endpoints
	mux.HandleFunc("/api/v1/auth/register", handler.Register)
	mux.HandleFunc("/api/v1/auth/login", handler.Login)

	// Metrics Endpoints
	mux.HandleFunc("/api/v1/metrics/summary", handler.GetSummary)
	mux.HandleFunc("/api/v1/metrics/timeseries", handler.GetTimeSeries)
	mux.HandleFunc("/api/v1/metrics/top-content", handler.GetTopContent)

	// Platform & Audience Endpoints
	mux.HandleFunc("/api/v1/platform-accounts", handler.GetPlatformAccounts)
	mux.HandleFunc("/api/v1/audience/insights", handler.GetAudienceInsights)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	addr := ":" + port

	log.Printf("Starting api-backend on %s...", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}