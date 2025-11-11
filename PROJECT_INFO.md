# Vigilon - Technical Documentation

## Project Overview

**Vigilon** is a comprehensive multi-platform service monitoring and alerting system designed to monitor services across Linux, Windows, and other platforms. It provides real-time status updates, historical data tracking, and instant Telegram notifications.

## Architecture

### Core Components

1. **Vigilon Server** - Central monitoring hub
   - RESTful API backend
   - Web-based UI
   - Authentication & Authorization system
   - Telegram bot integration
   - Service monitoring engine
   - SQLite database

2. **Vigilon Agent** - Lightweight monitoring agent
   - Runs on target servers
   - Reports service status to server
   - Dynamically fetches service list from API
   - Auto-refreshes monitored services
   - Platform-specific service checkers (systemd, Windows services)

3. **Web UI** - Modern responsive interface
   - Dashboard with real-time status
   - Server management
   - Service configuration
   - Alert management
   - User & role management
   - Agent installation script generator

## Database Schema

### Core Tables

#### `servers`
```sql
- id (PRIMARY KEY)
- name (UNIQUE)
- hostname
- ip_address
- port
- os (linux, windows, etc.)
- monitoring_mode (pull, push, hybrid)
- ssh_user, ssh_key_path (for pull mode)
- ssh_jump_host, ssh_jump_user, ssh_jump_key_path (for SSH tunneling)
- agent_token (for push mode)
- check_interval
- connection_status (not_connected, connected, idle, disconnected)
- enabled (BOOLEAN)
- last_seen (DATETIME)
- notify_telegram (BOOLEAN)
- created_at, updated_at
```

#### `services`
```sql
- id (PRIMARY KEY)
- server_id (FOREIGN KEY -> servers)
- name (service name, e.g., nginx.service)
- display_name
- description
- enabled (BOOLEAN)
- created_at, updated_at
- UNIQUE(server_id, name)
```

#### `service_checks`
```sql
- id (PRIMARY KEY)
- service_id (FOREIGN KEY -> services)
- status (running, stopped, failed, unknown, degraded)
- response_time_ms
- error_message
- checked_at (DATETIME)
- pid
- memory_kb
- cpu_percent
- uptime_seconds
```

#### `alerts`
```sql
- id (PRIMARY KEY)
- service_id (FOREIGN KEY -> services)
- server_id (FOREIGN KEY -> servers)
- status
- message
- sent_via (telegram, email, etc.)
- acknowledged (BOOLEAN)
- archived (BOOLEAN)
- created_at
- acknowledged_at (DATETIME)
- archived_at (DATETIME)
```

### Authentication Tables

#### `users`
```sql
- id (PRIMARY KEY)
- username (UNIQUE)
- email (UNIQUE)
- password_hash (bcrypt)
- role_id (FOREIGN KEY -> roles)
- enabled (BOOLEAN)
- created_at, updated_at
- last_login_at (DATETIME)
```

#### `roles`
```sql
- id (PRIMARY KEY)
- name (UNIQUE)
- display_name
- description
- is_super_admin (BOOLEAN) - Cannot be deleted
- is_system (BOOLEAN) - Cannot be modified
- created_at, updated_at
```

#### `permissions`
```sql
- id (PRIMARY KEY)
- name (UNIQUE, e.g., servers.view, users.create)
- display_name
- description
- category (servers, services, alerts, users, roles, settings)
- created_at
```

#### `role_permissions`
```sql
- role_id (FOREIGN KEY -> roles)
- permission_id (FOREIGN KEY -> permissions)
- PRIMARY KEY (role_id, permission_id)
- created_at
```

#### `sessions`
```sql
- id (PRIMARY KEY)
- user_id (FOREIGN KEY -> users)
- token (UNIQUE)
- expires_at (DATETIME)
- created_at
- ip_address
- user_agent
```

## Permission System

### Permission Categories & Names

**Servers:**
- `servers.view` - View server list and details
- `servers.create` - Add new servers
- `servers.edit` - Modify server settings
- `servers.delete` - Remove servers
- `servers.toggle` - Enable/disable server monitoring

