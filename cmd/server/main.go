package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/harungecit/vigilon/internal/api"
	"github.com/harungecit/vigilon/internal/config"
	"github.com/harungecit/vigilon/internal/database"
	"github.com/harungecit/vigilon/internal/models"
	"github.com/harungecit/vigilon/internal/monitor"
	"github.com/harungecit/vigilon/internal/telegram"
)

var (
	configPath = flag.String("config", "configs/config.yaml", "Path to configuration file")
	version    = "1.0.0"
)

func main() {
	flag.Parse()

	log.Printf("Vigilon Server v%s starting...", version)

	// Load configuration
	cfg, err := loadOrCreateConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Initialize database
	db, err := database.New(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()
	log.Println("Database initialized")

	// Sync config file servers to database
	if err := syncConfigToDatabase(cfg, db); err != nil {
		log.Printf("Warning: Failed to sync config to database: %v", err)
	}

	// Initialize Telegram notifier
	telegramNotifier, err := telegram.New(&cfg.Telegram, db)
	if err != nil {
		log.Printf("Warning: Failed to initialize Telegram: %v", err)
	}

	// Start Telegram bot in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if telegramNotifier != nil {
		go telegramNotifier.Start(ctx)
	}

	// Initialize monitor
	mon := monitor.New(db, cfg.Monitoring.CheckInterval, cfg.Monitoring.AlertCooldown)

	// Start monitoring in background
	go mon.Start(ctx)
	log.Printf("Monitor started (check interval: %v)", cfg.Monitoring.CheckInterval)

	// Initialize API
	apiHandler := api.New(db, telegramNotifier)

	// Create HTTP server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      apiHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // Disable write timeout for SSE
		IdleTimeout:  120 * time.Second,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("Server listening on http://%s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	mon.Stop()
	cancel() // Stop Telegram bot

	log.Println("Server stopped")
}

// loadOrCreateConfig loads config or creates a default one
func loadOrCreateConfig(path string) (*config.AppConfig, error) {
	// Try to load existing config
	cfg, err := config.LoadFromFile(path)
	if err != nil {
		// If file doesn't exist, create default config
		if os.IsNotExist(err) {
			log.Printf("Config file not found, creating default config at %s", path)
			cfg = config.GetDefaultConfig()
			if err := config.SaveToFile(cfg, path); err != nil {
				return nil, fmt.Errorf("failed to save default config: %w", err)
			}
			return cfg, nil
		}
		return nil, err
	}

	return cfg, nil
}

// syncConfigToDatabase syncs servers from config file to database
func syncConfigToDatabase(cfg *config.AppConfig, db *database.DB) error {
	for _, serverDef := range cfg.Servers {
		// Check if server already exists by name
		servers, err := db.GetAllServers()
		if err != nil {
			return err
		}

		exists := false
		var existingServer *models.Server
		for _, s := range servers {
			if s.Name == serverDef.Name {
				exists = true
				existingServer = s
				break
			}
		}

		if !exists {
			// Create new server
			server := &models.Server{
				Name:           serverDef.Name,
				Hostname:       serverDef.Hostname,
				IPAddress:      serverDef.IPAddress,
				Port:           serverDef.Port,
				OS:             serverDef.OS,
				MonitoringMode: serverDef.MonitoringMode,
				SSHUser:        serverDef.SSHUser,
				SSHKeyPath:     serverDef.SSHKeyPath,
				AgentToken:     serverDef.AgentToken,
				Enabled:        serverDef.Enabled,
				NotifyTelegram: serverDef.NotifyTelegram,
			}

			if err := db.CreateServer(server); err != nil {
				log.Printf("Failed to create server %s: %v", serverDef.Name, err)
				continue
			}

			log.Printf("Created server: %s", serverDef.Name)

			// Create services for this server
			for _, serviceDef := range serverDef.Services {
				service := &models.Service{
					ServerID:    server.ID,
					Name:        serviceDef.Name,
					DisplayName: serviceDef.DisplayName,
					Description: serviceDef.Description,
					Enabled:     serviceDef.Enabled,
				}

				if err := db.CreateService(service); err != nil {
					log.Printf("Failed to create service %s: %v", serviceDef.Name, err)
					continue
				}

				log.Printf("Created service: %s for server %s", serviceDef.Name, serverDef.Name)
			}
		} else if existingServer != nil {
			// Sync services for existing server
			existingServices, _ := db.GetServicesByServer(existingServer.ID)
			existingServiceNames := make(map[string]bool)
			for _, s := range existingServices {
				existingServiceNames[s.Name] = true
			}

			for _, serviceDef := range serverDef.Services {
				if !existingServiceNames[serviceDef.Name] {
					service := &models.Service{
						ServerID:    existingServer.ID,
						Name:        serviceDef.Name,
						DisplayName: serviceDef.DisplayName,
						Description: serviceDef.Description,
						Enabled:     serviceDef.Enabled,
					}

					if err := db.CreateService(service); err != nil {
						log.Printf("Failed to create service %s: %v", serviceDef.Name, err)
						continue
					}

					log.Printf("Created service: %s for server %s", serviceDef.Name, serverDef.Name)
				}
			}
		}
	}

	return nil
}
