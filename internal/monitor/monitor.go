package monitor

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/harungecit/vigilon/internal/database"
	"github.com/harungecit/vigilon/internal/models"
)

// Monitor handles service monitoring
type Monitor struct {
	db            *database.DB
	interval      time.Duration
	alertCooldown time.Duration
	lastAlerts    map[string]time.Time // key: "serverID:serviceID"
	mu            sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup
	maxWorkers    int           // Maximum concurrent workers
	workerSem     chan struct{} // Semaphore for limiting workers
}

// New creates a new Monitor instance
func New(db *database.DB, interval, alertCooldown time.Duration) *Monitor {
	maxWorkers := 10 // Limit concurrent workers to 10
	return &Monitor{
		db:            db,
		interval:      interval,
		alertCooldown: alertCooldown,
		lastAlerts:    make(map[string]time.Time),
		stopCh:        make(chan struct{}),
		maxWorkers:    maxWorkers,
		workerSem:     make(chan struct{}, maxWorkers),
	}
}

// Start begins the monitoring loop
func (m *Monitor) Start(ctx context.Context) {
	log.Println("Starting monitor...")
	ticker := time.NewTicker(m.interval)
	defer ticker.Stop()

	// Run initial check
	m.checkAllServers(ctx)

	for {
		select {
		case <-ticker.C:
			m.checkAllServers(ctx)
		case <-m.stopCh:
			return
		case <-ctx.Done():
			return
		}
	}
}

// Stop stops the monitoring loop
func (m *Monitor) Stop() {
	close(m.stopCh)
	m.wg.Wait()
}

// checkAllServers checks all enabled servers
func (m *Monitor) checkAllServers(ctx context.Context) {
	servers, err := m.db.GetAllServers()
	if err != nil {
		log.Printf("Failed to get servers: %v", err)
		return
	}

	for _, server := range servers {
		// Check for idle connections (last seen > 5 minutes ago)
		m.checkIdleStatus(server)

		if !server.Enabled {
			continue
		}

		// Acquire worker slot (blocks if limit reached)
		m.workerSem <- struct{}{}

		m.wg.Add(1)
		go func(srv *models.Server) {
			defer func() {
				<-m.workerSem // Release worker slot
				m.wg.Done()
			}()
			m.checkServer(ctx, srv)
		}(server)
	}
}

// checkIdleStatus checks if a server connection should be marked as idle
func (m *Monitor) checkIdleStatus(server *models.Server) {
	// Only check for push mode servers that are currently connected
	if server.MonitoringMode != models.ModePush || server.ConnectionStatus != models.ConnectionConnected {
		return
	}

	// If last seen is more than 5 minutes ago, mark as idle
	if server.LastSeen != nil {
		idleThreshold := 5 * time.Minute
		if time.Since(*server.LastSeen) > idleThreshold {
			if err := m.db.UpdateServerConnectionStatus(server.ID, models.ConnectionIdle); err != nil {
				log.Printf("Failed to update server %s to idle: %v", server.Name, err)
			} else {
				log.Printf("Server %s marked as idle (no activity for %v)", server.Name, time.Since(*server.LastSeen))
			}
		}
	}
}

// checkServer checks a single server and its services
func (m *Monitor) checkServer(ctx context.Context, server *models.Server) {
	services, err := m.db.GetServicesByServer(server.ID)
	if err != nil {
		log.Printf("Failed to get services for server %s: %v", server.Name, err)
		return
	}

	for _, service := range services {
		if !service.Enabled {
			continue
		}

		var check *models.ServiceCheck
		switch server.MonitoringMode {
		case models.ModePull:
			check = m.checkServicePull(ctx, server, service)
		case models.ModePush:
			// For push mode, we just check the last reported status
			check = m.checkServicePush(service)
		case models.ModeHybrid:
			check = m.checkServiceHybrid(ctx, server, service)
		default:
			log.Printf("Unknown monitoring mode %s for server %s", server.MonitoringMode, server.Name)
			continue
		}

		if check != nil {
			if err := m.db.CreateServiceCheck(check); err != nil {
				log.Printf("Failed to save check result: %v", err)
			}

			// Check if we need to send an alert
			m.handleAlert(server, service, check)
		}
	}

	// Update last seen
	if err := m.db.UpdateServerLastSeen(server.ID); err != nil {
		log.Printf("Failed to update last seen for server %s: %v", server.Name, err)
	}
}

