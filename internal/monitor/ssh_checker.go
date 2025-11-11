package monitor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/harungecit/vigilon/internal/models"
)

// ServiceInfo holds detailed information about a service
type ServiceInfo struct {
	PID    int
	Memory int64   // in KB
	CPU    float64 // percentage
	Uptime int64   // in seconds
}

// SSHChecker checks services via SSH
type SSHChecker struct {
	server *models.Server
}

// NewSSHChecker creates a new SSH checker
func NewSSHChecker(server *models.Server) *SSHChecker {
	return &SSHChecker{server: server}
}

// CheckService checks a service status via SSH
func (c *SSHChecker) CheckService(ctx context.Context, serviceName string) (models.ServiceStatus, *ServiceInfo, error) {
	// Determine OS type and use appropriate command
	switch c.server.OS {
	case "linux":
		return c.checkLinuxService(ctx, serviceName)
	case "windows":
		return c.checkWindowsService(ctx, serviceName)
	default:
		return models.StatusUnknown, nil, fmt.Errorf("unsupported OS: %s", c.server.OS)
	}
}

// checkLinuxService checks a systemd service on Linux
func (c *SSHChecker) checkLinuxService(ctx context.Context, serviceName string) (models.ServiceStatus, *ServiceInfo, error) {
	// Build SSH command
	sshCmd := c.buildSSHCommand()

	// Check service status using systemctl
	statusCmd := fmt.Sprintf("systemctl is-active %s", serviceName)
	output, err := c.executeSSH(ctx, sshCmd, statusCmd)

	status := models.StatusUnknown
	if err == nil {
		output = strings.TrimSpace(output)
		switch output {
		case "active":
			status = models.StatusRunning
		case "inactive":
			status = models.StatusStopped
		case "failed":
			status = models.StatusFailed
		case "activating", "deactivating":
			status = models.StatusDegraded
		default:
			status = models.StatusUnknown
		}
	} else {
		// If systemctl command fails, service might not exist
		return models.StatusUnknown, nil, fmt.Errorf("failed to check service: %w", err)
	}

	// Get service info if running
	var info *ServiceInfo
	if status == models.StatusRunning {
		info = c.getLinuxServiceInfo(ctx, sshCmd, serviceName)
	}

	return status, info, nil
}

// getLinuxServiceInfo gets detailed info about a Linux service
func (c *SSHChecker) getLinuxServiceInfo(ctx context.Context, sshCmd []string, serviceName string) *ServiceInfo {
	info := &ServiceInfo{}

	// Get PID
	pidCmd := fmt.Sprintf("systemctl show -p MainPID --value %s", serviceName)
	if output, err := c.executeSSH(ctx, sshCmd, pidCmd); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(output)); err == nil {
			info.PID = pid

			// Get memory and CPU usage using ps
			if pid > 0 {
				psCmd := fmt.Sprintf("ps -p %d -o rss=,%%cpu= 2>/dev/null", pid)
				if output, err := c.executeSSH(ctx, sshCmd, psCmd); err == nil {
					fields := strings.Fields(output)
					if len(fields) >= 2 {
						if mem, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
							info.Memory = mem
						}
						if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
							info.CPU = cpu
						}
					}
				}
			}
		}
	}

	// Get uptime (in seconds)
	uptimeCmd := fmt.Sprintf("systemctl show -p ActiveEnterTimestamp --value %s", serviceName)
	if output, err := c.executeSSH(ctx, sshCmd, uptimeCmd); err == nil {
		output = strings.TrimSpace(output)
		if output != "" && output != "n/a" {
			// Parse timestamp and calculate uptime
			if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", output); err == nil {
				info.Uptime = int64(time.Since(t).Seconds())
			}
		}
	}

	return info
}

