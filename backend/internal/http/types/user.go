package types

// RegisterRequest 表示注册接口请求体。
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginRequest 表示登录接口请求体。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UpdateProfileRequest 表示资料更新接口请求体。
type UpdateProfileRequest struct {
	Nickname  *string `json:"nickname"`
	AvatarURL *string `json:"avatar_url"`
	Signature *string `json:"signature"`
	Gender    *int8   `json:"gender"`
	Birthday  *string `json:"birthday"`
}

// ChangePasswordRequest 表示修改密码接口请求体。
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

type UpdateUsernameRequest struct {
	Username string `json:"username"`
}

// RelationActionRequest 表示关注操作请求体。
type RelationActionRequest struct {
	ToUserID   FlexibleInt64 `json:"to_user_id"`
	ActionType int8          `json:"action_type"`
}

// APIResponse 表示统一 JSON 响应结构。
type APIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// LoginData 表示登录成功后的响应体。
type LoginData struct {
	Token     string      `json:"token"`
	ExpiresAt string      `json:"expires_at"`
	User      ProfileData `json:"user"`
}

// ProfileData 表示用户资料接口返回结构。
type ProfileData struct {
	ID              int64  `json:"id,string"`
	Username        string `json:"username"`
	Nickname        string `json:"nickname"`
	AvatarURL       string `json:"avatar_url"`
	Signature       string `json:"signature"`
	Gender          int8   `json:"gender"`
	Birthday        string `json:"birthday,omitempty"`
	Status          int8   `json:"status"`
	FollowCount     int64  `json:"follow_count"`
	FollowerCount   int64  `json:"follower_count"`
	TotalLikedCount int64  `json:"total_liked_count"`
	WorkCount       int64  `json:"work_count"`
	FavoriteCount   int64  `json:"favorite_count"`
	IsFollowed      bool   `json:"is_followed"`
	IsMutual        bool   `json:"is_mutual"`
	CreatedAt       string `json:"created_at"`
}

// RelationActionData 表示关注操作响应。
type RelationActionData struct {
	ToUserID   int64  `json:"to_user_id,string"`
	ActionType int8   `json:"action_type"`
	IsFollowed bool   `json:"is_followed"`
	IsMutual   bool   `json:"is_mutual"`
	FollowedAt string `json:"followed_at,omitempty"`
}

// RelationUserData 表示关注列表中的用户项。
type RelationUserData struct {
	ID            int64  `json:"id,string"`
	Username      string `json:"username"`
	Nickname      string `json:"nickname"`
	AvatarURL     string `json:"avatar_url"`
	Signature     string `json:"signature"`
	Gender        int8   `json:"gender"`
	Status        int8   `json:"status"`
	FollowCount   int64  `json:"follow_count"`
	FollowerCount int64  `json:"follower_count"`
	IsFollowed    bool   `json:"is_followed"`
	IsMutual      bool   `json:"is_mutual"`
	CreatedAt     string `json:"created_at"`
}

