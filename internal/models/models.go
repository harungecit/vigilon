package models

import "time"

// MonitoringMode defines how the server is monitored
type MonitoringMode string

const (
	ModePull   MonitoringMode = "pull"   // Central server pulls data
	ModePush   MonitoringMode = "push"   // Agent pushes data
	ModeHybrid MonitoringMode = "hybrid" // SSH + local script
)

// ServiceStatus represents the current status of a service
type ServiceStatus string

const (
	StatusRunning  ServiceStatus = "running"
	StatusStopped  ServiceStatus = "stopped"
	StatusFailed   ServiceStatus = "failed"
	StatusUnknown  ServiceStatus = "unknown"
	StatusDegraded ServiceStatus = "degraded"
)

// ConnectionStatus represents the connection state of a server
type ConnectionStatus string

const (
	ConnectionNotConnected ConnectionStatus = "not_connected" // Never connected
	ConnectionConnected    ConnectionStatus = "connected"     // Currently active
	ConnectionIdle         ConnectionStatus = "idle"          // Was connected, but no recent activity
	ConnectionDisconnected ConnectionStatus = "disconnected"  // Manually disconnected
)

// Server represents a monitored server
type Server struct {
	ID               int              `json:"id"`
	Name             string           `json:"name"`
	Hostname         string           `json:"hostname"`
	IPAddress        string           `json:"ip_address"`
	Port             int              `json:"port"`
	OS               string           `json:"os"` // linux, windows, etc.
	MonitoringMode   MonitoringMode   `json:"monitoring_mode"`
	SSHUser          string           `json:"ssh_user,omitempty"`
	SSHKeyPath       string           `json:"ssh_key_path,omitempty"`
	SSHJumpHost      string           `json:"ssh_jump_host,omitempty"`     // Jump host for SSH tunnel
	SSHJumpUser      string           `json:"ssh_jump_user,omitempty"`     // Jump host user
	SSHJumpKeyPath   string           `json:"ssh_jump_key_path,omitempty"` // Jump host key
	AgentToken       string           `json:"agent_token,omitempty"`
	CheckInterval    int              `json:"check_interval"` // Check interval in seconds (0 = use default)
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	Enabled          bool             `json:"enabled"`
	LastSeen         *time.Time       `json:"last_seen"`
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
	NotifyTelegram   bool             `json:"notify_telegram"`
}

// Service represents a service to monitor on a server
type Service struct {
	ID          int       `json:"id"`
	ServerID    int       `json:"server_id"`
	Name        string    `json:"name"` // e.g., "rftt.service", "nginx", etc.
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// ServiceCheck represents a monitoring check result
type ServiceCheck struct {
	ID           int           `json:"id"`
	ServiceID    int           `json:"service_id"`
	Status       ServiceStatus `json:"status"`
	ResponseTime int64         `json:"response_time_ms"` // in milliseconds
	ErrorMessage string        `json:"error_message,omitempty"`
	CheckedAt    time.Time     `json:"checked_at"`
	PID          int           `json:"pid,omitempty"`
	Memory       int64         `json:"memory_kb,omitempty"` // in KB
	CPU          float64       `json:"cpu_percent,omitempty"`
	Uptime       int64         `json:"uptime_seconds,omitempty"`
}

// Alert represents a notification sent
type Alert struct {
	ID             int           `json:"id"`
	ServiceID      int           `json:"service_id"`
	ServerID       int           `json:"server_id"`
	Status         ServiceStatus `json:"status"`
	Message        string        `json:"message"`
	SentVia        string        `json:"sent_via"` // telegram, email, etc.
	Acknowledged   bool          `json:"acknowledged"`
	Archived       bool          `json:"archived"`
	CreatedAt      time.Time     `json:"created_at"`
	AcknowledgedAt *time.Time    `json:"acknowledged_at,omitempty"`
	ArchivedAt     *time.Time    `json:"archived_at,omitempty"`
}

// Config represents application configuration
type Config struct {
	ID        int       `json:"id"`
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// TelegramConfig holds Telegram bot configuration
type TelegramConfig struct {
	BotToken string   `json:"bot_token"`
	ChatIDs  []string `json:"chat_ids"`
	Enabled  bool     `json:"enabled"`
}

// User represents a system user
type User struct {
	ID           int        `json:"id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"` // Never send password hash to client
	RoleID       int        `json:"role_id"`
	Role         *Role      `json:"role,omitempty"`
	Enabled      bool       `json:"enabled"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

// Role represents a user role with permissions
type Role struct {
	ID           int          `json:"id"`
	Name         string       `json:"name"`
	DisplayName  string       `json:"display_name"`
	Description  string       `json:"description"`
	IsSuperAdmin bool         `json:"is_super_admin"` // Super admin cannot be deleted
	IsSystem     bool         `json:"is_system"`      // System roles cannot be modified
	Permissions  []Permission `json:"permissions,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Permission represents an action that can be performed
type Permission struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	DisplayName string    `json:"display_name"`
	Description string    `json:"description"`
	Category    string    `json:"category"` // servers, services, alerts, users
	CreatedAt   time.Time `json:"created_at"`
}

// RolePermission links roles to permissions
type RolePermission struct {
	RoleID       int       `json:"role_id"`
	PermissionID int       `json:"permission_id"`
	CreatedAt    time.Time `json:"created_at"`
}

// Session represents a user session
type Session struct {
	ID        string    `json:"id"`
	UserID    int       `json:"user_id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
	IPAddress string    `json:"ip_address"`
	UserAgent string    `json:"user_agent"`
}
