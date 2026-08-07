/*
services provides core application services.

These are the Go equivalents of VS Code's platform services.
Each service is defined as an interface (port) with a production
implementation and a null implementation for the MVP.

The init data for the extension host is built using data from
these services.
*/

package services

import (
	"path/filepath"
	"time"
)

// EnvironmentService manages environment and paths.
// Replaces src/vs/platform/environment/electron-main/environmentMainService.ts
type EnvironmentService struct {
	UserDataDir       string
	ExtensionsDir     string
	BuildOutputDir    string
	Locale            string
	Commit            string
	Version           string
	logsHome          string
	globalStorageHome string
	workspaceStorageHome string
}

func NewEnvironmentService(userDataDir, extensionsDir, buildOutputDir, locale string) *EnvironmentService {
	logsHome := filepath.Join(userDataDir, "logs")
	globalStorageHome := filepath.Join(userDataDir, "global-storage")
	workspaceStorageHome := filepath.Join(userDataDir, "workspace-storage")

	return &EnvironmentService{
		UserDataDir:        userDataDir,
		ExtensionsDir:      extensionsDir,
		BuildOutputDir:     buildOutputDir,
		Locale:             locale,
		Commit:             "dev",
		Version:            "0.1.0",
		logsHome:           logsHome,
		globalStorageHome:  globalStorageHome,
		workspaceStorageHome: workspaceStorageHome,
	}
}

func (e *EnvironmentService) LogsHome() string { return e.logsHome }
func (e *EnvironmentService) GlobalStorageHome() string { return e.globalStorageHome }
func (e *EnvironmentService) WorkspaceStorageHome() string { return e.workspaceStorageHome }

// FileService manages file operations.
// Replaces src/vs/platform/files/common/files.ts
type FileService struct {
	env *EnvironmentService
}

func NewFileService(env *EnvironmentService) *FileService {
	return &FileService{env: env}
}

func (f *FileService) ReadFile(path string) ([]byte, error) {
	// Simple file read implementation for MVP
	fullPath := filepath.Join(f.env.UserDataDir, "files", path)
	return os.ReadFile(fullPath)
}

func (f *FileService) WriteFile(path string, data []byte) error {
	fullPath := filepath.Join(f.env.UserDataDir, "files", path)
	return os.WriteFile(fullPath, data, 0644)
}

func (f *FileService) FileExists(path string) bool {
	fullPath := filepath.Join(f.env.UserDataDir, "files", path)
	_, err := os.Stat(fullPath)
	return err == nil
}

func (f *FileService) ListFiles(dir string) ([]string, error) {
	fullPath := filepath.Join(f.env.UserDataDir, "files", dir)
	entries, err := os.ReadDir(fullPath)
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

// ConfigurationService manages user/workspace configuration.
// Replaces src/vs/platform/configuration/common/configurationService.ts
type ConfigurationService struct {
	env   *EnvironmentService
	log   *LogService
	store map[string]interface{}
}

func NewConfigurationService(env *EnvironmentService, log *LogService) *ConfigurationService {
	return &ConfigurationService{
		env:   env,
		log:   log,
		store: make(map[string]interface{}),
	}
}

func (c *ConfigurationService) Initialize() error {
	// TODO: Load settings.json from UserDataDir
	return nil
}

func (c *ConfigurationService) GetValue(key string, defaultValue interface{}) interface{} {
	if v, ok := c.store[key]; ok {
		return v
	}
	return defaultValue
}

// StateService manages global state (like VS Code's global state).
// Replaces src/vs/platform/state/node/stateService.ts
type StateService struct {
	env   *EnvironmentService
	log   *LogService
	store map[string]interface{}
}

func NewStateService(env *EnvironmentService, log *LogService) *StateService {
	return &StateService{
		env:   env,
		log:   log,
		store: make(map[string]interface{}),
	}
}

func (s *StateService) Init() error {
	// TODO: Load state file
	return nil
}

func (s *StateService) Get(key string, defaultValue interface{}) interface{} {
	if v, ok := s.store[key]; ok {
		return v
	}
	return defaultValue
}

func (s *StateService) Set(key string, value interface{}) {
	s.store[key] = value
}

// LifecycleService manages application lifecycle events.
// Replaces src/vs/platform/lifecycle/electron-main/lifecycleMainService.ts
type LifecycleService struct {
	log *LogService
}

func NewLifecycleService(log *LogService) *LifecycleService {
	return &LifecycleService{log: log}
}

// LogService provides logging.
// Replaces src/vs/platform/log/common/log.ts
type LogService struct {
	logsHome string
	level    string
}

func NewLogService(env *EnvironmentService, level string) *LogService {
	return &LogService{
		logsHome: env.LogsHome(),
		level:    level,
	}
}

func (l *LogService) Info(msg string)  { logPrint("INFO", msg) }
func (l *LogService) Warn(msg string)  { logPrint("WARN", msg) }
func (l *LogService) Error(msg string) { logPrint("ERROR", msg) }
func (l *LogService) Trace(msg string) { logPrint("TRACE", msg) }

func logPrint(level, msg string) {
	// Simple stdout logging for MVP
	println(time.Now().Format("15:04:05.000"), level, msg)
}
