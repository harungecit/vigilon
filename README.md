# Vigilon

**Vigilon** - Multi-platform Service Monitoring and Alerting System

Vigilon is a comprehensive monitoring solution that tracks service status across multiple servers (Linux, Windows, Raspberry Pi, etc.) and sends real-time alerts via Telegram when services fail or become degraded.

## Features

### Core Monitoring
- **Multi-Platform Support**: Monitor services on Linux (systemd), Windows, and other platforms
- **Flexible Monitoring Modes**:
  - **Pull Mode**: Central server connects via SSH to check services
  - **Push Mode**: Lightweight agents on servers report status to central server
  - **Hybrid Mode**: Combination of SSH and local scripts
- **Real-Time Updates**: Server-Sent Events (SSE) for live dashboard updates without page refresh
- **Dynamic Service Configuration**: Add/remove services via UI, agents auto-update
- **One-Line Agent Installer**: Install agent with a single command from UI
- **Historical Data**: Track service uptime and performance metrics with detailed check history
- **Connection Status Tracking**: Monitor agent connectivity in real-time

### User Interface
- **Modern Web Dashboard**: Clean, responsive UI with real-time updates
- **Server Management**: View server details, service history, and health status
- **Alert Dashboard**: Visual indicators for active, acknowledged, and archived alerts
- **Service Detail Views**: Interactive modals with service check history
- **Dark Theme**: Professional dark color scheme for reduced eye strain

### Security & Access Control
- **Advanced RBAC**: Hierarchical role-based access control system
  - **Super Admin**: Full system access, can manage all roles except Super Admin itself
  - **Admin**: Can manage User and Custom roles, full operational access
  - **User**: Read-only access with configurable permissions
  - **Custom Roles**: Create roles with specific permission combinations
- **Session Management**: Secure cookie-based authentication with session timeout
- **Permission System**: Granular permissions for servers, services, alerts, users, roles, and settings
- **Password Security**: Minimum 4-character passwords, hashed with bcrypt
- **Admin Password Reset**: Admins can reset user passwords without knowing current password

### Integrations & Alerts
- **Telegram Integration**: Receive instant alerts when services fail
- **Alert Management**: Acknowledge, archive, and track alert history
- **Alert Cooldown**: Prevent notification spam with configurable cooldown periods
- **REST API**: Full API for automation and integration with token-based authentication

## Recent Updates

### Version 1.1.0 (November 2025)

#### Real-Time Updates
- ✅ **Server-Sent Events (SSE)**: Live updates for dashboard, servers, and service history
- ✅ **Auto-refresh**: Dashboard, server list, and service details update every 5 seconds
- ✅ **Modal Updates**: Service history modals update in real-time without closing

#### Security & Access Control
- ✅ **Enhanced RBAC**: Hierarchical role management with proper permission checks
- ✅ **Role Hierarchy**: Super Admin → Admin → User/Custom roles
- ✅ **Permission-Based UI**: Buttons and actions dynamically shown based on permissions
- ✅ **Admin Password Reset**: Admins can reset user passwords without current password
- ✅ **Session-Based Auth**: Secure cookie authentication with proper session management

#### API Improvements
- ✅ **Route Optimization**: Fixed `/api/users/me` route collision with `/api/users/{id}`
- ✅ **Permission Middleware**: All endpoints properly protected with permission checks
- ✅ **Current User API**: New `/api/users/me` endpoint for frontend user context
- ✅ **Password API**: Support both PUT and POST methods for password changes

#### Database
- ✅ **WAL Mode**: SQLite WAL (Write-Ahead Logging) enabled for better concurrency
- ✅ **CGO Build**: Compiled with CGO_ENABLED=1 for native SQLite performance
- ✅ **Auto-migrations**: Database schema updates on startup

#### Bug Fixes
- ✅ Fixed role management visibility for Super Admin
- ✅ Fixed permission checks in role editing
- ✅ Fixed duplicate modals and script tags in user management
- ✅ Fixed password change 405 errors
- ✅ Removed all debug console logs from production

## Quick Start