// checkServicePull checks a service in pull mode (SSH connection)
func (m *Monitor) checkServicePull(ctx context.Context, server *models.Server, service *models.Service) *models.ServiceCheck {
	start := time.Now()
	check := &models.ServiceCheck{
		ServiceID: service.ID,
		CheckedAt: start,
	}

	// Use the SSH checker
	checker := NewSSHChecker(server)
	status, info, err := checker.CheckService(ctx, service.Name)

	check.ResponseTime = time.Since(start).Milliseconds()
	check.Status = status

	if err != nil {
		check.ErrorMessage = err.Error()
	}

	if info != nil {
		check.PID = info.PID
		check.Memory = info.Memory
		check.CPU = info.CPU
		check.Uptime = info.Uptime
	}

	return check
}

// checkServicePush checks a service in push mode (agent reports)
func (m *Monitor) checkServicePush(service *models.Service) *models.ServiceCheck {
	// Get the last check from database
	lastCheck, err := m.db.GetLatestServiceCheck(service.ID)
	if err != nil {
		// No previous check, mark as unknown
		return &models.ServiceCheck{
			ServiceID:    service.ID,
			Status:       models.StatusUnknown,
			ErrorMessage: "No data received from agent",
			CheckedAt:    time.Now(),
		}
	}

	// If last check is older than 2 * interval, consider it stale
	if time.Since(lastCheck.CheckedAt) > 2*m.interval {
		return &models.ServiceCheck{
			ServiceID:    service.ID,
			Status:       models.StatusUnknown,
			ErrorMessage: fmt.Sprintf("Agent not reporting (last seen: %v)", lastCheck.CheckedAt),
			CheckedAt:    time.Now(),
		}
	}

	// Return the current status (already in DB from agent push)
	return nil
}

// checkServiceHybrid checks a service in hybrid mode (SSH + local script)
func (m *Monitor) checkServiceHybrid(ctx context.Context, server *models.Server, service *models.Service) *models.ServiceCheck {
	// Similar to pull mode but executes a pre-installed script
	return m.checkServicePull(ctx, server, service)
}

// handleAlert checks if an alert should be sent
func (m *Monitor) handleAlert(server *models.Server, service *models.Service, check *models.ServiceCheck) {
	// Only alert on non-running status
	if check.Status == models.StatusRunning {
		return
	}

	// Check cooldown
	alertKey := fmt.Sprintf("%d:%d", server.ID, service.ID)
	m.mu.RLock()
	lastAlert, exists := m.lastAlerts[alertKey]
	m.mu.RUnlock()

	if exists && time.Since(lastAlert) < m.alertCooldown {
		return
	}

	// Create alert
	message := fmt.Sprintf("🚨 Service '%s' on server '%s' is %s",
		service.DisplayName, server.Name, check.Status)
	if check.ErrorMessage != "" {
		message += fmt.Sprintf("\nError: %s", check.ErrorMessage)
	}

	alert := &models.Alert{
		ServiceID: service.ID,
		ServerID:  server.ID,
		Status:    check.Status,
		Message:   message,
		SentVia:   "telegram",
	}

	if err := m.db.CreateAlert(alert); err != nil {
		log.Printf("Failed to create alert: %v", err)
		return
	}

	// Update last alert time
	m.mu.Lock()
	m.lastAlerts[alertKey] = time.Now()
	m.mu.Unlock()

	log.Printf("Alert created: %s", message)
}
