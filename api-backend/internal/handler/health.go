package handler

import (
	"encoding/json"
	"net/http"
)

type HealthResponse struct {
	Status  string `json:"status"`
	Service string `json:"service"`
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	// Simple CORS header so the local frontend can call it directly
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	res := HealthResponse{
		Status:  "ok",
		Service: "api-backend",
	}

	_ = json.NewEncoder(w).Encode(res)
}