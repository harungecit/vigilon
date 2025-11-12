package api

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/harungecit/vigilon/internal/auth"
	"github.com/harungecit/vigilon/internal/database"
	"github.com/harungecit/vigilon/internal/models"
	"github.com/harungecit/vigilon/internal/sse"
	"github.com/harungecit/vigilon/internal/telegram"
)

// API handles HTTP requests
type API struct {
	db             *database.DB
	router         *mux.Router
	templates      *template.Template
	telegram       *telegram.Notifier
	authMiddleware *auth.Middleware
	sseManager     *sse.Manager
}

// New creates a new API instance
func New(db *database.DB, telegramNotifier *telegram.Notifier) *API {
	api := &API{
		db:             db,
		router:         mux.NewRouter(),
		telegram:       telegramNotifier,
		authMiddleware: auth.NewMiddleware(db),
		sseManager:     sse.NewManager(),
	}

	// Start SSE manager
	go api.sseManager.Start(context.Background())
	
	// Setup SSE broadcaster
	api.sseManager.SetBroadcaster(api.sseBroadcaster)

	// Load templates
	api.loadTemplates()

	// Setup routes
	api.setupRoutes()

	return api
}

// loadTemplates loads HTML templates
func (a *API) loadTemplates() {
	var err error
	a.templates, err = template.ParseGlob("web/templates/*.html")
	if err != nil {
		log.Printf("Warning: Failed to load templates: %v", err)
	}
}

