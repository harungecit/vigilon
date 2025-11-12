package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/harungecit/vigilon/internal/models"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type DB struct {
	conn *sql.DB
}

// New creates a new database connection
func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable WAL mode for better performance
	if _, err := conn.Exec("PRAGMA journal_mode=WAL;"); err != nil {
		return nil, fmt.Errorf("failed to enable WAL mode: %w", err)
	}

	// Set other performance optimizations
	conn.Exec("PRAGMA synchronous=NORMAL;")
	conn.Exec("PRAGMA cache_size=-64000;") // 64MB cache
	conn.Exec("PRAGMA busy_timeout=5000;")
	conn.Exec("PRAGMA foreign_keys=ON;")

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

// Close closes the database connection
func (db *DB) Close() error {
	return db.conn.Close()
}

// initSchema creates all necessary tables
func (db *DB) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS servers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		hostname TEXT NOT NULL,
		ip_address TEXT NOT NULL,
		port INTEGER DEFAULT 22,
		os TEXT NOT NULL,
		monitoring_mode TEXT NOT NULL CHECK(monitoring_mode IN ('pull', 'push', 'hybrid')),
		ssh_user TEXT,
		ssh_key_path TEXT,
		ssh_jump_host TEXT,
		ssh_jump_user TEXT,
		ssh_jump_key_path TEXT,
		agent_token TEXT,
		check_interval INTEGER DEFAULT 0,
		connection_status TEXT DEFAULT 'not_connected' CHECK(connection_status IN ('not_connected', 'connected', 'idle', 'disconnected')),
		enabled BOOLEAN DEFAULT 1,
		last_seen DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		notify_telegram BOOLEAN DEFAULT 1
	);

	CREATE TABLE IF NOT EXISTS services (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		server_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		display_name TEXT NOT NULL,
		description TEXT,
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE,
		UNIQUE(server_id, name)
	);

	CREATE TABLE IF NOT EXISTS service_checks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id INTEGER NOT NULL,
		status TEXT NOT NULL CHECK(status IN ('running', 'stopped', 'failed', 'unknown', 'degraded')),
		response_time_ms INTEGER DEFAULT 0,
		error_message TEXT,
		checked_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		pid INTEGER,
		memory_kb INTEGER,
		cpu_percent REAL,
		uptime_seconds INTEGER,
		FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS alerts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		service_id INTEGER NOT NULL,
		server_id INTEGER NOT NULL,
		status TEXT NOT NULL,
		message TEXT NOT NULL,
		sent_via TEXT NOT NULL,
		acknowledged BOOLEAN DEFAULT 0,
		archived BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		acknowledged_at DATETIME,
		archived_at DATETIME,
		FOREIGN KEY (service_id) REFERENCES services(id) ON DELETE CASCADE,
		FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS config (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key TEXT NOT NULL UNIQUE,
		value TEXT NOT NULL,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		username TEXT NOT NULL UNIQUE,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		role_id INTEGER NOT NULL,
		enabled BOOLEAN DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		last_login_at DATETIME,
		FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE RESTRICT
	);

	CREATE TABLE IF NOT EXISTS roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL,
		description TEXT,
		is_super_admin BOOLEAN DEFAULT 0,
		is_system BOOLEAN DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS permissions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL UNIQUE,
		display_name TEXT NOT NULL,
		description TEXT,
		category TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS role_permissions (
		role_id INTEGER NOT NULL,
		permission_id INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		PRIMARY KEY (role_id, permission_id),
		FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE,
		FOREIGN KEY (permission_id) REFERENCES permissions(id) ON DELETE CASCADE
	);

	CREATE TABLE IF NOT EXISTS sessions (
		id TEXT PRIMARY KEY,
		user_id INTEGER NOT NULL,
		token TEXT NOT NULL UNIQUE,
		expires_at DATETIME NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		ip_address TEXT,
		user_agent TEXT,
		FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_servers_enabled ON servers(enabled);
	CREATE INDEX IF NOT EXISTS idx_services_server_id ON services(server_id);
	CREATE INDEX IF NOT EXISTS idx_service_checks_service_id ON service_checks(service_id);
	CREATE INDEX IF NOT EXISTS idx_service_checks_checked_at ON service_checks(checked_at);
	CREATE INDEX IF NOT EXISTS idx_alerts_acknowledged ON alerts(acknowledged);
	CREATE INDEX IF NOT EXISTS idx_alerts_created_at ON alerts(created_at);
	CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
	CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id);
	CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(token);
	CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
	`

	_, err := db.conn.Exec(schema)
	if err != nil {
		return err
	}

	// Migration: Add connection_status column if it doesn't exist
	migrationQuery := `
		ALTER TABLE servers ADD COLUMN connection_status TEXT DEFAULT 'not_connected'
		CHECK(connection_status IN ('not_connected', 'connected', 'idle', 'disconnected'));
	`
	// Try to add the column, ignore error if it already exists
	db.conn.Exec(migrationQuery)

	// Migration: Add archived and archived_at columns to alerts if they don't exist
	// Check if archived column exists
	var columnExists int
	checkQuery := `SELECT COUNT(*) FROM pragma_table_info('alerts') WHERE name='archived'`
	db.conn.QueryRow(checkQuery).Scan(&columnExists)

	if columnExists == 0 {
		// Column doesn't exist, add it
		db.conn.Exec(`ALTER TABLE alerts ADD COLUMN archived BOOLEAN DEFAULT 0;`)
		db.conn.Exec(`ALTER TABLE alerts ADD COLUMN archived_at DATETIME;`)
	}

	// Create index for archived column (will be ignored if already exists)
	db.conn.Exec(`CREATE INDEX IF NOT EXISTS idx_alerts_archived ON alerts(archived);`)

	// Initialize default roles and permissions
	if err := db.initializeAuthDefaults(); err != nil {
		return fmt.Errorf("failed to initialize auth defaults: %w", err)
	}

	return nil
}

// initializeAuthDefaults creates default roles, permissions and super admin user
func (db *DB) initializeAuthDefaults() error {
	// Check if roles already exist
	var count int
	db.conn.QueryRow("SELECT COUNT(*) FROM roles").Scan(&count)
	if count > 0 {
		return nil // Already initialized
	}

	// Create permissions
	permissions := []struct {
		name, displayName, description, category string
	}{
		// Server permissions
		{"servers.view", "View Servers", "View server list and details", "servers"},
		{"servers.create", "Create Servers", "Add new servers", "servers"},
		{"servers.edit", "Edit Servers", "Modify server settings", "servers"},
		{"servers.delete", "Delete Servers", "Remove servers", "servers"},
		{"servers.toggle", "Enable/Disable Servers", "Enable or disable server monitoring", "servers"},

		// Service permissions
		{"services.view", "View Services", "View service list and details", "services"},
		{"services.create", "Create Services", "Add new services", "services"},
		{"services.edit", "Edit Services", "Modify service settings", "services"},
		{"services.delete", "Delete Services", "Remove services", "services"},
		{"services.toggle", "Enable/Disable Services", "Enable or disable service monitoring", "services"},

		// Alert permissions
		{"alerts.view", "View Alerts", "View alerts", "alerts"},
		{"alerts.acknowledge", "Acknowledge Alerts", "Acknowledge alerts", "alerts"},
		{"alerts.archive", "Archive Alerts", "Archive alerts", "alerts"},

		// User permissions
		{"users.view", "View Users", "View user list", "users"},
		{"users.create", "Create Users", "Add new users", "users"},
		{"users.edit", "Edit Users", "Modify user settings", "users"},
		{"users.delete", "Delete Users", "Remove users", "users"},

		// Role permissions
		{"roles.view", "View Roles", "View role list", "roles"},
		{"roles.create", "Create Roles", "Add new roles", "roles"},
		{"roles.edit", "Edit Roles", "Modify role settings", "roles"},
		{"roles.delete", "Delete Roles", "Remove roles", "roles"},

		// Settings permissions
		{"settings.view", "View Settings", "View system settings", "settings"},
		{"settings.edit", "Edit Settings", "Modify system settings", "settings"},
	}

	for _, p := range permissions {
		_, err := db.conn.Exec(`
			INSERT INTO permissions (name, display_name, description, category)
			VALUES (?, ?, ?, ?)
		`, p.name, p.displayName, p.description, p.category)
		if err != nil {
			return fmt.Errorf("failed to create permission %s: %w", p.name, err)
		}
	}

	// Create roles
	roles := []struct {
		name, displayName, description string
		isSuperAdmin, isSystem         bool
	}{
		{"super_admin", "Super Administrator", "Full system access, cannot be deleted", true, true},
		{"admin", "Administrator", "Full access except super admin management", false, true},
		{"user", "User", "Read-only access to all resources", false, true},
	}

	for _, r := range roles {
		_, err := db.conn.Exec(`
			INSERT INTO roles (name, display_name, description, is_super_admin, is_system)
			VALUES (?, ?, ?, ?, ?)
		`, r.name, r.displayName, r.description, r.isSuperAdmin, r.isSystem)
		if err != nil {
			return fmt.Errorf("failed to create role %s: %w", r.name, err)
		}
	}

	// Assign permissions to roles
	// Super Admin gets all permissions
	db.conn.Exec(`
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT 1, id FROM permissions
	`)

	// Admin gets all permissions except user deletion
	db.conn.Exec(`
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT 2, id FROM permissions WHERE name != 'users.delete'
	`)

	// User gets only view permissions
	db.conn.Exec(`
		INSERT INTO role_permissions (role_id, permission_id)
		SELECT 3, id FROM permissions WHERE name LIKE '%.view'
	`)

	// Create default super admin user (username: root, password: toor)
	// IMPORTANT: Change this password immediately after first login
	passwordHash, err := bcrypt.GenerateFromPassword([]byte("toor"), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash default password: %w", err)
	}

	_, err = db.conn.Exec(`
		INSERT INTO users (username, email, password_hash, role_id, enabled)
		VALUES (?, ?, ?, ?, ?)
	`, "root", "root@vigilon.local", string(passwordHash), 1, true)

	if err != nil {
		return fmt.Errorf("failed to create default admin user: %w", err)
	}

	return nil
}

// Server operations

func (db *DB) CreateServer(server *models.Server) error {
	// Set default connection status if empty
	if server.ConnectionStatus == "" {
		server.ConnectionStatus = models.ConnectionNotConnected
	}

	query := `
		INSERT INTO servers (name, hostname, ip_address, port, os, monitoring_mode,
			ssh_user, ssh_key_path, ssh_jump_host, ssh_jump_user, ssh_jump_key_path,
			agent_token, check_interval, connection_status, enabled, notify_telegram)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.conn.Exec(query, server.Name, server.Hostname, server.IPAddress,
		server.Port, server.OS, server.MonitoringMode, server.SSHUser, server.SSHKeyPath,
		server.SSHJumpHost, server.SSHJumpUser, server.SSHJumpKeyPath,
		server.AgentToken, server.CheckInterval, server.ConnectionStatus, server.Enabled, server.NotifyTelegram)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	server.ID = int(id)
	return nil
}

