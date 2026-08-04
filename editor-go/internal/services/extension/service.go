package extension

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type ExtensionInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	Publisher   string                 `json:"publisher,omitempty"`
	InstallPath string                 `json:"installPath"`
	Activated   bool                   `json:"activated"`
	Contributes map[string]interface{} `json:"contributes,omitempty"`
}

type ExtensionService struct {
	extensionsDir string
	extensions    map[string]*ExtensionInfo
	mu            sync.RWMutex
}

func NewExtensionService(extensionsDir string) *ExtensionService {
	return &ExtensionService{
		extensionsDir: extensionsDir,
		extensions:    make(map[string]*ExtensionInfo),
	}
}

func (es *ExtensionService) ScanExtensions() error {
	es.mu.Lock()
	defer es.mu.Unlock()

	entries, err := os.ReadDir(es.extensionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		extPath := filepath.Join(es.extensionsDir, entry.Name())
		pkgFile := filepath.Join(extPath, "package.json")
		data, err := os.ReadFile(pkgFile)
		if err != nil {
			continue
		}
		var pkg struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Publisher   string `json:"publisher"`
			Engines     struct {
				VSCode string `json:"vscode"`
			} `json:"engines"`
		}
		if err := json.Unmarshal(data, &pkg); err != nil {
			continue
		}
		id := pkg.Publisher + "." + pkg.Name
		es.extensions[id] = &ExtensionInfo{
			ID:          id,
			Name:        pkg.Name,
			Version:     pkg.Version,
			Description: pkg.Description,
			Publisher:   pkg.Publisher,
			InstallPath: extPath,
			Activated:   false,
		}
	}
	return nil
}

func (es *ExtensionService) GetExtension(id string) (*ExtensionInfo, error) {
	es.mu.RLock()
	defer es.mu.RUnlock()
	ext, ok := es.extensions[id]
	if !ok {
		return nil, fmt.Errorf("extension %s not found", id)
	}
	return ext, nil
}

func (es *ExtensionService) ListExtensions() []*ExtensionInfo {
	es.mu.RLock()
	defer es.mu.RUnlock()
	result := make([]*ExtensionInfo, 0, len(es.extensions))
	for _, ext := range es.extensions {
		result = append(result, ext)
	}
	return result
}

func (es *ExtensionService) ActivateExtension(id string) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	ext, ok := es.extensions[id]
	if !ok {
		return fmt.Errorf("extension %s not found", id)
	}
	ext.Activated = true
	return nil
}

func (es *ExtensionService) DeactivateExtension(id string) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	ext, ok := es.extensions[id]
	if !ok {
		return fmt.Errorf("extension %s not found", id)
	}
	ext.Activated = false
	return nil
}

func (es *ExtensionService) InstallExtension(extPath string) error {
	pkgFile := filepath.Join(extPath, "package.json")
	data, err := os.ReadFile(pkgFile)
	if err != nil {
		return err
	}
	var pkg struct {
		Name      string `json:"name"`
		Version   string `json:"version"`
		Publisher string `json:"publisher"`
	}
	if err := json.Unmarshal(data, &pkg); err != nil {
		return err
	}
	id := pkg.Publisher + "." + pkg.Name
	es.mu.Lock()
	es.extensions[id] = &ExtensionInfo{
		ID:          id,
		Name:        pkg.Name,
		Version:     pkg.Version,
		Publisher:   pkg.Publisher,
		InstallPath: extPath,
		Activated:   false,
	}
	es.mu.Unlock()
	return nil
}