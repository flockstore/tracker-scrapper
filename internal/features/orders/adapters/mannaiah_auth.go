package adapter

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"tracker-scrapper/internal/core/config"
	"tracker-scrapper/internal/core/logger"

	"go.uber.org/zap"
)

type mannaiahTokenCache struct {
	mu          sync.Mutex
	accessToken string
	expiresAt   time.Time
}

func (a *MannaiahAdapter) bearerToken() (string, error) {
	a.token.mu.Lock()
	defer a.token.mu.Unlock()

	if a.token.accessToken != "" && time.Now().Before(a.token.expiresAt) {
		return a.token.accessToken, nil
	}

	form := url.Values{}
	form.Set("grant_type", "client_credentials")
	form.Set("client_id", a.config.AppID)
	form.Set("client_secret", a.config.AppSecret)
	form.Set("resource", strings.TrimRight(a.config.BackendURL, "/"))
	form.Set("scope", a.config.Scope)

	tokenEndpoint, err := resolveTokenEndpoint(a.config)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest(http.MethodPost, tokenEndpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", fmt.Errorf("failed to create Mannaiah token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	start := time.Now()
	resp, err := a.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to request Mannaiah token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body := readBodyExcerpt(resp.Body)
		logger.Get().Error("Mannaiah M2M token request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("token_endpoint", sanitizedURL(tokenEndpoint)),
			zap.String("body", body),
			zap.Duration("duration", time.Since(start)),
		)
		return "", &MannaiahAPIError{
			Status:   resp.StatusCode,
			Endpoint: "m2m-token",
			Body:     body,
		}
	}

	var token mannaiahTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return "", fmt.Errorf("failed to decode Mannaiah token response: %w", err)
	}

	if token.AccessToken == "" {
		return "", fmt.Errorf("Mannaiah token response did not include access_token")
	}

	expiresIn := time.Duration(token.ExpiresIn) * time.Second
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}

	a.token.accessToken = token.AccessToken
	a.token.expiresAt = time.Now().Add(expiresIn - 15*time.Second)

	return a.token.accessToken, nil
}

func resolveTokenEndpoint(cfg config.MannaiahConfig) (string, error) {
	if strings.TrimSpace(cfg.TokenEndpoint) != "" {
		return strings.TrimSpace(cfg.TokenEndpoint), nil
	}

	endpoint := strings.TrimRight(strings.TrimSpace(cfg.Endpoint), "/")
	if endpoint == "" {
		return "", fmt.Errorf("missing required configuration: LOGTO_M2M_TOKEN_ENDPOINT or LOGTO_M2M_ENDPOINT")
	}

	return endpoint + "/oidc/token", nil
}