### Prerequisites

- Go 1.21 or higher (for building from source)
- OR Docker/Podman (for containerized deployment)
- SSH access to target servers (for pull/hybrid modes - optional)

### Default Credentials

⚠️ **IMPORTANT SECURITY NOTE:**
- Default username: `root`
- Default password: `toor`
- **CHANGE THIS IMMEDIATELY** after first login!

### Installation

#### Option 1: Docker/Podman (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/harungecit/vigilon.git
cd vigilon
```

2. Configure:
```bash
cp configs/config.example.yaml configs/config.yaml
nano configs/config.yaml
```

3. Start with Docker Compose:
```bash
# Using Docker
docker-compose up -d

# Using Podman
podman-compose up -d
```

Access the web interface at `http://localhost:8090`

#### Option 2: Build from Source

1. Clone the repository:
```bash
git clone https://github.com/harungecit/vigilon.git
cd vigilon
```

2. Install dependencies:
```bash
go mod download
```

3. Build the server:
```bash
go build -o vigilon-server cmd/server/main.go
```

4. Build the agent (optional, for push mode):
```bash
go build -o vigilon-agent cmd/agent/main.go
```

### Configuration

1. Copy the example config:
```bash
cp configs/config.example.yaml configs/config.yaml
```

2. Edit `configs/config.yaml` with your settings:
```yaml
server:
  host: 0.0.0.0
  port: 8090

telegram:
  enabled: true
  bot_token: "YOUR_BOT_TOKEN"
  chat_ids:
    - "YOUR_CHAT_ID"

monitoring:
  check_interval: 30s
  retention_days: 30
  alert_cooldown: 5m
```

3. Add your servers and services to the config file or use the Web UI.

### Running the Server

```bash
./vigilon-server -config configs/config.yaml
```

The server will start on `http://localhost:8090`

## Docker/Podman Deployment

### Using Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Rebuild after changes
docker-compose up -d --build
```

### Using Podman

Podman is a Docker-compatible alternative that doesn't require root:

```bash
# Build server image
podman build --target server -t vigilon-server:latest .

# Build agent image
podman build --target agent -t vigilon-agent:latest .

# Run server
podman run -d \
  --name vigilon-server \
  -p 8090:8090 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  -v vigilon-data:/app/data \
  vigilon-server:latest

# Run agent
podman run -d \
  --name vigilon-agent \
  -v $(pwd)/configs/agent-config.yaml:/app/config.yaml:ro \
  vigilon-agent:latest
```

### Pre-built Images

```bash
# Pull from Docker Hub
docker pull harungecit/vigilon-server:latest
docker pull harungecit/vigilon-agent:latest

# Or with Podman
podman pull harungecit/vigilon-server:latest
podman pull harungecit/vigilon-agent:latest
```

### Running the Agent (Push Mode)

On each target server:

1. Copy the agent binary and config:
```bash
sudo cp vigilon-agent /usr/local/bin/
sudo mkdir -p /etc/vigilon-agent
sudo cp configs/agent-config.example.yaml /etc/vigilon-agent/config.yaml
```

2. Edit `/etc/vigilon-agent/config.yaml`:
```yaml
server_url: http://your-server-ip:8080
token: your-secure-token-here
check_interval: 30s

services:
  - rftt.service
  - nginx.service
```

3. Create a systemd service (Linux):
```bash
sudo tee /etc/systemd/system/vigilon-agent.service > /dev/null <<EOF
[Unit]
Description=Vigilon Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/vigilon-agent -config /etc/vigilon-agent/config.yaml
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl enable vigilon-agent
sudo systemctl start vigilon-agent
```

## Monitoring Modes

### Pull Mode (SSH)
The central server connects to remote servers via SSH and checks service status.

**Pros:**
- No agent installation needed
- Centralized control
- SSH jump host support

**Cons:**
- Requires SSH access and credentials
- Higher network overhead

**Configuration:**
```yaml
servers:
  - name: my-server
    ip_address: 192.168.1.100
    monitoring_mode: pull
    ssh_user: admin
    ssh_key_path: /path/to/ssh/key