func (db *DB) GetServer(id int) (*models.Server, error) {
	query := `
		SELECT id, name, hostname, ip_address, port, os, monitoring_mode,
			ssh_user, ssh_key_path, ssh_jump_host, ssh_jump_user, ssh_jump_key_path,
			agent_token, check_interval, connection_status, enabled, last_seen,
			created_at, updated_at, notify_telegram
		FROM servers WHERE id = ?
	`
	server := &models.Server{}
	err := db.conn.QueryRow(query, id).Scan(
		&server.ID, &server.Name, &server.Hostname, &server.IPAddress,
		&server.Port, &server.OS, &server.MonitoringMode, &server.SSHUser,
		&server.SSHKeyPath, &server.SSHJumpHost, &server.SSHJumpUser, &server.SSHJumpKeyPath,
		&server.AgentToken, &server.CheckInterval, &server.ConnectionStatus, &server.Enabled, &server.LastSeen,
		&server.CreatedAt, &server.UpdatedAt, &server.NotifyTelegram,
	)
	if err != nil {
		return nil, err
	}
	return server, nil
}

func (db *DB) GetAllServers() ([]*models.Server, error) {
	query := `
		SELECT id, name, hostname, ip_address, port, os, monitoring_mode,
			ssh_user, ssh_key_path, ssh_jump_host, ssh_jump_user, ssh_jump_key_path,
			agent_token, check_interval, connection_status, enabled, last_seen,
			created_at, updated_at, notify_telegram
		FROM servers ORDER BY name
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*models.Server
	for rows.Next() {
		server := &models.Server{}
		err := rows.Scan(
			&server.ID, &server.Name, &server.Hostname, &server.IPAddress,
			&server.Port, &server.OS, &server.MonitoringMode, &server.SSHUser,
			&server.SSHKeyPath, &server.SSHJumpHost, &server.SSHJumpUser, &server.SSHJumpKeyPath,
			&server.AgentToken, &server.CheckInterval, &server.ConnectionStatus, &server.Enabled, &server.LastSeen,
			&server.CreatedAt, &server.UpdatedAt, &server.NotifyTelegram,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}
	return servers, nil
}

func (db *DB) UpdateServer(server *models.Server) error {
	query := `
		UPDATE servers SET name = ?, hostname = ?, ip_address = ?, port = ?, os = ?,
			monitoring_mode = ?, ssh_user = ?, ssh_key_path = ?, ssh_jump_host = ?,
			ssh_jump_user = ?, ssh_jump_key_path = ?, agent_token = ?, check_interval = ?,
			connection_status = ?, enabled = ?, notify_telegram = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, server.Name, server.Hostname, server.IPAddress,
		server.Port, server.OS, server.MonitoringMode, server.SSHUser, server.SSHKeyPath,
		server.SSHJumpHost, server.SSHJumpUser, server.SSHJumpKeyPath,
		server.AgentToken, server.CheckInterval, server.ConnectionStatus, server.Enabled, server.NotifyTelegram, server.ID)
	return err
}

