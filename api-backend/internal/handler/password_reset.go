package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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

const passwordResetMessage = "If an account exists for that email, a reset link has been sent."

type resendEmailRequest struct {
	From    string   `json:"from"`
	To      []string `json:"to"`
	Subject string   `json:"subject"`
	HTML    string   `json:"html"`
}

func sendPasswordResetEmail(to, resetLink string) error {
	apiKey := os.Getenv("RESEND_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("RESEND_API_KEY is not configured")
	}

	from := os.Getenv("EMAIL_FROM")
	if from == "" {
		// Resend permits this sender for testing to the account owner's address.
		from = "Metrix <onboarding@resend.dev>"
	}

	payload, err := json.Marshal(resendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: "Reset your Metrix password",
		HTML:    fmt.Sprintf(`<p>We received a request to reset your Metrix password.</p><p><a href="%s">Reset your password</a></p><p>This link expires in one hour. If you did not request it, you can safely ignore this email.</p>`, html.EscapeString(resetLink)),
	})
	if err != nil {
		return fmt.Errorf("encode email payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, "https://api.resend.com/emails", bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create resend request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: 10 * time.Second}).Do(req)
	if err != nil {
		return fmt.Errorf("send reset email: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("resend returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
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
			Message: passwordResetMessage,
		})
		return
	}

	token, err := db.CreatePasswordResetToken(r.Context(), user.ID)
	if err != nil {
		http.Error(w, `{"error":"Failed to create reset token"}`, http.StatusInternalServerError)
		return
	}

	appURL := strings.TrimSuffix(os.Getenv("PUBLIC_APP_URL"), "/")
	if appURL == "" {
		appURL = "https://app.metrix.com"
	}
	resetLink := fmt.Sprintf("%s/?token=%s&email=%s", appURL, url.QueryEscape(token.Token), url.QueryEscape(user.Email))

	if err := sendPasswordResetEmail(user.Email, resetLink); err != nil {
		log.Printf("[Metrix Password Reset] failed to send reset email for user %s: %v", user.ID, err)
		http.Error(w, `{"error":"Unable to send reset email. Please try again later."}`, http.StatusBadGateway)
		return
	}

	_ = json.NewEncoder(w).Encode(PasswordResetResponse{
		Message: passwordResetMessage,
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
