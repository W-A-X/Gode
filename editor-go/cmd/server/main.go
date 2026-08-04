package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"gode/editor/internal/services/configuration"
	"gode/editor/internal/services/environment"
	"gode/editor/internal/services/extension"
	"gode/editor/internal/services/filesystem"
	"gode/editor/internal/services/httpresource"
	"gode/editor/internal/services/logging"
	"gode/editor/internal/services/mcp"
	"gode/editor/internal/services/terminal"
)

var (
	port     = flag.Int("port", 0, "Port for the IPC server (0 = auto-assign)")
	rootPath = flag.String("root", ".", "Server root directory")
	logLevel = flag.String("log-level", "info", "Log level: trace, debug, info, warn, error")
)

func main() {
	flag.Parse()

	absRoot, err := filepath.Abs(*rootPath)
	if err != nil {
		log.Fatalf("Failed to resolve root path: %v", err)
	}

	env := environment.NewEnvironmentService(absRoot)
	if err := env.EnsureDirectories(); err != nil {
		log.Fatalf("Failed to create directories: %v", err)
	}

	logSvc := logging.NewLogService(filepath.Join(env.GetStorageDir(), "logs"))
	defer logSvc.Dispose()

	configSvc := configuration.NewConfigurationService(env.GetConfigDir())

	fsSvc := filesystem.NewFileSystemService()
	termSvc := terminal.NewTerminalService()
	extSvc := extension.NewExtensionService(env.GetExtensionsDir())
	mcpSvc := mcp.NewMCPService()
	_ = mcpSvc
	httpSvc := httpresource.NewHTTPResourceService()

	if err := extSvc.ScanExtensions(); err != nil {
		logSvc.Error("Failed to scan extensions:", err)
	}

	startPort := *port
	if startPort == 0 {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatalf("Failed to find free port: %v", err)
		}
		startPort = listener.Addr().(*net.TCPAddr).Port
		listener.Close()
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/fs/stat", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		stat, err := fsSvc.Stat(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(stat)
	})

	mux.HandleFunc("/fs/readdir", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		entries, err := fsSvc.Readdir(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(entries)
	})

	mux.HandleFunc("/fs/readfile", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Query().Get("path")
		data, err := fsSvc.ReadFile(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Write(data)
	})

	mux.HandleFunc("/fs/writefile", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
			Data []byte `json:"data"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := fsSvc.WriteFile(req.Path, req.Data, "w")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/fs/mkdir", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path string `json:"path"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := fsSvc.Mkdir(req.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/fs/delete", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Path      string `json:"path"`
			Recursive bool   `json:"recursive"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := fsSvc.Delete(req.Path, req.Recursive)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/fs/rename", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Source    string `json:"source"`
			Target    string `json:"target"`
			Overwrite bool   `json:"overwrite"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := fsSvc.Rename(req.Source, req.Target, req.Overwrite)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/terminal/create", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID      string   `json:"id"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
			Cols    int      `json:"cols"`
			Rows    int      `json:"rows"`
			WorkDir string   `json:"workDir"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if req.Cols == 0 {
			req.Cols = 80
		}
		if req.Rows == 0 {
			req.Rows = 24
		}
		info, err := termSvc.CreateTerminal(req.ID, req.Command, req.Args, req.Cols, req.Rows, req.WorkDir, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(info)
	})

	mux.HandleFunc("/terminal/input", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID   string `json:"id"`
			Data string `json:"data"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := termSvc.SendInput(req.ID, req.Data)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/terminal/kill", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := termSvc.KillTerminal(req.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/terminal/info", func(w http.ResponseWriter, r *http.Request) {
		id := r.URL.Query().Get("id")
		info, err := termSvc.GetProcessInfo(id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(info)
	})

	mux.HandleFunc("/config/get", func(w http.ResponseWriter, r *http.Request) {
		key := r.URL.Query().Get("key")
		val := configSvc.Get(key)
		json.NewEncoder(w).Encode(map[string]interface{}{"key": key, "value": val})
	})

	mux.HandleFunc("/config/set", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Key   string      `json:"key"`
			Value interface{} `json:"value"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := configSvc.Set(req.Key, req.Value)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/config/all", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(configSvc.GetAll())
	})

	mux.HandleFunc("/extensions/list", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(extSvc.ListExtensions())
	})

	mux.HandleFunc("/extensions/activate", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"id"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		err := extSvc.ActivateExtension(req.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	mux.HandleFunc("/env/info", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"serverRoot":    env.GetServerRoot(),
			"userHomeDir":   env.GetUserHomeDir(),
			"extensionsDir": env.GetExtensionsDir(),
			"configDir":     env.GetConfigDir(),
			"storageDir":    env.GetStorageDir(),
			"platform":      env.GetPlatform(),
			"arch":          env.GetArch(),
		})
	})

	mux.HandleFunc("/http/fetch", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		}
		json.NewDecoder(r.Body).Decode(&req)
		if err := httpSvc.ValidateURL(req.URL); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		body, contentType, err := httpSvc.Fetch(req.URL, req.Headers)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", contentType)
		w.Write(body)
	})

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	})

	addr := fmt.Sprintf("127.0.0.1:%d", startPort)
	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("Gode Go backend starting on %s", addr)
		log.Printf("Environment: platform=%s arch=%s", env.GetPlatform(), env.GetArch())
		log.Printf("Root: %s", absRoot)
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	portFile := filepath.Join(env.GetStorageDir(), "gode-go-server.port")
	os.WriteFile(portFile, []byte(fmt.Sprintf("%d", startPort)), 0644)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigCh
	log.Printf("Received signal %v, shutting down...", sig)

	termSvc.Cleanup()
	server.Close()
	os.Remove(portFile)
	logSvc.Info("Server stopped")
}