func (db *DB) UpdateServerLastSeen(id int) error {
	query := `UPDATE servers SET last_seen = ?, connection_status = 'connected' WHERE id = ?`
	_, err := db.conn.Exec(query, time.Now(), id)
	return err
}

func (db *DB) UpdateServerConnectionStatus(id int, status models.ConnectionStatus) error {
	query := `UPDATE servers SET connection_status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, status, id)
	return err
}

func (db *DB) DeleteServer(id int) error {
	query := `DELETE FROM servers WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// Service operations

func (db *DB) CreateService(service *models.Service) error {
	query := `
		INSERT INTO services (server_id, name, display_name, description, enabled)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := db.conn.Exec(query, service.ServerID, service.Name,
		service.DisplayName, service.Description, service.Enabled)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	service.ID = int(id)
	return nil
}

func (db *DB) GetService(id int) (*models.Service, error) {
	query := `
		SELECT id, server_id, name, display_name, description, enabled,
			created_at, updated_at
		FROM services WHERE id = ?
	`
	service := &models.Service{}
	err := db.conn.QueryRow(query, id).Scan(
		&service.ID, &service.ServerID, &service.Name, &service.DisplayName,
		&service.Description, &service.Enabled, &service.CreatedAt, &service.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return service, nil
}

func (db *DB) GetServicesByServer(serverID int) ([]*models.Service, error) {
	query := `
		SELECT id, server_id, name, display_name, description, enabled,
			created_at, updated_at
		FROM services WHERE server_id = ? ORDER BY name
	`
	rows, err := db.conn.Query(query, serverID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []*models.Service
	for rows.Next() {
		service := &models.Service{}
		err := rows.Scan(
			&service.ID, &service.ServerID, &service.Name, &service.DisplayName,
			&service.Description, &service.Enabled, &service.CreatedAt, &service.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		services = append(services, service)
	}
	return services, nil
}

func (db *DB) UpdateService(service *models.Service) error {
	query := `
		UPDATE services SET name = ?, display_name = ?, description = ?,
			enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, service.Name, service.DisplayName,
		service.Description, service.Enabled, service.ID)
	return err
}

