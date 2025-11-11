# Vigilon v1.0.0 - Release Notes

**Release Date:** November 11, 2025

## 🎉 Initial Release

Vigilon is a comprehensive multi-platform service monitoring and alerting system that tracks service status across Linux, Windows, Raspberry Pi, and other platforms with real-time Telegram notifications.

## ✨ Key Features

### Core Functionality
- **Multi-Platform Support**: Monitor services on Linux (systemd), Windows, and Raspberry Pi
- **Flexible Monitoring Modes**:
  - **Pull Mode**: SSH-based monitoring with jump host support
  - **Push Mode**: Lightweight agent with auto-configuration
  - **Hybrid Mode**: Combined SSH and local script approach
- **Real-Time Dashboard**: Modern web UI with live service status
- **Alert Management**: Acknowledge and archive alerts with history

### Security & Access Control
- **Role-Based Access Control (RBAC)**: 28 granular permissions across 6 categories
- **3 Default Roles**: Super Admin, Admin, and User with customizable permissions
- **Session Management**: Secure cookie-based authentication with bcrypt
- **Default Credentials**: root/toor (⚠️ MUST change immediately!)

### Agent Management
- **One-Line Installer**: `curl -fsSL http://server:8090/install.sh?token=TOKEN | sudo bash`
- **Dynamic Service Configuration**: Add/remove services via UI, agent auto-updates
- **Auto-Discovery**: Agent fetches service list from API every 5 minutes
- **Platform Detection**: Automatic OS and architecture detection

### Monitoring & Metrics
- **Service Status**: Running, Stopped, Failed, Degraded, Unknown
- **Health Metrics**: PID, CPU usage, Memory (KB), Uptime
- **Historical Data**: 30-day retention (configurable)
- **Connection Tracking**: Monitor agent connectivity in real-time
- **Response Time**: Track service response times

### Integration & Deployment
- **Telegram Bot**: Instant alerts with bot commands (/status, /servers, /alerts)
- **Docker/Podman**: Full containerization support with docker-compose
- **REST API**: Complete API for automation (80+ endpoints)
- **Systemd Services**: Production-ready service files included

## 📦 Installation

### Quick Start with Docker
```bash
git clone https://github.com/harungecit/vigilon.git
cd vigilon
cp configs/config.example.yaml configs/config.yaml
# Edit config.yaml with your settings
docker-compose up -d
```

### Build from Source
```bash
git clone https://github.com/harungecit/vigilon.git
cd vigilon
make build
./vigilon-server -config configs/config.yaml
```

### Access
- Web UI: http://localhost:8090
- Default Login: `root` / `toor` (⚠️ Change immediately!)

## 📊 Technical Specifications

### Technology Stack
- **Language**: Go 1.21+
- **Database**: SQLite3 with WAL mode
- **Web Framework**: gorilla/mux
- **Authentication**: bcrypt
- **Frontend**: Vanilla JavaScript, HTML5, CSS3
- **API**: Telegram Bot API

### Performance
- **Resource Usage**: ~50MB RAM for agent, ~100MB for server
- **Database Optimization**: 64MB cache, WAL mode, indexed queries
- **Check Interval**: Configurable (default: 30s)
- **Concurrent Monitoring**: Parallel service checks

### Security Features
- Bcrypt password hashing (cost 10)
- 32-byte secure session tokens
- IP and user agent tracking
- Privilege separation (systemd)
- Read-only system paths

## 📋 Requirements

### Server
- Go 1.21+ (for building) OR Docker/Podman
- 512MB RAM minimum
- 100MB disk space
- Linux, Windows, or macOS

### Agent
- 50MB RAM
- 20MB disk space
- SSH access (for pull mode) OR agent binary (for push mode)

## 🔧 Configuration

### Server Configuration
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
```

### Agent Configuration (Auto-generated via UI)
```yaml
server_url: http://server:8090
token: auto-generated-secure-token
check_interval: 30s
service_refresh_interval: 5m
services: []  # Fetched from API
```

## 📚 Documentation

- **README.md**: Quick start guide and feature overview
- **INSTALL.md**: Detailed installation instructions for all platforms
- **PROJECT_INFO.md**: Complete technical documentation (database schema, API endpoints, architecture)

## 🎯 Use Cases

- Monitor critical services across multiple servers
- Track application uptime and performance
- Receive instant notifications when services fail
- Manage server infrastructure with role-based access
- Automate service monitoring with REST API
- Deploy monitoring in containerized environments

## 🔐 Security Notice

**IMPORTANT**: The default credentials are:
- Username: `root`
- Password: `toor`

**You MUST change these immediately after first login!**

1. Login to web UI
2. Go to Users page
3. Change root password or create new admin user
4. Disable root user if not needed

## 🐛 Known Issues

None in this release.

## 🚀 Future Enhancements

Potential features for future releases:
- Email notifications
- Webhook support
- Prometheus metrics export
- Grafana integration
- Multi-language support
- Advanced alerting rules
- Service dependency management
- Custom health checks
- Performance graphs
- Log aggregation

## 📝 Changelog

### [1.0.0] - 2025-11-11

#### Added
- Initial release
- Multi-platform service monitoring
- Role-based access control
- Dynamic service configuration
- One-line agent installer
- Telegram bot integration
- Docker/Podman support
- Complete documentation
- Systemd service files
- Alert management
- Historical data tracking
- Connection status monitoring
- Health metrics collection

## 👨‍💻 Author

**Harun Geçit**
- Website: [harungecit.com](https://harungecit.com)
- Email: info@harungecit.com
- GitHub: [@harungecit](https://github.com/harungecit)
- Twitter/X: [@harungecit_](https://twitter.com/harungecit_)
- Instagram: [@harungecit.dev](https://instagram.com/harungecit.dev)

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Thanks to the Go community for excellent libraries
- Inspired by modern monitoring solutions
- Built with ❤️ for system administrators and DevOps engineers

## 💬 Support

For issues, questions, or feature requests:
- GitHub Issues: https://github.com/harungecit/vigilon/issues
- Email: info@harungecit.com
- Documentation: See PROJECT_INFO.md for technical details

---

**Download Release**: https://github.com/harungecit/vigilon/releases/tag/v1.0.0
