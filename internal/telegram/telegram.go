package telegram

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/harungecit/vigilon/internal/database"
	"github.com/harungecit/vigilon/internal/models"
	tele "gopkg.in/telebot.v3"
)

// Notifier handles Telegram notifications
type Notifier struct {
	bot    *tele.Bot
	config *models.TelegramConfig
	db     *database.DB
}

// New creates a new Telegram notifier
func New(config *models.TelegramConfig, db *database.DB) (*Notifier, error) {
	if !config.Enabled || config.BotToken == "" {
		return &Notifier{config: config, db: db}, nil
	}

	pref := tele.Settings{
		Token:  config.BotToken,
		Poller: &tele.LongPoller{Timeout: 10 * time.Second},
	}

	bot, err := tele.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("failed to create telegram bot: %w", err)
	}

	notifier := &Notifier{
		bot:    bot,
		config: config,
		db:     db,
	}

	// Set up command handlers
	notifier.setupHandlers()

	return notifier, nil
}

// Start starts the Telegram bot
func (n *Notifier) Start(ctx context.Context) {
	if n.bot == nil {
		log.Println("Telegram notifications disabled")
		return
	}

	log.Println("Starting Telegram bot...")
	go n.bot.Start()

	// Wait for context cancellation
	<-ctx.Done()
	n.bot.Stop()
}

// SendAlert sends an alert message to all configured chat IDs
func (n *Notifier) SendAlert(alert *models.Alert) error {
	if n.bot == nil || !n.config.Enabled {
		return nil
	}

	for _, chatID := range n.config.ChatIDs {
		recipient := &tele.Chat{ID: parseInt64(chatID)}
		_, err := n.bot.Send(recipient, alert.Message, &tele.SendOptions{
			ParseMode: "Markdown",
		})
		if err != nil {
			log.Printf("Failed to send alert to chat %s: %v", chatID, err)
			continue
		}
	}

	return nil
}

// SendMessage sends a custom message to all configured chat IDs
func (n *Notifier) SendMessage(message string) error {
	if n.bot == nil || !n.config.Enabled {
		return nil
	}

	for _, chatID := range n.config.ChatIDs {
		recipient := &tele.Chat{ID: parseInt64(chatID)}
		_, err := n.bot.Send(recipient, message)
		if err != nil {
			log.Printf("Failed to send message to chat %s: %v", chatID, err)
			continue
		}
	}

	return nil
}

// setupHandlers sets up bot command handlers
func (n *Notifier) setupHandlers() {
	// /start command
	n.bot.Handle("/start", func(c tele.Context) error {
		return c.Send("👋 Welcome to Vigilon Bot!\n\nAvailable commands:\n" +
			"/status - Get current status of all services\n" +
			"/servers - List all servers\n" +
			"/alerts - View recent alerts\n" +
			"/help - Show help message")
	})

	// /help command
	n.bot.Handle("/help", func(c tele.Context) error {
		return c.Send("📚 Vigilon Bot Commands:\n\n" +
			"/status - Get current status of all services\n" +
			"/servers - List all monitored servers\n" +
			"/alerts - View recent alerts (unacknowledged)\n" +
			"/ack <id> - Acknowledge an alert\n" +
			"/help - Show this help message")
	})

	// /status command
	n.bot.Handle("/status", func(c tele.Context) error {
		servers, err := n.db.GetAllServers()
		if err != nil {
			return c.Send("❌ Failed to get servers")
		}

		if len(servers) == 0 {
			return c.Send("ℹ️ No servers configured")
		}

		message := "📊 *Service Status Overview*\n\n"
		for _, server := range servers {
			if !server.Enabled {
				continue
			}

			message += fmt.Sprintf("🖥 *%s* (%s)\n", server.Name, server.IPAddress)

			services, err := n.db.GetServicesByServer(server.ID)
			if err != nil {
				message += "  ❌ Failed to get services\n"
				continue
			}

			if len(services) == 0 {
				message += "  ℹ️ No services configured\n\n"
				continue
			}

			for _, service := range services {
				if !service.Enabled {
					continue
				}

				check, err := n.db.GetLatestServiceCheck(service.ID)
				if err != nil {
					message += fmt.Sprintf("  ❓ %s: Unknown\n", service.DisplayName)
					continue
				}

				statusIcon := getStatusIcon(check.Status)
				message += fmt.Sprintf("  %s %s: %s\n", statusIcon, service.DisplayName, check.Status)
			}
			message += "\n"
		}

		return c.Send(message, &tele.SendOptions{ParseMode: "Markdown"})
	})

	// /servers command
	n.bot.Handle("/servers", func(c tele.Context) error {
		servers, err := n.db.GetAllServers()
		if err != nil {
			return c.Send("❌ Failed to get servers")
		}

		if len(servers) == 0 {
			return c.Send("ℹ️ No servers configured")
		}

		message := "🖥 *Monitored Servers*\n\n"
		for _, server := range servers {
			status := "✅ Enabled"
			if !server.Enabled {
				status = "⏸ Disabled"
			}

			lastSeen := "Never"
			if server.LastSeen != nil {
				lastSeen = server.LastSeen.Format("2006-01-02 15:04:05")
			}

			message += fmt.Sprintf("*%s*\n", server.Name)
			message += fmt.Sprintf("  IP: %s\n", server.IPAddress)
			message += fmt.Sprintf("  OS: %s\n", server.OS)
			message += fmt.Sprintf("  Mode: %s\n", server.MonitoringMode)
			message += fmt.Sprintf("  Status: %s\n", status)
			message += fmt.Sprintf("  Last Seen: %s\n\n", lastSeen)
		}

		return c.Send(message, &tele.SendOptions{ParseMode: "Markdown"})
	})

	// /alerts command
	n.bot.Handle("/alerts", func(c tele.Context) error {
		alerts, err := n.db.GetRecentAlerts(10)
		if err != nil {
			return c.Send("❌ Failed to get alerts")
		}

		// Filter unacknowledged alerts
		unacked := make([]*models.Alert, 0)
		for _, alert := range alerts {
			if !alert.Acknowledged {
				unacked = append(unacked, alert)
			}
		}

		if len(unacked) == 0 {
			return c.Send("✅ No pending alerts")
		}

		message := "🚨 *Recent Alerts*\n\n"
		for _, alert := range unacked {
			message += fmt.Sprintf("*Alert #%d*\n", alert.ID)
			message += fmt.Sprintf("%s\n", alert.Message)
			message += fmt.Sprintf("Time: %s\n", alert.CreatedAt.Format("2006-01-02 15:04:05"))
			message += fmt.Sprintf("Use /ack %d to acknowledge\n\n", alert.ID)
		}

		return c.Send(message, &tele.SendOptions{ParseMode: "Markdown"})
	})
}

// getStatusIcon returns an emoji icon for a service status
func getStatusIcon(status models.ServiceStatus) string {
	switch status {
	case models.StatusRunning:
		return "✅"
	case models.StatusStopped:
		return "🔴"
	case models.StatusFailed:
		return "❌"
	case models.StatusDegraded:
		return "⚠️"
	default:
		return "❓"
	}
}

// parseInt64 parses a string to int64
func parseInt64(s string) int64 {
	var i int64
	fmt.Sscanf(s, "%d", &i)
	return i
}