**Services:**
- `services.view` - View service list and details
- `services.create` - Add new services
- `services.edit` - Modify service settings
- `services.delete` - Remove services
- `services.toggle` - Enable/disable service monitoring

**Alerts:**
- `alerts.view` - View alerts
- `alerts.acknowledge` - Acknowledge alerts
- `alerts.archive` - Archive alerts

**Users:**
- `users.view` - View user list
- `users.create` - Add new users
- `users.edit` - Modify user settings
- `users.delete` - Remove users

**Roles:**
- `roles.view` - View role list
- `roles.create` - Add new roles
- `roles.edit` - Modify role settings
- `roles.delete` - Remove roles

**Settings:**
- `settings.view` - View system settings
- `settings.edit` - Modify system settings

### Default Roles

1. **Super Administrator** (super_admin)
   - Has ALL permissions
   - Cannot be deleted
   - Default user: `root` / `toor` (must change immediately!)

2. **Administrator** (admin)
   - All permissions except `users.delete`
   - System role (cannot be deleted)

3. **User** (user)
   - Only view permissions (*.view)
   - Read-only access
   - System role (cannot be deleted)

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login with username/password
- `POST /api/auth/logout` - Logout and invalidate session
- `GET /login` - Login page

### Servers
- `GET /api/servers` - List all servers [Permission: servers.view]
- `POST /api/servers` - Create new server [Permission: servers.create]
- `GET /api/servers/{id}` - Get server details [Permission: servers.view]
- `PUT /api/servers/{id}` - Update server [Permission: servers.edit]
- `DELETE /api/servers/{id}` - Delete server [Permission: servers.delete]
- `POST /api/servers/{id}/disconnect` - Disconnect server [Permission: servers.edit]

### Services
- `GET /api/servers/{id}/services` - List services for server [Permission: services.view]
- `POST /api/services` - Create new service [Permission: services.create]
- `PUT /api/services/{id}` - Update service [Permission: services.edit]
- `DELETE /api/services/{id}` - Delete service [Permission: services.delete]
- `GET /api/services/{id}/status` - Get service status [Permission: services.view]
- `GET /api/services/{id}/checks` - Get check history [Permission: services.view]

### Alerts
- `GET /api/alerts` - List recent alerts [Permission: alerts.view]
- `GET /api/alerts/archived` - List archived alerts [Permission: alerts.view]
- `POST /api/alerts/{id}/acknowledge` - Acknowledge alert [Permission: alerts.acknowledge]
- `POST /api/alerts/{id}/archive` - Archive alert [Permission: alerts.archive]
- `POST /api/alerts/{id}/unarchive` - Unarchive alert [Permission: alerts.archive]
- `POST /api/alerts/archive-all` - Archive all alerts [Permission: alerts.archive]

### Users
- `GET /api/users` - List all users [Permission: users.view]
- `POST /api/users` - Create new user [Permission: users.create]
- `GET /api/users/{id}` - Get user details [Permission: users.view]
- `PUT /api/users/{id}` - Update user [Permission: users.edit]
- `DELETE /api/users/{id}` - Delete user [Permission: users.delete]
- `PUT /api/users/{id}/password` - Change user password [Authenticated user only]

### Roles & Permissions
- `GET /api/roles` - List all roles [Permission: roles.view]
- `POST /api/roles` - Create new role [Permission: roles.edit]
- `GET /api/roles/{id}` - Get role details [Permission: roles.view]
- `PUT /api/roles/{id}` - Update role [Permission: roles.edit]
- `DELETE /api/roles/{id}` - Delete role [Permission: roles.edit]
- `PUT /api/roles/{id}/permissions` - Update role permissions [Permission: roles.edit]
- `GET /api/permissions` - List all permissions [Permission: roles.view]

### Agent Endpoints (No session auth, uses token)
- `POST /api/agent/report` - Agent reports service status
- `GET /api/agent/services?token={token}` - Get service list for agent
- `POST /api/agent/install-script` - Generate install script
- `GET /install.sh?token={token}` - One-line installer script

