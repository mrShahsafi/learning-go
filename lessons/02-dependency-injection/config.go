package main

import "os"

// Config holds all runtime configuration for the app.
// In Django this is settings.py — a module-level singleton loaded by the framework.
// In Go you own it: a plain struct, populated once at startup, passed explicitly.
type Config struct {
	DatabaseURL string
	Port        string
}

func LoadConfig() Config {
	return Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://localhost:5432/coordeck"),
		Port:        getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
