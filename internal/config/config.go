package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	configDirName  = "google-cli"
	configFileName = "config.json"
	tokensDirName  = "tokens"
)

// AccountConfig holds configuration for a single account
type AccountConfig struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	CalendarID   string `json:"calendar_id,omitempty"`
}

// Config holds the overall configuration
type Config struct {
	DefaultAccount string                   `json:"default_account"`
	Accounts       map[string]AccountConfig `json:"accounts"`
}

// GetConfigDir returns the path to the config directory (~/.config/google-cli)
func GetConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(homeDir, ".config", configDirName)
	return configDir, nil
}

// GetTokensDir returns the path to the tokens directory
func GetTokensDir() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, tokensDirName), nil
}

// GetTokenPath returns the path to the token file for a specific account
func GetTokenPath(accountName string) (string, error) {
	tokensDir, err := GetTokensDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(tokensDir, accountName+".json"), nil
}

// GetConfigPath returns the path to the config file
func GetConfigPath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, configFileName), nil
}

// EnsureConfigDir ensures the config directory exists
func EnsureConfigDir() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	tokensDir, err := GetTokensDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.MkdirAll(tokensDir, 0700); err != nil {
		return fmt.Errorf("failed to create tokens directory: %w", err)
	}

	return nil
}

// Load loads the configuration from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return empty config if file doesn't exist
			return &Config{
				Accounts: make(map[string]AccountConfig),
			}, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if cfg.Accounts == nil {
		cfg.Accounts = make(map[string]AccountConfig)
	}

	return &cfg, nil
}

// Save saves the configuration to disk
func (c *Config) Save() error {
	if err := EnsureConfigDir(); err != nil {
		return err
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// AddAccount adds a new account to the configuration
func (c *Config) AddAccount(name string, account AccountConfig) error {
	if _, exists := c.Accounts[name]; exists {
		return fmt.Errorf("account '%s' already exists", name)
	}

	c.Accounts[name] = account

	// If this is the first account, set it as default
	if c.DefaultAccount == "" {
		c.DefaultAccount = name
	}

	return c.Save()
}

// UpdateAccount updates an existing account
func (c *Config) UpdateAccount(name string, account AccountConfig) error {
	if _, exists := c.Accounts[name]; !exists {
		return fmt.Errorf("account '%s' does not exist", name)
	}

	c.Accounts[name] = account
	return c.Save()
}

// RemoveAccount removes an account from the configuration
func (c *Config) RemoveAccount(name string) error {
	if _, exists := c.Accounts[name]; !exists {
		return fmt.Errorf("account '%s' does not exist", name)
	}

	delete(c.Accounts, name)

	// If we removed the default account, set a new default
	if c.DefaultAccount == name {
		c.DefaultAccount = ""
		for accountName := range c.Accounts {
			c.DefaultAccount = accountName
			break
		}
	}

	// Also remove the token file
	tokenPath, err := GetTokenPath(name)
	if err == nil {
		os.Remove(tokenPath)
	}

	return c.Save()
}

// SetDefault sets the default account
func (c *Config) SetDefault(name string) error {
	if _, exists := c.Accounts[name]; !exists {
		return fmt.Errorf("account '%s' does not exist", name)
	}

	c.DefaultAccount = name
	return c.Save()
}

// GetAccount returns an account by name, or the default account if name is empty
func (c *Config) GetAccount(name string) (string, AccountConfig, error) {
	if name == "" {
		name = c.DefaultAccount
	}

	if name == "" {
		return "", AccountConfig{}, fmt.Errorf("no account specified and no default account set")
	}

	account, exists := c.Accounts[name]
	if !exists {
		return "", AccountConfig{}, fmt.Errorf("account '%s' does not exist", name)
	}

	return name, account, nil
}

// GetAllAccounts returns all account names
func (c *Config) GetAllAccounts() []string {
	names := make([]string, 0, len(c.Accounts))
	for name := range c.Accounts {
		names = append(names, name)
	}
	return names
}

// HasAccounts returns true if there are any configured accounts
func (c *Config) HasAccounts() bool {
	return len(c.Accounts) > 0
}