// RelationUserListData 表示关注列表响应。
type RelationUserListData struct {
	Items      []RelationUserData `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

// UploadCredentialRequest 表示获取上传凭证接口请求体。
type UploadCredentialRequest struct {
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	ObjectKey   string `json:"object_key"`
}

// UploadCredentialData 表示上传凭证响应结构。
type UploadCredentialData struct {
	ObjectKey    string `json:"object_key"`
	UploadURL    string `json:"upload_url"`
	UploadMethod string `json:"upload_method"`
	ContentType  string `json:"content_type"`
	ExpiresAt    string `json:"expires_at"`
	UploadToken  string `json:"upload_token"`
}

// UploadFileData 表示服务端代理上传完成后的响应结构。
type UploadFileData struct {
	ObjectKey string `json:"object_key"`
}

// PublishVideoRequest 表示发布视频请求体。
type PublishVideoRequest struct {
	ObjectKey    string   `json:"object_key"`
	Title        string   `json:"title"`
	HashtagIDs   []int64  `json:"hashtag_ids"`
	HashtagNames []string `json:"hashtag_names"`
	AllowComment int8     `json:"allow_comment"`
	Visibility   int8     `json:"visibility"`
}

// PublishVideoData 表示发布视频响应结构。
type PublishVideoData struct {
	VideoID         int64  `json:"video_id,string"`
	ObjectKey       string `json:"object_key"`
	SourceURL       string `json:"source_url"`
	TranscodeStatus int8   `json:"transcode_status"`
}

type VideoDetailData struct {
	VideoID             int64  `json:"video_id,string"`
	UserID              int64  `json:"user_id,string"`
	Title               string `json:"title"`
	ObjectKey           string `json:"object_key"`
	SourceURL           string `json:"source_url"`
	CoverURL            string `json:"cover_url"`
	DurationMS          int32  `json:"duration_ms"`
	AllowComment        int8   `json:"allow_comment"`
	Visibility          int8   `json:"visibility"`
	TranscodeStatus     int8   `json:"transcode_status"`
	AuditStatus         int8   `json:"audit_status"`
	TranscodeFailReason string `json:"transcode_fail_reason"`
	AuditRemark         string `json:"audit_remark"`
	PlayCount           int64  `json:"play_count"`
	LikeCount           int64  `json:"like_count"`
	CommentCount        int64  `json:"comment_count"`
	FavoriteCount       int64  `json:"favorite_count"`
	CreatedAt           string `json:"created_at"`
	UpdatedAt           string `json:"updated_at"`
}

type VideoResourceData struct {
	VideoID    int64  `json:"video_id,string"`
	Resolution string `json:"resolution"`
	FileURL    string `json:"file_url"`
	FileSize   int64  `json:"file_size"`
	Bitrate    int32  `json:"bitrate"`
	CreatedAt  string `json:"created_at"`
}

type VideoResourceListData struct {
	Items []VideoResourceData `json:"items"`
}

type VideoPlayReportRequest struct {
	VideoID FlexibleInt64 `json:"video_id"`
}

type UserVideoListData struct {
	Items      []VideoDetailData `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

// SaveDraftRequest 表示保存草稿请求体。
type SaveDraftRequest struct {
	DraftID      FlexibleInt64 `json:"draft_id"`
	ObjectKey    string        `json:"object_key"`
	CoverURL     string        `json:"cover_url"`
	Title        string        `json:"title"`
	TagNames     string        `json:"tag_names"`
	AllowComment int8          `json:"allow_comment"`
	Visibility   int8          `json:"visibility"`
}

// DraftData 表示草稿响应结构。
type DraftData struct {
	ID           int64  `json:"id,string"`
	ObjectKey    string `json:"object_key"`
	SourceURL    string `json:"source_url"`
	CoverURL     string `json:"cover_url"`
	Title        string `json:"title"`
	TagNames     string `json:"tag_names"`
	AllowComment int8   `json:"allow_comment"`
	Visibility   int8   `json:"visibility"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

// DraftListData 表示草稿列表响应结构。
type DraftListData struct {
	Items []DraftData `json:"items"`
}

// HashtagData 表示话题详情响应结构。
type HashtagData struct {
	ID        int64  `json:"id,string"`
	Name      string `json:"name"`
	UseCount  int64  `json:"use_count"`
	CreatedAt string `json:"created_at"`
}

type HashtagListData struct {
	Items []HashtagData `json:"items"`
}

// CreateHashtagRequest 表示创建话题请求体。
type CreateHashtagRequest struct {
	Name string `json:"name"`
}

// HashtagVideoData 表示话题下视频项。
type HashtagVideoData struct {
	VideoID         int64  `json:"video_id,string"`
	UserID          int64  `json:"user_id,string"`
	Title           string `json:"title"`
	ObjectKey       string `json:"object_key"`
	SourceURL       string `json:"source_url"`
	CoverURL        string `json:"cover_url"`
	Visibility      int8   `json:"visibility"`
	TranscodeStatus int8   `json:"transcode_status"`
	AuditStatus     int8   `json:"audit_status"`
	CreatedAt       string `json:"created_at"`
}

// HashtagVideoListData 表示话题视频列表响应结构。
type HashtagVideoListData struct {
	Items      []HashtagVideoData `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type FeedVideoData struct {
	VideoID             int64            `json:"video_id,string"`
	UserID              int64            `json:"user_id,string"`
	Title               string           `json:"title"`
	ObjectKey           string           `json:"object_key"`
	SourceURL           string           `json:"source_url"`
	CoverURL            string           `json:"cover_url"`
	DurationMS          int32            `json:"duration_ms"`
	AllowComment        int8             `json:"allow_comment"`
	Visibility          int8             `json:"visibility"`
	TranscodeStatus     int8             `json:"transcode_status"`
	AuditStatus         int8             `json:"audit_status"`
	TranscodeFailReason string           `json:"transcode_fail_reason"`
	AuditRemark         string           `json:"audit_remark"`
	PlayCount           int64            `json:"play_count"`
	LikeCount           int64            `json:"like_count"`
	CommentCount        int64            `json:"comment_count"`
	FavoriteCount       int64            `json:"favorite_count"`
	CreatedAt           string           `json:"created_at"`
	UpdatedAt           string           `json:"updated_at"`
	Author              FeedAuthorData   `json:"author"`
	Interact            FeedInteractData `json:"interact"`
}

type FeedVideoListData struct {
	Items      []FeedVideoData `json:"items"`
	NextCursor string          `json:"next_cursor,omitempty"`
}

type SearchUsersResponseData struct {
	Items      []SearchUserData `json:"items"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type SearchUserData struct {
	ID            int64  `json:"id,string"`
	Username      string `json:"username"`
	Nickname      string `json:"nickname"`
	AvatarURL     string `json:"avatar_url"`
	Signature     string `json:"signature"`
	FollowerCount int64  `json:"follower_count"`
	FollowCount   int64  `json:"follow_count"`
	WorkCount     int64  `json:"work_count"`
	IsFollowed    bool   `json:"is_followed"`
	IsMutual      bool   `json:"is_mutual"`
}

type SearchHashtagsResponseData struct {
	Items      []SearchHashtagData `json:"items"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type SearchHashtagData struct {
	ID       int64  `json:"id,string"`
	Name     string `json:"name"`
	UseCount int64  `json:"use_count"`
}

type SearchVideosResponseData struct {
	Items      []SearchVideoData `json:"items"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type SearchVideoData struct {
	VideoID         int64          `json:"video_id,string"`
	UserID          int64          `json:"user_id,string"`
	Title           string         `json:"title"`
	CoverURL        string         `json:"cover_url"`
	SourceURL       string         `json:"source_url"`
	PlayCount       int64          `json:"play_count"`
	LikeCount       int64          `json:"like_count"`
	CommentCount    int64          `json:"comment_count"`
	FavoriteCount   int64          `json:"favorite_count"`
	Visibility      int8           `json:"visibility"`
	AuditStatus     int8           `json:"audit_status"`
	TranscodeStatus int8           `json:"transcode_status"`
	Author          FeedAuthorData `json:"author"`
	Interact        FeedInteractData `json:"interact"`
}

type SearchAllResponseData struct {
	Users    []SearchUserData    `json:"users"`
	Hashtags []SearchHashtagData `json:"hashtags"`
	Videos   []SearchVideoData   `json:"videos"`
}

type FeedAuthorData struct {
	ID        int64  `json:"id,string"`
	Username  string `json:"username"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url"`
}

type FeedInteractData struct {
	IsFollowed  bool `json:"is_followed"`
	IsLiked     bool `json:"is_liked"`
	IsFavorited bool `json:"is_favorited"`
}

type MessageData struct {
	ID         int64  `json:"id,string"`
	ReceiverID int64  `json:"receiver_id,string"`
	SenderID   int64  `json:"sender_id,string"`
	Type       int8   `json:"type"`
	RelatedID  int64  `json:"related_id,string"`
	Content    string `json:"content"`
	IsRead     int8   `json:"is_read"`
	CreatedAt  string `json:"created_at"`
}

type MessageListData struct {
	Items      []MessageData `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

type MessageUnreadCountData struct {
	Items map[string]int64 `json:"items"`
}

type MessageReadRequest struct {
	MsgID *FlexibleInt64 `json:"msg_id"`
	Type  *int8          `json:"type"`
}

type InteractActionRequest struct {
	VideoID    FlexibleInt64 `json:"video_id"`
	ActionType int8          `json:"action_type"`
}

type FavoriteVideoData struct {
	VideoID         int64  `json:"video_id,string"`
	UserID          int64  `json:"user_id,string"`
	Title           string `json:"title"`
	ObjectKey       string `json:"object_key"`
	SourceURL       string `json:"source_url"`
	CoverURL        string `json:"cover_url"`
	DurationMS      int32  `json:"duration_ms"`
	AllowComment    int8   `json:"allow_comment"`
	Visibility      int8   `json:"visibility"`
	TranscodeStatus int8   `json:"transcode_status"`
	AuditStatus     int8   `json:"audit_status"`
	LikeCount       int64  `json:"like_count"`
	CommentCount    int64  `json:"comment_count"`
	FavoriteCount   int64  `json:"favorite_count"`
	CreatedAt       string `json:"created_at"`
	UpdatedAt       string `json:"updated_at"`
	FavoritedAt     string `json:"favorited_at"`
}

type FavoriteVideoListData struct {
	Items      []FavoriteVideoData `json:"items"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type CommentPublishRequest struct {
	VideoID  FlexibleInt64 `json:"video_id"`
	Content  string        `json:"content"`
	ParentID FlexibleInt64 `json:"parent_id"`
	RootID   FlexibleInt64 `json:"root_id"`
	ToUserID FlexibleInt64 `json:"to_user_id"`
}

type CommentData struct {
	ID        int64  `json:"id,string"`
	VideoID   int64  `json:"video_id,string"`
	UserID    int64  `json:"user_id,string"`
	Content   string `json:"content"`
	ParentID  int64  `json:"parent_id,string"`
	RootID    int64  `json:"root_id,string"`
	ToUserID  int64  `json:"to_user_id,string"`
	LikeCount int64  `json:"like_count"`
	IsDeleted bool   `json:"is_deleted"`
	CreatedAt string `json:"created_at"`
}

type CommentListData struct {
	Items      []CommentData `json:"items"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

type CommentLikeRequest struct {
	CommentID  FlexibleInt64 `json:"comment_id"`
	ActionType int8          `json:"action_type"`
}

type CommentDeleteRequest struct {
	CommentID FlexibleInt64 `json:"comment_id"`
}
