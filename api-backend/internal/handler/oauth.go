package handler

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
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

// isDemoMode reports whether the OAuth flow should be bypassed with simulated
// data instead of a real provider authorization. The demo path is the default
// for this portfolio deployment; set DEMO_MODE=false to enable real OAuth.
func isDemoMode() bool {
	return os.Getenv("DEMO_MODE") != "false"
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
	issuedAt, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil || time.Since(time.Unix(issuedAt, 0)) > 10*time.Minute || time.Unix(issuedAt, 0).After(time.Now().Add(time.Minute)) {
		return "", false
	}
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
	Platform    string `json:"platform"`
	DisplayName string `json:"display_name"`
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
	platform := strings.ToLower(strings.TrimSpace(req.Platform))
	cfg, ok := oauthConfig[platform]
	if !ok {
		http.Error(w, `{"error":"Unsupported platform"}`, http.StatusBadRequest)
		return
	}

	// Demo mode bypasses the real OAuth dance: the platform account is created
	// (or reconnected) directly with status "connected" and seeded with
	// simulated history. The real OAuth path below stays intact for when real
	// client credentials are configured.
	if isDemoMode() {
		existing, err := db.GetPlatformAccountForUser(r.Context(), userID, platform)
		if err != nil {
			http.Error(w, `{"error":"Database query failed"}`, http.StatusInternalServerError)
			return
		}
		if existing != nil && existing.Status == "connected" {
			http.Error(w, `{"error":"This platform is already connected"}`, http.StatusConflict)
			return
		}

		displayName := strings.TrimSpace(req.DisplayName)
		if displayName == "" {
			displayName = platformDefaultName(platform)
		}

		if existing != nil {
			if _, err := db.UpdatePlatformAccountStatus(r.Context(), existing.ID, userID, "connected"); err != nil {
				http.Error(w, `{"error":"Failed to reconnect platform"}`, http.StatusInternalServerError)
				return
			}
		} else {
			if _, err := db.CreatePlatformAccountWithStatus(r.Context(), userID, platform, displayName, "connected"); err != nil {
				http.Error(w, `{"error":"Failed to create platform account"}`, http.StatusInternalServerError)
				return
			}
		}

		if err := db.SeedMetricsForPlatform(r.Context(), userID, platform); err != nil {
			http.Error(w, `{"error":"Failed to sync platform data"}`, http.StatusInternalServerError)
			return
		}

		_ = json.NewEncoder(w).Encode(map[string]any{
			"demo":         true,
			"status":       "connected",
			"platform":     platform,
			"display_name": displayName,
		})
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

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = platformDefaultName(platform)
	}

	var accountID string
	if existing != nil {
		accountID = existing.ID
		if displayName != platformDefaultName(platform) {
			if err := db.UpdatePlatformAccountName(r.Context(), existing.ID, userID, displayName); err != nil {
				http.Error(w, `{"error":"Failed to save platform display name"}`, http.StatusInternalServerError)
				return
			}
		}
	} else {
		acc, err := db.CreatePlatformAccountWithStatus(r.Context(), userID, platform, displayName, "pending")
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

	authURL, err := addOAuthParams(cfg.AuthURL, url.Values{
		"client_id":     {cfg.ClientID},
		"redirect_uri":  {cfg.RedirectURI},
		"response_type": {"code"},
		"state":         {state},
		"scope":         {"read"},
	})
	if err != nil {
		http.Error(w, `{"error":"Failed to build authorization URL"}`, http.StatusInternalServerError)
		return
	}

	res := map[string]string{
		"url":   authURL,
		"state": state,
	}
	_ = json.NewEncoder(w).Encode(res)
}

func addOAuthParams(rawURL string, params url.Values) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for key, values := range params {
		q[key] = values
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
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
	account, err := db.GetPlatformAccountByID(r.Context(), accountID)
	if err != nil || account == nil || account.Platform != platform {
		http.Error(w, `{"error":"OAuth state does not match a platform account"}`, http.StatusBadRequest)
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

	if ok, err := db.UpdatePlatformAccountStatusByID(r.Context(), accountID, "connected"); err != nil || !ok {
		http.Error(w, `{"error":"Failed to update platform status"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = fmt.Fprintf(w, `<!doctype html><title>Metrix connected</title><script>if(window.opener){window.opener.postMessage({type:"metrix-oauth-complete",platform:%q},"*");window.close()}</script><p>%s connected. You can close this window.</p>`, platform, platform)
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
