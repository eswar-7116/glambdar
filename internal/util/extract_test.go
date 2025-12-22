package util_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eswar-7116/glambdar/internal/util"
)

var (
	validZipFile   = filepath.Join("..", "..", "test_data", "zip", "valid.zip")
	invalidZipFile = filepath.Join("..", "..", "test_data", "zip", "invalid.zip")
	emptyZipFile   = filepath.Join("..", "..", "test_data", "zip", "empty.zip")
	noZipFile      = filepath.Join("..", "..", "test_data", "zip", "none.zip")
)

func TestExtractZIP_ValidZIP(t *testing.T) {
	util.InitPaths()
	extractedDir, err := util.ExtractZIP(validZipFile, "vaild")
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
	util.InitPaths()
	_, err := util.ExtractZIP(invalidZipFile, "invalid")
	defer os.RemoveAll(filepath.Join(util.FunctionsDir, "invalid"))
	if err == nil || !strings.Contains(err.Error(), "error opening zip") {
		t.Fatalf("Expected error opening zip, but got: %v", err)
	}
}

func TestExtractZIP_EmptyZIP(t *testing.T) {
	util.InitPaths()
	extractedDir, err := util.ExtractZIP(emptyZipFile, "empty")
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
	util.InitPaths()
	_, err := util.ExtractZIP(noZipFile, "noZip")
	defer os.RemoveAll(filepath.Join(util.FunctionsDir, "noZip"))
	if err == nil {
		t.Fatalf("Expected error from os.MkdirAll, but got: %v", err)
	}
	os.RemoveAll("functions")
}
