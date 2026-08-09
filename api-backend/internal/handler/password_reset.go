package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/crypto/bcrypt"

	"metrix/api-backend/internal/db"
)

type PasswordResetRequest struct {
	Email string `json:"email"`
}

type PasswordResetConfirm struct {
	Email       string `json:"email"`
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
}

type PasswordResetResponse struct {
	Message string `json:"message"`
}

func SendPasswordReset(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req PasswordResetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" {
		http.Error(w, `{"error":"Email is required"}`, http.StatusBadRequest)
		return
	}

	user, err := db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		http.Error(w, `{"error":"Database query failed"}`, http.StatusInternalServerError)
		return
	}

	if user == nil {
		_ = json.NewEncoder(w).Encode(PasswordResetResponse{
			Message: "If an account exists for that email, a reset link has been sent.",
		})
		return
	}

	token, err := db.CreatePasswordResetToken(r.Context(), user.ID)
	if err != nil {
		http.Error(w, `{"error":"Failed to create reset token"}`, http.StatusInternalServerError)
		return
	}

	resetLink := fmt.Sprintf("https://app.metrix.com/reset-password?token=%s&email=%s", token.Token, user.Email)

	fmt.Printf("[Metrix Password Reset] User: %s (%s) — Reset Link: %s\n", user.Name, user.Email, resetLink)

	_ = json.NewEncoder(w).Encode(PasswordResetResponse{
		Message: "If an account exists for that email, a reset link has been sent.",
	})
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req PasswordResetConfirm
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}

	if req.Email == "" || req.Token == "" || req.NewPassword == "" {
		http.Error(w, `{"error":"Email, token, and new password are required"}`, http.StatusBadRequest)
		return
	}

	if len(req.NewPassword) < 8 {
		http.Error(w, `{"error":"New password must be at least 8 characters"}`, http.StatusBadRequest)
		return
	}

	user, err := db.GetUserByEmail(r.Context(), req.Email)
	if err != nil || user == nil {
		http.Error(w, `{"error":"Invalid reset token"}`, http.StatusUnauthorized)
		return
	}

	validUser, err := db.GetUserByResetToken(r.Context(), req.Token)
	if err != nil || validUser == nil {
		http.Error(w, `{"error":"Invalid or expired reset token"}`, http.StatusUnauthorized)
		return
	}

	if validUser.ID != user.ID {
		http.Error(w, `{"error":"Invalid reset token"}`, http.StatusUnauthorized)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		http.Error(w, `{"error":"Failed to hash password"}`, http.StatusInternalServerError)
		return
	}

	if err := db.UpdateUserPassword(r.Context(), user.ID, string(hashedPassword)); err != nil {
		http.Error(w, `{"error":"Failed to update password"}`, http.StatusInternalServerError)
		return
	}

	if err := db.InvalidateResetToken(r.Context(), req.Token); err != nil {
		fmt.Printf("[Metrix Password Reset] Warning: failed to invalidate token: %v\n", err)
	}

	_ = json.NewEncoder(w).Encode(map[string]string{"status": "password_reset_success"})
}
