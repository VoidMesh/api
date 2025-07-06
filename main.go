package main

import (
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Get the directory where the binary is located
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		panic(err)
	}

	// Execute the server binary
	cmd := exec.Command(filepath.Join(dir, "cmd", "server", "main.go"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir

	// Execute with go run
	goCmd := exec.Command("go", "run", "./cmd/server")
	goCmd.Stdout = os.Stdout
	goCmd.Stderr = os.Stderr
	goCmd.Dir = dir

	if err := goCmd.Run(); err != nil {
		panic(err)
	}
}
