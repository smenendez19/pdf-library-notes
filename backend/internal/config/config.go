package config

import (
	"os"
	"strconv"
	"strings"
)

const (
	DefaultPort           = "8080"
	DefaultStorageRoot    = "./data/files"
	DefaultDatabasePath   = "./data/catalog.db"
	DefaultAllowedOrigins = "*"
	DefaultMaxUploadBytes = 100 << 20 // 100 MiB
)

type Config struct {
	Port           string
	StorageRoot    string
	DatabasePath   string
	AllowedOrigins []string
	MaxUploadBytes int64
}

func Load() Config {
	return Config{
		Port:           envOr(DefaultPort, "PORT"),
		StorageRoot:    envOr(DefaultStorageRoot, "STORAGE_ROOT"),
		DatabasePath:   envOr(DefaultDatabasePath, "DATABASE_PATH"),
		AllowedOrigins: parseOrigins(envOr(DefaultAllowedOrigins, "ALLOWED_ORIGINS")),
		MaxUploadBytes: parseBytes(envOr(strconv.FormatInt(DefaultMaxUploadBytes, 10), "MAX_UPLOAD_BYTES")),
	}
}

func envOr(def string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			return v
		}
	}
	return def
}

func parseOrigins(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	if len(out) == 0 {
		return []string{DefaultAllowedOrigins}
	}
	return out
}

func parseBytes(s string) int64 {
	s = strings.TrimSpace(s)
	if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 {
		return n
	}
	return DefaultMaxUploadBytes
}
