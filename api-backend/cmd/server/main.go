package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"metrix/api-backend/internal/db"
	"metrix/api-backend/internal/handler"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("DATABASE_URL environment variable is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.InitDB(ctx, dbURL); err != nil {
		log.Fatalf("Failed to initialize database pool: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HealthCheck)

	// Auth Endpoints (Public)
	mux.HandleFunc("/api/v1/auth/register", handler.Register)
	mux.HandleFunc("/api/v1/auth/login", handler.Login)
	mux.HandleFunc("/api/v1/auth/reset-password", handler.SendPasswordReset)
	mux.HandleFunc("/api/v1/auth/reset-password/confirm", handler.ResetPassword)

	// OAuth Endpoints
	mux.HandleFunc("/api/v1/oauth/connect", handler.AuthMiddleware(handler.GetOAuthConnectURL))
	mux.HandleFunc("/api/v1/oauth/callback/", handler.OAuthCallback)

	// Auth Endpoints (Protected)
	mux.HandleFunc("/api/v1/auth/me", handler.AuthMiddleware(handler.GetMe))
	mux.HandleFunc("/api/v1/auth/change-password", handler.AuthMiddleware(handler.ChangePassword))
	mux.HandleFunc("/api/v1/auth/account", handler.AuthMiddleware(handler.DeleteAccount))
	mux.HandleFunc("/api/v1/settings/profile", handler.AuthMiddleware(handler.UpdateProfile))

	// Metrics Endpoints (Protected)
	mux.HandleFunc("/api/v1/metrics/summary", handler.AuthMiddleware(handler.GetSummary))
	mux.HandleFunc("/api/v1/metrics/timeseries", handler.AuthMiddleware(handler.GetTimeSeries))
	mux.HandleFunc("/api/v1/metrics/top-content", handler.AuthMiddleware(handler.GetTopContent))

	// Platform & Audience Endpoints (Protected)
	mux.HandleFunc("/api/v1/platform-accounts", handler.AuthMiddleware(handler.PlatformAccountsRouter))
	mux.HandleFunc("/api/v1/platform-accounts/", handler.AuthMiddleware(handler.HandlePlatformAccountAction))
	mux.HandleFunc("/api/v1/audience/insights", handler.AuthMiddleware(handler.GetAudienceInsights))

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
