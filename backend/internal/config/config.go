// Package config loads runtime configuration from environment variables.
package config

import (
	"log"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Env  string
	Host string
	Port string

	AllowedOrigins []string
	TrustedProxies []string

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	ContactTo    string

	RateLimitPerMinute int
}

func Load() *Config {
	cfg := &Config{
		Env: getEnv("APP_ENV", "development"),
		// Listen on loopback only by default: on EC2 Nginx is the public entry
		// point and port 8080 should never be reachable from the internet.
		// Docker sets HOST=0.0.0.0 so the port mapping works.
		Host: getEnv("HOST", "127.0.0.1"),
		Port: getEnv("PORT", "8080"),

		AllowedOrigins: splitCSV(getEnv("ALLOWED_ORIGINS", "http://localhost:8765")),
		TrustedProxies: splitCSV(getEnv("TRUSTED_PROXIES", "127.0.0.1,::1")),

		SMTPHost:     getEnv("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnvInt("SMTP_PORT", 587),
		SMTPUsername: getEnv("SMTP_USERNAME", ""),
		SMTPPassword: getEnv("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnv("SMTP_FROM", ""),
		ContactTo:    getEnv("CONTACT_TO_EMAIL", "sumaiya.rahim234@gmail.com"),

		RateLimitPerMinute: getEnvInt("RATE_LIMIT_PER_MINUTE", 5),
	}

	if cfg.SMTPFrom == "" {
		cfg.SMTPFrom = cfg.SMTPUsername
	}

	if cfg.Env == "production" {
		if cfg.SMTPUsername == "" || cfg.SMTPPassword == "" {
			log.Fatal("config: SMTP_USERNAME and SMTP_PASSWORD are required in production")
		}
	}

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
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

func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
