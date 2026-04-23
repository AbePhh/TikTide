package rediskey

import (
	"crypto/sha256"
	"encoding/hex"
)

// JWTBlacklist 返回 JWT 黑名单对应的 Redis Key。
func JWTBlacklist(token string) string {
	sum := sha256.Sum256([]byte(token))
	return "jwt:blacklist:" + hex.EncodeToString(sum[:])
}
