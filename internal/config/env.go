package config

import (
	"os"
	"strconv"
)

type Env struct {
	Port               int
	ProviderConfigPath string
	Environment        string
}

func LoadEnv() Env {
	return Env{
		Port:               getInt("PORT", 8080),
		ProviderConfigPath: getString("PROVIDER_CONFIG_PATH", "provider_config.example.json"),
		Environment:        getString("APP_ENV", "development"),
	}
}

func getString(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}

	return n
}
