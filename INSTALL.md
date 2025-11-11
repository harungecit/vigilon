# Vigilon Installation Guide

This guide will help you install and configure Vigilon on your servers.

## Table of Contents

1. [System Requirements](#system-requirements)
2. [Server Installation](#server-installation)
   - [Option 1: Build from Source](#option-1-build-from-source)
   - [Option 2: Pre-compiled Binary](#option-2-pre-compiled-binary)
   - [Option 3: Docker/Podman](#option-3-dockerpodman)
3. [Agent Installation](#agent-installation)
4. [Telegram Bot Setup](#telegram-bot-setup)
5. [Configuration](#configuration)
6. [Troubleshooting](#troubleshooting)

## System Requirements

### Server Requirements
- Go 1.21+ (for building from source)
- 512MB RAM minimum
- 100MB disk space
- Linux, Windows, or macOS

### Agent Requirements
- Go 1.21+ (for building from source) or pre-compiled binary
- 50MB RAM
- 20MB disk space
- SSH access (for pull mode)

## Server Installation

### Option 1: Build from Source

1. Install Go (if not already installed):
```bash
# Ubuntu/Debian
sudo apt update
sudo apt install golang-go

# CentOS/RHEL
sudo yum install golang
```

2. Clone and build:
```bash
git clone https://github.com/harungecit/vigilon.git
cd vigilon
make build
```

3. Create configuration:
```bash
cp configs/config.example.yaml configs/config.yaml
nano configs/config.yaml
```

4. Create user and directories:
```bash
sudo useradd -r -s /bin/false vigilon
sudo mkdir -p /opt/vigilon
sudo cp vigilon-server /opt/vigilon/
sudo cp -r configs /opt/vigilon/
sudo cp -r web /opt/vigilon/
sudo chown -R vigilon:vigilon /opt/vigilon
```

5. Install systemd service:
```bash
sudo cp configs/vigilon-server.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable vigilon-server
sudo systemctl start vigilon-server
```

6. Check status:
```bash
sudo systemctl status vigilon-server
```

### Option 2: Pre-compiled Binary

1. Download the latest release:
```bash
wget https://github.com/harungecit/vigilon/releases/download/v1.0.0/vigilon-server-linux-amd64
chmod +x vigilon-server-linux-amd64
sudo mv vigilon-server-linux-amd64 /opt/vigilon/vigilon-server
```

2. Follow steps 3-6 from Option 1.

### Option 3: Docker/Podman

You can run Vigilon using either Docker or Podman (compatible alternative).

#### Using Docker Compose (Recommended)

1. Clone the repository:
```bash
git clone https://github.com/harungecit/vigilon.git
cd vigilon
```

2. Create your configuration:
```bash
cp configs/config.example.yaml configs/config.yaml
nano configs/config.yaml
```

3. Start the server:
```bash
# Using Docker Compose
docker-compose up -d

# Using Podman Compose
podman-compose up -d
```

4. Check status:
```bash
# Docker
docker-compose ps
docker-compose logs -f vigilon-server

# Podman
podman-compose ps
podman-compose logs -f vigilon-server
```

#### Using Docker/Podman Directly

1. Build the image:
```bash
# Docker
docker build --target server -t vigilon-server:latest .

# Podman
podman build --target server -t vigilon-server:latest .
```

2. Run the container:
```bash
# Docker
docker run -d \
  --name vigilon-server \
  -p 8090:8090 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  -v vigilon-data:/app/data \
  --restart unless-stopped \
  vigilon-server:latest

# Podman
podman run -d \
  --name vigilon-server \
  -p 8090:8090 \
  -v $(pwd)/configs/config.yaml:/app/configs/config.yaml:ro \
  -v vigilon-data:/app/data \
  --restart unless-stopped \
  vigilon-server:latest
```

3. View logs:
```bash
# Docker
docker logs -f vigilon-server

# Podman
podman logs -f vigilon-server
```

#### Docker Hub Images

Pre-built images are available on Docker Hub:
```bash
# Pull the server image
docker pull harungecit/vigilon-server:latest
# or
podman pull harungecit/vigilon-server:latest

# Run directly
docker run -d -p 8090:8090 --name vigilon-server harungecit/vigilon-server:latest
```

### Accessing the Web Interface

Once installed, access the web interface at:
```
http://your-server-ip:8090
```

**Default Login Credentials:**
- Username: `root`
- Password: `toor`

⚠️ **CRITICAL**: Change the default password immediately after first login for security!

**First Steps:**
1. Login with default credentials
2. Go to Users page
3. Change root password or create a new admin user
4. Create additional users with appropriate roles
5. Configure Telegram bot (optional)
6. Add your servers

## Agent Installation

### Recommended: UI-Based Installation (Push Mode)

The easiest way to install an agent is through the web interface:

1. **Login to Vigilon UI** at `http://your-server:8090`

2. **Add a New Server:**
   - Go to "Servers" page
   - Click "Add Server"
   - Fill in server details:
     - Name: `my-server`
     - IP Address: `192.168.1.100`
     - OS: `linux` or `windows`
     - Monitoring Mode: Select `push`
     - Port: `22` (not used for push mode)
     - Enable: ✓
     - Notify Telegram: ✓

3. **Generate Token:**
   - Click "Generate Token" button (auto-generates secure token)
   - Or enter your own secure token

4. **Save Server:**
   - Click "Add Server"
   - System will create the server and display installation instructions

5. **Add Services:**
   - After creating server, go to server details page
   - Click "Add Service"
   - Add services to monitor (e.g., `nginx.service`, `postgresql.service`)
   - Services are immediately available to the agent

6. **Install Agent:**
   - Copy the one-line installation command shown in the UI
   - SSH to your target server
   - Run the command:
   ```bash
   curl -fsSL http://your-server:8090/install.sh?token=YOUR_TOKEN | sudo bash
   ```

7. **Verify Installation:**
   - Agent will start automatically
   - Check status: `sudo systemctl status vigilon-agent`
   - View logs: `sudo journalctl -u vigilon-agent -f`
   - In UI, server status will change to "connected"

**That's it!** The agent will:
- Automatically fetch the service list from the UI
- Monitor all enabled services
- Report status every 30 seconds
- Refresh service list every 5 minutes
- No need to restart agent when you add/remove services

### Alternative: Manual Installation

#### Linux Systems (systemd)

1. Build or download agent binary:
```bash
# Build from source
make agent

# Or download pre-compiled
wget https://github.com/harungecit/vigilon/releases/download/v1.0.0/vigilon-agent-linux-amd64
chmod +x vigilon-agent-linux-amd64
```

2. Install agent:
```bash
sudo cp vigilon-agent /usr/local/bin/
sudo mkdir -p /etc/vigilon-agent
```

3. Create configuration:
```bash
sudo tee /etc/vigilon-agent/config.yaml > /dev/null <<EOF
server_url: http://YOUR_SERVER_IP:8090
token: YOUR_SECURE_TOKEN
check_interval: 30s
service_refresh_interval: 5m
services: []  # Optional fallback, agent fetches from UI
EOF
```

**Note:** The `services` field is optional. The agent will automatically fetch the service list from the UI. You can use it as a fallback if the API is unreachable.

4. Install systemd service:
```bash
sudo cp configs/vigilon-agent.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable vigilon-agent
sudo systemctl start vigilon-agent
```

5. Verify agent is running:
```bash
sudo systemctl status vigilon-agent
sudo journalctl -u vigilon-agent -f
```

### Raspberry Pi (ARM)

Use ARM binary:
```bash
wget https://github.com/harungecit/vigilon/releases/download/v1.0.0/vigilon-agent-linux-arm64
chmod +x vigilon-agent-linux-arm64
sudo mv vigilon-agent-linux-arm64 /usr/local/bin/vigilon-agent
```

Then follow steps 2-5 from Linux installation.

### Docker/Podman Agent

You can also run the agent in a container:

1. Build or pull the agent image:
```bash
# Build locally
docker build --target agent -t vigilon-agent:latest .

# Or pull from Docker Hub
docker pull harungecit/vigilon-agent:latest
```

2. Create agent config:
```bash
mkdir -p configs
cat > configs/agent-config.yaml <<EOF
server_url: http://YOUR_SERVER_IP:8090
token: YOUR_SECURE_TOKEN
check_interval: 30s
service_refresh_interval: 5m
services: []
EOF
```

**Note:** Services are managed through the UI, not the config file.

3. Run the agent container:
```bash
# Docker
docker run -d \
  --name vigilon-agent \
  -v $(pwd)/configs/agent-config.yaml:/app/config.yaml:ro \
  -v /var/run/docker.sock:/var/run/docker.sock:ro \
  --restart unless-stopped \
  vigilon-agent:latest

# Podman
podman run -d \
  --name vigilon-agent \
  -v $(pwd)/configs/agent-config.yaml:/app/config.yaml:ro \
  -v /run/podman/podman.sock:/var/run/docker.sock:ro \
  --restart unless-stopped \
  vigilon-agent:latest
```

**Note:** For monitoring host services, you may need to run with `--privileged` or mount additional volumes.

### Windows Systems

1. Download Windows agent:
```powershell
# Download vigilon-agent-windows-amd64.exe
```

2. Create config file at `C:\ProgramData\vigilon-agent\config.yaml`:
```yaml
server_url: http://YOUR_SERVER_IP:8090
token: YOUR_SECURE_TOKEN
check_interval: 30s
service_refresh_interval: 5m
services: []  # Optional fallback
```

**Note:** Agent fetches service list from UI automatically.

3. Install as Windows Service:
```powershell
# Using NSSM or Windows Service Manager
sc.exe create VigilonAgent binPath= "C:\Path\To\vigilon-agent.exe -config C:\ProgramData\vigilon-agent\config.yaml"
sc.exe start VigilonAgent
```

## Telegram Bot Setup

1. Create a new bot with @BotFather on Telegram:
   - Open Telegram and search for `@BotFather`
   - Send `/newbot`
   - Follow instructions and save your bot token

2. Get your Chat ID:
   - Send a message to your bot
   - Visit: `https://api.telegram.org/bot<YOUR_BOT_TOKEN>/getUpdates`
   - Find your chat ID in the response

3. Update server config:
```yaml
telegram:
  enabled: true
  bot_token: "123456789:ABCdefGHIjklMNOpqrsTUVwxyz"
  chat_ids:
    - "123456789"
```

4. Restart server:
```bash
sudo systemctl restart vigilon-server
```

## Configuration

### Server Configuration (`configs/config.yaml`)

```yaml
server:
  host: 0.0.0.0          # Listen on all interfaces
  port: 8090             # HTTP port

database:
  path: ./vigilon.db    # SQLite database path

telegram:
  enabled: true
  bot_token: "YOUR_BOT_TOKEN"
  chat_ids:
    - "YOUR_CHAT_ID"

monitoring:
  check_interval: 30s     # How often to check services
  retention_days: 30      # How long to keep history
  alert_cooldown: 5m      # Minimum time between alerts
```

### User Management

After installation, manage users through the web UI:

1. **Login as root** (username: `root`, password: `toor`)
2. **Change password immediately!**
   - Click on username in top-right
   - Select "Change Password"
3. **Create additional users:**
   - Go to "Users" page
   - Click "Add User"
   - Assign appropriate roles
4. **Manage roles and permissions:**
   - Go to "Users" page
   - Click on "Roles" tab
   - View/edit role permissions
   - Create custom roles

**Default Roles:**
- Super Administrator - Full access (cannot be deleted)
- Administrator - Full access except user deletion
- User - Read-only access

### Adding Servers via Web UI

1. Navigate to `http://your-server-ip:8090/servers`
2. Click "Add Server"
3. Fill in the form:
   - **Name**: Unique server identifier
   - **IP Address**: Server IP or hostname
   - **OS**: linux, windows, etc.
   - **Monitoring Mode**: 
     - `push` - Agent-based (recommended, one-line install)
     - `pull` - SSH-based
     - `hybrid` - Combined approach
   - **Token**: Auto-generate for push mode
   - **SSH Details**: For pull/hybrid mode
4. Click "Add Server"
5. Follow installation instructions shown for push mode
6. Add services for the server

### Adding Services Dynamically

**For Push Mode (Recommended):**
1. Go to server details page
2. Click "Add Service"
3. Enter service details:
   - **Name**: Actual service name (e.g., `nginx.service`, `W3SVC`)
   - **Display Name**: Human-readable name
   - **Description**: Optional description
   - **Enabled**: Check to monitor
4. Click "Add Service"
5. **Agent automatically fetches updated service list** (within 5 minutes, or restart agent)

**For Pull Mode:**
Services are checked via SSH on-demand.

### Adding Servers via Config File (Optional)

### Adding Servers via Config File (Optional)

You can also define servers in the config file (they will be synced to database on startup):

```yaml
servers:
  - name: web-server-1
    hostname: web1.example.com
    ip_address: 192.168.1.10
    port: 22
    os: linux
    monitoring_mode: pull  # pull, push, or hybrid
    ssh_user: admin
    ssh_key_path: /home/vigilon/.ssh/id_rsa
    enabled: true
    notify_telegram: true
    services:
      - name: nginx.service
        display_name: Nginx Web Server
        description: Main web server
        enabled: true
```

**Note:** For push mode, it's easier to use the UI which auto-generates tokens and installation commands.

### Adding Servers via Web UI

1. Navigate to `http://your-server-ip:8090/servers`
2. Click "Add Server"
3. Add services for the server

### Adding Servers via Web UI

1. Navigate to `http://your-server-ip:8080/servers`
2. Click "Add Server"
3. Fill in the form
4. Add services for the server

## Pull Mode Configuration (SSH)

1. Generate SSH key on server:
```bash
sudo -u vigilon ssh-keygen -t rsa -b 4096 -f /opt/vigilon/.ssh/id_rsa -N ""
```

2. Copy public key to target servers:
```bash
ssh-copy-id -i /opt/vigilon/.ssh/id_rsa.pub user@target-server
```

3. Test SSH connection:
```bash
sudo -u vigilon ssh -i /opt/vigilon/.ssh/id_rsa user@target-server
```

4. Add server in config with pull mode:
```yaml
monitoring_mode: pull
ssh_user: user
ssh_key_path: /opt/vigilon/.ssh/id_rsa
```

## Push Mode Configuration (Agent)

1. Generate secure token:
```bash
openssl rand -hex 32
```

2. Add server in config with push mode:
```yaml
monitoring_mode: push
agent_token: "your-secure-token-here"
```

3. Install agent on target server with matching token

## Troubleshooting

### Server won't start
```bash
# Check logs
sudo journalctl -u vigilon-server -n 50

# Check if port is available
sudo netstat -tlnp | grep 8080

# Verify config syntax
cat configs/config.yaml
```

### Agent not connecting
```bash
# Check agent logs
sudo journalctl -u vigilon-agent -n 50

# Test connectivity
curl http://your-server-ip:8080/api/servers

# Verify token matches
```

### Services not showing
```bash
# For systemd services, verify name
systemctl list-units --type=service

# Check agent config
cat /etc/vigilon-agent/config.yaml
```

### Telegram not working
```bash
# Test bot token
curl https://api.telegram.org/bot<TOKEN>/getMe

# Check chat ID
curl https://api.telegram.org/bot<TOKEN>/getUpdates

# Verify server logs
sudo journalctl -u vigilon-server | grep telegram
```

### SSH authentication fails (Pull mode)
```bash
# Test SSH manually
sudo -u vigilon ssh -i /path/to/key user@target-server

# Check key permissions
ls -la /opt/vigilon/.ssh/

# Verify authorized_keys on target
cat ~/.ssh/authorized_keys
```

## Updating

### Server Update
```bash
# Stop service
sudo systemctl stop vigilon-server

# Backup database
cp /opt/vigilon/vigilon.db /opt/vigilon/vigilon.db.backup

# Update binary
sudo cp vigilon-server /opt/vigilon/

# Start service
sudo systemctl start vigilon-server
```

### Agent Update
```bash
sudo systemctl stop vigilon-agent
sudo cp vigilon-agent /usr/local/bin/
sudo systemctl start vigilon-agent
```

## Backup and Restore

### Backup
```bash
# Backup database
cp /opt/vigilon/vigilon.db /backup/vigilon-$(date +%Y%m%d).db

# Backup config
cp /opt/vigilon/configs/config.yaml /backup/config-$(date +%Y%m%d).yaml
```

### Restore
```bash
sudo systemctl stop vigilon-server
cp /backup/vigilon-20240101.db /opt/vigilon/vigilon.db
sudo systemctl start vigilon-server
```

## Uninstallation

### Server
```bash
sudo systemctl stop vigilon-server
sudo systemctl disable vigilon-server
sudo rm /etc/systemd/system/vigilon-server.service
sudo rm -rf /opt/vigilon
sudo userdel vigilon
```

### Agent
```bash
sudo systemctl stop vigilon-agent
sudo systemctl disable vigilon-agent
sudo rm /etc/systemd/system/vigilon-agent.service
sudo rm /usr/local/bin/vigilon-agent
sudo rm -rf /etc/vigilon-agent
```

## Support

For issues or questions:
- Technical Documentation: [PROJECT_INFO.md](PROJECT_INFO.md)
- GitHub Issues: https://github.com/harungecit/vigilon/issues
- Website: https://harungecit.com
- Email: info@harungecit.com

**Author:** Harun Geçit
- GitHub: [@harungecit](https://github.com/harungecit)
- Twitter/X: [@harungecit_](https://twitter.com/harungecit_)
- Instagram: [@harungecit.dev](https://instagram.com/harungecit.dev)
