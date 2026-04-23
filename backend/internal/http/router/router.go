package router

import (
	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	"github.com/AbePhh/TikTide/backend/internal/http/handler"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
)

// NewEngine 构建 Gin 路由引擎。
func NewEngine(appCtx *app.Context) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())
	engine.Use(ginmiddleware.CORS(appCtx.Config.CORSAllowedOrigins))

	userHandler := handler.NewUserHandler(appCtx)
	videoHandler := handler.NewVideoHandler(appCtx)

	engine.GET("/healthz", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"code": 0,
			"msg":  "success",
			"data": gin.H{"status": "ok"},
		})
	})
	engine.GET("/swagger", handler.SwaggerHome)
	engine.GET("/swagger/", handler.SwaggerHome)
	engine.GET("/swagger/index.html", handler.SwaggerHome)
	engine.GET("/swagger/openapi.yaml", handler.SwaggerDoc)

	api := engine.Group("/api/v1")
	{
		api.POST("/user/register", userHandler.Register)
		api.POST("/user/login", userHandler.Login)
	}

	authenticated := api.Group("/")
	authenticated.Use(ginmiddleware.Auth(appCtx.JWTManager, appCtx.TokenBlacklist))
	{
		authenticated.POST("/user/logout", userHandler.Logout)
		authenticated.GET("/user/profile", userHandler.GetProfile)
		authenticated.PUT("/user/profile", userHandler.UpdateProfile)
		authenticated.PUT("/user/password", userHandler.ChangePassword)
		authenticated.GET("/user/:uid", userHandler.GetHomepage)
		authenticated.POST("/video/upload-credential", videoHandler.CreateUploadCredential)
		authenticated.POST("/video/publish", videoHandler.PublishVideo)
	}

	return engine
}