// setupRoutes sets up all HTTP routes
func (a *API) setupRoutes() {
	// Public routes (no auth required)
	a.router.HandleFunc("/login", a.handleLoginPage).Methods("GET")
	a.router.HandleFunc("/api/auth/login", a.handleLogin).Methods("POST")
	a.router.HandleFunc("/api/auth/logout", a.handleLogout).Methods("POST")

	// One-line installer (no auth)
	a.router.HandleFunc("/install.sh", a.handleInstallScript).Methods("GET")

	// Agent endpoints (no session auth, uses token)
	a.router.HandleFunc("/api/agent/report", a.handleAgentReport).Methods("POST")
	a.router.HandleFunc("/api/agent/install-script", a.handleAgentInstallScript).Methods("POST")
	a.router.HandleFunc("/api/agent/services", a.handleAgentServices).Methods("GET")

	// SSE endpoints (protected with auth)
	a.router.Handle("/api/sse/dashboard", a.authMiddleware.RequireAuth(http.HandlerFunc(a.handleSSEDashboard))).Methods("GET")
	a.router.Handle("/api/sse/servers", a.authMiddleware.RequireAuth(http.HandlerFunc(a.handleSSEServers))).Methods("GET")
	a.router.Handle("/api/sse/server/{id}", a.authMiddleware.RequireAuth(http.HandlerFunc(a.handleSSEServerDetail))).Methods("GET")
	a.router.Handle("/api/sse/service/{id}/history", a.authMiddleware.RequireAuth(http.HandlerFunc(a.handleSSEServiceHistory))).Methods("GET")

	// Protected Web UI routes
	a.router.Handle("/", a.authMiddleware.RequireAuth(http.HandlerFunc(a.handleIndex))).Methods("GET")
	a.router.Handle("/servers", a.authMiddleware.RequireAuth(
		a.authMiddleware.RequirePermission("servers.view")(http.HandlerFunc(a.handleServersPage)))).Methods("GET")
	a.router.Handle("/server/{id}", a.authMiddleware.RequireAuth(
		a.authMiddleware.RequirePermission("servers.view")(http.HandlerFunc(a.handleServerDetailPage)))).Methods("GET")
	a.router.Handle("/alerts", a.authMiddleware.RequireAuth(
		a.authMiddleware.RequirePermission("alerts.view")(http.HandlerFunc(a.handleAlertsPage)))).Methods("GET")
	a.router.Handle("/alerts/archived", a.authMiddleware.RequireAuth(
		a.authMiddleware.RequirePermission("alerts.view")(http.HandlerFunc(a.handleArchivedAlertsPage)))).Methods("GET")
	a.router.Handle("/users", a.authMiddleware.RequireAuth(
		a.authMiddleware.RequirePermission("users.view")(http.HandlerFunc(a.handleUsersPage)))).Methods("GET")

	// Protected API routes - Servers
	a.router.Handle("/api/servers", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("servers.view")(http.HandlerFunc(a.handleGetServers)))).Methods("GET")
	a.router.Handle("/api/servers", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("servers.create")(http.HandlerFunc(a.handleCreateServer)))).Methods("POST")
	a.router.Handle("/api/servers/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("servers.view")(http.HandlerFunc(a.handleGetServer)))).Methods("GET")
	a.router.Handle("/api/servers/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("servers.edit")(http.HandlerFunc(a.handleUpdateServer)))).Methods("PUT")
	a.router.Handle("/api/servers/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("servers.delete")(http.HandlerFunc(a.handleDeleteServer)))).Methods("DELETE")
	a.router.Handle("/api/servers/{id}/disconnect", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("servers.edit")(http.HandlerFunc(a.handleDisconnectServer)))).Methods("POST")

	// Protected API routes - Services
	a.router.Handle("/api/servers/{id}/services", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("services.view")(http.HandlerFunc(a.handleGetServices)))).Methods("GET")
	a.router.Handle("/api/services", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("services.create")(http.HandlerFunc(a.handleCreateService)))).Methods("POST")
	a.router.Handle("/api/services/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("services.edit")(http.HandlerFunc(a.handleUpdateService)))).Methods("PUT")
	a.router.Handle("/api/services/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("services.delete")(http.HandlerFunc(a.handleDeleteService)))).Methods("DELETE")

	// Protected API routes - Service Checks
	a.router.Handle("/api/services/{id}/checks", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("services.view")(http.HandlerFunc(a.handleGetServiceChecks)))).Methods("GET")
	a.router.Handle("/api/services/{id}/status", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("services.view")(http.HandlerFunc(a.handleGetServiceStatus)))).Methods("GET")

	// Protected API routes - Alerts
	a.router.Handle("/api/alerts", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("alerts.view")(http.HandlerFunc(a.handleGetAlerts)))).Methods("GET")
	a.router.Handle("/api/alerts/archived", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("alerts.view")(http.HandlerFunc(a.handleGetArchivedAlerts)))).Methods("GET")
	a.router.Handle("/api/alerts/{id}/acknowledge", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("alerts.acknowledge")(http.HandlerFunc(a.handleAcknowledgeAlert)))).Methods("POST")
	a.router.Handle("/api/alerts/{id}/archive", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("alerts.archive")(http.HandlerFunc(a.handleArchiveAlert)))).Methods("POST")
	a.router.Handle("/api/alerts/{id}/unarchive", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("alerts.archive")(http.HandlerFunc(a.handleUnarchiveAlert)))).Methods("POST")
	a.router.Handle("/api/alerts/archive-all", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("alerts.archive")(http.HandlerFunc(a.handleArchiveAllAlerts)))).Methods("POST")

	// Protected API routes - Users
	a.router.Handle("/api/users", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("users.view")(http.HandlerFunc(a.handleGetUsers)))).Methods("GET")
	a.router.Handle("/api/users", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("users.create")(http.HandlerFunc(a.handleCreateUser)))).Methods("POST")
	// /api/users/me must come BEFORE /api/users/{id} to avoid route collision
	a.router.Handle("/api/users/me", a.authMiddleware.RequireAuthAPI(http.HandlerFunc(a.handleGetCurrentUser))).Methods("GET")
	a.router.Handle("/api/users/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("users.view")(http.HandlerFunc(a.handleGetUser)))).Methods("GET")
	a.router.Handle("/api/users/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("users.edit")(http.HandlerFunc(a.handleUpdateUser)))).Methods("PUT")
	a.router.Handle("/api/users/{id}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("users.delete")(http.HandlerFunc(a.handleDeleteUser)))).Methods("DELETE")
	a.router.Handle("/api/users/{id}/password", a.authMiddleware.RequireAuthAPI(http.HandlerFunc(a.handleChangePassword))).Methods("PUT", "POST")

	// Protected API routes - Roles
	a.router.Handle("/api/roles", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("roles.view")(http.HandlerFunc(a.handleGetRoles)))).Methods("GET")
	a.router.Handle("/api/roles", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("roles.edit")(http.HandlerFunc(a.handleCreateRole)))).Methods("POST")
	a.router.Handle("/api/roles/{id:[0-9]+}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("roles.view")(http.HandlerFunc(a.handleGetRole)))).Methods("GET")
	a.router.Handle("/api/roles/{id:[0-9]+}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("roles.edit")(http.HandlerFunc(a.handleUpdateRole)))).Methods("PUT")
	a.router.Handle("/api/roles/{id:[0-9]+}", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("roles.edit")(http.HandlerFunc(a.handleDeleteRole)))).Methods("DELETE")
	a.router.Handle("/api/roles/{id:[0-9]+}/permissions", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("roles.edit")(http.HandlerFunc(a.handleUpdateRolePermissions)))).Methods("PUT")
	a.router.Handle("/api/permissions", a.authMiddleware.RequireAuthAPI(
		a.authMiddleware.RequirePermissionAPI("roles.view")(http.HandlerFunc(a.handleGetPermissions)))).Methods("GET")

	// Static files
	a.router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
}

// ServeHTTP implements http.Handler
func (a *API) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.router.ServeHTTP(w, r)
}

// Web UI Handlers

