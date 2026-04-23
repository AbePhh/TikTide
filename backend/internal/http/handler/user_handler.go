package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/app"
	ginmiddleware "github.com/AbePhh/TikTide/backend/internal/http/middleware"
	"github.com/AbePhh/TikTide/backend/internal/http/types"
	userservice "github.com/AbePhh/TikTide/backend/internal/user/service"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

// UserHandler 处理用户与鉴权相关的 HTTP 请求。
type UserHandler struct {
	appCtx *app.Context
}

// NewUserHandler 创建用户处理器。
func NewUserHandler(appCtx *app.Context) *UserHandler {
	return &UserHandler{appCtx: appCtx}
}

// Register 处理用户注册请求。
func (h *UserHandler) Register(ctx *gin.Context) {
	var req types.RegisterRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.UserService.Register(ctx.Request.Context(), userservice.RegisterRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, gin.H{
		"user": buildProfileData(result.User),
	})
}

// Login 处理用户登录请求。
func (h *UserHandler) Login(ctx *gin.Context) {
	var req types.LoginRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	result, err := h.appCtx.UserService.Login(ctx.Request.Context(), userservice.LoginRequest{
		Username: req.Username,
		Password: req.Password,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, types.LoginData{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt.Format(time.RFC3339),
		User:      buildProfileData(result.User),
	})
}

// Logout 处理用户退出登录请求。
func (h *UserHandler) Logout(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	if err := h.appCtx.UserService.Logout(ctx.Request.Context(), authInfo.Token); err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, gin.H{"logged_out": true})
}

// GetProfile 返回当前登录用户资料。
func (h *UserHandler) GetProfile(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	profile, err := h.appCtx.UserService.GetProfile(ctx.Request.Context(), authInfo.UserID)
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, buildProfileData(*profile))
}

// UpdateProfile 更新当前用户资料。
func (h *UserHandler) UpdateProfile(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.UpdateProfileRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	updateReq := userservice.UpdateProfileRequest{
		Nickname:  req.Nickname,
		AvatarURL: req.AvatarURL,
		Signature: req.Signature,
		Gender:    req.Gender,
	}
	if req.Birthday != nil {
		updateReq.BirthdayIsSet = true
		if *req.Birthday != "" {
			birthday, err := time.Parse("2006-01-02", *req.Birthday)
			if err != nil {
				fail(ctx, errno.ErrInvalidParam)
				return
			}
			updateReq.Birthday = &birthday
		}
	}

	profile, err := h.appCtx.UserService.UpdateProfile(ctx.Request.Context(), authInfo.UserID, updateReq)
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, buildProfileData(*profile))
}

// ChangePassword 修改当前用户密码。
func (h *UserHandler) ChangePassword(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	var req types.ChangePasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	err := h.appCtx.UserService.ChangePassword(ctx.Request.Context(), authInfo.UserID, userservice.ChangePasswordRequest{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	})
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, gin.H{"changed": true})
}

// GetHomepage 返回目标用户主页信息。
func (h *UserHandler) GetHomepage(ctx *gin.Context) {
	authInfo, ok := ginmiddleware.GetAuthInfo(ctx)
	if !ok {
		fail(ctx, errno.ErrUnauthorized)
		return
	}

	uid, err := strconv.ParseInt(ctx.Param("uid"), 10, 64)
	if err != nil || uid <= 0 {
		fail(ctx, errno.ErrInvalidParam)
		return
	}

	profile, err := h.appCtx.UserService.GetHomepage(ctx.Request.Context(), authInfo.UserID, uid)
	if err != nil {
		fail(ctx, err)
		return
	}

	success(ctx, buildProfileData(*profile))
}

func buildProfileData(profile userservice.Profile) types.ProfileData {
	response := types.ProfileData{
		ID:              profile.ID,
		Username:        profile.Username,
		Nickname:        profile.Nickname,
		AvatarURL:       profile.AvatarURL,
		Signature:       profile.Signature,
		Gender:          profile.Gender,
		Status:          profile.Status,
		FollowCount:     profile.FollowCount,
		FollowerCount:   profile.FollowerCount,
		TotalLikedCount: profile.TotalLikedCount,
		WorkCount:       profile.WorkCount,
		FavoriteCount:   profile.FavoriteCount,
		IsFollowed:      profile.IsFollowed,
		IsMutual:        profile.IsMutual,
		CreatedAt:       profile.CreatedAt.Format(time.RFC3339),
	}
	if profile.Birthday != nil {
		response.Birthday = profile.Birthday.Format("2006-01-02")
	}
	return response
}
