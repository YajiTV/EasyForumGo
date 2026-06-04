package utils

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
)

const MaxUploadSize = 20 << 20

var ErrInvalidMIME = errors.New("type de fichier non autorisé (JPEG, PNG ou GIF uniquement)")
var ErrFileTooLarge = errors.New("fichier trop volumineux (max 20 Mo)")

var allowedMIMEs = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
}

func SaveUploadedImage(file multipart.File, header *multipart.FileHeader, uploadDir string) (string, error) {
	if header.Size > MaxUploadSize {
		return "", ErrFileTooLarge
	}

	buf := make([]byte, 512)
	if _, err := file.Read(buf); err != nil {
		return "", fmt.Errorf("lecture fichier: %w", err)
	}

	mimeType := http.DetectContentType(buf)
	ext, ok := allowedMIMEs[mimeType]
	if !ok {
		return "", ErrInvalidMIME
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seek fichier: %w", err)
	}

	filename := NewUUID() + ext
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		return "", fmt.Errorf("creation fichier: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("ecriture fichier: %w", err)
	}

	return filename, nil
}
