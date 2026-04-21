package dto

import "github.com/gin-gonic/gin"

// dto/response.go (thêm vào)

func SendSuccessWithPagination(ctx *gin.Context, statusCode int, message string, data interface{}, skip, limit int64, total int64) {
	page := int64(1)
	if limit > 0 {
		page = (skip / limit) + 1
	}

	totalPages := int64(1)
	if limit > 0 && total > 0 {
		totalPages = (total + limit - 1) / limit
	}

	ctx.JSON(statusCode, gin.H{
		"success": true,
		"message": message,
		"data":    data,
		"pagination": gin.H{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
			"has_next":    page < totalPages,
			"has_prev":    page > 1,
		},
	})
}
