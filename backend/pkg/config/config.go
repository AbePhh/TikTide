package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

// Config 保存运行时配置。
type Config struct {
	AppName            string
	AppEnv             string
	HTTPAddr           string
	MySQLDSN           string
	RedisAddr          string
	RedisPassword      string
	RedisDB            int
	JWTIssuer          string
	JWTAudience        string
	JWTTTL             time.Duration
	JWTSecret          string
	CORSAllowedOrigins []string
}

// Load 从环境变量读取配置并应用默认值。
func Load() (Config, error) {
	_ = godotenv.Load(".env")

	ttl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_TTL: %w", err)
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("parse REDIS_DB: %w", err)
	}

	cfg := Config{
		AppName:            getEnv("APP_NAME", "TikTide"),
		AppEnv:             getEnv("APP_ENV", "dev"),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		MySQLDSN:           getEnv("MYSQL_DSN", "tiktide:tiktide@tcp(127.0.0.1:13306)/tiktide?charset=utf8mb4&parseTime=true&loc=Local"),
		RedisAddr:          getEnv("REDIS_ADDR", "127.0.0.1:16379"),
		RedisPassword:      os.Getenv("REDIS_PASSWORD"),
		RedisDB:            redisDB,
		JWTIssuer:          getEnv("JWT_ISSUER", "tiktide-gateway"),
		JWTAudience:        getEnv("JWT_AUDIENCE", "tiktide-web"),
		JWTTTL:             ttl,
		JWTSecret:          getEnv("JWT_SECRET", "tiktide-system"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:8081,http://127.0.0.1:8081,http://localhost:8080,http://127.0.0.1:8080")),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func splitCSV(raw string) []string {
	items := strings.Split(raw, ",")
	values := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed != "" {
			values = append(values, trimmed)
		}
	}
	return values
}
