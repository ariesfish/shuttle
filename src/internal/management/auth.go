package management

import (
	"context"
	"net/http"
	"strings"
)

type AuthConfig struct {
	Enabled bool
	Token   string
}

type Actor struct {
	Name      string
	ProjectID string
	Role      string
}

type actorContextKey struct{}

func ActorFromContext(ctx context.Context) Actor {
	actor, _ := ctx.Value(actorContextKey{}).(Actor)
	if actor.Name == "" {
		actor.Name = "anonymous"
	}
	return actor
}

func withActor(ctx context.Context, actor Actor) context.Context {
	return context.WithValue(ctx, actorContextKey{}, actor)
}

func authMiddleware(config AuthConfig, next http.Handler) http.Handler {
	if !config.Enabled {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor := Actor{
				Name:      headerOrDefault(r, "X-Actor", "dev"),
				ProjectID: r.Header.Get("X-Project-ID"),
				Role:      headerOrDefault(r, "X-Role", "admin"),
			}
			next.ServeHTTP(w, r.WithContext(withActor(r.Context(), actor)))
		})
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" {
			next.ServeHTTP(w, r)
			return
		}
		if !validBearerToken(r.Header.Get("Authorization"), config.Token) {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		actor := Actor{
			Name:      headerOrDefault(r, "X-Actor", "api-user"),
			ProjectID: r.Header.Get("X-Project-ID"),
			Role:      headerOrDefault(r, "X-Role", "viewer"),
		}
		next.ServeHTTP(w, r.WithContext(withActor(r.Context(), actor)))
	})
}

func requireRole(w http.ResponseWriter, r *http.Request, roles ...string) bool {
	actor := ActorFromContext(r.Context())
	for _, role := range roles {
		if actor.Role == role {
			return true
		}
	}
	writeError(w, http.StatusForbidden, "insufficient role")
	return false
}

func validBearerToken(header string, token string) bool {
	if token == "" {
		return false
	}
	prefix := "Bearer "
	return strings.HasPrefix(header, prefix) && strings.TrimSpace(strings.TrimPrefix(header, prefix)) == token
}

func headerOrDefault(r *http.Request, name string, fallback string) string {
	value := strings.TrimSpace(r.Header.Get(name))
	if value == "" {
		return fallback
	}
	return value
}
