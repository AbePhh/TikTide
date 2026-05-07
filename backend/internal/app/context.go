package app

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	feedservice "github.com/AbePhh/TikTide/backend/internal/feed/service"
	interactmodel "github.com/AbePhh/TikTide/backend/internal/interact/model"
	interactservice "github.com/AbePhh/TikTide/backend/internal/interact/service"
	messagemodel "github.com/AbePhh/TikTide/backend/internal/message/model"
	messageservice "github.com/AbePhh/TikTide/backend/internal/message/service"
	recommendservice "github.com/AbePhh/TikTide/backend/internal/recommend/service"
	relationmodel "github.com/AbePhh/TikTide/backend/internal/relation/model"
	relationservice "github.com/AbePhh/TikTide/backend/internal/relation/service"
	searchmodel "github.com/AbePhh/TikTide/backend/internal/search/model"
	searchservice "github.com/AbePhh/TikTide/backend/internal/search/service"
	usermodel "github.com/AbePhh/TikTide/backend/internal/user/model"
	userservice "github.com/AbePhh/TikTide/backend/internal/user/service"
	videomodel "github.com/AbePhh/TikTide/backend/internal/video/model"
	videoservice "github.com/AbePhh/TikTide/backend/internal/video/service"
	videotranscode "github.com/AbePhh/TikTide/backend/internal/video/transcode"
	"github.com/AbePhh/TikTide/backend/pkg/config"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
	ossclient "github.com/AbePhh/TikTide/backend/pkg/oss"
	"github.com/AbePhh/TikTide/backend/pkg/utils"
)

// Context 保存单体后端运行所需的共享依赖。
type Context struct {
	Config           config.Config
	DB               *gorm.DB
	Redis            *redis.Client
	JWTManager       *jwt.Manager
	TokenBlacklist   jwt.TokenBlacklist
	UserService      userservice.UserService
	RelationService  relationservice.RelationService
	VideoService     videoservice.VideoService
	InteractService  interactservice.InteractService
	FeedService      feedservice.FeedService
	RecommendService recommendservice.RecommendService
	MessageService   messageservice.MessageService
	SearchService    searchservice.SearchService
	TranscodeWorker  *videotranscode.Worker
	cancel           context.CancelFunc
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

	userRepo := usermodel.NewMySQLRepository(db)
	relationRepo := relationmodel.NewMySQLRepository(db)
	blocklist := jwt.NewRedisBlacklistStore(redisClient)
	messageRepo := messagemodel.NewMySQLRepository(db)
	messageSvc := messageservice.New(messageRepo, redisClient)
	relationService := relationservice.New(relationRepo, userRepo, messageSvc)
	userService := userservice.New(userRepo, relationService, idGenerator, jwtManager, blocklist)

	storageClient, err := ossclient.NewQiniuClient(cfg)
	if err != nil {
		_ = sqlDB.Close()
		_ = redisClient.Close()
		return nil, fmt.Errorf("create oss client: %w", err)
	}

	videoRepo := videomodel.NewMySQLRepository(db)
	interactRepo := interactmodel.NewMySQLRepository(db)
	videoService := videoservice.New(videoRepo, storageClient, idGenerator, cfg)
	videoService.SetRedisClient(redisClient)
	interactSvc := interactservice.New(interactRepo, userRepo, videoRepo, nil, idGenerator)
	feedSvc := feedservice.New(redisClient, relationRepo, videoService, userRepo, relationService, interactRepo)
	recommendSvc := recommendservice.New(redisClient, videoRepo, interactRepo, videoService, userRepo, relationService)
	interactSvc = interactservice.New(interactRepo, userRepo, videoRepo, messageSvc, idGenerator)

	var searchSvc searchservice.SearchService
	if cfg.SearchEnabled {
		searchRepo, searchErr := searchmodel.NewElasticRepository(cfg)
		if searchErr != nil {
			_ = sqlDB.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("create search repository: %w", searchErr)
		}
		searchSvc = searchservice.New(searchRepo, userRepo, videoRepo, videoService, relationService)
		if searchErr := searchSvc.Initialize(context.Background()); searchErr != nil {
			_ = sqlDB.Close()
			_ = redisClient.Close()
			return nil, fmt.Errorf("initialize search service: %w", searchErr)
		}
		if cfg.SearchBootstrapRebuild {
			if searchErr := searchSvc.RebuildAll(context.Background()); searchErr != nil {
				_ = sqlDB.Close()
				_ = redisClient.Close()
				return nil, fmt.Errorf("bootstrap rebuild search indexes: %w", searchErr)
			}
		}
	}

	workerCtx, cancel := context.WithCancel(context.Background())
	transcodeWorker := videotranscode.NewWorker(cfg, redisClient, videoService, storageClient, feedSvc, messageSvc)
	videoService.SetTranscodeDispatcher(transcodeWorker)
	transcodeWorker.Start(workerCtx)

	return &Context{
		Config:           cfg,
		DB:               db,
		Redis:            redisClient,
		JWTManager:       jwtManager,
		TokenBlacklist:   blocklist,
		UserService:      userService,
		RelationService:  relationService,
		VideoService:     videoService,
		InteractService:  interactSvc,
		FeedService:      feedSvc,
		RecommendService: recommendSvc,
		MessageService:   messageSvc,
		SearchService:    searchSvc,
		TranscodeWorker:  transcodeWorker,
		cancel:           cancel,
	}, nil
}

// NewForTest 创建测试用应用上下文。
func NewForTest(
	cfg config.Config,
	userService userservice.UserService,
	relationService relationservice.RelationService,
	videoService videoservice.VideoService,
	interactService interactservice.InteractService,
	feedService feedservice.FeedService,
	recommendService recommendservice.RecommendService,
	messageService messageservice.MessageService,
	searchService searchservice.SearchService,
	jwtManager *jwt.Manager,
	blocklist jwt.TokenBlacklist,
) *Context {
	return &Context{
		Config:           cfg,
		JWTManager:       jwtManager,
		TokenBlacklist:   blocklist,
		UserService:      userService,
		RelationService:  relationService,
		VideoService:     videoService,
		InteractService:  interactService,
		FeedService:      feedService,
		RecommendService: recommendService,
		MessageService:   messageService,
		SearchService:    searchService,
	}
}

// Close 关闭应用上下文持有的资源。
func (c *Context) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.TranscodeWorker != nil {
		c.TranscodeWorker.Stop()
	}
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
