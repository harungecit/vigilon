# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.1.2] - 2025-11-14

### Fixed
- **Agent Memory Leak**: Fixed critical memory leak in agent causing continuous growth from 3MB to 11MB+
  - Added aggressive GC strategy with SetGCPercent(50) and forced cleanup after each check
  - Implemented context timeouts (5s) on all system commands to prevent hanging processes
  - Explicit buffer cleanup by setting output variables to nil after use
  - Added cleanupServiceStates() to remove orphaned service state entries
  - Memory now stays stable at ~3-4MB

### Changed
- **Agent Logging**: Optimized logging to only show service status changes instead of every check
  - Reduces log noise significantly
  - Shows previous state for context: "Service X: Y (changed from Z)"
  - Added summary log when changes detected

### Added
- **Build System**: Enhanced Makefile to copy all agent binaries to web/static/bin/
  - Now includes Linux AMD64, Linux ARM64, and Windows AMD64 binaries
  - Automatic deployment to static download directory

## [1.1.1] - 2025-11-14

### Added
- **Toast Notification System**: Modern notification system replacing browser alerts
  - Right-side toast notifications with auto-dismiss
  - Custom confirmation modals with promise-based async/await
  - Color-coded severity (success, error, warning, info)
  - Smooth animations and responsive design
  - Consistent UX across all pages

### Changed
- **UI/UX Improvements**: Converted all native browser alerts and confirms
  - alerts.js: acknowledge, archive, load more functions
  - users.js: user/role management, permissions (15+ conversions)
  - main.js: logout confirmation
  - servers.js: disconnect/delete confirmations
  - server_detail.js: server/service toggle, delete, edit functions

## [1.1.0] - 2025-11-14

### Added
- **Real-time Updates**: Server-Sent Events (SSE) for live dashboard updates
  - New internal/sse package for SSE management
  - SSE client library with auto-reconnect
  - Real-time updates for dashboard, servers, and service history (5s interval)
  - SSE endpoints: /api/sse/dashboard, /api/sse/servers, /api/sse/service-history

- **Enhanced RBAC**: Hierarchical role-based access control
  - Role hierarchy: Super Admin > Admin > User
  - Permission-based UI controls with dynamic button visibility
  - Admin password reset capability without current password
  - Fixed role management permissions and visibility

- **Current User API**: New /api/users/me endpoint
  - Provides frontend context for current user
  - Fixed route collision with /api/users/{id}

### Changed
- **Password Management**: Improved password change workflow
  - API supports both PUT and POST methods
  - Fixed 405 errors on password change
  - Admin can reset user passwords without knowing current password

- **Database**: Enabled WAL mode for SQLite
  - Requires CGO_ENABLED=1 for better performance
  - Improved concurrent access handling

### Fixed
- **Super Admin Visibility**: Fixed role management UI for Super Admin
- **Duplicate Modals**: Removed duplicate modals and script tags in user management
- **Route Ordering**: Fixed /api/users/me vs /api/users/{id} collision
- **Permission Checks**: Improved permission middleware with detailed logging
- **Debug Logs**: Removed console.log statements from production

### Documentation
- Updated README.md with all recent changes
- Added SSE endpoints documentation
- Enhanced RBAC documentation with role hierarchy
- Added installation and configuration guides

## [1.0.0] - 2025-11-13

### Added
- Initial release of Vigilon Server and Agent
- Server monitoring with systemd service checks
- Web-based dashboard
- Alert system with Telegram notifications
- User authentication and basic RBAC
- RESTful API
- Multi-platform agent support (Linux, Windows, ARM)
- SSH connectivity monitoring
- Service status tracking (running, stopped, failed)
- Real-time metrics: CPU, Memory, Uptime
- SQLite database with automatic migrations

### Features
- **Server Management**
  - Add/remove/edit servers
  - Monitor server connectivity
  - Track server status and metrics

- **Service Monitoring**
  - Monitor systemd services (Linux) and Windows services
  - Track service status, CPU, memory usage
  - Alert on service failures

- **Alert System**
  - Configurable alert rules
  - Telegram integration
  - Alert history and archiving
  - Acknowledgment system

- **User Management**
  - User authentication with JWT
  - Role-based access control
  - Password management

- **Agent**
  - Lightweight monitoring agent
  - HTTP-based reporting
  - Service list fetching from API
  - Automatic service discovery
  - Memory-optimized with GC tuning

[1.1.2]: https://github.com/harungecit/vigilon/releases/tag/v1.1.2
[1.1.1]: https://github.com/harungecit/vigilon/releases/tag/v1.1.1
[1.1.0]: https://github.com/harungecit/vigilon/releases/tag/v1.1.0
[1.0.0]: https://github.com/harungecit/vigilon/releases/tag/v1.0.0