### Web UI Pages
- `GET /` - Dashboard
- `GET /servers` - Server management
- `GET /server/{id}` - Server detail page
- `GET /alerts` - Active alerts
- `GET /alerts/archived` - Archived alerts
- `GET /users` - User management

## Monitoring Modes

### 1. Pull Mode (SSH-based)
- Server connects to target via SSH
- Executes systemctl commands remotely
- Requires SSH credentials and key
- Supports SSH jump hosts for bastion access

**Features:**
- No agent installation needed
- Centralized control
- SSH tunnel support
- Real-time on-demand checks

**Configuration:**
```yaml
monitoring_mode: pull
ssh_user: admin
ssh_key_path: /path/to/key
ssh_jump_host: bastion.example.com (optional)
ssh_jump_user: jump_user (optional)
```

### 2. Push Mode (Agent-based)
- Lightweight agent runs on target server
- Agent fetches service list from API dynamically
- Reports status periodically
- Auto-refreshes service list

**Features:**
- No SSH required
- Works behind NAT/firewall
- Lower network overhead
- Dynamic service configuration

**Configuration:**
```yaml
monitoring_mode: push
agent_token: secure-random-token
```

**Agent Config:**
```yaml
server_url: http://server:8090
token: secure-random-token
check_interval: 30s
service_refresh_interval: 5m
```

### 3. Hybrid Mode
- Combines SSH access with local scripts
- Flexible monitoring options

## Agent Installation Flow

### UI-Based Installation

1. **Add Server in UI** (Push Mode)
   - User creates server with monitoring_mode=push
   - System auto-generates secure token
   - Token is associated with server

2. **Generate Install Script**
   - UI displays installation command
   - One-line installer: `curl -fsSL http://server:8090/install.sh?token=TOKEN | sudo bash`
   - Or download platform-specific script (Linux, Windows)

3. **Script Execution**
   - Downloads agent binary from `/static/bin/`
   - Creates config file with server URL and token
   - Installs as systemd service (Linux) or Windows service
   - Starts agent automatically

4. **Agent Operation**
   - Agent starts and fetches service list from `/api/agent/services?token=TOKEN`
   - API returns all enabled services for that server
   - Agent checks services periodically
   - Reports status to `/api/agent/report`
   - Refreshes service list every 5 minutes (configurable)

### Manual Installation

Users can also install agent manually:
1. Download binary from releases or `/static/bin/`
2. Create config file with server URL and token
3. Install as service
4. Start service

## Service Status Flow

### For Push Mode (Agent)

```
Agent -> Fetch Services (/api/agent/services?token=X)
  <- Returns: {server_id: 1, services: [{name: "nginx.service", enabled: true}, ...]}

Agent -> Check each service locally (systemctl/PowerShell)
  -> Collect: status, PID, memory, CPU, uptime

Agent -> Report to Server (/api/agent/report)
  -> Send: {token: X, services: [{name, status, pid, memory, cpu, uptime}, ...]}

Server -> Update service_checks table
Server -> Check for status changes
Server -> Send Telegram alert if service failed/stopped
```

### For Pull Mode (SSH)

```
Monitor -> Select enabled server (pull mode)
Monitor -> SSH connect to server
Monitor -> Execute: systemctl is-active service.name
Monitor -> Parse output
Monitor -> Store in service_checks table
Monitor -> Check for status changes
Monitor -> Send Telegram alert if needed
```

## Telegram Integration

### Bot Commands
- `/start` - Welcome message
- `/status` - Current status of all services
- `/servers` - List all monitored servers
- `/alerts` - View recent alerts
- `/help` - Command help

### Alert Messages
- Sent when service status changes to failed/stopped
- Cooldown period to prevent spam (default: 5 minutes)
- Server-level notification toggle
- Includes server name, service name, status, timestamp

### Configuration
```yaml
telegram:
  enabled: true
  bot_token: "123456789:ABCdefGHI..."
  chat_ids:
    - "123456789"
    - "987654321"
```

## Authentication & Session Management

