package utils

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

const MaxUploadSize = 20 << 20

var ErrInvalidMIME = errors.New("type de fichier non autorisé (JPEG, PNG ou GIF uniquement)")
var ErrFileTooLarge = errors.New("fichier trop volumineux (max 20 Mo)")

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

func SaveUploadedImage(file multipart.File, header *multipart.FileHeader, uploadDir string) (string, error) {
	if header.Size > MaxUploadSize {
		return "", ErrFileTooLarge
	}

	buf := make([]byte, 8)
	if _, err := io.ReadFull(file, buf); err != nil {
		return "", fmt.Errorf("lecture fichier: %w", err)
	}

	ext, ok := detectImageExtension(buf)
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
