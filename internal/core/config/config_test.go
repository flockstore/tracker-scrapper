package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_Defaults verifies that default values are used when env vars are missing.
func TestLoad_Defaults(t *testing.T) {
	os.Unsetenv("APP_ENV")
	os.Unsetenv("LOG_LEVEL")
	os.Unsetenv("SERVER_PORT")

	os.Setenv("LOGTO_M2M_APP_ID", "m2m_default")
	os.Setenv("LOGTO_M2M_APP_SECRET", "secret_default")
	os.Setenv("COURIER_COORDINADORA_CO", "https://coordinadora.test")
	os.Setenv("COURIER_SERVIENTREGA_CO", "https://servientrega.test")
	os.Setenv("COURIER_INTERRAPIDISIMO_CO", "https://interrapidisimo.test")
	os.Setenv("CACHE_REDIS_URL", "redis://localhost:6379")
	defer func() {
		os.Unsetenv("LOGTO_M2M_APP_ID")
		os.Unsetenv("LOGTO_M2M_APP_SECRET")
		os.Unsetenv("COURIER_COORDINADORA_CO")
		os.Unsetenv("COURIER_SERVIENTREGA_CO")
		os.Unsetenv("COURIER_INTERRAPIDISIMO_CO")
		os.Unsetenv("CACHE_REDIS_URL")
	}()

	cfg, err := Load(".")
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "development", cfg.Environment)
	assert.Equal(t, "info", cfg.LogLevel)
	assert.Equal(t, 8080, cfg.ServerPort)
}

// TestLoad_EnvVars verifies that environment variables override defaults.
func TestLoad_EnvVars(t *testing.T) {
	os.Setenv("APP_ENV", "production")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("MANNAIAH_BACKEND_URL", "https://api.example.com")
	os.Setenv("LOGTO_M2M_TOKEN_ENDPOINT", "https://auth.example.com/oidc/token")
	os.Setenv("LOGTO_M2M_APP_ID", "m2m_123")
	os.Setenv("LOGTO_M2M_APP_SECRET", "secret_123")
	os.Setenv("LOGTO_M2M_SCOPE", "order:view contact:view")
	os.Setenv("COURIER_COORDINADORA_CO", "https://coordinadora.test")
	os.Setenv("COURIER_SERVIENTREGA_CO", "https://servientrega.test")
	os.Setenv("COURIER_INTERRAPIDISIMO_CO", "https://interrapidisimo.test")
	os.Setenv("CACHE_REDIS_URL", "redis://localhost:6379")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("MANNAIAH_BACKEND_URL")
		os.Unsetenv("LOGTO_M2M_TOKEN_ENDPOINT")
		os.Unsetenv("LOGTO_M2M_APP_ID")
		os.Unsetenv("LOGTO_M2M_APP_SECRET")
		os.Unsetenv("LOGTO_M2M_SCOPE")
		os.Unsetenv("COURIER_COORDINADORA_CO")
		os.Unsetenv("COURIER_SERVIENTREGA_CO")
		os.Unsetenv("COURIER_INTERRAPIDISIMO_CO")
		os.Unsetenv("CACHE_REDIS_URL")
	}()

	cfg, err := Load(".")
	require.NoError(t, err)

	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 9090, cfg.ServerPort)
	assert.Equal(t, "https://api.example.com", cfg.Mannaiah.BackendURL)
	assert.Equal(t, "https://auth.example.com/oidc/token", cfg.Mannaiah.TokenEndpoint)
	assert.Equal(t, "m2m_123", cfg.Mannaiah.AppID)
	assert.Equal(t, "order:view contact:view", cfg.Mannaiah.Scope)
}

// TestLoad_File verifies that values are loaded from a .env file.
func TestLoad_File(t *testing.T) {
	content := []byte(`
APP_ENV=staging
LOG_LEVEL=warn
SERVER_PORT=7070
MANNAIAH_BACKEND_URL=https://api.staging.example.com
LOGTO_M2M_TOKEN_ENDPOINT=https://auth.staging.example.com/oidc/token
LOGTO_M2M_APP_ID=m2m_staging
LOGTO_M2M_APP_SECRET=secret_staging
COURIER_COORDINADORA_CO=https://coordinadora.test
COURIER_SERVIENTREGA_CO=https://servientrega.test
COURIER_INTERRAPIDISIMO_CO=https://interrapidisimo.test
CACHE_REDIS_URL=redis://localhost:6379
`)
	err := os.WriteFile(".env", content, 0644)
	require.NoError(t, err)
	defer os.Remove(".env")

	cfg, err := Load(".")
	require.NoError(t, err)

	assert.Equal(t, "staging", cfg.Environment)
	assert.Equal(t, "warn", cfg.LogLevel)
	assert.Equal(t, 7070, cfg.ServerPort)
}

// TestLoad_ValidationFailure verifies that missing required fields return an error.
func TestLoad_ValidationFailure(t *testing.T) {
	os.Unsetenv("LOGTO_M2M_APP_ID")
	os.Unsetenv("LOGTO_M2M_APP_SECRET")

	cfg, err := Load(".")
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "missing required configuration")
}