func (a *API) handleIndex(w http.ResponseWriter, r *http.Request) {
	// Get user from context
	user := auth.GetUserFromContext(r.Context())
	
	servers, err := a.db.GetAllServers()
	if err != nil {
		http.Error(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	// Get service status for each server
	type ServerWithServices struct {
		Server   *models.Server
		Services []*models.Service
		Statuses map[int]*models.ServiceCheck
	}

	serverData := make([]*ServerWithServices, 0)
	for _, server := range servers {
		services, _ := a.db.GetServicesByServer(server.ID)
		statuses := make(map[int]*models.ServiceCheck)

		for _, service := range services {
			if check, err := a.db.GetLatestServiceCheck(service.ID); err == nil {
				statuses[service.ID] = check
			}
		}

		serverData = append(serverData, &ServerWithServices{
			Server:   server,
			Services: services,
			Statuses: statuses,
		})
	}

	// Check for error message
	errorMsg := ""
	if r.URL.Query().Get("error") == "forbidden" {
		errorMsg = "You don't have permission to access that page."
	}

	data := map[string]interface{}{
		"Title":   "Vigilon - Service Monitor",
		"Servers": serverData,
		"Error":   errorMsg,
		"User":    user,
	}

	if err := a.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *API) handleServersPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	
	servers, err := a.db.GetAllServers()
	if err != nil {
		http.Error(w, "Failed to get servers", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":   "Servers - Vigilon",
		"Servers": servers,
		"User":    user,
	}

	if err := a.templates.ExecuteTemplate(w, "servers.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *API) handleServerDetailPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	server, err := a.db.GetServer(id)
	if err != nil {
		http.Error(w, "Server not found", http.StatusNotFound)
		return
	}

	services, _ := a.db.GetServicesByServer(id)

	data := map[string]interface{}{
		"Title":    server.Name + " - Vigilon",
		"Server":   server,
		"Services": services,
		"User":     user,
	}

	if err := a.templates.ExecuteTemplate(w, "server_detail.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *API) handleAlertsPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	alerts, err := a.db.GetRecentAlerts(50)
	if err != nil {
		http.Error(w, "Failed to get alerts", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":  "Alerts - Vigilon",
		"Alerts": alerts,
		"User":   user,
	}

	if err := a.templates.ExecuteTemplate(w, "alerts.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *API) handleArchivedAlertsPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	alerts, err := a.db.GetArchivedAlerts(100, 0)
	if err != nil {
		http.Error(w, "Failed to get archived alerts", http.StatusInternalServerError)
		return
	}

	data := map[string]interface{}{
		"Title":  "Archived Alerts - Vigilon",
		"Alerts": alerts,
		"User":   user,
	}

	if err := a.templates.ExecuteTemplate(w, "archived_alerts.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// API Handlers - Servers

func (a *API) handleGetServers(w http.ResponseWriter, r *http.Request) {
	servers, err := a.db.GetAllServers()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, servers)
}

func (a *API) handleGetServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	server, err := a.db.GetServer(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Server not found"})
		return
	}
	respondJSON(w, http.StatusOK, server)
}

func (a *API) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	var server models.Server
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := a.db.CreateServer(&server); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, server)
}

func (a *API) handleUpdateServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var server models.Server
	if err := json.NewDecoder(r.Body).Decode(&server); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	server.ID = id
	if err := a.db.UpdateServer(&server); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, server)
}

func (a *API) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if err := a.db.DeleteServer(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Server deleted"})
}

func (a *API) handleDisconnectServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	// Update server connection status to disconnected
	if err := a.db.UpdateServerConnectionStatus(id, models.ConnectionDisconnected); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Server disconnected"})
}

// API Handlers - Services

func (a *API) handleGetServices(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID, _ := strconv.Atoi(vars["id"])

	services, err := a.db.GetServicesByServer(serverID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, services)
}

func (a *API) handleCreateService(w http.ResponseWriter, r *http.Request) {
	var service models.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := a.db.CreateService(&service); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, service)
}

func (a *API) handleUpdateService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var service models.Service
	if err := json.NewDecoder(r.Body).Decode(&service); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	service.ID = id
	if err := a.db.UpdateService(&service); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, service)
}

func (a *API) handleDeleteService(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if err := a.db.DeleteService(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Service deleted"})
}

// API Handlers - Service Checks

func (a *API) handleGetServiceChecks(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, _ := strconv.Atoi(vars["id"])

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	checks, err := a.db.GetServiceCheckHistory(serviceID, limit)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, checks)
}

func (a *API) handleGetServiceStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serviceID, _ := strconv.Atoi(vars["id"])

	check, err := a.db.GetLatestServiceCheck(serviceID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "No status available"})
		return
	}
	respondJSON(w, http.StatusOK, check)
}

// API Handlers - Alerts

func (a *API) handleGetAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	alerts, err := a.db.GetRecentAlertsWithOffset(limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, alerts)
}

func (a *API) handleAcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if err := a.db.AcknowledgeAlert(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Alert acknowledged"})
}

func (a *API) handleArchiveAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if err := a.db.ArchiveAlert(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Alert archived"})
}

func (a *API) handleArchiveAllAlerts(w http.ResponseWriter, r *http.Request) {
	if err := a.db.ArchiveAllAlerts(); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "All alerts archived"})
}

