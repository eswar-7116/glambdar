package util_test

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/util"
)

var (
	validZipFile   = filepath.Join("..", "..", "test_data", "zip", "valid.zip")
	invalidZipFile = filepath.Join("..", "..", "test_data", "zip", "invalid.zip")
	emptyZipFile   = filepath.Join("..", "..", "test_data", "zip", "empty.zip")
	noZipFile      = filepath.Join("..", "..", "test_data", "zip", "none.zip")
)

func init() {
	tempHome, _ := os.MkdirTemp("", "home-*")
	os.Setenv("HOME", tempHome)
}

func helperOpenZip(t *testing.T, path string) (*os.File, int64) {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		return nil, 0
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0
	}
	return f, st.Size()
}

func TestExtractZIP_ValidZIP(t *testing.T) {
	config.InitPaths()
	f, size := helperOpenZip(t, validZipFile)
	if f != nil {
		defer f.Close()
	}
	extractedDir, err := util.ExtractZIP(f, size, "vaild")
	defer os.RemoveAll(extractedDir)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	fpath := filepath.Join(extractedDir, "index.js")
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist, but it does not", fpath)
	}
}

func TestExtractZIP_InvalidZIP(t *testing.T) {
	config.InitPaths()
	f, size := helperOpenZip(t, invalidZipFile)
	if f != nil {
		defer f.Close()
	}
	_, err := util.ExtractZIP(f, size, "invalid")
	defer os.RemoveAll(filepath.Join(config.FunctionsDir, "invalid"))
	if err == nil || !strings.Contains(err.Error(), "error opening zip") {
		t.Fatalf("Expected error opening zip, but got: %v", err)
	}
}

func TestExtractZIP_EmptyZIP(t *testing.T) {
	config.InitPaths()
	f, size := helperOpenZip(t, emptyZipFile)
	if f != nil {
		defer f.Close()
	}
	extractedDir, err := util.ExtractZIP(f, size, "empty")
	defer os.RemoveAll(extractedDir)
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	files, err := os.ReadDir(extractedDir)
	if err != nil {
		t.Fatalf("Error reading destination directory: %v", err)
	}

	if len(files) > 0 {
		t.Fatalf("Expected no files to be extracted, but found: %v", files)
	}
}

func TestExtractZIP_NoZip(t *testing.T) {
	config.InitPaths()
	f, size := helperOpenZip(t, noZipFile)
	if f != nil {
		defer f.Close()
	}
	var rAt io.ReaderAt = f
	if f == nil {
		rAt = bytes.NewReader(nil)
	}
	_, err := util.ExtractZIP(rAt, size, "noZip")
	defer os.RemoveAll(filepath.Join(config.FunctionsDir, "noZip"))
	if err == nil {
		t.Fatalf("Expected error from opening zip, but got nil")
	}
}