func (db *DB) DeleteService(id int) error {
	query := `DELETE FROM services WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// ServiceCheck operations

func (db *DB) CreateServiceCheck(check *models.ServiceCheck) error {
	query := `
		INSERT INTO service_checks (service_id, status, response_time_ms, error_message,
			pid, memory_kb, cpu_percent, uptime_seconds)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`
	result, err := db.conn.Exec(query, check.ServiceID, check.Status, check.ResponseTime,
		check.ErrorMessage, check.PID, check.Memory, check.CPU, check.Uptime)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	check.ID = int(id)
	return nil
}

func (db *DB) GetLatestServiceCheck(serviceID int) (*models.ServiceCheck, error) {
	query := `
		SELECT id, service_id, status, response_time_ms, error_message, checked_at,
			pid, memory_kb, cpu_percent, uptime_seconds
		FROM service_checks WHERE service_id = ?
		ORDER BY checked_at DESC LIMIT 1
	`
	check := &models.ServiceCheck{}
	err := db.conn.QueryRow(query, serviceID).Scan(
		&check.ID, &check.ServiceID, &check.Status, &check.ResponseTime,
		&check.ErrorMessage, &check.CheckedAt, &check.PID, &check.Memory,
		&check.CPU, &check.Uptime,
	)
	if err != nil {
		return nil, err
	}
	return check, nil
}

func (db *DB) GetServiceCheckHistory(serviceID int, limit int) ([]*models.ServiceCheck, error) {
	query := `
		SELECT id, service_id, status, response_time_ms, error_message, checked_at,
			pid, memory_kb, cpu_percent, uptime_seconds
		FROM service_checks WHERE service_id = ?
		ORDER BY checked_at DESC LIMIT ?
	`
	rows, err := db.conn.Query(query, serviceID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []*models.ServiceCheck
	for rows.Next() {
		check := &models.ServiceCheck{}
		err := rows.Scan(
			&check.ID, &check.ServiceID, &check.Status, &check.ResponseTime,
			&check.ErrorMessage, &check.CheckedAt, &check.PID, &check.Memory,
			&check.CPU, &check.Uptime,
		)
		if err != nil {
			return nil, err
		}
		checks = append(checks, check)
	}
	return checks, nil
}

// Alert operations

func (db *DB) CreateAlert(alert *models.Alert) error {
	query := `
		INSERT INTO alerts (service_id, server_id, status, message, sent_via)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := db.conn.Exec(query, alert.ServiceID, alert.ServerID,
		alert.Status, alert.Message, alert.SentVia)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	alert.ID = int(id)
	return nil
}

