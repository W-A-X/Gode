/*
app manages the Gode application lifecycle.

Replaces src/vs/code/electron-main/main.ts (CodeMain class).
Coordinates:
  - Core services (configuration, environment, lifecycle, state)
  - Extension host process (via extproc)
  - IPC bridge
  - UI layer (gogpu/ui)
*/

package app

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/microsoft/gode/internal/extproc"
	"github.com/microsoft/gode/internal/ipc"
	"github.com/microsoft/gode/internal/services"
	"github.com/microsoft/gode/internal/ui"
)

type Config struct {
	NodeBin        string
	BuildOutputDir string
	UserDataDir    string
	ExtensionsDir  string
	Locale         string
	LogLevel       string
}

const (
	appName = "gode"
)

type App struct {
	cfg Config
	mu  sync.Mutex

	// Services
	envSvc     *services.EnvironmentService
	configSvc  *services.ConfigurationService
	stateSvc   *services.StateService
	lifecycle  *services.LifecycleService
	logSvc     *services.LogService

	// Extension host
	extProc  *extproc.Manager
	readyCh  chan struct{}

	// Shutdown state
	shutdownOnce sync.Once
	shutdownCh   chan struct{}
}

func New(cfg *Config) *App {
	log.Printf("Gode starting (v0.1.0-mvp, commit: placeholder)")

	return &App{
		cfg:        *cfg,
		readyCh:    make(chan struct{}),
		shutdownCh: make(chan struct{}),
	}
}

func (a *App) Run() error {
	// Initialize services
	if err := a.initServices(); err != nil {
		return fmt.Errorf("init services: %w", err)
	}

	// Start extension host
	if err := a.startExtensionHost(); err != nil {
		return fmt.Errorf("start extension host: %w", err)
	}

	// Start UI (gogpu/ui) immediately, in parallel with extension host
	uiApp := ui.NewApp()
	go func() {
		if err := uiApp.Run(); err != nil {
			log.Printf("UI error: %v", err)
			a.Shutdown()
		}
	}()

	// Wait for extension host to be ready (or timeout)
	select {
	case <-a.readyCh:
		log.Println("extension host ready")
	case <-time.After(60 * time.Second):
		log.Println("extension host did not become ready in 60s, continuing without it")
	case <-a.shutdownCh:
		return nil
	}

	log.Println("Gode is ready")

	// Block until shutdown
	<-a.shutdownCh
	return nil
}

func (a *App) initServices() error {
	// Set up paths
	userDataDir := a.cfg.UserDataDir
	if userDataDir == "" {
		home := filepath.Join(os.Getenv("HOME"), ".gode")
		userDataDir = home
	}

	extensionsDir := a.cfg.ExtensionsDir
	if extensionsDir == "" {
		extensionsDir = filepath.Join(userDataDir, "extensions")
	}

	// Initialize services
	a.envSvc = services.NewEnvironmentService(userDataDir, extensionsDir, a.cfg.BuildOutputDir, a.cfg.Locale)
	a.logSvc = services.NewLogService(a.envSvc, a.cfg.LogLevel)
	a.configSvc = services.NewConfigurationService(a.envSvc, a.logSvc)
	a.stateSvc = services.NewStateService(a.envSvc, a.logSvc)
	a.lifecycle = services.NewLifecycleService(a.logSvc)

	return nil
}

func (a *App) startExtensionHost() error {
	initData := a.buildInitData()

	a.extProc = extproc.NewManager(extproc.Config{
		NodeBin:        a.cfg.NodeBin,
		BuildOutputDir: a.cfg.BuildOutputDir,
		UserDataDir:    a.envSvc.UserDataDir,
		ExtensionsDir:  a.envSvc.ExtensionsDir,
		LogDir:         a.envSvc.LogsHome(),
		Commit:         a.envSvc.Commit,
		Version:        a.envSvc.Version,
		AppName:        appName,
		AppLanguage:    a.cfg.Locale,
	})

	ready, err := a.extProc.Start(initData)
	if err != nil {
		return err
	}

	go func() {
		select {
		case <-ready:
			close(a.readyCh)
		case <-a.extProc.Done():
			if err := a.extProc.Err(); err != nil {
				log.Printf("extension host error: %v", err)
			}
		}
	}()

	return nil
}

func (a *App) buildInitData() *ipc.ExtensionHostInitData {
	return &ipc.ExtensionHostInitData{
		Version:   "0.1.0",
		Commit:    a.envSvc.Commit,
		ParentPID: os.Getpid(),
		Environment: ipc.ExtensionHostEnvironment{
			AppName:  appName,
			AppHost:  "desktop",
			AppLanguage: a.cfg.Locale,
			AppURIScheme: "file",
			IsExtensionTelemetryLoggingOnly: false,
			GlobalStorageHome:  ipc.NewURI("vscode-userdata", a.envSvc.GlobalStorageHome()),
			WorkspaceStorageHome: ipc.NewURI("vscode-userdata", a.envSvc.WorkspaceStorageHome()),
		},
		Extensions: ipc.ExtensionDescriptionSnapshot{
			VersionID: 1,
			AllExtensions: []ipc.ExtensionDescription{}, // TODO: populate from extensions dir
			ActivationEvents: map[string][]string{},
			MyExtensions: []ipc.ExtensionIdentifier{},
		},
		TelemetryInfo: ipc.TelemetryInfo{
			SessionID: fmt.Sprintf("gode-session-%d", time.Now().UnixNano()),
			MachineID: "local",
			SquareID: "local",
			DeviceID: "local",
			FirstSessionDate: time.Now().Format("2006-01-02"),
		},
		LogLevel: ipc.LogLevelInfo,
		Loggers: []ipc.LoggerResource{
			{
				Resource: ipc.NewURI("file", filepath.Join(a.envSvc.LogsHome(), "main.log")),
				ID:       "main",
				Name:     "Main",
			},
			{
				Resource: ipc.NewURI("file", filepath.Join(a.envSvc.LogsHome(), "exthost.log")),
				ID:       "exthost",
				Name:     "ExtensionHost",
			},
		},
		LogsLocation: ipc.NewURI("file", a.envSvc.LogsHome()),
		AutoStart: true,
		Remote: ipc.RemoteInfo{
			IsRemote: false,
		},
		ConsoleForward: ipc.ConsoleForward{
			IncludeStack: true,
			LogNative: true,
		},
		UIKind: 1, // UIKind.Desktop = 1
	}
}

func (a *App) Shutdown() {
	a.shutdownOnce.Do(func() {
		log.Println("shutting down services...")

		if a.extProc != nil {
			a.extProc.Stop()
		}

		<-time.After(100 * time.Millisecond) // graceful shutdown grace

		close(a.shutdownCh)
	})
}
