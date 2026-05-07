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
	relationHandler := handler.NewRelationHandler(appCtx)
	videoHandler := handler.NewVideoHandler(appCtx)
	interactHandler := handler.NewInteractHandler(appCtx)
	feedHandler := handler.NewFeedHandler(appCtx)
	messageHandler := handler.NewMessageHandler(appCtx)
	searchHandler := handler.NewSearchHandler(appCtx)

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
		authenticated.PUT("/user/username", userHandler.UpdateUsername)
		authenticated.PUT("/user/profile", userHandler.UpdateProfile)
		authenticated.PUT("/user/password", userHandler.ChangePassword)
		authenticated.GET("/user/:uid", userHandler.GetHomepage)
		authenticated.POST("/relation/action", relationHandler.Action)
		authenticated.GET("/relation/following/:uid", relationHandler.ListFollowing)
		authenticated.GET("/relation/followers/:uid", relationHandler.ListFollowers)
		authenticated.POST("/hashtag", videoHandler.CreateHashtag)
		authenticated.GET("/hashtag/:hid", videoHandler.GetHashtag)
		authenticated.GET("/hashtag/hot", videoHandler.ListHotHashtags)
		authenticated.GET("/hashtag/:hid/videos", videoHandler.ListHashtagVideos)
		authenticated.GET("/feed/following", feedHandler.ListFollowing)
		authenticated.GET("/feed/recommend", feedHandler.ListRecommend)
		authenticated.GET("/search/users", searchHandler.SearchUsers)
		authenticated.GET("/search/hashtags", searchHandler.SearchHashtags)
		authenticated.GET("/search/videos", searchHandler.SearchVideos)
		authenticated.GET("/search/all", searchHandler.SearchAll)
		authenticated.GET("/message/unread-count", messageHandler.GetUnreadCount)
		authenticated.GET("/message/list", messageHandler.List)
		authenticated.POST("/message/read", messageHandler.Read)
		authenticated.POST("/interact/like", interactHandler.LikeVideo)
		authenticated.POST("/interact/favorite", interactHandler.FavoriteVideo)
		authenticated.GET("/interact/favorite/list", interactHandler.ListFavorites)
		authenticated.POST("/interact/comment/publish", interactHandler.PublishComment)
		authenticated.GET("/interact/comment/list", interactHandler.ListComments)
		authenticated.POST("/interact/comment/delete", interactHandler.DeleteComment)
		authenticated.POST("/interact/comment/like", interactHandler.LikeComment)
		authenticated.POST("/draft", videoHandler.SaveDraft)
		authenticated.GET("/draft/:id", videoHandler.GetDraft)
		authenticated.GET("/draft/list", videoHandler.ListDrafts)
		authenticated.DELETE("/draft/:id", videoHandler.DeleteDraft)
		authenticated.POST("/video/upload-credential", videoHandler.CreateUploadCredential)
		authenticated.POST("/video/publish", videoHandler.PublishVideo)
		authenticated.GET("/user/:uid/videos", videoHandler.ListUserVideos)
		authenticated.GET("/video/:vid", videoHandler.GetVideo)
		authenticated.GET("/video/:vid/resources", videoHandler.GetVideoResources)
		authenticated.POST("/video/play/report", videoHandler.ReportPlay)
	}

	return engine
}