func (a *API) handleGetArchivedAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		limit, _ = strconv.Atoi(l)
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		offset, _ = strconv.Atoi(o)
	}

	alerts, err := a.db.GetArchivedAlerts(limit, offset)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, alerts)
}

func (a *API) handleUnarchiveAlert(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	if err := a.db.UnarchiveAlert(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Alert unarchived"})
}

// API Handlers - Agent

type AgentReport struct {
	Token    string               `json:"token"`
	Services []AgentServiceReport `json:"services"`
}

type AgentServiceReport struct {
	Name         string               `json:"name"`
	Status       models.ServiceStatus `json:"status"`
	ErrorMessage string               `json:"error_message,omitempty"`
	PID          int                  `json:"pid,omitempty"`
	Memory       int64                `json:"memory_kb,omitempty"`
	CPU          float64              `json:"cpu_percent,omitempty"`
	Uptime       int64                `json:"uptime_seconds,omitempty"`
}

func (a *API) handleAgentReport(w http.ResponseWriter, r *http.Request) {
	var report AgentReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Find server by agent token
	servers, err := a.db.GetAllServers()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var server *models.Server
	for _, s := range servers {
		if s.AgentToken == report.Token {
			server = s
			break
		}
	}

	if server == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid token"})
		return
	}

	// Process each service report
	for _, svcReport := range report.Services {
		// Find or create service
		services, _ := a.db.GetServicesByServer(server.ID)
		var service *models.Service
		for _, s := range services {
			if s.Name == svcReport.Name {
				service = s
				break
			}
		}

		if service == nil {
			// Auto-create service
			service = &models.Service{
				ServerID:    server.ID,
				Name:        svcReport.Name,
				DisplayName: svcReport.Name,
				Enabled:     true,
			}
			if err := a.db.CreateService(service); err != nil {
				log.Printf("Failed to create service: %v", err)
				continue
			}
		}

		// Create service check
		check := &models.ServiceCheck{
			ServiceID:    service.ID,
			Status:       svcReport.Status,
			ErrorMessage: svcReport.ErrorMessage,
			PID:          svcReport.PID,
			Memory:       svcReport.Memory,
			CPU:          svcReport.CPU,
			Uptime:       svcReport.Uptime,
		}

		if err := a.db.CreateServiceCheck(check); err != nil {
			log.Printf("Failed to save check: %v", err)
		}
	}

	// Update server last seen
	a.db.UpdateServerLastSeen(server.ID)

	respondJSON(w, http.StatusOK, map[string]string{"message": "Report received"})
}

// Helper functions

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// handleAgentInstallScript generates an installation script for the agent
func (a *API) handleAgentInstallScript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ServerURL string `json:"server_url"`
		Token     string `json:"token"`
		OS        string `json:"os"`
		Arch      string `json:"arch"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	var script string

	switch req.OS {
	case "linux":
		script = generateLinuxInstallScript(req.ServerURL, req.Token, req.Arch)
	case "windows":
		script = generateWindowsInstallScript(req.ServerURL, req.Token)
	default:
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Unsupported OS"})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"script": script})
}

func generateLinuxInstallScript(serverURL, token, arch string) string {
	if arch == "" {
		arch = "amd64"
	}

	return fmt.Sprintf(`#!/bin/bash
set -e

echo "Installing Vigilon Agent..."

# Download agent binary
AGENT_URL="%s/static/bin/vigilon-agent-linux-%s"
curl -fsSL "$AGENT_URL" -o /tmp/vigilon-agent || wget -q "$AGENT_URL" -O /tmp/vigilon-agent

# Install binary
sudo install -m 755 /tmp/vigilon-agent /usr/local/bin/vigilon-agent
rm /tmp/vigilon-agent

# Create config directory
sudo mkdir -p /etc/vigilon-agent

# Create configuration
sudo tee /etc/vigilon-agent/config.yaml > /dev/null <<EOF
server_url: %s
token: %s
check_interval: 30s
services: []
EOF

# Create systemd service
sudo tee /etc/systemd/system/vigilon-agent.service > /dev/null <<'SVCEOF'
[Unit]
Description=Vigilon Agent - Service Monitor Agent
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/vigilon-agent -config /etc/vigilon-agent/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SVCEOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable vigilon-agent
sudo systemctl start vigilon-agent

echo "Vigilon Agent installed successfully!"
echo "Check status: sudo systemctl status vigilon-agent"
echo ""
echo "To add services to monitor, edit /etc/vigilon-agent/config.yaml"
echo "Example:"
echo "  services:"
echo "    - nginx.service"
echo "    - postgresql.service"
echo ""
echo "After editing, restart: sudo systemctl restart vigilon-agent"
`, serverURL, arch, serverURL, token)
}

func generateWindowsInstallScript(serverURL, token string) string {
	return fmt.Sprintf(`# Vigilon Agent Installation Script for Windows
