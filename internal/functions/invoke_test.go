package functions_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/internal/functions"
	"github.com/eswar-7116/glambdar/internal/util"
)

func setupInvokeEnv(t *testing.T) func() {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test")
	}

	tmp := t.TempDir()
	util.InitPaths()
	util.FunctionsDir = tmp
	util.UDSPath = filepath.Join(tmp, "glambdar.sock")
	return func() {
		os.RemoveAll(tmp)
	}
}

func TestInvoke_FunctionNotFound(t *testing.T) {
	cleanup := setupInvokeEnv(t)
	defer cleanup()

	_, err := functions.Invoke("missingFunc", functions.InvokeRequest{})
	if err == nil {
		t.Fatalf("expected error for missing function")
	}
}

func TestInvoke_HappyPath(t *testing.T) {
	cleanup := setupInvokeEnv(t)
	defer cleanup()

	funcDir := filepath.Join(util.FunctionsDir, "valid")
	if _, err := os.Stat(funcDir); os.IsNotExist(err) {
		if err := functions.Deploy(validZipFile, "valid"); err != nil {
			t.Fatalf("deploy failed: %v", err)
		}
	}

	req := functions.InvokeRequest{
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"name":"Eswar"}`,
	}

	res, err := functions.Invoke("valid", req)
	if err != nil {
		t.Fatalf("invoke failed: %v", err)
	}

	if res.StatusCode != 200 {
		t.Fatalf("expected status 200, got %d", res.StatusCode)
	}

	var body map[string]any
	if err := json.Unmarshal(res.Body, &body); err != nil {
		t.Fatalf("invalid response body")
	}

	if _, ok := body["json"]; !ok {
		t.Fatalf("expected json field in response body")
	}
}
