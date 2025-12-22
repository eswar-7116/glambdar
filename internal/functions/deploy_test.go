package functions_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/eswar-7116/glambdar/internal/util"
)

var validZipFile = filepath.Join("..", "..", "test_data", "zip", "valid.zip")

func TestDeploy_CreatesFunctionAndMetadata(t *testing.T) {
	tmp := t.TempDir()
	util.FunctionsDir = tmp

	t.Log("Deploying...")
	err := functions.Deploy(validZipFile, "testFunc")
	if err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	funcDir := filepath.Join(util.FunctionsDir, "testFunc")
	if _, err := os.Stat(funcDir); err != nil {
		t.Fatalf("function directory not created")
	}

	md, err := functions.LoadMetadata(funcDir)
	if err != nil {
		t.Fatalf("metadata not created")
	}

	if md.Name != "testFunc" {
		t.Fatalf("metadata name mismatch")
	}
	if md.InvokeCount != 0 {
		t.Fatalf("invoke count should be zero")
	}
}