func (db *DB) GetRecentAlerts(limit int) ([]*models.Alert, error) {
	return db.GetRecentAlertsWithOffset(limit, 0)
}

func (db *DB) GetRecentAlertsWithOffset(limit, offset int) ([]*models.Alert, error) {
	query := `
		SELECT id, service_id, server_id, status, message, sent_via,
			acknowledged, archived, created_at, acknowledged_at, archived_at
		FROM alerts WHERE archived = 0 ORDER BY created_at DESC LIMIT ? OFFSET ?
	`
	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		alert := &models.Alert{}
		err := rows.Scan(
			&alert.ID, &alert.ServiceID, &alert.ServerID, &alert.Status,
			&alert.Message, &alert.SentVia, &alert.Acknowledged, &alert.Archived,
			&alert.CreatedAt, &alert.AcknowledgedAt, &alert.ArchivedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

func (db *DB) AcknowledgeAlert(id int) error {
	query := `UPDATE alerts SET acknowledged = 1, acknowledged_at = ? WHERE id = ?`
	_, err := db.conn.Exec(query, time.Now(), id)
	return err
}

func (db *DB) ArchiveAlert(id int) error {
	query := `UPDATE alerts SET archived = 1, archived_at = ? WHERE id = ?`
	_, err := db.conn.Exec(query, time.Now(), id)
	return err
}

func (db *DB) ArchiveAllAlerts() error {
	query := `UPDATE alerts SET archived = 1, archived_at = ? WHERE archived = 0`
	_, err := db.conn.Exec(query, time.Now())
	return err
}

func (db *DB) GetArchivedAlerts(limit, offset int) ([]*models.Alert, error) {
	query := `
		SELECT id, service_id, server_id, status, message, sent_via,
			acknowledged, archived, created_at, acknowledged_at, archived_at
		FROM alerts WHERE archived = 1 ORDER BY archived_at DESC LIMIT ? OFFSET ?
	`
	rows, err := db.conn.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		alert := &models.Alert{}
		err := rows.Scan(
			&alert.ID, &alert.ServiceID, &alert.ServerID, &alert.Status,
			&alert.Message, &alert.SentVia, &alert.Acknowledged, &alert.Archived,
			&alert.CreatedAt, &alert.AcknowledgedAt, &alert.ArchivedAt,
		)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}
	return alerts, nil
}

func (db *DB) UnarchiveAlert(id int) error {
	query := `UPDATE alerts SET archived = 0, archived_at = NULL WHERE id = ?`
	_, err := db.conn.Exec(query, id)
	return err
}

// Config operations

func (db *DB) SetConfig(key, value string) error {
	query := `
		INSERT INTO config (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = ?, updated_at = CURRENT_TIMESTAMP
	`
	_, err := db.conn.Exec(query, key, value, value)
	return err
}

func (db *DB) GetConfig(key string) (string, error) {
	query := `SELECT value FROM config WHERE key = ?`
	var value string
	err := db.conn.QueryRow(query, key).Scan(&value)
	return value, err
}

// User operations

func (db *DB) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (username, email, password_hash, role_id, enabled)
		VALUES (?, ?, ?, ?, ?)
	`
	result, err := db.conn.Exec(query, user.Username, user.Email, user.PasswordHash, user.RoleID, user.Enabled)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = int(id)
	return nil
}

func (db *DB) GetUser(id int) (*models.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.role_id, u.enabled,
			u.created_at, u.updated_at, u.last_login_at,
			r.id, r.name, r.display_name, r.description, r.is_super_admin, r.is_system
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.id = ?
	`
	user := &models.User{Role: &models.Role{}}
	err := db.conn.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.RoleID, &user.Enabled,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.Role.ID, &user.Role.Name, &user.Role.DisplayName, &user.Role.Description,
		&user.Role.IsSuperAdmin, &user.Role.IsSystem,
	)
	if err != nil {
		return nil, err
	}

	// Load role permissions
	if user.Role != nil && user.Role.ID > 0 {
		permQuery := `
			SELECT p.id, p.name, p.display_name, p.description, p.category
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			WHERE rp.role_id = ?
			ORDER BY p.category, p.name
		`
		rows, err := db.conn.Query(permQuery, user.Role.ID)
		if err != nil {
			return user, nil // Return user even if permissions fail
		}
		defer rows.Close()

		for rows.Next() {
			perm := models.Permission{}
			err := rows.Scan(&perm.ID, &perm.Name, &perm.DisplayName, &perm.Description, &perm.Category)
			if err != nil {
				continue
			}
			user.Role.Permissions = append(user.Role.Permissions, perm)
		}
	}

	return user, nil
}

