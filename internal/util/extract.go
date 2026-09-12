package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/eswar-7116/glambdar/v3/internal/config"
)

func ExtractZIP(r io.ReaderAt, size int64, funcName string) (string, error) {
	destDir := filepath.Join(config.FunctionsDir, funcName)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("error creating function dir: %w", err)
	}

	zipReader, err := zip.NewReader(r, size)
	if err != nil {
		return "", fmt.Errorf("error opening zip: %w", err)
	}

	for _, f := range zipReader.File {
		fpath := filepath.Join(destDir, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(destDir)+string(os.PathSeparator)) {
			return "", fmt.Errorf("illegal file path: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, f.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return "", fmt.Errorf("error extracting zip: %w", err)
		}

		dst, err := os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
		if err != nil {
			return "", fmt.Errorf("error extracting zip: %w", err)
		}

		src, err := f.Open()
		if err != nil {
			dst.Close()
			return "", fmt.Errorf("error extracting zip: %w", err)
		}

		_, copyErr := io.Copy(dst, src)
		src.Close()
		dst.Close()
		if copyErr != nil {
			return "", fmt.Errorf("error extracting zip: %w", copyErr)
		}
	}

	return destDir, nil
}
