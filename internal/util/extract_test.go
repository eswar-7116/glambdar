package util_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eswar-7116/glambdar/internal/util"
)

const (
	validZipFile   = "../../test_data/zip/valid.zip"
	invalidZipFile = "../../test_data/zip/invalid.zip"
	emptyZipFile   = "../../test_data/zip/empty.zip"
	noZipFile      = "../../test_data/zip/none.zip"
)

func TestExtractZIP_ValidZIP(t *testing.T) {
	destDir, err := util.ExtractZIP(validZipFile, "vaild")
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	fpath := filepath.Join(destDir, "index.js")
	if _, err := os.Stat(fpath); os.IsNotExist(err) {
		t.Fatalf("Expected file %s to exist, but it does not", fpath)
	}
}

func TestExtractZIP_InvalidZIP(t *testing.T) {
	_, err := util.ExtractZIP(invalidZipFile, "invalid")
	if err == nil || !strings.Contains(err.Error(), "error opening zip") {
		t.Fatalf("Expected error opening zip, but got: %v", err)
	}
}

func TestExtractZIP_EmptyZIP(t *testing.T) {
	destDir, err := util.ExtractZIP(emptyZipFile, "empty")
	if err != nil {
		t.Fatalf("Expected no error, but got: %v", err)
	}

	files, err := os.ReadDir(destDir)
	if err != nil {
		t.Fatalf("Error reading destination directory: %v", err)
	}

	if len(files) > 0 {
		t.Fatalf("Expected no files to be extracted, but found: %v", files)
	}
}

func TestExtractZIP_NoZip(t *testing.T) {
	_, err := util.ExtractZIP(noZipFile, "noZip")
	if err == nil {
		t.Fatalf("Expected error from os.MkdirAll, but got: %v", err)
	}
	os.RemoveAll("functions")
}