func (db *DB) GetUserByUsername(username string) (*models.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.role_id, u.enabled,
			u.created_at, u.updated_at, u.last_login_at,
			r.id, r.name, r.display_name, r.description, r.is_super_admin, r.is_system
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		WHERE u.username = ?
	`
	user := &models.User{Role: &models.Role{}}
	err := db.conn.QueryRow(query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.RoleID, &user.Enabled,
		&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		&user.Role.ID, &user.Role.Name, &user.Role.DisplayName, &user.Role.Description,
		&user.Role.IsSuperAdmin, &user.Role.IsSystem,
	)
	if err != nil {
		return nil, err
	}

	// Load role permissions
	if user.Role != nil && user.Role.ID > 0 {
		permQuery := `
			SELECT p.id, p.name, p.display_name, p.description, p.category
			FROM permissions p
			JOIN role_permissions rp ON p.id = rp.permission_id
			WHERE rp.role_id = ?
			ORDER BY p.category, p.name
		`
		rows, err := db.conn.Query(permQuery, user.Role.ID)
		if err != nil {
			return user, nil
		}
		defer rows.Close()

		for rows.Next() {
			perm := models.Permission{}
			err := rows.Scan(&perm.ID, &perm.Name, &perm.DisplayName, &perm.Description, &perm.Category)
			if err != nil {
				continue
			}
			user.Role.Permissions = append(user.Role.Permissions, perm)
		}
	}

	return user, nil
}

func (db *DB) GetAllUsers() ([]*models.User, error) {
	query := `
		SELECT u.id, u.username, u.email, u.password_hash, u.role_id, u.enabled,
			u.created_at, u.updated_at, u.last_login_at,
			r.id, r.name, r.display_name, r.description, r.is_super_admin, r.is_system
		FROM users u
		LEFT JOIN roles r ON u.role_id = r.id
		ORDER BY u.created_at DESC
	`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{Role: &models.Role{}}
		err := rows.Scan(
			&user.ID, &user.Username, &user.Email, &user.PasswordHash, &user.RoleID, &user.Enabled,
			&user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
			&user.Role.ID, &user.Role.Name, &user.Role.DisplayName, &user.Role.Description,
			&user.Role.IsSuperAdmin, &user.Role.IsSystem,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

func (db *DB) UpdateUser(user *models.User) error {
	query := `
		UPDATE users SET username = ?, email = ?, role_id = ?, enabled = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	_, err := db.conn.Exec(query, user.Username, user.Email, user.RoleID, user.Enabled, user.ID)
	return err
}

