package errno

import (
	"errors"
	"net/http"
)

// Error represents a business error with a code and HTTP status.
type Error struct {
	Code    int
	Message string
	Status  int
}

func (e *Error) Error() string {
	return e.Message
}

// New creates a business error.
func New(code int, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// From normalizes any error into a business error.
func From(err error) *Error {
	if err == nil {
		return nil
	}

	var target *Error
	if errors.As(err, &target) {
		return target
	}

	return ErrInternalRPC
}

// IsCode reports whether the error is the given business code.
func IsCode(err error, code int) bool {
	var target *Error
	if errors.As(err, &target) {
		return target.Code == code
	}
	return false
}

var (
	ErrInvalidParam     = New(100001, "参数不合法", http.StatusBadRequest)
	ErrUnauthorized     = New(100002, "未登录或 Token 无效", http.StatusUnauthorized)
	ErrTooFrequent      = New(100003, "请求过于频繁", http.StatusTooManyRequests)
	ErrInternalRPC      = New(100004, "内部服务通信失败", http.StatusInternalServerError)
	ErrResourceNotFound = New(100005, "资源不存在", http.StatusNotFound)
	ErrDuplicateRequest = New(100006, "重复请求或幂等冲突", http.StatusConflict)

	ErrUsernameExists    = New(200101, "用户名已存在", http.StatusConflict)
	ErrInvalidCredential = New(200102, "用户名或密码错误", http.StatusUnauthorized)
	ErrUserBanned        = New(200103, "用户已被封禁", http.StatusForbidden)
	ErrWrongOldPassword  = New(200104, "原密码错误", http.StatusBadRequest)
	ErrDuplicateFollow   = New(200105, "重复关注", http.StatusConflict)
	ErrRelationNotFound  = New(200106, "关注关系不存在", http.StatusNotFound)

	ErrUploadObjectNotFound = New(300101, "上传对象不存在", http.StatusBadRequest)
	ErrVideoTranscoding     = New(300102, "视频转码处理中", http.StatusConflict)
	ErrVideoTranscodeFailed = New(300103, "视频转码失败", http.StatusInternalServerError)
	ErrVideoInvisible       = New(300104, "视频不可见", http.StatusForbidden)
	ErrHashtagNotFound      = New(300105, "话题不存在", http.StatusBadRequest)
	ErrDraftNotFound        = New(300106, "草稿不存在", http.StatusNotFound)

	ErrDuplicateLike     = New(400101, "重复点赞", http.StatusConflict)
	ErrDuplicateFavorite = New(400102, "重复收藏", http.StatusConflict)
	ErrInvalidComment    = New(400103, "评论内容不合法", http.StatusBadRequest)
	ErrCommentNotFound   = New(400104, "评论不存在", http.StatusNotFound)
	ErrCommentDeleted    = New(400105, "评论已删除", http.StatusConflict)
	ErrCommentForbidden  = New(400106, "视频不允许评论", http.StatusForbidden)

	ErrFeedInvalidCursor = New(500101, "Feed 游标不合法", http.StatusBadRequest)
	ErrFeedFetchFailed   = New(500102, "Feed 拉取失败", http.StatusInternalServerError)

	ErrSearchUnavailable   = New(600101, "搜索服务暂不可用", http.StatusServiceUnavailable)
	ErrSearchFailed        = New(600102, "搜索失败", http.StatusInternalServerError)
	ErrSearchInvalidCursor = New(600103, "搜索游标不合法", http.StatusBadRequest)
)
