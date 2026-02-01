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

	os.Setenv("WC_URL", "https://default.com")
	os.Setenv("WC_CONSUMER_KEY", "ck_default")
	os.Setenv("WC_CONSUMER_SECRET", "cs_default")
	os.Setenv("COURIER_COORDINADORA_CO", "https://coordinadora.test")
	os.Setenv("COURIER_SERVIENTREGA_CO", "https://servientrega.test")
	os.Setenv("COURIER_INTERRAPIDISIMO_CO", "https://interrapidisimo.test")
	defer func() {
		os.Unsetenv("WC_URL")
		os.Unsetenv("WC_CONSUMER_KEY")
		os.Unsetenv("WC_CONSUMER_SECRET")
		os.Unsetenv("COURIER_COORDINADORA_CO")
		os.Unsetenv("COURIER_SERVIENTREGA_CO")
		os.Unsetenv("COURIER_INTERRAPIDISIMO_CO")
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
	os.Setenv("WC_URL", "https://example.com")
	os.Setenv("WC_CONSUMER_KEY", "ck_123")
	os.Setenv("WC_CONSUMER_SECRET", "cs_123")
	os.Setenv("COURIER_COORDINADORA_CO", "https://coordinadora.test")
	os.Setenv("COURIER_SERVIENTREGA_CO", "https://servientrega.test")
	os.Setenv("COURIER_INTERRAPIDISIMO_CO", "https://interrapidisimo.test")
	defer func() {
		os.Unsetenv("APP_ENV")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("WC_URL")
		os.Unsetenv("WC_CONSUMER_KEY")
		os.Unsetenv("WC_CONSUMER_SECRET")
		os.Unsetenv("COURIER_COORDINADORA_CO")
		os.Unsetenv("COURIER_SERVIENTREGA_CO")
		os.Unsetenv("COURIER_INTERRAPIDISIMO_CO")
	}()

	cfg, err := Load(".")
	require.NoError(t, err)

	assert.Equal(t, "production", cfg.Environment)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, 9090, cfg.ServerPort)
	assert.Equal(t, "https://example.com", cfg.WooCommerce.URL)
	assert.Equal(t, "ck_123", cfg.WooCommerce.ConsumerKey)
}

// TestLoad_File verifies that values are loaded from a .env file.
func TestLoad_File(t *testing.T) {
	content := []byte(`
APP_ENV=staging
LOG_LEVEL=warn
SERVER_PORT=7070
WC_URL=https://staging.example.com
WC_CONSUMER_KEY=ck_staging
WC_CONSUMER_SECRET=cs_staging
COURIER_COORDINADORA_CO=https://coordinadora.test
COURIER_SERVIENTREGA_CO=https://servientrega.test
COURIER_INTERRAPIDISIMO_CO=https://interrapidisimo.test
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
	os.Unsetenv("WC_URL")
	os.Unsetenv("WC_CONSUMER_KEY")
	os.Unsetenv("WC_CONSUMER_SECRET")

	cfg, err := Load(".")
	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "missing required configuration")
}
