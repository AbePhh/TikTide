package config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

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
	OSSRegion          string
	OSSEndpoint        string
	OSSDownloadDomain  string
	OSSBucket          string
	OSSAccessKeyID     string
	OSSAccessKeySecret string
	OSSUploadExpire    time.Duration
	OSSReadExpire      time.Duration
	FFmpegPath         string
	FFprobePath        string
	TranscodeWorkDir   string
	TranscodeQueueSize int
	TranscodeWorkers   int
	TranscodeMaxRetry  int
	TranscodeLockTTL   time.Duration
	CORSAllowedOrigins []string
	SearchEnabled          bool
	SearchAddresses        []string
	SearchUsername         string
	SearchPassword         string
	SearchUsersAlias       string
	SearchHashtagsAlias    string
	SearchVideosAlias      string
	SearchUseIK            bool
	SearchBootstrapRebuild bool
}

func Load() (Config, error) {
	loadedEnvPath, err := loadDotEnv()
	if err != nil {
		return Config{}, err
	}

	ttl, err := time.ParseDuration(getEnv("JWT_TTL", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_TTL: %w", err)
	}

	ossUploadExpire, err := time.ParseDuration(getEnv("OSS_UPLOAD_EXPIRE", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("parse OSS_UPLOAD_EXPIRE: %w", err)
	}

	ossReadExpire, err := time.ParseDuration(getEnv("OSS_READ_EXPIRE", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("parse OSS_READ_EXPIRE: %w", err)
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("parse REDIS_DB: %w", err)
	}

	transcodeQueueSize, err := strconv.Atoi(getEnv("TRANSCODE_QUEUE_SIZE", "128"))
	if err != nil {
		return Config{}, fmt.Errorf("parse TRANSCODE_QUEUE_SIZE: %w", err)
	}

	transcodeWorkers, err := strconv.Atoi(getEnv("TRANSCODE_WORKERS", "1"))
	if err != nil {
		return Config{}, fmt.Errorf("parse TRANSCODE_WORKERS: %w", err)
	}

	transcodeMaxRetry, err := strconv.Atoi(getEnv("TRANSCODE_MAX_RETRY", "3"))
	if err != nil {
		return Config{}, fmt.Errorf("parse TRANSCODE_MAX_RETRY: %w", err)
	}

	transcodeLockTTL, err := time.ParseDuration(getEnv("TRANSCODE_LOCK_TTL", "10m"))
	if err != nil {
		return Config{}, fmt.Errorf("parse TRANSCODE_LOCK_TTL: %w", err)
	}

	cfg := Config{
		AppName:            getEnv("APP_NAME", "TikTide"),
		AppEnv:             getEnv("APP_ENV", "dev"),
		HTTPAddr:           getEnv("HTTP_ADDR", ":8080"),
		MySQLDSN:           getEnv("MYSQL_DSN", "tiktide:tiktide@tcp(127.0.0.1:13306)/tiktide?charset=utf8mb4&parseTime=true&loc=Local"),
		RedisAddr:          getEnv("REDIS_ADDR", "127.0.0.1:16379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            redisDB,
		JWTIssuer:          getEnv("JWT_ISSUER", "tiktide-gateway"),
		JWTAudience:        getEnv("JWT_AUDIENCE", "tiktide-web"),
		JWTTTL:             ttl,
		JWTSecret:          getEnv("JWT_SECRET", "tiktide-system"),
		OSSRegion:          getEnv("OSS_REGION", "z0"),
		OSSEndpoint:        getEnv("OSS_ENDPOINT", "https://tiktide.s3.cn-east-1.qiniucs.com"),
		OSSDownloadDomain:  getEnv("OSS_DOWNLOAD_DOMAIN", ""),
		OSSBucket:          getEnv("OSS_BUCKET", "tiktide"),
		OSSAccessKeyID:     getEnv("OSS_ACCESS_KEY_ID", ""),
		OSSAccessKeySecret: getEnv("OSS_ACCESS_KEY_SECRET", ""),
		OSSUploadExpire:    ossUploadExpire,
		OSSReadExpire:      ossReadExpire,
		FFmpegPath:         getEnv("FFMPEG_PATH", "ffmpeg"),
		FFprobePath:        getEnv("FFPROBE_PATH", "ffprobe"),
		TranscodeWorkDir:   getEnv("TRANSCODE_WORK_DIR", os.TempDir()),
		TranscodeQueueSize: transcodeQueueSize,
		TranscodeWorkers:   transcodeWorkers,
		TranscodeMaxRetry:  transcodeMaxRetry,
		TranscodeLockTTL:   transcodeLockTTL,
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173,http://localhost:8081,http://127.0.0.1:8081,http://localhost:8080,http://127.0.0.1:8080")),
		SearchEnabled:          getEnvBool("SEARCH_ENABLED", false),
		SearchAddresses:        splitCSV(getEnv("SEARCH_ELASTIC_ADDRESSES", "http://127.0.0.1:9200")),
		SearchUsername:         getEnv("SEARCH_ELASTIC_USERNAME", ""),
		SearchPassword:         getEnv("SEARCH_ELASTIC_PASSWORD", ""),
		SearchUsersAlias:       getEnv("SEARCH_USERS_ALIAS", "tiktide_users"),
		SearchHashtagsAlias:    getEnv("SEARCH_HASHTAGS_ALIAS", "tiktide_hashtags"),
		SearchVideosAlias:      getEnv("SEARCH_VIDEOS_ALIAS", "tiktide_videos"),
		SearchUseIK:            getEnvBool("SEARCH_USE_IK", false),
		SearchBootstrapRebuild: getEnvBool("SEARCH_BOOTSTRAP_REBUILD", true),
	}

	workingDir, _ := os.Getwd()
	log.Printf(
		"config source: cwd=%s dotenv=%s env_oss_access_key=%s final_oss_access_key=%s",
		workingDir,
		loadedEnvPath,
		maskConfigValue(os.Getenv("OSS_ACCESS_KEY_ID")),
		maskConfigValue(cfg.OSSAccessKeyID),
	)

	return cfg, nil
}

func loadDotEnv() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}

	seen := make(map[string]struct{})
	current := workingDir
	for {
		candidates := []string{
			filepath.Join(current, ".env"),
			filepath.Join(current, "backend", ".env"),
		}
		for _, candidate := range candidates {
			cleaned := filepath.Clean(candidate)
			if _, exists := seen[cleaned]; exists {
				continue
			}
			seen[cleaned] = struct{}{}

			if _, statErr := os.Stat(cleaned); statErr == nil {
				if loadErr := godotenv.Overload(cleaned); loadErr != nil {
					return cleaned, fmt.Errorf("load %s: %w", cleaned, loadErr)
				}
				return cleaned, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", nil
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

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		switch strings.ToLower(strings.TrimSpace(value)) {
		case "1", "true", "yes", "on":
			return true
		case "0", "false", "no", "off":
			return false
		}
	}
	return fallback
}

func maskConfigValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 8 {
		return trimmed
	}
	return trimmed[:4] + "..." + trimmed[len(trimmed)-4:]
}
