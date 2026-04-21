// service/media_service.go
package service

import (
	"errors"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/luong-vh/Digimart_Backend/internal/model"
	"github.com/luong-vh/Digimart_Backend/internal/platform/cloudinary"
)

type MediaService interface {
	UploadImage(file *multipart.FileHeader) (*model.Image, error)
	UploadImages(files []*multipart.FileHeader) ([]*model.Image, error)
	UploadVideo(file *multipart.FileHeader) (*model.Video, error)
	UploadVideos(files []*multipart.FileHeader) ([]*model.Video, error)
}

type mediaService struct{}

func NewMediaService() MediaService {
	return &mediaService{}
}

// Allowed extensions & sizes
var (
	allowedImageExts = []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	allowedVideoExts = []string{".mp4", ".mov", ".avi", ".webm", ".mkv"}
	maxImageSize     = int64(10 * 1024 * 1024)  // 10MB
	maxVideoSize     = int64(100 * 1024 * 1024) // 100MB
	maxImageCount    = 10
	maxVideoCount    = 5
)

// ============ UPLOAD IMAGE ============

func (s *mediaService) UploadImage(file *multipart.FileHeader) (*model.Image, error) {
	if err := s.validateImage(file); err != nil {
		return nil, err
	}

	images, err := cloudinary.UploadImages([]*multipart.FileHeader{file})
	if err != nil {
		return nil, err
	}

	if len(images) == 0 {
		return nil, errors.New("upload failed")
	}

	return images[0], nil
}

// ============ UPLOAD MULTIPLE IMAGES ============

func (s *mediaService) UploadImages(files []*multipart.FileHeader) ([]*model.Image, error) {
	if len(files) == 0 {
		return nil, errors.New("no files provided")
	}

	if len(files) > maxImageCount {
		return nil, errors.New("maximum 10 images allowed")
	}

	// Validate all files first
	for _, file := range files {
		if err := s.validateImage(file); err != nil {
			return nil, err
		}
	}

	return cloudinary.UploadImages(files)
}

// ============ UPLOAD VIDEO ============

func (s *mediaService) UploadVideo(file *multipart.FileHeader) (*model.Video, error) {
	if err := s.validateVideo(file); err != nil {
		return nil, err
	}

	videos, err := cloudinary.UploadVideos([]*multipart.FileHeader{file})
	if err != nil {
		return nil, err
	}

	if len(videos) == 0 {
		return nil, errors.New("upload failed")
	}

	return videos[0], nil
}

// ============ UPLOAD MULTIPLE VIDEOS ============

func (s *mediaService) UploadVideos(files []*multipart.FileHeader) ([]*model.Video, error) {
	if len(files) == 0 {
		return nil, errors.New("no files provided")
	}

	if len(files) > maxVideoCount {
		return nil, errors.New("maximum 5 videos allowed")
	}

	// Validate all files first
	for _, file := range files {
		if err := s.validateVideo(file); err != nil {
			return nil, err
		}
	}

	return cloudinary.UploadVideos(files)
}

// ============ VALIDATION ============

func (s *mediaService) validateImage(file *multipart.FileHeader) error {
	if file.Size > maxImageSize {
		return errors.New("image size must be less than 10MB")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedExt(ext, allowedImageExts) {
		return errors.New("invalid image format. Allowed: jpg, jpeg, png, gif, webp")
	}

	return nil
}

func (s *mediaService) validateVideo(file *multipart.FileHeader) error {
	if file.Size > maxVideoSize {
		return errors.New("video size must be less than 100MB")
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedExt(ext, allowedVideoExts) {
		return errors.New("invalid video format. Allowed: mp4, mov, avi, webm, mkv")
	}

	return nil
}

func (s *mediaService) isAllowedExt(ext string, allowed []string) bool {
	for _, a := range allowed {
		if ext == a {
			return true
		}
	}
	return false
}
