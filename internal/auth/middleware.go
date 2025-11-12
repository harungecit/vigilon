package auth

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/harungecit/vigilon/internal/database"
	"github.com/harungecit/vigilon/internal/models"
)

type contextKey string

const (
	UserContextKey    contextKey = "user"
	SessionContextKey contextKey = "session"
)

// Middleware handles authentication and authorization
type Middleware struct {
	db *database.DB
}

// NewMiddleware creates a new auth middleware
func NewMiddleware(db *database.DB) *Middleware {
	return &Middleware{db: db}
}

// RequireAuth checks if user is authenticated
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for session token in cookie
		cookie, err := r.Cookie("session_token")
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate session
		session, err := m.db.GetSessionByToken(cookie.Value)
		if err != nil {
			http.SetCookie(w, &http.Cookie{
				Name:   "session_token",
				Value:  "",
				Path:   "/",
				MaxAge: -1,
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Get user
		user, err := m.db.GetUser(session.UserID)
		if err != nil || !user.Enabled {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Add user and session to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		ctx = context.WithValue(ctx, SessionContextKey, session)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermission checks if user has specific permission
func (m *Middleware) RequirePermission(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Super admin has all permissions
			if user.Role != nil && user.Role.IsSuperAdmin {
				next.ServeHTTP(w, r)
				return
			}

			// Check permission
			hasPermission, err := m.db.UserHasPermission(user.ID, permission)
			if err != nil || !hasPermission {
				// For web UI, redirect to home with error message
				http.Redirect(w, r, "/?error=forbidden", http.StatusSeeOther)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAuthAPI checks authentication for API endpoints
func (m *Middleware) RequireAuthAPI(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for session token in cookie or Authorization header
		var token string

		// Try cookie first
		cookie, err := r.Cookie("session_token")
		if err == nil {
			token = cookie.Value
		} else {
			// Try Authorization header
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				token = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		if token == "" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Validate session
		session, err := m.db.GetSessionByToken(token)
		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Get user
		user, err := m.db.GetUser(session.UserID)
		if err != nil || !user.Enabled {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Add user and session to context
		ctx := context.WithValue(r.Context(), UserContextKey, user)
		ctx = context.WithValue(ctx, SessionContextKey, session)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RequirePermissionAPI checks permission for API endpoints
func (m *Middleware) RequirePermissionAPI(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				log.Printf("[PERMISSION] User not in context for %s %s", r.Method, r.URL.Path)
				respondJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
				return
			}

			log.Printf("[PERMISSION] User: %s (ID:%d), Role: %s (SuperAdmin:%v), Required: %s, Path: %s %s",
				user.Username, user.ID, user.Role.Name, user.Role.IsSuperAdmin, permission, r.Method, r.URL.Path)

			// Super admin has all permissions
			if user.Role != nil && user.Role.IsSuperAdmin {
				log.Printf("[PERMISSION] ✓ Super admin access granted")
				next.ServeHTTP(w, r)
				return
			}

			// Check permission
			hasPermission, err := m.db.UserHasPermission(user.ID, permission)
			if err != nil {
				log.Printf("[PERMISSION] ✗ Error checking permission: %v", err)
				respondJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden: insufficient permissions"})
				return
			}
			
			if !hasPermission {
				log.Printf("[PERMISSION] ✗ User does not have permission '%s'", permission)
				respondJSON(w, http.StatusForbidden, map[string]string{"error": "Forbidden: insufficient permissions"})
				return
			}

			log.Printf("[PERMISSION] ✓ Permission granted")
			next.ServeHTTP(w, r)
		})
	}
}

// GetUserFromContext retrieves user from request context
func GetUserFromContext(ctx context.Context) *models.User {
	if user, ok := ctx.Value(UserContextKey).(*models.User); ok {
		return user
	}
	return nil
}

// GetSessionFromContext retrieves session from request context
func GetSessionFromContext(ctx context.Context) *models.Session {
	if session, ok := ctx.Value(SessionContextKey).(*models.Session); ok {
		return session
	}
	return nil
}

// Helper function to respond with JSON
func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
