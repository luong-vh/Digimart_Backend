package cloudinary

import (
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/luong-vh/Digimart_Backend/internal/config"
	"github.com/luong-vh/Digimart_Backend/internal/model"
)

func newCld() (*cloudinary.Cloudinary, error) {
	if config.Cfg.Cloudinary.CloudName == "" || config.Cfg.Cloudinary.APIKey == "" || config.Cfg.Cloudinary.APISecret == "" {
		return nil, errors.New("cloudinary configuration is incomplete")
	}

	return cloudinary.NewFromParams(config.Cfg.Cloudinary.CloudName, config.Cfg.Cloudinary.APIKey, config.Cfg.Cloudinary.APISecret)
}

func Upload(file multipart.File) (*uploader.UploadResult, error) {
	cld, err := newCld()
	if err != nil {
		return nil, err
	}

	params := uploader.UploadParams{
		Folder: config.Cfg.Cloudinary.UploadFolder,
	}
	if config.Cfg.Cloudinary.UploadPreset != "" {
		params.UploadPreset = config.Cfg.Cloudinary.UploadPreset
	}

	return cld.Upload.Upload(context.Background(), file, params)
}

func UploadVideo(file multipart.File) (*uploader.UploadResult, error) {
	cld, err := newCld()
	if err != nil {
		return nil, err
	}

	params := uploader.UploadParams{
		Folder:       config.Cfg.Cloudinary.UploadFolder,
		ResourceType: "video",
	}
	if config.Cfg.Cloudinary.UploadPreset != "" {
		params.UploadPreset = config.Cfg.Cloudinary.UploadPreset
	}

	return cld.Upload.Upload(context.Background(), file, params)
}

func Delete(publicID string) (*uploader.DestroyResult, error) {
	cld, err := newCld()
	if err != nil {
		return nil, err
	}

	return cld.Upload.Destroy(context.Background(), uploader.DestroyParams{PublicID: publicID})
}

// UploadImages uploads multiple images and returns model.Image slice.
// This function handles both single and multiple image uploads.
func UploadImages(files []*multipart.FileHeader) ([]*model.Image, error) {
	if len(files) == 0 {
		return nil, errors.New("no images provided")
	}

	var uploadedImages []*model.Image
	var lastErr error

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			lastErr = err
			continue
		}

		result, err := Upload(file)
		file.Close() // Close immediately, not defer in loop

		if err != nil {
			lastErr = err
			continue
		}
		if result == nil {
			lastErr = errors.New("cloudinary upload returned empty response")
			continue
		}
		if result.Error.Message != "" {
			lastErr = fmt.Errorf("cloudinary upload failed: %s", result.Error.Message)
			continue
		}

		imageURL := result.SecureURL
		if imageURL == "" {
			imageURL = result.URL
		}
		if imageURL == "" || result.PublicID == "" {
			lastErr = errors.New("cloudinary upload returned empty url or public_id")
			continue
		}

		uploadedImages = append(uploadedImages, &model.Image{
			URL:        imageURL,
			PublicID:   result.PublicID,
			UploadedAt: time.Now(),
		})
	}

	if len(uploadedImages) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("all image uploads failed: %w", lastErr)
		}
		return nil, errors.New("all image uploads failed")
	}

	return uploadedImages, nil
}

// UploadVideos uploads multiple videos and returns model.Video slice.
func UploadVideos(files []*multipart.FileHeader) ([]*model.Video, error) {
	if len(files) == 0 {
		return nil, errors.New("no videos provided")
	}

	var uploadedVideos []*model.Video
	var lastErr error

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			lastErr = err
			continue
		}

		result, err := UploadVideo(file) // Use UploadVideo
		file.Close()

		if err != nil {
			lastErr = err
			continue
		}
		if result == nil {
			lastErr = errors.New("cloudinary upload returned empty response")
			continue
		}
		if result.Error.Message != "" {
			lastErr = fmt.Errorf("cloudinary upload failed: %s", result.Error.Message)
			continue
		}

		videoURL := result.SecureURL
		if videoURL == "" {
			videoURL = result.URL
		}
		if videoURL == "" || result.PublicID == "" {
			lastErr = errors.New("cloudinary upload returned empty url or public_id")
			continue
		}

		uploadedVideos = append(uploadedVideos, &model.Video{
			URL:        videoURL,
			PublicID:   result.PublicID,
			UploadedAt: time.Now(),
		})
	}

	if len(uploadedVideos) == 0 {
		if lastErr != nil {
			return nil, fmt.Errorf("all video uploads failed: %w", lastErr)
		}
		return nil, errors.New("all video uploads failed")
	}

	return uploadedVideos, nil
}
