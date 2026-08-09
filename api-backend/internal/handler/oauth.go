package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"metrix/api-backend/internal/db"
)

type oauthProviderConfig struct {
	AuthURL     string
	TokenURL    string
	ClientID    string
	RedirectURI string
}

var oauthConfig = map[string]oauthProviderConfig{
	"youtube": {
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		ClientID:    os.Getenv("OAUTH_YOUTUBE_CLIENT_ID"),
		RedirectURI: os.Getenv("OAUTH_YOUTUBE_REDIRECT_URI"),
	},
	"instagram": {
		AuthURL:     "https://api.instagram.com/oauth/authorize",
		TokenURL:    "https://api.instagram.com/oauth/access_token",
		ClientID:    os.Getenv("OAUTH_INSTAGRAM_CLIENT_ID"),
		RedirectURI: os.Getenv("OAUTH_INSTAGRAM_REDIRECT_URI"),
	},
	"tiktok": {
		AuthURL:     "https://open.tiktok.com/platform/oauth/connect/",
		TokenURL:    "https://open.tiktok.com/oauth/access_token/",
		ClientID:    os.Getenv("OAUTH_TIKTOK_CLIENT_KEY"),
		RedirectURI: os.Getenv("OAUTH_TIKTOK_REDIRECT_URI"),
	},
}

var stateSecret = []byte(os.Getenv("OAUTH_STATE_SECRET"))

func init() {
	if len(stateSecret) == 0 {
		stateSecret = []byte("metrix-oauth-state-secret-2026")
	}
}

func generateState(accountID string) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	nonceStr := hex.EncodeToString(nonce)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	payload := fmt.Sprintf("%s:%s:%s", accountID, nonceStr, timestamp)
	mac := hmac.New(sha256.New, stateSecret)
	mac.Write([]byte(payload))
	signature := hex.EncodeToString(mac.Sum(nil))
	state := fmt.Sprintf("%s:%s:%s:%s", accountID, nonceStr, timestamp, signature)
	return state, nil
}

func verifyState(state string) (string, bool) {
	parts := strings.Split(state, ":")
	if len(parts) != 4 {
		return "", false
	}
	accountID, nonceStr, timestamp, signature := parts[0], parts[1], parts[2], parts[3]
	payload := fmt.Sprintf("%s:%s:%s", accountID, nonceStr, timestamp)
	mac := hmac.New(sha256.New, stateSecret)
	mac.Write([]byte(payload))
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expectedSig)) != 1 {
		return "", false
	}
	return accountID, true
}

type GetOAuthConnectURLRequest struct {
	Platform string `json:"platform"`
}

func GetOAuthConnectURL(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	userID := GetUserIDFromContext(r.Context())
	if userID == "" {
		http.Error(w, `{"error":"Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var req GetOAuthConnectURLRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"Invalid JSON payload"}`, http.StatusBadRequest)
		return
	}
	_ = req

	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	cfg, ok := oauthConfig[platform]
	if !ok {
		http.Error(w, `{"error":"Unsupported platform"}`, http.StatusBadRequest)
		return
	}
	if cfg.ClientID == "" || cfg.RedirectURI == "" {
		http.Error(w, `{"error":"OAuth not configured for this platform"}`, http.StatusServiceUnavailable)
		return
	}

	existing, err := db.GetPlatformAccountForUser(r.Context(), userID, platform)
	if err != nil {
		http.Error(w, `{"error":"Database query failed"}`, http.StatusInternalServerError)
		return
	}
	if existing != nil && existing.Status == "connected" {
		http.Error(w, `{"error":"This platform is already connected"}`, http.StatusConflict)
		return
	}

	var accountID string
	if existing != nil {
		accountID = existing.ID
	} else {
		acc, err := db.CreatePlatformAccount(r.Context(), userID, platform, platformDefaultName(platform))
		if err != nil {
			http.Error(w, `{"error":"Failed to create platform account"}`, http.StatusInternalServerError)
			return
		}
		accountID = acc.ID
	}

	state, err := generateState(accountID)
	if err != nil {
		http.Error(w, `{"error":"Failed to generate OAuth state"}`, http.StatusInternalServerError)
		return
	}

	authURL := fmt.Sprintf("%s?client_id=%s&redirect_uri=%s&response_type=code&state=%s&scope=read",
		cfg.AuthURL, cfg.ClientID, cfg.RedirectURI, state)

	res := map[string]string{
		"url":   authURL,
		"state": state,
	}
	_ = json.NewEncoder(w).Encode(res)
}

func OAuthCallback(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	platform := strings.ToLower(strings.TrimPrefix(r.URL.Path, "/api/v1/oauth/callback/"))

	if code == "" || state == "" {
		http.Error(w, `{"error":"Code and state parameters required"}`, http.StatusBadRequest)
		return
	}

	accountID, ok := verifyState(state)
	if !ok {
		http.Error(w, `{"error":"Invalid or expired OAuth state"}`, http.StatusBadRequest)
		return
	}

	cfg, ok := oauthConfig[platform]
	if !ok {
		http.Error(w, `{"error":"Unsupported platform"}`, http.StatusBadRequest)
		return
	}

	tokenResp, expiresAt, err := exchangeCodeForToken(cfg.TokenURL, code, cfg.ClientID, cfg.RedirectURI)
	if err != nil {
		http.Error(w, fmt.Sprintf(`{"error":"Token exchange failed: %s"}`, err.Error()), http.StatusInternalServerError)
		return
	}

	if err := db.UpsertOAuthToken(r.Context(), accountID, tokenResp.AccessToken, tokenResp.RefreshToken, expiresAt); err != nil {
		http.Error(w, `{"error":"Failed to store OAuth token"}`, http.StatusInternalServerError)
		return
	}

	if _, err := db.UpdatePlatformAccountStatus(r.Context(), accountID, "", "connected"); err != nil {
		http.Error(w, `{"error":"Failed to update platform status"}`, http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]string{
		"status":   "connected",
		"platform": platform,
	})
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

func exchangeCodeForToken(tokenURL, code, clientID, redirectURI string) (tokenResponse, time.Time, error) {
	req, err := http.NewRequest("POST", tokenURL, strings.NewReader(fmt.Sprintf(
		"grant_type=authorization_code&code=%s&client_id=%s&redirect_uri=%s",
		code, clientID, redirectURI,
	)))
	if err != nil {
		return tokenResponse{}, time.Time{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return tokenResponse{}, time.Time{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return tokenResponse{}, time.Time{}, fmt.Errorf("token endpoint returned %d", resp.StatusCode)
	}

	var tr tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return tokenResponse{}, time.Time{}, err
	}

	var expiresAt time.Time
	if tr.ExpiresIn > 0 {
		expiresAt = time.Now().Add(time.Duration(tr.ExpiresIn) * time.Second)
	} else {
		expiresAt = time.Now().Add(1 * time.Hour)
	}

	return tr, expiresAt, nil
}
