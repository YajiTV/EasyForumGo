package utils

import (
	"errors"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

const DefaultMaxUploadSize int64 = 20 << 20

var ErrInvalidMIME = errors.New("type de fichier non autorisé (JPEG, PNG ou GIF uniquement)")
var ErrInvalidImage = errors.New("image invalide ou corrompue")
var ErrFileTooLarge = errors.New("fichier trop volumineux")

type imageSignature struct {
	magic []byte
	ext   string
}

var imageSignatures = []imageSignature{
	{magic: []byte{0xFF, 0xD8, 0xFF}, ext: ".jpg"},
	{magic: []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, ext: ".png"},
	{magic: []byte{0x47, 0x49, 0x46, 0x38, 0x37, 0x61}, ext: ".gif"},
	{magic: []byte{0x47, 0x49, 0x46, 0x38, 0x39, 0x61}, ext: ".gif"},
}

func detectImageExtension(buf []byte) (string, bool) {
	for _, sig := range imageSignatures {
		if len(buf) >= len(sig.magic) {
			match := true
			for i, b := range sig.magic {
				if buf[i] != b {
					match = false
					break
				}
			}
			if match {
				return sig.ext, true
			}
		}
	}
	return "", false
}

// MaxMultipartBodySize allows enough room for the image and multipart metadata.
func MaxMultipartBodySize(maxUploadBytes int64) int64 {
	return normalizedMaxUploadBytes(maxUploadBytes) + 1<<20
}

// UploadSizeLabel formats an upload limit for user-facing messages.
func UploadSizeLabel(maxUploadBytes int64) string {
	return fmt.Sprintf("%d Mo", normalizedMaxUploadBytes(maxUploadBytes)/(1024*1024))
}

// SaveUploadedImage saves uploaded data
func SaveUploadedImage(file multipart.File, header *multipart.FileHeader, uploadDir string, maxUploadBytes int64) (filename string, err error) {
	maxUploadBytes = normalizedMaxUploadBytes(maxUploadBytes)
	if header.Size > maxUploadBytes {
		return "", fmt.Errorf("%w (max %s)", ErrFileTooLarge, UploadSizeLabel(maxUploadBytes))
	}

	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		return "", fmt.Errorf("création du dossier d'upload: %w", err)
	}

	tmp, err := os.CreateTemp(uploadDir, ".upload-*")
	if err != nil {
		return "", fmt.Errorf("création du fichier temporaire: %w", err)
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		if filename == "" {
			_ = os.Remove(tmpPath)
		}
	}()

	written, err := io.Copy(tmp, io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		return "", fmt.Errorf("écriture du fichier temporaire: %w", err)
	}
	if written > maxUploadBytes {
		return "", fmt.Errorf("%w (max %s)", ErrFileTooLarge, UploadSizeLabel(maxUploadBytes))
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("lecture du fichier temporaire: %w", err)
	}

	buf := make([]byte, 8)
	if _, err := io.ReadFull(tmp, buf); err != nil {
		return "", ErrInvalidImage
	}
	ext, ok := detectImageExtension(buf)
	if !ok {
		return "", ErrInvalidMIME
	}
	if _, err := tmp.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("lecture du fichier temporaire: %w", err)
	}
	_, format, err := image.Decode(tmp)
	if err != nil {
		return "", ErrInvalidImage
	}
	if (format == "jpeg" && ext != ".jpg") || (format == "png" && ext != ".png") || (format == "gif" && ext != ".gif") {
		return "", ErrInvalidImage
	}

	if err := tmp.Sync(); err != nil {
		return "", fmt.Errorf("synchronisation du fichier temporaire: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("fermeture du fichier temporaire: %w", err)
	}
	filename = NewUUID() + ext
	if err := os.Rename(tmpPath, filepath.Join(uploadDir, filename)); err != nil {
		filename = ""
		return "", fmt.Errorf("enregistrement de l'image: %w", err)
	}
	return filename, nil
}

func normalizedMaxUploadBytes(maxUploadBytes int64) int64 {
	if maxUploadBytes <= 0 {
		return DefaultMaxUploadSize
	}
	return maxUploadBytes
}
