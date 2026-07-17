package functions_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/eswar-7116/glambdar/v3/internal/config"
	"github.com/eswar-7116/glambdar/v3/internal/docker"
	"github.com/eswar-7116/glambdar/v3/internal/functions"
	"gorm.io/gorm/logger"
)

func setupInvokeEnv(t *testing.T) func() {
	t.Helper()

	if os.Getenv("RUN_INTEGRATION_TESTS") == "" {
		t.Skip("skipping integration test")
	}

	tmp := t.TempDir()
	config.InitPathsWithBase(tmp)
	config.DB.AutoMigrate(&functions.Metadata{}, &functions.Log{})
	config.DB.Logger = logger.Default.LogMode(logger.Silent)

	// In tests, we need to point to the worker script in the repository
	workerPath, err := filepath.Abs(filepath.Join("..", "..", "worker", "glambdar-worker.js"))
	if err != nil {
		t.Fatalf("failed to get absolute path for worker: %v", err)
	}
	config.WorkerPath = workerPath

	return func() {
		os.RemoveAll(tmp)
	}
}

func TestInvoke_FunctionNotFound(t *testing.T) {
	cleanup := setupInvokeEnv(t)
	defer cleanup()

	d := &docker.Docker{
		WorkerPath: config.WorkerPath,
	}
	defer d.Close()
	ctx := t.Context()

	_, err := functions.Invoke(ctx, d, "missingFunc", functions.InvokeRequest{})
	if err == nil {
		t.Fatalf("expected error for missing function")
	}
}

func TestInvoke_HappyPath(t *testing.T) {
	cleanup := setupInvokeEnv(t)
	defer cleanup()

	funcDir := filepath.Join(config.FunctionsDir, "valid")
	if _, err := os.Stat(funcDir); os.IsNotExist(err) {
		if err := functions.Deploy(validZipFile, "valid", 0); err != nil {
			t.Fatalf("deploy failed: %v", err)
		}
	}

	req := functions.InvokeRequest{
		Method: "POST",
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: `{"name":"Eswar"}`,
	}

	d := &docker.Docker{
		WorkerPath: config.WorkerPath,
	}
	defer d.Close()
	ctx := t.Context()
	defer config.PoolManager.DeleteAllContainers(ctx, d)

	res, err := functions.Invoke(ctx, d, "valid", req)
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

	if body["method"] != "POST" {
		t.Errorf("expected default method POST, got %s", body["method"])
	}
}

func TestInvoke_Methods(t *testing.T) {
	cleanup := setupInvokeEnv(t)
	defer cleanup()

	funcName := "methods-func"
	if err := functions.Deploy(validZipFile, funcName, 0); err != nil {
		t.Fatalf("deploy failed: %v", err)
	}

	d := &docker.Docker{
		WorkerPath: config.WorkerPath,
	}
	defer d.Close()
	ctx := t.Context()
	defer config.PoolManager.DeleteAllContainers(ctx, d)

	methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			req := functions.InvokeRequest{
				Method: method,
			}
			res, err := functions.Invoke(ctx, d, funcName, req)
			if err != nil {
				t.Fatalf("invoke failed for %s: %v", method, err)
			}

			if res.StatusCode != 200 {
				t.Fatalf("expected status 200, got %d", res.StatusCode)
			}

			var body map[string]any
			if err := json.Unmarshal(res.Body, &body); err != nil {
				t.Fatalf("invalid response body: %v", err)
			}

			if body["method"] != method {
				t.Errorf("expected method %s, got %s", method, body["method"])
			}
		})
	}
}