```

### Push Mode (Agent)
Lightweight agents run on each server and report status to the central server.

**Pros:**
- No SSH required
- Lower network overhead
- Works behind firewalls/NAT
- **Dynamic service list** - Agent fetches services from UI
- **One-line installation** from web UI

**Cons:**
- Requires agent installation
- Need to manage agent tokens

**Installation Flow:**
1. Add server in UI with Push mode
2. System generates secure token automatically
3. Copy one-line install command from UI
4. Run on target server: `curl -fsSL http://server:8090/install.sh?token=TOKEN | sudo bash`
5. Agent auto-starts and fetches service list from UI
6. Add services in UI, agent updates automatically

**Configuration:**
```yaml
servers:
  - name: my-server
    ip_address: 192.168.1.100
    monitoring_mode: push
    agent_token: auto-generated-secure-token
```

### Hybrid Mode
Combines SSH access with local scripts for optimal flexibility.

## Project Structure

```
vigilon/
├── cmd/
│   ├── server/          # Main server application
│   └── agent/           # Agent for push mode
├── internal/
│   ├── api/             # HTTP API handlers with SSE support
│   ├── auth/            # Authentication & authorization middleware
│   ├── config/          # Configuration management
│   ├── database/        # SQLite database layer (WAL mode enabled)
│   ├── models/          # Data models (User, Role, Permission, Server, Service, Alert)
│   ├── monitor/         # Monitoring logic (SSH checker, status tracker)
│   ├── telegram/        # Telegram bot integration
│   └── sse/             # Server-Sent Events manager
├── web/
│   ├── templates/       # HTML templates (Dashboard, Servers, Users, Alerts)
│   ├── static/          
│   │   ├── css/         # Stylesheets (dark theme)
│   │   ├── js/          
│   │   │   ├── main.js        # Dashboard & auth
│   │   │   ├── servers.js     # Server management
│   │   │   ├── users.js       # User & role management
│   │   │   ├── alerts.js      # Alert management
│   │   │   ├── sse.js         # SSE client library
│   │   │   └── server_detail.js  # Server detail view
│   │   └── bin/         # Pre-compiled agent binaries (vigilon-agent-*)
├── configs/             # Configuration files
│   ├── config.example.yaml
│   ├── agent-config.example.yaml
│   ├── vigilon-server.service
│   └── vigilon-agent.service
├── vigilon.db          # SQLite database (auto-created, WAL mode)
├── Dockerfile          # Multi-stage Docker build
├── docker-compose.yml  # Docker Compose config
├── Makefile            # Build automation
├── go.mod              # Go module dependencies
├── PROJECT_INFO.md     # Complete technical documentation
└── README.md           # This file
```

## Database

Vigilon uses **SQLite** with **WAL (Write-Ahead Logging)** mode for optimal performance:

- **WAL Mode**: Enabled by default for better concurrency and crash recovery
- **CGO Build**: Compiled with `CGO_ENABLED=1` for native SQLite performance
- **Auto-migration**: Schema updates automatically on server startup
- **Location**: `vigilon.db` in the application directory
- **Backup**: Copy `vigilon.db` file (WAL checkpoints automatically merge changes)

### Database Schema

- **users**: User accounts with bcrypt password hashes
- **roles**: Role definitions with system flags
- **permissions**: Granular permission definitions
- **role_permissions**: Many-to-many role-permission mapping
- **servers**: Monitored server configurations
- **services**: Service definitions per server
- **service_checks**: Historical service check results
- **alerts**: Alert records with status tracking
- **sessions**: User session management

## Deployment Options

### Development
```bash
go run cmd/server/main.go -config configs/config.yaml
```

### Production - Binary
```bash
make build
./vigilon-server -config configs/config.yaml
```

### Production - Docker
```bash
docker-compose up -d
```

### Production - Podman
```bash
podman-compose up -d
```

### Production - Systemd
```bash
sudo systemctl enable vigilon-server
sudo systemctl start vigilon-server
```

## API Endpoints