func (db *DB) UpdateUserPassword(userID int, passwordHash string) error {
	query := `UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, passwordHash, userID)
	return err
}

func (db *DB) UpdateUserLastLogin(userID int) error {
	query := `UPDATE users SET last_login_at = ? WHERE id = ?`
	_, err := db.conn.Exec(query, time.Now(), userID)
	return err
}

func (db *DB) DeleteUser(id int) error {
	// Check if user is super admin
	var isSuperAdmin bool
	err := db.conn.QueryRow(`
		SELECT r.is_super_admin FROM users u
		JOIN roles r ON u.role_id = r.id
		WHERE u.id = ?
	`, id).Scan(&isSuperAdmin)

	if err != nil {
		return err
	}

	if isSuperAdmin {
		return fmt.Errorf("cannot delete super admin user")
	}

	query := `DELETE FROM users WHERE id = ?`
	_, err = db.conn.Exec(query, id)
	return err
}

// Role operations

func (db *DB) GetAllRoles() ([]*models.Role, error) {
	query := `SELECT id, name, display_name, description, is_super_admin, is_system, created_at, updated_at FROM roles ORDER BY name`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []*models.Role
	for rows.Next() {
		role := &models.Role{}
		err := rows.Scan(&role.ID, &role.Name, &role.DisplayName, &role.Description,
			&role.IsSuperAdmin, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	return roles, nil
}

func (db *DB) GetRole(id int) (*models.Role, error) {
	query := `SELECT id, name, display_name, description, is_super_admin, is_system, created_at, updated_at FROM roles WHERE id = ?`
	role := &models.Role{}
	err := db.conn.QueryRow(query, id).Scan(&role.ID, &role.Name, &role.DisplayName, &role.Description,
		&role.IsSuperAdmin, &role.IsSystem, &role.CreatedAt, &role.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Load permissions
	permissions, err := db.GetRolePermissions(id)
	if err == nil {
		role.Permissions = permissions
	}

	return role, nil
}

func (db *DB) GetRolePermissions(roleID int) ([]models.Permission, error) {
	query := `
		SELECT p.id, p.name, p.display_name, p.description, p.category, p.created_at
		FROM permissions p
		JOIN role_permissions rp ON p.id = rp.permission_id
		WHERE rp.role_id = ?
		ORDER BY p.category, p.name
	`
	rows, err := db.conn.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		perm := models.Permission{}
		err := rows.Scan(&perm.ID, &perm.Name, &perm.DisplayName, &perm.Description, &perm.Category, &perm.CreatedAt)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

func (db *DB) GetAllPermissions() ([]models.Permission, error) {
	query := `SELECT id, name, display_name, description, category, created_at FROM permissions ORDER BY category, name`
	rows, err := db.conn.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var permissions []models.Permission
	for rows.Next() {
		perm := models.Permission{}
		err := rows.Scan(&perm.ID, &perm.Name, &perm.DisplayName, &perm.Description, &perm.Category, &perm.CreatedAt)
		if err != nil {
			return nil, err
		}
		permissions = append(permissions, perm)
	}
	return permissions, nil
}

func (db *DB) UserHasPermission(userID int, permissionName string) (bool, error) {
	query := `
		SELECT COUNT(*) FROM role_permissions rp
		JOIN permissions p ON rp.permission_id = p.id
		JOIN users u ON u.role_id = rp.role_id
		WHERE u.id = ? AND p.name = ?
	`
	var count int
	err := db.conn.QueryRow(query, userID, permissionName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (db *DB) UpdateRolePermissions(roleID int, permissionIDs []int) error {
	// Start transaction
	tx, err := db.conn.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete existing permissions
	_, err = tx.Exec("DELETE FROM role_permissions WHERE role_id = ?", roleID)
	if err != nil {
		return err
	}

	// Insert new permissions
	stmt, err := tx.Prepare("INSERT INTO role_permissions (role_id, permission_id) VALUES (?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, permID := range permissionIDs {
		_, err = stmt.Exec(roleID, permID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (db *DB) CreateRole(role *models.Role) error {
	query := `INSERT INTO roles (name, display_name, description, is_super_admin, is_system) VALUES (?, ?, ?, 0, 0)`
	result, err := db.conn.Exec(query, role.Name, role.DisplayName, role.Description)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	role.ID = int(id)
	return nil
}

func (db *DB) UpdateRole(role *models.Role) error {
	query := `UPDATE roles SET name = ?, display_name = ?, description = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := db.conn.Exec(query, role.Name, role.DisplayName, role.Description, role.ID)
	return err
}

