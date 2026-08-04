package environment

import (
	"os"
	"path/filepath"
	"runtime"
)

type EnvironmentService struct {
	serverRoot    string
	userHomeDir   string
	extensionsDir string
	configDir     string
	storageDir    string
}

func NewEnvironmentService(serverRoot string) *EnvironmentService {
	homeDir, _ := os.UserHomeDir()
	return &EnvironmentService{
		serverRoot:    serverRoot,
		userHomeDir:   homeDir,
		extensionsDir: filepath.Join(serverRoot, "extensions"),
		configDir:     filepath.Join(homeDir, ".gode", "config"),
		storageDir:    filepath.Join(homeDir, ".gode", "storage"),
	}
}

func (s *EnvironmentService) GetServerRoot() string      { return s.serverRoot }
func (s *EnvironmentService) GetUserHomeDir() string      { return s.userHomeDir }
func (s *EnvironmentService) GetExtensionsDir() string   { return s.extensionsDir }
func (s *EnvironmentService) GetConfigDir() string       { return s.configDir }
func (s *EnvironmentService) GetStorageDir() string      { return s.storageDir }
func (s *EnvironmentService) GetPlatform() string        { return runtime.GOOS }
func (s *EnvironmentService) GetArch() string            { return runtime.GOARCH }

func (s *EnvironmentService) EnsureDirectories() error {
	dirs := []string{s.extensionsDir, s.configDir, s.storageDir}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}
	return nil
}