# Run this in PowerShell as Administrator

$ErrorActionPreference = "Stop"

Write-Host "Installing Vigilon Agent..." -ForegroundColor Green

# Download agent
$AgentURL = "%s/static/bin/vigilon-agent-windows-amd64.exe"
$AgentPath = "C:\Program Files\VigilonAgent\vigilon-agent.exe"
$ConfigDir = "C:\ProgramData\vigilon-agent"

# Create directories
New-Item -ItemType Directory -Force -Path "C:\Program Files\VigilonAgent" | Out-Null
New-Item -ItemType Directory -Force -Path $ConfigDir | Out-Null

# Download binary
Write-Host "Downloading agent..."
Invoke-WebRequest -Uri $AgentURL -OutFile $AgentPath

# Create configuration
$Config = @"
server_url: %s
token: %s
check_interval: 30s
services: []
"@

$Config | Out-File -FilePath "$ConfigDir\config.yaml" -Encoding UTF8

# Install as Windows Service using NSSM or sc.exe
Write-Host "Installing Windows Service..."
sc.exe create VigilonAgent binPath= "$AgentPath -config $ConfigDir\config.yaml" start= auto
sc.exe description VigilonAgent "Vigilon Service Monitor Agent"
sc.exe start VigilonAgent

Write-Host "Vigilon Agent installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "To add services to monitor, edit: $ConfigDir\config.yaml"
Write-Host "Example:"
Write-Host "  services:"
Write-Host "    - W3SVC"
Write-Host "    - MSSQLSERVER"
Write-Host ""
Write-Host "After editing, restart: Restart-Service VigilonAgent"
`, serverURL, serverURL, token)
}

// handleAgentServices returns the list of services for an agent to monitor
func (a *API) handleAgentServices(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "token required"})
		return
	}

	// Find server by token
	servers, err := a.db.GetAllServers()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	var server *models.Server
	for _, s := range servers {
		if s.AgentToken == token {
			server = s
			break
		}
	}

	if server == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid token"})
		return
	}

	// Get enabled services for this server
	allServices, err := a.db.GetServicesByServer(server.ID)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Filter only enabled services
	var enabledServices []*models.Service
	for _, svc := range allServices {
		if svc.Enabled {
			enabledServices = append(enabledServices, svc)
		}
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"server_id": server.ID,
		"services":  enabledServices,
	})
}

// handleInstallScript serves the one-line installer script
func (a *API) handleInstallScript(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Error: token parameter required\n")
		fmt.Fprintf(w, "Usage: curl -fsSL http://your-server:8090/install.sh?token=YOUR_TOKEN | sudo bash\n")
		return
	}

	serverURL := fmt.Sprintf("%s://%s", "http", r.Host)

	script := fmt.Sprintf(`#!/bin/bash
set -e

# Vigilon Agent One-Line Installer
# Generated at: %s

echo "=================================="
echo "  Vigilon Agent Installer"
echo "=================================="
echo ""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and Architecture
OS="unknown"
ARCH=$(uname -m)

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    echo -e "${RED}Error: macOS is not supported yet${NC}"
    exit 1
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
    echo -e "${RED}Error: Windows detected. Please use PowerShell installer.${NC}"
    exit 1
fi

# Map architecture
case "$ARCH" in
    x86_64|amd64)
        AGENT_ARCH="amd64"
        ;;
    aarch64|arm64)
        AGENT_ARCH="arm64"
        ;;
    *)
        echo -e "${RED}Error: Unsupported architecture: $ARCH${NC}"
        exit 1
        ;;
esac

echo -e "${GREEN}Detected:${NC} $OS-$AGENT_ARCH"
echo ""

# Check root/sudo
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Error: This script must be run as root or with sudo${NC}"
    exit 1
fi

# Download URL
AGENT_URL="%s/static/bin/vigilon-agent-$OS-$AGENT_ARCH"
TOKEN="%s"

echo -e "${YELLOW}[1/5]${NC} Downloading agent binary..."
if command -v curl &> /dev/null; then
    curl -fsSL "$AGENT_URL" -o /tmp/vigilon-agent
elif command -v wget &> /dev/null; then
    wget -q "$AGENT_URL" -O /tmp/vigilon-agent
else
    echo -e "${RED}Error: Neither curl nor wget found. Please install one.${NC}"
    exit 1
fi

if [ ! -f /tmp/vigilon-agent ]; then
    echo -e "${RED}Error: Failed to download agent binary${NC}"
    exit 1
fi

echo -e "${GREEN}✓${NC} Downloaded successfully"
echo ""

echo -e "${YELLOW}[2/5]${NC} Installing agent binary..."
install -m 755 /tmp/vigilon-agent /usr/local/bin/vigilon-agent
rm /tmp/vigilon-agent
echo -e "${GREEN}✓${NC} Installed to /usr/local/bin/vigilon-agent"
echo ""

echo -e "${YELLOW}[3/5]${NC} Creating configuration..."
mkdir -p /etc/vigilon-agent

