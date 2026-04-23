package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/AbePhh/TikTide/backend/internal/http/types"
	"github.com/AbePhh/TikTide/backend/pkg/errno"
)

func success(ctx *gin.Context, data any) {
	ctx.JSON(200, types.APIResponse{
		Code: 0,
		Msg:  "success",
		Data: data,
	})
}

func fail(ctx *gin.Context, err error) {
	bizErr := errno.From(err)
	ctx.JSON(bizErr.Status, types.APIResponse{
		Code: bizErr.Code,
		Msg:  bizErr.Message,
	})
}