### Login Flow
1. User submits username/password to `/api/auth/login`
2. Server validates credentials (bcrypt)
3. Server creates session with expiry
4. Server generates secure session token
5. Token stored in cookie (`session_token`)
6. User redirected to dashboard

### Middleware
- `RequireAuth` - Checks session cookie, validates token
- `RequirePermission` - Checks if user has specific permission
- Super admins bypass all permission checks

### Session Storage
- Sessions stored in database
- Contains user ID, token, expiry, IP, user agent
- Tokens are cryptographically secure random strings
- Sessions expire after configured period

## Configuration Files

### Server Config (`configs/config.yaml`)
```yaml
server:
  host: 0.0.0.0
  port: 8090

database:
  path: ./vigilon.db

telegram:
  enabled: true
  bot_token: "YOUR_BOT_TOKEN"
  chat_ids:
    - "YOUR_CHAT_ID"

monitoring:
  check_interval: 30s
  retention_days: 30
  alert_cooldown: 5m

servers:
  - name: example-server
    hostname: example.com
    ip_address: 192.168.1.100
    port: 22
    os: linux
    monitoring_mode: pull
    ssh_user: admin
    ssh_key_path: /path/to/key
    enabled: true
    notify_telegram: true
    services:
      - name: nginx.service
        display_name: Nginx
        description: Web server
        enabled: true
```

### Agent Config (`/etc/vigilon-agent/config.yaml`)
```yaml
server_url: http://server:8090
token: secure-token-from-server
check_interval: 30s
service_refresh_interval: 5m
services: []  # Fallback if API fetch fails
```

## Technology Stack

### Backend
- **Language:** Go 1.21+
- **Web Framework:** gorilla/mux (HTTP routing)
- **Database:** SQLite3 with WAL mode
- **Authentication:** bcrypt for password hashing
- **Session:** Cookie-based with database storage

### Frontend
- **HTML Templates:** Go templates
- **CSS:** Custom responsive design
- **JavaScript:** Vanilla JS (no frameworks)
- **AJAX:** Fetch API for dynamic updates

### External Dependencies
- `github.com/mattn/go-sqlite3` - SQLite driver
- `github.com/gorilla/mux` - HTTP router
- `golang.org/x/crypto/bcrypt` - Password hashing
- `gopkg.in/yaml.v3` - YAML parsing
- Telegram Bot API

## File Structure

```
vigilon/
├── cmd/
│   ├── server/main.go          # Server entry point
│   └── agent/main.go           # Agent entry point
├── internal/
│   ├── api/api.go              # HTTP handlers & routing (1493 lines)
│   ├── auth/
│   │   ├── auth.go             # Password & token utilities
│   │   └── middleware.go       # Auth middleware (191 lines)
│   ├── config/config.go        # Configuration management
│   ├── database/database.go    # Database layer (1093 lines)
│   ├── models/models.go        # Data models (177 lines)
│   ├── monitor/
│   │   ├── monitor.go          # Monitoring engine
│   │   └── ssh_checker.go      # SSH-based checker
│   └── telegram/telegram.go    # Telegram bot integration
├── web/
│   ├── templates/
│   │   ├── index.html          # Dashboard
│   │   ├── servers.html        # Server management
│   │   ├── server_detail.html  # Server details
│   │   ├── alerts.html         # Active alerts
│   │   ├── archived_alerts.html
│   │   ├── users.html          # User management
│   │   └── login.html          # Login page
│   └── static/
│       ├── css/style.css
│       ├── js/
│       │   ├── main.js
│       │   ├── servers.js      # Server management JS
│       │   ├── users.js        # User/role management JS (632 lines)
│       │   ├── alerts.js
│       │   └── server_detail.js
│       └── bin/                # Agent binaries
│           ├── vigilon-agent-linux-amd64
│           ├── vigilon-agent-linux-arm64
│           └── vigilon-agent-windows-amd64.exe
├── configs/
│   ├── config.yaml
│   ├── config.example.yaml
│   ├── agent-config.example.yaml
│   ├── vigilon-server.service
│   └── vigilon-agent.service
├── Dockerfile                  # Multi-stage Docker build
├── docker-compose.yml
├── .dockerignore
├── Makefile
├── go.mod
├── go.sum
├── README.md
├── INSTALL.md
└── PROJECT_INFO.md (this file)
```