cat > /etc/vigilon-agent/config.yaml <<EOF
server_url: %s
token: $TOKEN
check_interval: 30s
services: []
EOF

echo -e "${GREEN}✓${NC} Configuration created at /etc/vigilon-agent/config.yaml"
echo ""

echo -e "${YELLOW}[4/5]${NC} Installing systemd service..."
cat > /etc/systemd/system/vigilon-agent.service <<'SVCEOF'
[Unit]
Description=Vigilon Agent - Service Monitor
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/vigilon-agent -config /etc/vigilon-agent/config.yaml
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
SVCEOF

systemctl daemon-reload
echo -e "${GREEN}✓${NC} Service installed"
echo ""

echo -e "${YELLOW}[5/5]${NC} Starting agent..."
systemctl enable vigilon-agent
systemctl start vigilon-agent
sleep 2

if systemctl is-active --quiet vigilon-agent; then
    echo -e "${GREEN}✓${NC} Agent started successfully!"
    echo ""
    echo "=================================="
    echo -e "${GREEN}  Installation Complete!${NC}"
    echo "=================================="
    echo ""
    echo "Agent is now running and will:"
    echo "  • Automatically fetch service list from Vigilon server"
    echo "  • Monitor services you add from the web panel"
    echo "  • Report status every 30 seconds"
    echo ""
    echo "Useful commands:"
    echo "  • Check status: sudo systemctl status vigilon-agent"
    echo "  • View logs:    sudo journalctl -u vigilon-agent -f"
    echo "  • Restart:      sudo systemctl restart vigilon-agent"
    echo ""
    echo "Add services to monitor from the web panel at:"
    echo "  %s"
else
    echo -e "${RED}✗${NC} Failed to start agent"
    echo "Check logs with: sudo journalctl -u vigilon-agent -xe"
    exit 1
fi
`, time.Now().Format(time.RFC3339), serverURL, token, serverURL, serverURL)

	w.Header().Set("Content-Type", "text/x-shellscript")
	w.Header().Set("Content-Disposition", "attachment; filename=install.sh")
	fmt.Fprint(w, script)
}

// Auth Handlers

func (a *API) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	// If already logged in, redirect to dashboard
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if session, err := a.db.GetSessionByToken(cookie.Value); err == nil {
			if user, err := a.db.GetUser(session.UserID); err == nil && user.Enabled {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
		}
	}

	data := map[string]interface{}{
		"Title": "Login - Vigilon",
	}

	if err := a.templates.ExecuteTemplate(w, "login.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *API) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Get user
	user, err := a.db.GetUserByUsername(req.Username)
	if err != nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	// Check if user is enabled
	if !user.Enabled {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Account is disabled"})
		return
	}

	// Verify password
	if !auth.CheckPassword(req.Password, user.PasswordHash) {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Invalid credentials"})
		return
	}

	// Create session
	token, err := auth.GenerateToken()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create session"})
		return
	}

	session := &models.Session{
		ID:        auth.GenerateSessionID(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		IPAddress: r.RemoteAddr,
		UserAgent: r.UserAgent(),
	}

	if err := a.db.CreateSession(session); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create session"})
		return
	}

	// Update last login
	a.db.UpdateUserLastLogin(user.ID)

	// Set cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "session_token",
		Value:    token,
		Path:     "/",
		MaxAge:   86400, // 24 hours
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
	})

	// Remove password hash from response
	user.PasswordHash = ""

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"user":    user,
		"token":   token,
	})
}

func (a *API) handleLogout(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("session_token")
	if err == nil {
		if session, err := a.db.GetSessionByToken(cookie.Value); err == nil {
			a.db.DeleteSession(session.ID)
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:   "session_token",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})

	respondJSON(w, http.StatusOK, map[string]string{"message": "Logged out successfully"})
}

// User Management Handlers

func (a *API) handleUsersPage(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUserFromContext(r.Context())
	data := map[string]interface{}{
		"Title": "Users - Vigilon",
		"User":  user,
	}

	if err := a.templates.ExecuteTemplate(w, "users.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *API) handleGetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := a.db.GetAllUsers()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	// Remove password hashes
	for _, user := range users {
		user.PasswordHash = ""
	}

	respondJSON(w, http.StatusOK, users)
}

func (a *API) handleGetUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	user, err := a.db.GetUser(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	user.PasswordHash = ""
	respondJSON(w, http.StatusOK, user)
}

func (a *API) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		RoleID   int    `json:"role_id"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	// Validate
	if req.Username == "" || req.Email == "" || req.Password == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Username, email and password are required"})
		return
	}

	// Hash password
	passwordHash, err := auth.HashPassword(req.Password)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
		return
	}

	user := &models.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: passwordHash,
		RoleID:       req.RoleID,
		Enabled:      req.Enabled,
	}

	if err := a.db.CreateUser(user); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	user.PasswordHash = ""
	respondJSON(w, http.StatusCreated, user)
}

