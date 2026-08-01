package main

import (
	"log"
	"net/http"

	"metrix/api-backend/internal/handler"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", handler.HealthCheck)

	port := ":8080"
	log.Printf("Starting api-backend on %s...", port)
	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}