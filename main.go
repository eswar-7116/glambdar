package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/eswar-7116/glambdar/cmd/glambdar"
)

//go:embed worker/glambdar-worker.js
var workerScript []byte

func init() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error getting user home directory:", err)
	}

	glambdarPath := filepath.Join(home, ".glambdar")

	// Ensure the worker directory exists
	err = os.MkdirAll(filepath.Join(glambdarPath, "worker"), 0755)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating directory $HOME/.glambdar/worker:", err)
	}

	// Always write the latest embedded worker script
	err = os.WriteFile(filepath.Join(glambdarPath, "worker", "glambdar-worker.js"), workerScript, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing worker script:", err)
	}
	workerScript = nil

	glambdar.Init()
}

func main() {
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--version" || arg == "-v" || arg == "version" {
			fmt.Printf("glambdar version %s\n", glambdar.VERSION)
			return
		}
	}
	glambdar.Start()
}