func (a *API) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		RoleID   int    `json:"role_id"`
		Enabled  bool   `json:"enabled"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	user, err := a.db.GetUser(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	// Check if trying to modify super admin
	if user.Role != nil && user.Role.IsSuperAdmin {
		currentUser := auth.GetUserFromContext(r.Context())
		if currentUser.ID != user.ID {
			respondJSON(w, http.StatusForbidden, map[string]string{"error": "Cannot modify super admin"})
			return
		}
	}

	user.Username = req.Username
	user.Email = req.Email
	user.RoleID = req.RoleID
	user.Enabled = req.Enabled

	if err := a.db.UpdateUser(user); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	user.PasswordHash = ""
	respondJSON(w, http.StatusOK, user)
}

func (a *API) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser.ID == id {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "Cannot delete yourself"})
		return
	}

	if err := a.db.DeleteUser(id); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "User deleted successfully"})
}

func (a *API) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	currentUser := auth.GetUserFromContext(r.Context())
	if currentUser == nil {
		respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Not authenticated"})
		return
	}

	user, err := a.db.GetUser(currentUser.ID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	user.PasswordHash = ""
	respondJSON(w, http.StatusOK, user)
}

func (a *API) handleChangePassword(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request"})
		return
	}

	currentUser := auth.GetUserFromContext(r.Context())

	// Users can only change their own password unless they're admin
	if currentUser.ID != id && !currentUser.Role.IsSuperAdmin {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden"})
		return
	}

	user, err := a.db.GetUser(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	// Verify current password if changing own password
	if currentUser.ID == id {
		if req.CurrentPassword == "" {
			respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Current password is required"})
			return
		}
		if !auth.CheckPassword(req.CurrentPassword, user.PasswordHash) {
			respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Current password is incorrect"})
			return
		}
	}

	// Hash new password
	passwordHash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to hash password"})
		return
	}

	if err := a.db.UpdateUserPassword(id, passwordHash); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Password updated successfully"})
}

func (a *API) handleGetRoles(w http.ResponseWriter, r *http.Request) {
	roles, err := a.db.GetAllRoles()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, roles)
}

func (a *API) handleGetRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid role ID"})
		return
	}

	role, err := a.db.GetRole(id)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Role not found"})
		return
	}
	respondJSON(w, http.StatusOK, role)
}

func (a *API) handleGetPermissions(w http.ResponseWriter, r *http.Request) {
	permissions, err := a.db.GetAllPermissions()
	if err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	respondJSON(w, http.StatusOK, permissions)
}

func (a *API) handleUpdateRolePermissions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid role ID"})
		return
	}

	var input struct {
		PermissionIDs []int `json:"permission_ids"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	// Check if role is super admin
	role, err := a.db.GetRole(roleID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Role not found"})
		return
	}

	if role.IsSuperAdmin {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "Cannot modify super admin role permissions"})
		return
	}

	// Update permissions
	if err := a.db.UpdateRolePermissions(roleID, input.PermissionIDs); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Permissions updated successfully"})
}

func (a *API) handleCreateRole(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if input.Name == "" || input.DisplayName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name and display name are required"})
		return
	}

	role := &models.Role{
		Name:        input.Name,
		DisplayName: input.DisplayName,
		Description: input.Description,
	}

	if err := a.db.CreateRole(role); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusCreated, role)
}

