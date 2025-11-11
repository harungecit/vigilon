package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
	"github.com/harungecit/vigilon/internal/models"
)

// AppConfig represents the application configuration
type AppConfig struct {
	Server    ServerConfig              `yaml:"server"`
	Database  DatabaseConfig            `yaml:"database"`
	Telegram  models.TelegramConfig     `yaml:"telegram"`
	Monitoring MonitoringConfig         `yaml:"monitoring"`
	Servers   []ServerDefinition        `yaml:"servers"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type DatabaseConfig struct {
	Path string `yaml:"path"`
}

type MonitoringConfig struct {
	CheckInterval    time.Duration `yaml:"check_interval"`
	RetentionDays    int           `yaml:"retention_days"`
	AlertCooldown    time.Duration `yaml:"alert_cooldown"`
}

type ServerDefinition struct {
	Name           string                  `yaml:"name"`
	Hostname       string                  `yaml:"hostname"`
	IPAddress      string                  `yaml:"ip_address"`
	Port           int                     `yaml:"port"`
	OS             string                  `yaml:"os"`
	MonitoringMode models.MonitoringMode   `yaml:"monitoring_mode"`
	SSHUser        string                  `yaml:"ssh_user,omitempty"`
	SSHKeyPath     string                  `yaml:"ssh_key_path,omitempty"`
	AgentToken     string                  `yaml:"agent_token,omitempty"`
	Enabled        bool                    `yaml:"enabled"`
	NotifyTelegram bool                    `yaml:"notify_telegram"`
	Services       []ServiceDefinition     `yaml:"services"`
}

type ServiceDefinition struct {
	Name        string `yaml:"name"`
	DisplayName string `yaml:"display_name"`
	Description string `yaml:"description"`
	Enabled     bool   `yaml:"enabled"`
}

// LoadFromFile loads configuration from a YAML file
func LoadFromFile(path string) (*AppConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.Server.Host == "" {
		config.Server.Host = "0.0.0.0"
	}
	if config.Server.Port == 0 {
		config.Server.Port = 8080
	}
	if config.Database.Path == "" {
		config.Database.Path = "./vigilon.db"
	}
	if config.Monitoring.CheckInterval == 0 {
		config.Monitoring.CheckInterval = 30 * time.Second
	}
	if config.Monitoring.RetentionDays == 0 {
		config.Monitoring.RetentionDays = 30
	}
	if config.Monitoring.AlertCooldown == 0 {
		config.Monitoring.AlertCooldown = 5 * time.Minute
	}

	return &config, nil
}

// SaveToFile saves configuration to a YAML file
func SaveToFile(config *AppConfig, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// GetDefaultConfig returns a default configuration
func GetDefaultConfig() *AppConfig {
	return &AppConfig{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{
			Path: "./vigilon.db",
		},
		Telegram: models.TelegramConfig{
			Enabled: false,
		},
		Monitoring: MonitoringConfig{
			CheckInterval: 30 * time.Second,
			RetentionDays: 30,
			AlertCooldown: 5 * time.Minute,
		},
		Servers: []ServerDefinition{},
	}
}
