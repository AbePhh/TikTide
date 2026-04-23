package app

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	"github.com/AbePhh/TikTide/backend/internal/user/model"
	userservice "github.com/AbePhh/TikTide/backend/internal/user/service"
	videomodel "github.com/AbePhh/TikTide/backend/internal/video/model"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	ossclient "github.com/AbePhh/TikTide/backend/pkg/oss"
	"github.com/AbePhh/TikTide/backend/pkg/utils"
)

// Context 保存单体后端运行所需的共享依赖。
type Context struct {
	Config         config.Config
	DB             *gorm.DB
	Redis          *redis.Client
	JWTManager     *jwt.Manager
	TokenBlacklist jwt.TokenBlacklist
	UserService    userservice.UserService
	VideoService   videoservice.VideoService
}

// New 创建应用上下文。
func New(cfg config.Config) (*Context, error) {
	db, err := gorm.Open(mysql.Open(cfg.MySQLDSN), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("open mysql: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("get sql db from gorm: %w", err)
	}
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(10)
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping mysql: %w", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	jwtManager, err := jwt.NewManager(cfg.JWTSecret, cfg.JWTIssuer, cfg.JWTAudience, cfg.JWTTTL)
	if err != nil {
		_ = sqlDB.Close()
		_ = redisClient.Close()
		return nil, fmt.Errorf("load jwt manager: %w", err)
	}

	idGenerator, err := utils.NewSnowflakeGenerator(1)
	if err != nil {
		_ = sqlDB.Close()
		_ = redisClient.Close()
		return nil, fmt.Errorf("create id generator: %w", err)
	}

	userRepo := model.NewMySQLRepository(db)
	blocklist := jwt.NewRedisBlacklistStore(redisClient)
	userService := userservice.New(userRepo, idGenerator, jwtManager, blocklist)
	aliyunOSSClient, err := ossclient.NewAliyunClient(cfg)
	if err != nil {
		_ = sqlDB.Close()
		_ = redisClient.Close()
		return nil, fmt.Errorf("create oss client: %w", err)
	}
	videoRepo := videomodel.NewMySQLRepository(db)
	videoService := videoservice.New(videoRepo, aliyunOSSClient, idGenerator, cfg)

	return &Context{
		Config:         cfg,
		DB:             db,
		Redis:          redisClient,
		JWTManager:     jwtManager,
		TokenBlacklist: blocklist,
		UserService:    userService,
		VideoService:   videoService,
	}, nil
}

// NewForTest 创建测试用应用上下文。
func NewForTest(
	cfg config.Config,
	userService userservice.UserService,
	videoService videoservice.VideoService,
	jwtManager *jwt.Manager,
	blocklist jwt.TokenBlacklist,
) *Context {
	return &Context{
		Config:         cfg,
		JWTManager:     jwtManager,
		TokenBlacklist: blocklist,
		UserService:    userService,
		VideoService:   videoService,
	}
}

// Close 关闭应用上下文持有的资源。
func (c *Context) Close() error {
	if c.Redis != nil {
		_ = c.Redis.Close()
	}
	if c.DB != nil {
		sqlDB, err := c.DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
