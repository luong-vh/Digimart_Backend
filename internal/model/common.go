package model

import "time"

// Image defines the structure for a stored image.
type Image struct {
	URL        string    `bson:"url" json:"url"`
	PublicID   string    `bson:"public_id" json:"public_id"`
	UploadedAt time.Time `bson:"uploaded_at" json:"uploaded_at"`
}

// Video defines the structure for a stored video.
type Video struct {
	URL        string    `bson:"url" json:"url"`
	PublicID   string    `bson:"public_id" json:"public_id"`
	UploadedAt time.Time `bson:"uploaded_at" json:"uploaded_at"`
}

func NewImage(url, publicID string) Image {
	return Image{
		URL:        url,
		PublicID:   publicID,
		UploadedAt: time.Now(),
	}
}
func GetFirstImage(images []Image) Image {
	if len(images) > 0 {
		return images[0]
	}
	return Image{}
}

// GetImageByIndex returns image at specific index
func GetImageByIndex(images []Image, index int) Image {
	if index >= 0 && index < len(images) {
		return images[index]
	}
	return Image{}
}