func (db *DB) DeleteRole(roleID int) error {
	query := `DELETE FROM roles WHERE id = ?`
	_, err := db.conn.Exec(query, roleID)
	return err
}

func (db *DB) GetUsersByRole(roleID int) ([]*models.User, error) {
	query := `SELECT id, username, email FROM users WHERE role_id = ?`
	rows, err := db.conn.Query(query, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		if err := rows.Scan(&user.ID, &user.Username, &user.Email); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, nil
}

// Session operations

func (db *DB) CreateSession(session *models.Session) error {
	query := `
		INSERT INTO sessions (id, user_id, token, expires_at, ip_address, user_agent)
		VALUES (?, ?, ?, ?, ?, ?)
	`
	_, err := db.conn.Exec(query, session.ID, session.UserID, session.Token, session.ExpiresAt, session.IPAddress, session.UserAgent)
	return err
}

func (db *DB) GetSessionByToken(token string) (*models.Session, error) {
	query := `SELECT id, user_id, token, expires_at, created_at, ip_address, user_agent FROM sessions WHERE token = ?`
	session := &models.Session{}
	err := db.conn.QueryRow(query, token).Scan(&session.ID, &session.UserID, &session.Token, &session.ExpiresAt, &session.CreatedAt, &session.IPAddress, &session.UserAgent)
	if err != nil {
		return nil, err
	}

	// Check if expired
	if time.Now().After(session.ExpiresAt) {
		db.DeleteSession(session.ID)
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func (db *DB) DeleteSession(sessionID string) error {
	query := `DELETE FROM sessions WHERE id = ?`
	_, err := db.conn.Exec(query, sessionID)
	return err
}

func (db *DB) DeleteExpiredSessions() error {
	query := `DELETE FROM sessions WHERE expires_at < ?`
	_, err := db.conn.Exec(query, time.Now())
	return err
}
