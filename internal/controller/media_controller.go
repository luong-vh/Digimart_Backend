// controller/media_controller.go
package controller

import (
	"net/http"

	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/apperror"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/dto"
	"github.com/DucLove1/SE357-ShoppingManagement-BE/internal/service"
	"github.com/gin-gonic/gin"
)

type MediaController struct {
	mediaService service.MediaService
}

func NewMediaController(mediaService service.MediaService) *MediaController {
	return &MediaController{mediaService: mediaService}
}

// UploadImage - Upload 1 ảnh
// POST /api/media/image
// Form: image (file)
func (c *MediaController) UploadImage(ctx *gin.Context) {
	file, err := ctx.FormFile("image")
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Image file is required", apperror.ErrBadRequest.Code)
		return
	}

	image, err := c.mediaService.UploadImage(file)
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, err.Error(), apperror.ErrBadRequest.Code)
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Image uploaded successfully", image)
}

// UploadImages - Upload nhiều ảnh
// POST /api/media/images
// Form: images (files)
func (c *MediaController) UploadImages(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid form data", apperror.ErrBadRequest.Code)
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		dto.SendError(ctx, http.StatusBadRequest, "At least one image is required", apperror.ErrBadRequest.Code)
		return
	}

	images, err := c.mediaService.UploadImages(files)
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, err.Error(), apperror.ErrBadRequest.Code)
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Images uploaded successfully", gin.H{
		"images": images,
		"count":  len(images),
	})
}

// UploadVideo - Upload 1 video
// POST /api/media/video
// Form: video (file)
func (c *MediaController) UploadVideo(ctx *gin.Context) {
	file, err := ctx.FormFile("video")
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Video file is required", apperror.ErrBadRequest.Code)
		return
	}

	video, err := c.mediaService.UploadVideo(file)
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, err.Error(), apperror.ErrBadRequest.Code)
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Video uploaded successfully", video)
}

// UploadVideos - Upload nhiều video
// POST /api/media/videos
// Form: videos (files)
func (c *MediaController) UploadVideos(ctx *gin.Context) {
	form, err := ctx.MultipartForm()
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, "Invalid form data", apperror.ErrBadRequest.Code)
		return
	}

	files := form.File["videos"]
	if len(files) == 0 {
		dto.SendError(ctx, http.StatusBadRequest, "At least one video is required", apperror.ErrBadRequest.Code)
		return
	}

	videos, err := c.mediaService.UploadVideos(files)
	if err != nil {
		dto.SendError(ctx, http.StatusBadRequest, err.Error(), apperror.ErrBadRequest.Code)
		return
	}

	dto.SendSuccess(ctx, http.StatusOK, "Videos uploaded successfully", gin.H{
		"videos": videos,
		"count":  len(videos),
	})
}