// checkWindowsService checks a Windows service
func (c *SSHChecker) checkWindowsService(ctx context.Context, serviceName string) (models.ServiceStatus, *ServiceInfo, error) {
	sshCmd := c.buildSSHCommand()

	// Check service status using PowerShell
	statusCmd := fmt.Sprintf("powershell -Command \"Get-Service -Name %s | Select-Object -ExpandProperty Status\"", serviceName)
	output, err := c.executeSSH(ctx, sshCmd, statusCmd)

	if err != nil {
		return models.StatusUnknown, nil, fmt.Errorf("failed to check service: %w", err)
	}

	output = strings.TrimSpace(output)
	status := models.StatusUnknown

	switch strings.ToLower(output) {
	case "running":
		status = models.StatusRunning
	case "stopped":
		status = models.StatusStopped
	case "paused":
		status = models.StatusDegraded
	default:
		status = models.StatusUnknown
	}

	// Get service info if running
	var info *ServiceInfo
	if status == models.StatusRunning {
		info = c.getWindowsServiceInfo(ctx, sshCmd, serviceName)
	}

	return status, info, nil
}

// getWindowsServiceInfo gets detailed info about a Windows service
func (c *SSHChecker) getWindowsServiceInfo(ctx context.Context, sshCmd []string, serviceName string) *ServiceInfo {
	info := &ServiceInfo{}

	// Get process ID
	pidCmd := fmt.Sprintf("powershell -Command \"Get-CimInstance Win32_Service -Filter \\\"Name='%s'\\\" | Select-Object -ExpandProperty ProcessId\"", serviceName)
	if output, err := c.executeSSH(ctx, sshCmd, pidCmd); err == nil {
		if pid, err := strconv.Atoi(strings.TrimSpace(output)); err == nil && pid > 0 {
			info.PID = pid

			// Get memory and CPU usage
			perfCmd := fmt.Sprintf("powershell -Command \"Get-Process -Id %d | Select-Object @{N='WS';E={$_.WS/1KB}},CPU | ConvertTo-Csv -NoTypeInformation\"", pid)
			if output, err := c.executeSSH(ctx, sshCmd, perfCmd); err == nil {
				lines := strings.Split(output, "\n")
				if len(lines) > 1 {
					fields := strings.Split(strings.Trim(lines[1], "\""), "\",\"")
					if len(fields) >= 2 {
						if mem, err := strconv.ParseFloat(fields[0], 64); err == nil {
							info.Memory = int64(mem)
						}
						if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
							info.CPU = cpu
						}
					}
				}
			}
		}
	}

	return info
}

// buildSSHCommand builds the base SSH command
func (c *SSHChecker) buildSSHCommand() []string {
	cmd := []string{"ssh"}

	// Add key if specified
	if c.server.SSHKeyPath != "" {
		cmd = append(cmd, "-i", c.server.SSHKeyPath)
	}

	// Add options with aggressive timeouts to prevent hanging
	cmd = append(cmd,
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=5",        // Reduced from 10
		"-o", "ServerAliveInterval=5",   // Send keepalive every 5s
		"-o", "ServerAliveCountMax=2",   // Disconnect after 2 failed keepalives
		"-o", "ConnectionAttempts=1",    // Don't retry
		"-o", "BatchMode=yes",           // Never ask for password
		"-p", strconv.Itoa(c.server.Port),
	)

	// Add user@host
	target := c.server.IPAddress
	if c.server.SSHUser != "" {
		target = c.server.SSHUser + "@" + target
	}
	cmd = append(cmd, target)

	return cmd
}

// executeSSH executes a command via SSH
func (c *SSHChecker) executeSSH(ctx context.Context, sshCmd []string, remoteCmd string) (string, error) {
	cmd := append(sshCmd, remoteCmd)

	execCmd := exec.CommandContext(ctx, cmd[0], cmd[1:]...)
	execCmd.Env = os.Environ()

	output, err := execCmd.Output()
	if err != nil {
		return "", err
	}

	return string(output), nil
}
