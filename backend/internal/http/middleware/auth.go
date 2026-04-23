package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/pkg/errno"
	"github.com/AbePhh/TikTide/backend/pkg/jwt"
)

const authContextKey = "auth_info"

// AuthInfo 保存从 JWT 中解析出的认证用户信息。
type AuthInfo struct {
	UserID   int64
	Username string
	Token    string
}

// Auth 校验 JWT，并将认证信息写入 Gin 上下文。
func Auth(manager *jwt.Manager, blocklist jwt.TokenBlacklist) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		rawToken := extractBearerToken(ctx.GetHeader("Authorization"))
		if rawToken == "" {
			writeUnauthorized(ctx)
			return
		}

		blocked, err := blocklist.Contains(ctx.Request.Context(), rawToken)
		if err != nil {
			writeInternal(ctx)
			return
		}
		if blocked {
			writeUnauthorized(ctx)
			return
		}

		claims, err := manager.ParseToken(rawToken)
		if err != nil {
			writeUnauthorized(ctx)
			return
		}

		ctx.Set(authContextKey, AuthInfo{
			UserID:   claims.UserID,
			Username: claims.Username,
			Token:    rawToken,
		})
		ctx.Next()
	}
}

// GetAuthInfo 从 Gin 上下文中读取认证用户信息。
func GetAuthInfo(ctx *gin.Context) (AuthInfo, bool) {
	value, ok := ctx.Get(authContextKey)
	if !ok {
		return AuthInfo{}, false
	}
	authInfo, ok := value.(AuthInfo)
	return authInfo, ok
}

func extractBearerToken(header string) string {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

func writeUnauthorized(ctx *gin.Context) {
	bizErr := errno.ErrUnauthorized
	ctx.AbortWithStatusJSON(bizErr.Status, gin.H{
		"code": bizErr.Code,
		"msg":  bizErr.Message,
	})
}

func writeInternal(ctx *gin.Context) {
	bizErr := errno.ErrInternalRPC
	ctx.AbortWithStatusJSON(bizErr.Status, gin.H{
		"code": bizErr.Code,
		"msg":  bizErr.Message,
	})
}
