/*
file provides file system operations.

This replaces parts of VS Code's file system abstractions.
*/

package services

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// FileService provides file operations.
type FileService struct {
	directory string
}

// NewFileService creates a new file service.
func NewFileService(directory string) *FileService {
	return &FileService{directory: directory}
}

// ReadFile reads the contents of a file.
func (f *FileService) ReadFile(path string) ([]byte, error) {
	filePath := filepath.Join(f.directory, path)
	return os.ReadFile(filePath)
}

// WriteFile writes data to a file.
func (f *FileService) WriteFile(path string, data []byte) error {
	filePath := filepath.Join(f.directory, path)
	return os.WriteFile(filePath, data, 0644)
}

// FileExists checks if a file exists.
func (f *FileService) FileExists(path string) bool {
	filePath := filepath.Join(f.directory, path)
	_, err := os.Stat(filePath)
	return err == nil
}

// ListFiles lists files in a directory.
func (f *FileService) ListFiles(dir string) ([]string, error) {
	dirPath := filepath.Join(f.directory, dir)
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// EnsureDirectory ensures that a directory exists.
func (f *FileService) EnsureDirectory(path string) error {
	dirPath := filepath.Join(f.directory, filepath.Dir(path))
	return os.MkdirAll(dirPath, 0755)
}

// OpenFile opens a file for reading.
func (f *FileService) OpenFile(path string) (io.ReadCloser, error) {
	filePath := filepath.Join(f.directory, path)
	return os.Open(filePath)
}

// GetMimeType returns the MIME type of a file based on extension.
func (f *FileService) GetMimeType(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".txt":
		return "text/plain"
	case ".js", ".ts":
		return "text/javascript"
	case ".json":
		return "application/json"
	case ".html", ".htm":
		return "text/html"
	case ".md":
		return "text/markdown"
	case ".css":
		return "text/css"
	default:
		return "application/octet-stream"
	}
}
