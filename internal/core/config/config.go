package config

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/spf13/viper"
)

// AppConfig holds the configuration for the application.
// Tags used:
// - mapstructure: used by viper to unmarshal
// - default: default value to set if missing
// - required: if "true", error if missing
type AppConfig struct {
	// Environment specifies the runtime environment (e.g., development, production).
	Environment string `mapstructure:"APP_ENV" default:"development"`
	// LogLevel defines the logging verbosity (e.g., debug, info, error).
	LogLevel string `mapstructure:"LOG_LEVEL" default:"info"`
	// ServerPort is the port where the server will listen.
	ServerPort int `mapstructure:"SERVER_PORT" default:"8080"`

	// Database holds the database configuration.
	Database DatabaseConfig `mapstructure:",squash"`

	// WooCommerce holds the WooCommerce API configuration.
	WooCommerce WooCommerceConfig `mapstructure:",squash"`

	// Couriers holds the courier tracking URL configuration.
	Couriers CourierConfig `mapstructure:",squash"`

	// Cache holds the Redis cache configuration.
	Cache CacheConfig `mapstructure:",squash"`
}

// WooCommerceConfig holds the credentials for the WooCommerce Store.
type WooCommerceConfig struct {
	// URL is the base URL of the WooCommerce store.
	URL string `mapstructure:"WC_URL" required:"true"`
	// ConsumerKey is the public key for API access.
	ConsumerKey string `mapstructure:"WC_CONSUMER_KEY" required:"true"`
	// ConsumerSecret is the secret key for API access.
	ConsumerSecret string `mapstructure:"WC_CONSUMER_SECRET" required:"true"`
}

// DatabaseConfig holds database connection details.
type DatabaseConfig struct {
	// Host is the database server hostname.
	Host string `mapstructure:"DB_HOST" default:"localhost"`
	// Port is the database connection port.
	Port int `mapstructure:"DB_PORT" default:"5432"`
}

// CourierConfig holds courier tracking API URLs.
type CourierConfig struct {
	// CoordinadoraURL is the Coordinadora tracking API base URL.
	CoordinadoraURL string `mapstructure:"COURIER_COORDINADORA_CO" required:"true"`
	// ServientregaURL is the Servientrega tracking API base URL.
	ServientregaURL string `mapstructure:"COURIER_SERVIENTREGA_CO" required:"true"`
	// InterrapidisimoURL is the Interrapidisimo tracking API base URL.
	InterrapidisimoURL string `mapstructure:"COURIER_INTERRAPIDISIMO_CO" required:"true"`
}

// CacheConfig holds Redis cache configuration.
type CacheConfig struct {
	// RedisURL is the Redis connection URL (format: redis://[:password@]host[:port][/database]).
	RedisURL string `mapstructure:"CACHE_REDIS_URL" required:"true"`
	// OrderTTL is the TTL in seconds for order cache entries.
	OrderTTL int `mapstructure:"CACHE_ORDER_TTL" default:"3600"`
	// TrackingTTL is the TTL in seconds for tracking cache entries.
	TrackingTTL int `mapstructure:"CACHE_TRACKING_TTL" default:"1800"`
}

// Load loads configuration from .env files and environment variables.
func Load(path string) (*AppConfig, error) {
	v := viper.New()

	v.AutomaticEnv()

	v.AddConfigPath(path)
	v.SetConfigName(".env")
	v.SetConfigType("env")

	if err := v.ReadInConfig(); err != nil {
		var configFileNotFoundError viper.ConfigFileNotFoundError
		if !errors.As(err, &configFileNotFoundError) {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config AppConfig

	if err := processTags(v, &config); err != nil {
		return nil, err
	}

	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode into struct: %w", err)
	}

	if err := validateRequired(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// processTags iterates over the struct fields and sets default values in Viper.
// processTags iterates over the struct fields and sets default values in Viper.
func processTags(v *viper.Viper, config interface{}) error {
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	t := val.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Type.Kind() == reflect.Struct {
			if err := processTags(v, val.Field(i).Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		key := field.Tag.Get("mapstructure")
		defaultValue := field.Tag.Get("default")

		if key != "" {
			v.BindEnv(key)
		}

		if key != "" && defaultValue != "" {
			v.SetDefault(key, defaultValue)
		}
	}
	return nil
}

// validateRequired checks if fields marked as required have non-zero values.
func validateRequired(config interface{}) error {
	val := reflect.ValueOf(config)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	t := val.Type()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		if field.Type.Kind() == reflect.Struct {
			if err := validateRequired(val.Field(i).Addr().Interface()); err != nil {
				return err
			}
			continue
		}

		required := field.Tag.Get("required")
		if required == "true" {
			value := val.Field(i)
			if isZero(value) {
				key := field.Tag.Get("mapstructure")
				return fmt.Errorf("missing required configuration: %s", key)
			}
		}
	}
	return nil
}

// isZero checks if a reflect.Value is the zero value for its type.
func isZero(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.String:
		return v.String() == ""
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Slice, reflect.Map:
		return v.Len() == 0
	default:
		return v.IsZero()
	}
}
