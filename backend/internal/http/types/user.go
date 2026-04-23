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
	ID              int64  `json:"id"`
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

// UploadCredentialRequest 表示获取上传凭证接口请求体。
type UploadCredentialRequest struct {
	FileName string `json:"file_name"`
}

// UploadCredentialData 表示上传凭证响应结构。
type UploadCredentialData struct {
	ObjectKey    string `json:"object_key"`
	UploadURL    string `json:"upload_url"`
	UploadMethod string `json:"upload_method"`
	ExpiresAt    string `json:"expires_at"`
}

// PublishVideoRequest 表示发布视频请求体。
type PublishVideoRequest struct {
	ObjectKey    string  `json:"object_key"`
	Title        string  `json:"title"`
	HashtagIDs   []int64 `json:"hashtag_ids"`
	AllowComment int8    `json:"allow_comment"`
	Visibility   int8    `json:"visibility"`
}

// PublishVideoData 表示发布视频响应结构。
type PublishVideoData struct {
	VideoID         int64  `json:"video_id"`
	ObjectKey       string `json:"object_key"`
	SourceURL       string `json:"source_url"`
	TranscodeStatus int8   `json:"transcode_status"`
}
