package util

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ExtractZIP(zipFilePath string, funcName string) (string, error) {
	destDir := filepath.Join(FunctionsDir, funcName)

	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "nil", fmt.Errorf("error creating function dir: %w", err)
	}

	r, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return "", fmt.Errorf("error opening zip: %w", err)
	}
	defer r.Close()

	for _, f := range r.File {
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
		defer dst.Close()

		src, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("error extracting zip: %w", err)
		}
		defer src.Close()

		if _, err = io.Copy(dst, src); err != nil {
			return "", fmt.Errorf("error extracting zip: %w", err)
		}
	}

	return destDir, nil
}