func (a *API) handleUpdateRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid role ID"})
		return
	}

	// Check if role is super admin
	existingRole, err := a.db.GetRole(roleID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Role not found"})
		return
	}

	if existingRole.IsSuperAdmin {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "Cannot modify super admin role"})
		return
	}

	var input struct {
		Name        string `json:"name"`
		DisplayName string `json:"display_name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid request body"})
		return
	}

	if input.Name == "" || input.DisplayName == "" {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Name and display name are required"})
		return
	}

	role := &models.Role{
		ID:          roleID,
		Name:        input.Name,
		DisplayName: input.DisplayName,
		Description: input.Description,
	}

	if err := a.db.UpdateRole(role); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, role)
}

func (a *API) handleDeleteRole(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	roleID, err := strconv.Atoi(vars["id"])
	if err != nil {
		respondJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid role ID"})
		return
	}

	// Check if role is a system role
	role, err := a.db.GetRole(roleID)
	if err != nil {
		respondJSON(w, http.StatusNotFound, map[string]string{"error": "Role not found"})
		return
	}

	if role.IsSystem {
		respondJSON(w, http.StatusForbidden, map[string]string{"error": "Cannot delete system role"})
		return
	}

	// Check if any users have this role
	users, err := a.db.GetUsersByRole(roleID)
	if err == nil && len(users) > 0 {
		respondJSON(w, http.StatusConflict, map[string]string{"error": fmt.Sprintf("Cannot delete role: %d users are assigned to this role", len(users))})
		return
	}

	if err := a.db.DeleteRole(roleID); err != nil {
		respondJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Role deleted successfully"})
}

// SSE Handlers

func (a *API) handleSSEDashboard(w http.ResponseWriter, r *http.Request) {
	a.sseManager.ServeHTTP(w, r)
}

func (a *API) handleSSEServers(w http.ResponseWriter, r *http.Request) {
	a.sseManager.ServeHTTP(w, r)
}

func (a *API) handleSSEServerDetail(w http.ResponseWriter, r *http.Request) {
	a.sseManager.ServeHTTP(w, r)
}

func (a *API) handleSSEServiceHistory(w http.ResponseWriter, r *http.Request) {
	a.sseManager.ServeHTTP(w, r)
}

// sseBroadcaster periodically broadcasts dashboard data
func (a *API) sseBroadcaster(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Only broadcast if there are connected clients
			if a.sseManager.ClientCount() == 0 {
				continue
			}

			// Fetch latest dashboard data
			servers, err := a.db.GetAllServers()
			if err != nil {
				continue
			}

			// Build dashboard data
			type ServerStatus struct {
				ServerID      int    `json:"server_id"`
				ServerName    string `json:"server_name"`
				Enabled       bool   `json:"enabled"`
				Status        string `json:"status"`
				LastSeen      *time.Time `json:"last_seen"`
				ServiceCount  int    `json:"service_count"`
				RunningCount  int    `json:"running_count"`
				StoppedCount  int    `json:"stopped_count"`
				FailedCount   int    `json:"failed_count"`
			}

			var dashboardData []ServerStatus

			for _, server := range servers {
				services, _ := a.db.GetServicesByServer(server.ID)
				
				running, stopped, failed := 0, 0, 0
				for _, service := range services {
					if check, err := a.db.GetLatestServiceCheck(service.ID); err == nil {
						switch check.Status {
						case models.StatusRunning:
							running++
						case models.StatusStopped:
							stopped++
						case models.StatusFailed:
							failed++
						}
					}
				}

				status := "active"
				if !server.Enabled {
					status = "disabled"
				} else if server.LastSeen == nil {
					status = "never_connected"
				} else if time.Since(*server.LastSeen) > 2*time.Minute {
					status = "offline"
				}

				dashboardData = append(dashboardData, ServerStatus{
					ServerID:     server.ID,
					ServerName:   server.Name,
					Enabled:      server.Enabled,
					Status:       status,
					LastSeen:     server.LastSeen,
					ServiceCount: len(services),
					RunningCount: running,
					StoppedCount: stopped,
					FailedCount:  failed,
				})
			}

			// Broadcast to all clients
			a.sseManager.Broadcast("dashboard_update", dashboardData)

			// Also broadcast servers list update with connection status
			type ServerListItem struct {
				ServerID         int        `json:"server_id"`
				ServerName       string     `json:"server_name"`
				Enabled          bool       `json:"enabled"`
				ConnectionStatus string     `json:"connection_status"`
				LastSeen         *time.Time `json:"last_seen"`
			}

			var serversListData []ServerListItem
			for _, server := range servers {
				connStatus := "not_connected"
				if server.LastSeen != nil {
					if time.Since(*server.LastSeen) < 2*time.Minute {
						connStatus = "connected"
					} else if time.Since(*server.LastSeen) < 10*time.Minute {
						connStatus = "idle"
					} else {
						connStatus = "disconnected"
					}
				}

				serversListData = append(serversListData, ServerListItem{
					ServerID:         server.ID,
					ServerName:       server.Name,
					Enabled:          server.Enabled,
					ConnectionStatus: connStatus,
					LastSeen:         server.LastSeen,
				})
			}

			a.sseManager.Broadcast("servers_update", serversListData)

			// Broadcast per-server detail updates
			for _, server := range servers {
				type ServerDetailUpdate struct {
					ServerID int        `json:"server_id"`
					Enabled  bool       `json:"enabled"`
					LastSeen *time.Time `json:"last_seen"`
				}

				a.sseManager.Broadcast("server_detail_update", ServerDetailUpdate{
					ServerID: server.ID,
					Enabled:  server.Enabled,
					LastSeen: server.LastSeen,
				})

				// Get services for this server
				services, _ := a.db.GetServicesByServer(server.ID)
				type ServiceUpdate struct {
					ServiceID int  `json:"service_id"`
					Enabled   bool `json:"enabled"`
				}

				var serviceUpdates []ServiceUpdate
				for _, svc := range services {
					serviceUpdates = append(serviceUpdates, ServiceUpdate{
						ServiceID: svc.ID,
						Enabled:   svc.Enabled,
					})
				}

				a.sseManager.Broadcast("service_update", serviceUpdates)

				// Broadcast service history for each service
				for _, svc := range services {
					checks, err := a.db.GetServiceCheckHistory(svc.ID, 20)
					if err == nil && len(checks) > 0 {
						a.sseManager.Broadcast("history_update", checks)
					}
				}
			}
		}
	}
}
