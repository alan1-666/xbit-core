package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Service struct {
	Name            string
	Addr            string
	DatabaseDSN     string
	JWTSigningKey   string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	ShutdownTimeout time.Duration
	DevAuthEnabled  bool
	MQTTBrokerURL   string
	MQTTClientID    string
	MQTTUsername    string
	MQTTPassword    string
	MQTTEnabled     bool
	HyperliquidMode string
	HyperliquidURL  string
	ProviderTimeout time.Duration
}

func LoadService(name string, defaultAddr string) (Service, error) {
	accessTTL, err := durationEnv("ACCESS_TOKEN_TTL", 15*time.Minute)
	if err != nil {
		return Service{}, err
	}
	refreshTTL, err := durationEnv("REFRESH_TOKEN_TTL", 30*24*time.Hour)
	if err != nil {
		return Service{}, err
	}
	shutdownTimeout, err := durationEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Service{}, err
	}
	providerTimeout, err := durationEnv("PROVIDER_TIMEOUT", 8*time.Second)
	if err != nil {
		return Service{}, err
	}

	return Service{
		Name:            name,
		Addr:            envOr("SERVICE_ADDR", defaultAddr),
		DatabaseDSN:     strings.TrimSpace(os.Getenv("POSTGRES_DSN")),
		JWTSigningKey:   envOr("JWT_SIGNING_KEY", "dev-only-change-me"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		ShutdownTimeout: shutdownTimeout,
		DevAuthEnabled:  boolEnv("DEV_AUTH_ENABLED", true),
		MQTTBrokerURL:   strings.TrimSpace(os.Getenv("MQTT_BROKER_URL")),
		MQTTClientID:    envOr("MQTT_CLIENT_ID", "xbit-"+name),
		MQTTUsername:    strings.TrimSpace(os.Getenv("MQTT_USERNAME")),
		MQTTPassword:    strings.TrimSpace(os.Getenv("MQTT_PASSWORD")),
		MQTTEnabled:     boolEnv("MQTT_ENABLED", false),
		HyperliquidMode: strings.ToLower(envOr("HYPERLIQUID_PROVIDER_MODE", "local")),
		HyperliquidURL:  envOr("HYPERLIQUID_API_URL", "https://api.hyperliquid.xyz"),
		ProviderTimeout: providerTimeout,
	}, nil
}

func envOr(key string, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("parse %s: %w", key, err)
	}
	return parsed, nil
}

func boolEnv(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}
