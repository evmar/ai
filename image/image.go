package image

import (
	"fmt"
	"os"
	"path"
)

type LoadedImage struct {
	MimeType string
	Data     []byte
}

func LoadImage(imagePath string) (*LoadedImage, error) {
	mimeType := ""
	switch ext := path.Ext(imagePath); ext {
	case ".jpg", ".jpeg":
		mimeType = "image/jpeg"
	case ".png":
		mimeType = "image/png"
	default:
		return nil, fmt.Errorf("unknown ext %s", ext)
	}

	data, err := os.ReadFile(imagePath)
	if err != nil {
		return nil, err
	}
	return &LoadedImage{MimeType: mimeType, Data: data}, nil
}
