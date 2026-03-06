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
	if _, err = os.Stat(glambdarPath); os.IsNotExist(err) {
		fmt.Println("$HOME/.glambdar not found. Creating one now...")

		err = os.MkdirAll(filepath.Join(glambdarPath, "worker"), 0755)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating directory $HOME/.glambdar/worker:", err)
		}

		err = os.WriteFile(filepath.Join(glambdarPath, "worker", "glambdar-worker.js"), workerScript, 0644)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error creating worker script in $HOME/.glambdar/worker:", err)
		}

		workerScript = nil
	}

	glambdar.Init()
}

func main() {
	glambdar.Start()
}
