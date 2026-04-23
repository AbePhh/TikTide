package middleware

import "github.com/gin-gonic/gin"

// CORS 为网页端提供基础跨域支持。
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAny := false
	originSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		if origin == "*" {
			allowAny = true
			break
		}
		originSet[origin] = struct{}{}
	}

	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")
		if origin != "" && (allowAny || containsOrigin(originSet, origin)) {
			ctx.Header("Access-Control-Allow-Origin", origin)
			ctx.Header("Vary", "Origin")
			ctx.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
			ctx.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			ctx.Header("Access-Control-Allow-Credentials", "true")
		}

		if ctx.Request.Method == "OPTIONS" {
			ctx.AbortWithStatus(204)
			return
		}

		ctx.Next()
	}
}

func containsOrigin(originSet map[string]struct{}, origin string) bool {
	_, ok := originSet[origin]
	return ok
}
