/*
Main entry point for Gode - a Go-based VS Code compatible editor.

Replaces src/main.ts (Electron main process) and src/vs/code/electron-main/main.ts.
The extension host remains in Node.js (preserved).
*/

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/microsoft/gode/internal/app"
)

func main() {
	var (
		userDataDir    = flag.String("user-data-dir", "", "User data directory")
		extensionsDir  = flag.String("extensions-dir", "", "Extensions directory")
		locale         = flag.String("locale", "en", "UI language")
		logLevel       = flag.String("log", "info", "Log level")
	)
	flag.Parse()

	repoRoot, err := findRepoRoot()
	if err != nil {
		log.Fatalf("find repo root: %v", err)
	}

	nodeBin := findNodeBin()
	if nodeBin == "" {
		log.Fatalf("node binary not found in PATH")
	}

	buildOutput := filepath.Join(repoRoot, "out")

	appInstance := app.New(&app.Config{
		NodeBin:        nodeBin,
		BuildOutputDir: buildOutput,
		UserDataDir:    *userDataDir,
		ExtensionsDir:  *extensionsDir,
		Locale:         *locale,
		LogLevel:       *logLevel,
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("shutting down...")
		appInstance.Shutdown()
	}()

	if err := appInstance.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func findRepoRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for dir := cwd; dir != "/"; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, "product.json")); err == nil {
			return dir, nil
		}
	}

	// Default: repo root is parent of go/ directory
	return filepath.Join(cwd, ".."), nil
}

func findNodeBin() string {
	home := os.Getenv("HOME")
	candidates := []string{
		filepath.Join(home, ".nvm/versions/node/v24.18.0/bin/node"),
		"/usr/local/bin/node",
		"/usr/bin/node",
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	if path, err := exec.LookPath("node"); err == nil {
		return path
	}

	return ""
}

// Ensure strings import is used
var _ = strings.TrimSpace
