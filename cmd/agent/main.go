package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// AgentConfig represents the agent configuration
type AgentConfig struct {
	ServerURL              string        `yaml:"server_url"`
	Token                  string        `yaml:"token"`
	CheckInterval          time.Duration `yaml:"check_interval"`
	ServiceRefreshInterval time.Duration `yaml:"service_refresh_interval"`
	Services               []string      `yaml:"services"` // Optional fallback if API fetch fails
}

// ServiceListResponse represents the API response for service list
type ServiceListResponse struct {
	ServerID int       `json:"server_id"`
	Services []Service `json:"services"`
}

// Service represents a service from the API
type Service struct {
	ID          int    `json:"id"`
	ServerID    int    `json:"server_id"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
	Enabled     bool   `json:"enabled"`
}

// ServiceStatus represents a service status
type ServiceStatus string

const (
	StatusRunning  ServiceStatus = "running"
	StatusStopped  ServiceStatus = "stopped"
	StatusFailed   ServiceStatus = "failed"
	StatusUnknown  ServiceStatus = "unknown"
	StatusDegraded ServiceStatus = "degraded"
)

// AgentReport represents the data sent to the server
type AgentReport struct {
	Token    string          `json:"token"`
	Services []ServiceReport `json:"services"`
}

// ServiceReport represents a single service status report
type ServiceReport struct {
	Name         string        `json:"name"`
	Status       ServiceStatus `json:"status"`
	ErrorMessage string        `json:"error_message,omitempty"`
	PID          int           `json:"pid,omitempty"`
	Memory       int64         `json:"memory_kb,omitempty"`
	CPU          float64       `json:"cpu_percent,omitempty"`
	Uptime       int64         `json:"uptime_seconds,omitempty"`
}

var (
	configPath = flag.String("config", "/etc/vigilon-agent/config.yaml", "Path to configuration file")
	version    = "1.0.0"

	// Cached service list from API
	cachedServices []string
	
	// Reusable HTTP client with connection pooling
	httpClient = &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        10,
			MaxIdleConnsPerHost: 2,
			IdleConnTimeout:     90 * time.Second,
		},
	}
)

func main() {
	flag.Parse()

	// Load configuration
	config, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Vigilon Agent v%s starting...", version)
	log.Printf("Server URL: %s", config.ServerURL)
	log.Printf("Check interval: %v", config.CheckInterval)
	log.Printf("Service refresh interval: %v", config.ServiceRefreshInterval)

	// Set GOMAXPROCS for better resource usage
	if runtime.NumCPU() > 2 {
		runtime.GOMAXPROCS(2) // Limit to 2 cores for agent
	}

	// Fetch initial service list from API
	if err := refreshServiceList(config); err != nil {
		log.Printf("Failed to fetch service list from API: %v", err)
		// Fall back to config file services if available
		if len(config.Services) > 0 {
			cachedServices = config.Services
			log.Printf("Using %d services from config file as fallback", len(cachedServices))
		} else {
			log.Printf("WARNING: No services to monitor. Add services in the panel or config file.")
		}
	}

	// Run initial check
	if err := checkAndReport(config); err != nil {
		log.Printf("Initial check failed: %v", err)
	}

	// Start periodic checking
	checkTicker := time.NewTicker(config.CheckInterval)
	defer checkTicker.Stop()

	// Start periodic service list refresh
	refreshTicker := time.NewTicker(config.ServiceRefreshInterval)
	defer refreshTicker.Stop()

	// Manual GC trigger every 10 minutes to prevent memory buildup
	gcTicker := time.NewTicker(10 * time.Minute)
	defer gcTicker.Stop()

	for {
		select {
		case <-checkTicker.C:
			if err := checkAndReport(config); err != nil {
				log.Printf("Check failed: %v", err)
			}
		case <-refreshTicker.C:
			if err := refreshServiceList(config); err != nil {
				log.Printf("Failed to refresh service list: %v", err)
			}
		case <-gcTicker.C:
			runtime.GC() // Force garbage collection
		}
	}
}

// loadConfig loads the agent configuration from a YAML file
func loadConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Set defaults
	if config.CheckInterval == 0 {
		config.CheckInterval = 30 * time.Second
	}
	if config.ServiceRefreshInterval == 0 {
		config.ServiceRefreshInterval = 5 * time.Minute
	}

	return &config, nil
}

// refreshServiceList fetches the service list from the API
func refreshServiceList(config *AgentConfig) error {
	url := fmt.Sprintf("%s/api/agent/services?token=%s", config.ServerURL, config.Token)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to fetch service list: %w", err)
	}
	defer resp.Body.Close()
	
	// Drain and close response body to reuse connection
	defer io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	var serviceList ServiceListResponse
	if err := json.NewDecoder(resp.Body).Decode(&serviceList); err != nil {
		return fmt.Errorf("failed to decode service list: %w", err)
	}

	// Extract service names from enabled services
	newServices := make([]string, 0, len(serviceList.Services))
	for _, service := range serviceList.Services {
		if service.Enabled {
			newServices = append(newServices, service.Name)
		}
	}

	// Check if service list changed
	if !servicesEqual(cachedServices, newServices) {
		log.Printf("Service list updated: %d services", len(newServices))
		for _, svc := range newServices {
			log.Printf("  - %s", svc)
		}
		cachedServices = newServices
	}

	return nil
}

// servicesEqual checks if two service lists are equal
func servicesEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// checkAndReport checks all services and reports to the server
func checkAndReport(config *AgentConfig) error {
	// Skip if no services to check
	if len(cachedServices) == 0 {
		return nil
	}

	report := AgentReport{
		Token:    config.Token,
		Services: make([]ServiceReport, 0, len(cachedServices)),
	}

	for _, serviceName := range cachedServices {
		serviceReport := checkService(serviceName)
		report.Services = append(report.Services, serviceReport)
		log.Printf("Service %s: %s", serviceName, serviceReport.Status)
	}

	// Send report to server
	return sendReport(config.ServerURL, report)
}

// checkService checks a single service status
func checkService(serviceName string) ServiceReport {
	report := ServiceReport{
		Name: serviceName,
	}

	switch runtime.GOOS {
	case "linux":
		return checkLinuxService(serviceName)
	case "windows":
		return checkWindowsService(serviceName)
	default:
		report.Status = StatusUnknown
		report.ErrorMessage = fmt.Sprintf("Unsupported OS: %s", runtime.GOOS)
	}

	return report
}

// checkLinuxService checks a systemd service on Linux
func checkLinuxService(serviceName string) ServiceReport {
	report := ServiceReport{
		Name: serviceName,
	}

	// Check service status
	cmd := exec.Command("systemctl", "is-active", serviceName)
	output, err := cmd.Output()
	statusStr := strings.TrimSpace(string(output))

	if err == nil {
		switch statusStr {
		case "active":
			report.Status = StatusRunning
		case "inactive":
			report.Status = StatusStopped
		case "failed":
			report.Status = StatusFailed
		case "activating", "deactivating":
			report.Status = StatusDegraded
		default:
			report.Status = StatusUnknown
		}
	} else {
		report.Status = StatusUnknown
		report.ErrorMessage = fmt.Sprintf("Failed to check service: %v", err)
		return report
	}

	// Get additional info if running
	if report.Status == StatusRunning {
		// Get PID
		cmd = exec.Command("systemctl", "show", "-p", "MainPID", "--value", serviceName)
		if output, err := cmd.Output(); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil && pid > 0 {
				report.PID = pid

				// Get memory and CPU
				cmd = exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "rss=,%cpu=")
				if output, err := cmd.Output(); err == nil {
					fields := strings.Fields(string(output))
					if len(fields) >= 2 {
						if mem, err := strconv.ParseInt(fields[0], 10, 64); err == nil {
							report.Memory = mem
						}
						if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
							report.CPU = cpu
						}
					}
				}
			}
		}

		// Get uptime
		cmd = exec.Command("systemctl", "show", "-p", "ActiveEnterTimestamp", "--value", serviceName)
		if output, err := cmd.Output(); err == nil {
			timestampStr := strings.TrimSpace(string(output))
			if timestampStr != "" && timestampStr != "n/a" {
				if t, err := time.Parse("Mon 2006-01-02 15:04:05 MST", timestampStr); err == nil {
					report.Uptime = int64(time.Since(t).Seconds())
				}
			}
		}
	}

	return report
}

// checkWindowsService checks a Windows service
func checkWindowsService(serviceName string) ServiceReport {
	report := ServiceReport{
		Name: serviceName,
	}

	// Check service status
	cmd := exec.Command("powershell", "-Command",
		fmt.Sprintf("Get-Service -Name %s | Select-Object -ExpandProperty Status", serviceName))
	output, err := cmd.Output()

	if err != nil {
		report.Status = StatusUnknown
		report.ErrorMessage = fmt.Sprintf("Failed to check service: %v", err)
		return report
	}

	statusStr := strings.TrimSpace(string(output))
	switch strings.ToLower(statusStr) {
	case "running":
		report.Status = StatusRunning
	case "stopped":
		report.Status = StatusStopped
	case "paused":
		report.Status = StatusDegraded
	default:
		report.Status = StatusUnknown
	}

	// Get additional info if running
	if report.Status == StatusRunning {
		// Get process ID
		cmd = exec.Command("powershell", "-Command",
			fmt.Sprintf("Get-CimInstance Win32_Service -Filter \"Name='%s'\" | Select-Object -ExpandProperty ProcessId", serviceName))
		if output, err := cmd.Output(); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(output))); err == nil && pid > 0 {
				report.PID = pid

				// Get memory and CPU
				cmd = exec.Command("powershell", "-Command",
					fmt.Sprintf("Get-Process -Id %d | Select-Object @{N='WS';E={$_.WS/1KB}},CPU | ConvertTo-Csv -NoTypeInformation", pid))
				if output, err := cmd.Output(); err == nil {
					lines := strings.Split(string(output), "\n")
					if len(lines) > 1 {
						fields := strings.Split(strings.Trim(lines[1], "\""), "\",\"")
						if len(fields) >= 2 {
							if mem, err := strconv.ParseFloat(fields[0], 64); err == nil {
								report.Memory = int64(mem)
							}
							if cpu, err := strconv.ParseFloat(fields[1], 64); err == nil {
								report.CPU = cpu
							}
						}
					}
				}
			}
		}
	}

	return report
}

// sendReport sends the report to the server
func sendReport(serverURL string, report AgentReport) error {
	url := fmt.Sprintf("%s/api/agent/report", serverURL)

	jsonData, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send report: %w", err)
	}
	defer resp.Body.Close()
	
	// Drain and close response body to reuse connection
	defer io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned status %d", resp.StatusCode)
	}

	return nil
}
