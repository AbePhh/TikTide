package model

import "time"

type UserDocument struct {
	ID            string    `json:"id"`
	Username      string    `json:"username"`
	Nickname      string    `json:"nickname"`
	Signature     string    `json:"signature"`
	AvatarURL     string    `json:"avatar_url"`
	Status        int8      `json:"status"`
	FollowerCount int64     `json:"follower_count"`
	FollowCount   int64     `json:"follow_count"`
	WorkCount     int64     `json:"work_count"`
	CreatedAt     time.Time `json:"created_at"`
}

type HashtagDocument struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	UseCount  int64     `json:"use_count"`
	CreatedAt time.Time `json:"created_at"`
}

type VideoDocument struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	UserID          string    `json:"user_id"`
	AuthorUsername  string    `json:"author_username"`
	AuthorNickname  string    `json:"author_nickname"`
	Hashtags        []string  `json:"hashtags"`
	CoverURL        string    `json:"cover_url"`
	PlayCount       int64     `json:"play_count"`
	LikeCount       int64     `json:"like_count"`
	CommentCount    int64     `json:"comment_count"`
	FavoriteCount   int64     `json:"favorite_count"`
	Visibility      int8      `json:"visibility"`
	AuditStatus     int8      `json:"audit_status"`
	TranscodeStatus int8      `json:"transcode_status"`
	CreatedAt       time.Time `json:"created_at"`
}

type SearchHit struct {
	ID         string
	SortValues []any
}

type SearchResult struct {
	Hits       []SearchHit
	NextCursor string
}

type SearchRequest struct {
	Query  string
	Cursor string
	Limit  int
}

