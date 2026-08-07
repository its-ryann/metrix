package handler

import (
	"encoding/json"
	"net/http"
)

type AudienceInsight struct {
	Category string  `json:"category"`
	Label    string  `json:"label"`
	Value    float64 `json:"value"`
}

type AudienceResponse struct {
	Demographics []AudienceInsight `json:"demographics"`
	Geography    []AudienceInsight `json:"geography"`
}

func GetAudienceInsights(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	res := AudienceResponse{
		Demographics: []AudienceInsight{
			{Category: "age", Label: "18-24", Value: 25.0},
			{Category: "age", Label: "25-34", Value: 45.0},
			{Category: "age", Label: "35-44", Value: 20.0},
			{Category: "gender", Label: "Male", Value: 52.0},
			{Category: "gender", Label: "Female", Value: 48.0},
		},
		Geography: []AudienceInsight{
			{Category: "country", Label: "USA", Value: 40.0},
			{Category: "country", Label: "UK", Value: 15.0},
			{Category: "country", Label: "Canada", Value: 10.0},
		},
	}

	json.NewEncoder(w).Encode(res)
}
