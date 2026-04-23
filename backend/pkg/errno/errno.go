package errno

import (
	"errors"
	"net/http"
)

// Error 表示带有稳定错误码和 HTTP 状态码的业务错误。
type Error struct {
	Code    int
	Message string
	Status  int
}

func (e *Error) Error() string {
	return e.Message
}

// New 创建一个业务错误。
func New(code int, message string, status int) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Status:  status,
	}
}

// From 将任意错误转换为统一业务错误。
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

// IsCode 判断错误是否为指定业务错误码。
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

	ErrUploadObjectNotFound = New(300101, "上传对象不存在", http.StatusBadRequest)
	ErrVideoTranscoding     = New(300102, "视频转码处理中", http.StatusConflict)
	ErrVideoTranscodeFailed = New(300103, "视频转码失败", http.StatusInternalServerError)
	ErrVideoInvisible       = New(300104, "视频不可见", http.StatusForbidden)
	ErrHashtagNotFound      = New(300105, "话题不存在", http.StatusBadRequest)
)
