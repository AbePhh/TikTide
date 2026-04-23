package jwt

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	gojwt "github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"

	"github.com/AbePhh/TikTide/backend/pkg/rediskey"
)

// Claims 定义网关使用的 JWT 载荷。
type Claims struct {
	UserID   int64  `json:"uid"`
	Username string `json:"username"`
	gojwt.RegisteredClaims
}

// TokenBlacklist 定义登出后 Token 黑名单能力。
type TokenBlacklist interface {
	Add(ctx context.Context, token string, expiresAt time.Time) error
	Contains(ctx context.Context, token string) (bool, error)
}

// Manager 负责签发和解析 JWT。
type Manager struct {
	secret   []byte
	issuer   string
	audience string
	ttl      time.Duration
}

// NewManager 使用固定密钥创建 JWT 管理器。
func NewManager(secret, issuer, audience string, ttl time.Duration) (*Manager, error) {
	if secret == "" {
		return nil, errors.New("jwt secret is empty")
	}

	return &Manager{
		secret:   []byte(secret),
		issuer:   issuer,
		audience: audience,
		ttl:      ttl,
	}, nil
}

// IssueToken 为指定用户签发 JWT。
func (m *Manager) IssueToken(userID int64, username string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(m.ttl)

	claims := Claims{
		UserID:   userID,
		Username: username,
		RegisteredClaims: gojwt.RegisteredClaims{
			Issuer:    m.issuer,
			Subject:   fmt.Sprintf("%d", userID),
			Audience:  []string{m.audience},
			ExpiresAt: gojwt.NewNumericDate(expiresAt),
			IssuedAt:  gojwt.NewNumericDate(now),
			NotBefore: gojwt.NewNumericDate(now),
			ID:        randomTokenID(),
		},
	}

	token := gojwt.NewWithClaims(gojwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, err
	}

	return signed, expiresAt, nil
}

// ParseToken 校验并解析 JWT。
func (m *Manager) ParseToken(tokenString string) (*Claims, error) {
	token, err := gojwt.ParseWithClaims(tokenString, &Claims{}, func(token *gojwt.Token) (any, error) {
		if token.Method != gojwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secret, nil
	}, gojwt.WithAudience(m.audience), gojwt.WithIssuer(m.issuer))
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// RedisBlacklistStore 将黑名单 Token 存储到 Redis。
type RedisBlacklistStore struct {
	client redis.Cmdable
}

// NewRedisBlacklistStore 创建 Redis 黑名单存储。
func NewRedisBlacklistStore(client redis.Cmdable) *RedisBlacklistStore {
	return &RedisBlacklistStore{client: client}
}

// Add 将 Token 加入黑名单直到过期。
func (s *RedisBlacklistStore) Add(ctx context.Context, token string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil
	}
	return s.client.Set(ctx, rediskey.JWTBlacklist(token), "1", ttl).Err()
}

// Contains 判断 Token 是否已经被拉黑。
func (s *RedisBlacklistStore) Contains(ctx context.Context, token string) (bool, error) {
	result, err := s.client.Exists(ctx, rediskey.JWTBlacklist(token)).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

func randomTokenID() string {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(raw)
}
