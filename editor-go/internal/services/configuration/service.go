package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type ConfigurationService struct {
	configFile string
	config     map[string]interface{}
	mu         sync.RWMutex
}

func NewConfigurationService(configDir string) *ConfigurationService {
	cs := &ConfigurationService{
		configFile: filepath.Join(configDir, "settings.json"),
		config:     make(map[string]interface{}),
	}
	cs.load()
	return cs
}

func (cs *ConfigurationService) load() {
	data, err := os.ReadFile(cs.configFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &cs.config)
}

func (cs *ConfigurationService) save() error {
	data, err := json.MarshalIndent(cs.config, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(cs.configFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(cs.configFile, data, 0644)
}

func (cs *ConfigurationService) Get(key string) interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.config[key]
}

func (cs *ConfigurationService) Set(key string, value interface{}) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.config[key] = value
	return cs.save()
}

func (cs *ConfigurationService) Delete(key string) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.config, key)
	return cs.save()
}

func (cs *ConfigurationService) GetAll() map[string]interface{} {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	result := make(map[string]interface{})
	for k, v := range cs.config {
		result[k] = v
	}
	return result
}

func (cs *ConfigurationService) GetString(key, defaultVal string) string {
	val := cs.Get(key)
	if val == nil {
		return defaultVal
	}
	if s, ok := val.(string); ok {
		return s
	}
	return defaultVal
}

func (cs *ConfigurationService) GetInt(key string, defaultVal int) int {
	val := cs.Get(key)
	if val == nil {
		return defaultVal
	}
	switch v := val.(type) {
	case float64:
		return int(v)
	case int:
		return v
	}
	return defaultVal
}

func (cs *ConfigurationService) GetBool(key string, defaultVal bool) bool {
	val := cs.Get(key)
	if val == nil {
		return defaultVal
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return defaultVal
}