### Authentication
- `POST /api/auth/login` - Login
- `POST /api/auth/logout` - Logout
- `GET /login` - Login page

### Servers
- `GET /api/servers` - List all servers
- `POST /api/servers` - Create a new server
- `GET /api/servers/{id}` - Get server details
- `PUT /api/servers/{id}` - Update server
- `DELETE /api/servers/{id}` - Delete server
- `POST /api/servers/{id}/disconnect` - Disconnect server

### Services
- `GET /api/servers/{id}/services` - List services for a server
- `POST /api/services` - Create a new service
- `PUT /api/services/{id}` - Update service
- `DELETE /api/services/{id}` - Delete service
- `GET /api/services/{id}/status` - Get current service status
- `GET /api/services/{id}/checks` - Get service check history

### Alerts
- `GET /api/alerts` - List recent alerts
- `GET /api/alerts/archived` - List archived alerts
- `POST /api/alerts/{id}/acknowledge` - Acknowledge an alert
- `POST /api/alerts/{id}/archive` - Archive an alert
- `POST /api/alerts/{id}/unarchive` - Unarchive an alert
- `POST /api/alerts/archive-all` - Archive all alerts

### Users & Roles
- `GET /api/users` - List all users (requires `users.view`)
- `POST /api/users` - Create a new user (requires `users.create`)
- `GET /api/users/me` - Get current user details (authenticated)
- `GET /api/users/{id}` - Get user details (requires `users.view`)
- `PUT /api/users/{id}` - Update user (requires `users.edit`)
- `DELETE /api/users/{id}` - Delete user (requires `users.delete`)
- `PUT /api/users/{id}/password` - Change user password
- `POST /api/users/{id}/password` - Change user password (alternative method)
- `GET /api/roles` - List all roles (requires `roles.view`, Admin/Super Admin only)
- `POST /api/roles` - Create a new role (requires `roles.create`)
- `GET /api/roles/{id}` - Get role details (requires `roles.view`)
- `PUT /api/roles/{id}` - Update role (requires `roles.edit`)
- `DELETE /api/roles/{id}` - Delete role (requires `roles.delete`)
- `PUT /api/roles/{id}/permissions` - Update role permissions (requires `roles.edit`)
- `GET /api/permissions` - List all permissions (requires permissions access)

### Real-Time Updates (SSE)
- `GET /api/sse/dashboard` - Dashboard real-time updates (5-second interval)
- `GET /api/sse/servers` - Server list real-time updates
- `GET /api/sse/server/{id}` - Single server detail updates
- `GET /api/sse/service/{id}/history` - Service history updates

### Agent (Token-based authentication)
- `POST /api/agent/report` - Agent endpoint to push status updates
- `GET /api/agent/services?token={token}` - Get service list for agent
- `POST /api/agent/install-script` - Generate installation script
- `GET /install.sh?token={token}` - One-line installer script

## Telegram Bot Commands

- `/start` - Welcome message and bot information
- `/status` - Get current status of all services
- `/servers` - List all monitored servers
- `/alerts` - View recent alerts
- `/help` - Show help message and available commands

## User Roles & Permissions

### Role Hierarchy

Vigilon implements a hierarchical role-based access control (RBAC) system:

1. **Super Administrator** (super_admin)
   - **Full system access** - Root-level privileges
   - Can manage all users and roles
   - **Can edit all roles EXCEPT Super Admin itself** (locked for security)
   - Can create, edit, and delete custom roles
   - Cannot be deleted (system role)
   - Default credentials: `root` / `toor` ⚠️ **CHANGE IMMEDIATELY**

2. **Administrator** (admin)
   - **Can manage User and Custom roles only**
   - Cannot edit Super Admin or Admin roles
   - Full operational access (servers, services, alerts)
   - Can create and delete custom roles
   - Can reset user passwords without current password
   - Cannot be deleted (system role)

3. **User** (user)
   - Read-only access by default
   - Cannot view or manage roles
   - Cannot access role management interface
   - Configurable permissions for operational tasks
   - System role

