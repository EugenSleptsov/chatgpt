package service

import (
	conf "GPTBot/config"
	"fmt"
	"sync"
)

// ConfigService provides thread-safe access to the application configuration.
// Read-only consumers receive specific values at construction time;
// admin commands use this service to reload or persist changes.
type ConfigService struct {
	mu         sync.RWMutex
	config     *conf.Config
	configPath string
}

func NewConfigService(config *conf.Config, configPath string) *ConfigService {
	return &ConfigService{config: config, configPath: configPath}
}

// Get returns the current configuration. The caller must NOT mutate the result.
func (cs *ConfigService) Get() *conf.Config {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.config
}

// Reload reads the configuration file and replaces the in-memory config.
// Returns the new config on success.
func (cs *ConfigService) Reload() (*conf.Config, error) {
	newConfig, err := conf.ReadConfig(cs.configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}
	cs.mu.Lock()
	defer cs.mu.Unlock()
	*cs.config = *newConfig
	return cs.config, nil
}

// SetAuthorizedUsers updates the authorized user list and persists to disk.
func (cs *ConfigService) SetAuthorizedUsers(ids []int64) error {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.config.AuthorizedUserIds = ids
	return conf.UpdateConfig(cs.configPath, cs.config)
}

// String returns a human-readable representation of the current config.
func (cs *ConfigService) String() string {
	cs.mu.RLock()
	defer cs.mu.RUnlock()
	return cs.config.String()
}