## Key Features

### Dynamic Service Management
- Services can be added/removed via UI
- Agent automatically fetches updated service list
- No need to restart agent when services change
- Fallback to config file if API unavailable

### Connection Status Tracking
- `not_connected` - Never connected
- `connected` - Currently active
- `idle` - Was connected, no recent activity
- `disconnected` - Manually disconnected

### Alert Management
- Acknowledge alerts to mark as seen
- Archive old/resolved alerts
- Archive all alerts at once
- View archived alerts separately
- Alert cooldown to prevent spam

### Role-Based Access Control (RBAC)
- Granular permissions per resource
- Custom roles with specific permissions
- System roles cannot be modified
- Super admin cannot be deleted

### One-Line Installer
- Generate installation script from UI
- Auto-detects OS and architecture
- Downloads correct binary
- Creates config with token
- Installs as system service
- No manual configuration needed

### Health Monitoring
- Service status (running/stopped/failed/degraded)
- Process metrics (PID, memory, CPU)
- Service uptime
- Response time tracking
- Historical data retention

### Multi-Platform Support
- Linux (systemd services)
- Windows (Windows services)
- Raspberry Pi (ARM support)
- Docker/Podman containers

## Security Considerations

### Authentication
- Bcrypt password hashing (cost 10)
- Secure session tokens (32-byte random)
- Session expiry
- IP and user agent tracking

### Agent Communication
- Token-based authentication
- Tokens are unique per server
- Tokens should be kept secure
- HTTPS recommended for production

### Default Credentials
- **CRITICAL:** Default super admin is `root`/`toor`
- **MUST** change password immediately after first login
- Create additional admin users
- Disable root user if not needed

### Permissions
- Least privilege principle
- Separate read and write permissions
- Super admin required for sensitive operations
- Cannot delete system roles

## Performance Optimizations

### Database
- SQLite WAL mode for better concurrency
- 64MB cache size
- Busy timeout 5 seconds
- Foreign key constraints enabled
- Indexes on frequently queried columns

### Monitoring
- Configurable check intervals
- Connection pooling for SSH
- Batch service checks
- Alert cooldown to reduce spam
- Data retention policy

### Agent
- Lightweight resource usage (~50MB RAM)
- Efficient service checking
- Periodic service list refresh (not every check)
- Minimal network traffic

## Future Enhancements (Potential)

- Email notifications
- Webhook support
- Prometheus metrics export
- Grafana integration
- Mobile app
- Multi-language support
- Advanced alerting rules
- Service dependency management
- Custom health checks
- Performance graphs
- Log aggregation

## Development & Building

### Build Commands
```bash
# Build server
go build -o vigilon-server cmd/server/main.go

# Build agent
go build -o vigilon-agent cmd/agent/main.go

# Build all
make build

# Cross-compile
GOOS=linux GOARCH=amd64 go build -o vigilon-agent-linux-amd64 cmd/agent/main.go
GOOS=linux GOARCH=arm64 go build -o vigilon-agent-linux-arm64 cmd/agent/main.go
GOOS=windows GOARCH=amd64 go build -o vigilon-agent-windows-amd64.exe cmd/agent/main.go
```

### Docker Build
```bash
# Build server image
docker build --target server -t vigilon-server:latest .

# Build agent image
docker build --target agent -t vigilon-agent:latest .

# Run with docker-compose
docker-compose up -d
```

## Contact & Support

**Developer:** Harun Geçit
**Email:** info@harungecit.com
**Website:** https://harungecit.com
**GitHub:** https://github.com/harungecit
**Twitter/X:** @harungecit_
**Instagram:** @harungecit.dev

**Project Repository:** https://github.com/harungecit/vigilon

## License

This project is licensed under the MIT License.

---

**Last Updated:** 2025-11-11
**Version:** 1.0.0