4. **Custom Roles**
   - Create roles with specific permission combinations
   - Can be edited by Super Admin and Admin
   - Can be deleted by Super Admin and Admin
   - Inherit from permission system

### Permission Categories

#### Servers
- `servers.view` - View server list and details
- `servers.create` - Add new servers
- `servers.edit` - Modify server configuration
- `servers.delete` - Remove servers
- `servers.toggle` - Enable/disable server monitoring

#### Services
- `services.view` - View service status and history
- `services.create` - Add services to servers
- `services.edit` - Modify service configuration
- `services.delete` - Remove services
- `services.toggle` - Enable/disable service monitoring

#### Alerts
- `alerts.view` - View alert dashboard and history
- `alerts.acknowledge` - Acknowledge active alerts
- `alerts.archive` - Archive alerts
- `alerts.unarchive` - Restore archived alerts

#### Users
- `users.view` - View user list and details
- `users.create` - Create new users
- `users.edit` - Modify user details and roles
- `users.delete` - Delete users
- User can always change their own password

#### Roles & Permissions
- `roles.view` - View role list (Admin and Super Admin only)
- `roles.create` - Create custom roles (Admin and Super Admin only)
- `roles.edit` - Modify role permissions (Admin and Super Admin only)
- `roles.delete` - Delete custom roles (Admin and Super Admin only)

#### Settings
- `settings.view` - View system settings
- `settings.edit` - Modify system configuration

### Password Management

- **Minimum Length**: 4 characters (configurable)
- **Hashing**: bcrypt with cost factor 10
- **Self-Service**: All users can change their own password
- **Admin Reset**: Super Admin and Admin can reset user passwords without knowing current password
- **Security**: Current password required for self-service changes

### UI Permission Controls

The interface dynamically adjusts based on user permissions:
- **Manage Roles** button: Only visible to Super Admin and Admin
- **Add User** button: Only visible to users with `users.create` permission
- **Edit/Delete** buttons: Shown based on role hierarchy and permissions
- **Role Permission Edit**: Only available for roles the current user can manage

## Building for Production

### Server

⚠️ **Important**: Server requires **CGO_ENABLED=1** for SQLite support.

```bash
# Linux (requires gcc)
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o vigilon-server-linux-amd64 cmd/server/main.go

# Cross-compile for Linux ARM64 (requires cross-compiler)
CGO_ENABLED=1 GOOS=linux GOARCH=arm64 CC=aarch64-linux-gnu-gcc go build -o vigilon-server-linux-arm64 cmd/server/main.go

# macOS (native build)
CGO_ENABLED=1 go build -o vigilon-server-darwin-amd64 cmd/server/main.go

# Windows (requires mingw-w64)
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -o vigilon-server-windows-amd64.exe cmd/server/main.go
```

### Agent

Agent does not require CGO:

```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o vigilon-agent-linux-amd64 cmd/agent/main.go

# ARM (Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o vigilon-agent-linux-arm64 cmd/agent/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o vigilon-agent-windows-amd64.exe cmd/agent/main.go

# macOS
GOOS=darwin GOARCH=amd64 go build -o vigilon-agent-darwin-amd64 cmd/agent/main.go
```

### Using Makefile

```bash
# Build server (CGO enabled automatically)
make build-server

# Build agent
make build-agent

# Build both
make build

# Clean binaries
make clean
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Technical Documentation

For complete technical documentation including database schema, authentication flow, and API details, see [PROJECT_INFO.md](PROJECT_INFO.md).

## License

This project is licensed under the MIT License.

## Author

**Harun Geçit**

- Website: [harungecit.com](https://harungecit.com)
- Email: [info@harungecit.com](mailto:info@harungecit.com)
- GitHub: [@harungecit](https://github.com/harungecit)
- Twitter/X: [@harungecit_](https://twitter.com/harungecit_)
- Instagram: [@harungecit.dev](https://instagram.com/harungecit.dev)

## Support

For issues and questions:
- Open an issue: [GitHub Issues](https://github.com/harungecit/vigilon/issues)
- Email: info@harungecit.com
