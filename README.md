# Vigilon

**Vigilon** - Multi-platform Service Monitoring and Alerting System

Vigilon is a comprehensive monitoring solution that tracks service status across multiple servers (Linux, Windows, Raspberry Pi, etc.) and sends real-time alerts via Telegram when services fail or become degraded.

## Features

- **Multi-Platform Support**: Monitor services on Linux (systemd), Windows, and other platforms
- **Flexible Monitoring Modes**:
  - **Pull Mode**: Central server connects via SSH to check services
  - **Push Mode**: Lightweight agents on servers report status to central server
  - **Hybrid Mode**: Combination of SSH and local scripts
- **Web Dashboard**: Clean, modern UI to view all service statuses
- **Role-Based Access Control**: User management with granular permissions
- **Dynamic Service Configuration**: Add/remove services via UI, agents auto-update
- **One-Line Agent Installer**: Install agent with a single command from UI
- **Telegram Integration**: Receive instant alerts when services fail
- **REST API**: Full API for automation and integration
- **Historical Data**: Track service uptime and performance metrics
- **Alert Management**: Acknowledge and archive alerts
- **Connection Status Tracking**: Monitor agent connectivity in real-time

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
│   ├── api/             # HTTP API handlers
│   ├── auth/            # Authentication & authorization
│   ├── config/          # Configuration management
│   ├── database/        # SQLite database layer
│   ├── models/          # Data models (User, Role, Permission, Server, Service)
│   ├── monitor/         # Monitoring logic
│   └── telegram/        # Telegram bot integration
├── web/
│   ├── templates/       # HTML templates (Dashboard, Servers, Users, Alerts)
│   ├── static/          
│   │   ├── css/         # Stylesheets
│   │   ├── js/          # JavaScript (Servers, Users, Alerts management)
│   │   └── bin/         # Pre-compiled agent binaries (vigilon-agent-*)
├── configs/             # Configuration files
│   ├── config.example.yaml
│   ├── agent-config.example.yaml
│   ├── vigilon-server.service
│   └── vigilon-agent.service
├── Dockerfile           # Multi-stage Docker build
├── docker-compose.yml   # Docker Compose config
├── PROJECT_INFO.md      # Complete technical documentation
└── README.md
```

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
- `GET /api/users` - List all users
- `POST /api/users` - Create a new user
- `GET /api/users/{id}` - Get user details
- `PUT /api/users/{id}` - Update user
- `DELETE /api/users/{id}` - Delete user
- `PUT /api/users/{id}/password` - Change user password
- `GET /api/roles` - List all roles
- `POST /api/roles` - Create a new role
- `GET /api/roles/{id}` - Get role details
- `PUT /api/roles/{id}` - Update role
- `DELETE /api/roles/{id}` - Delete role
- `PUT /api/roles/{id}/permissions` - Update role permissions
- `GET /api/permissions` - List all permissions

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

### Default Roles

1. **Super Administrator** (super_admin)
   - Full system access
   - Cannot be deleted
   - Default credentials: `root` / `toor`

2. **Administrator** (admin)
   - Full access except user deletion
   - System role

3. **User** (user)
   - Read-only access
   - System role

### Permission Categories

- **Servers**: view, create, edit, delete, toggle
- **Services**: view, create, edit, delete, toggle
- **Alerts**: view, acknowledge, archive
- **Users**: view, create, edit, delete
- **Roles**: view, create, edit, delete
- **Settings**: view, edit

Custom roles can be created with specific permission combinations.

## Building for Production

### Server
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o vigilon-server-linux-amd64 cmd/server/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o vigilon-server-windows-amd64.exe cmd/server/main.go
```

### Agent
```bash
# Linux
GOOS=linux GOARCH=amd64 go build -o vigilon-agent-linux-amd64 cmd/agent/main.go

# ARM (Raspberry Pi)
GOOS=linux GOARCH=arm64 go build -o vigilon-agent-linux-arm64 cmd/agent/main.go

# Windows
GOOS=windows GOARCH=amd64 go build -o vigilon-agent-windows-amd64.exe cmd/agent/main.go
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
