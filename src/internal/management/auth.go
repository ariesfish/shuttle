package management

import (
	"context"
	"net/http"
	"strings"
)

type AuthConfig struct {
	Enabled bool
	Token   string
	Tokens  map[string]Actor
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
		token, ok := bearerToken(r.Header.Get("Authorization"))
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
		}
		actor, ok := config.ActorForToken(token)
		if !ok {
			writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
			return
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

func (config AuthConfig) ActorForToken(token string) (Actor, bool) {
	token = strings.TrimSpace(token)
	if token == "" {
		return Actor{}, false
	}
	if actor, ok := config.Tokens[token]; ok {
		return normalizedActor(actor), true
	}
	if config.Token != "" && token == config.Token {
		return Actor{Name: "api-token", Role: "admin"}, true
	}
	return Actor{}, false
}

func bearerToken(header string) (string, bool) {
	prefix := "Bearer "
	if !strings.HasPrefix(header, prefix) {
		return "", false
	}
	token := strings.TrimSpace(strings.TrimPrefix(header, prefix))
	return token, token != ""
}

func normalizedActor(actor Actor) Actor {
	actor.Name = strings.TrimSpace(actor.Name)
	actor.ProjectID = strings.TrimSpace(actor.ProjectID)
	actor.Role = strings.TrimSpace(actor.Role)
	if actor.Name == "" {
		actor.Name = "api-token"
	}
	if actor.Role == "" {
		actor.Role = "viewer"
	}
	return actor
}

func headerOrDefault(r *http.Request, name string, fallback string) string {
	value := strings.TrimSpace(r.Header.Get(name))
	if value == "" {
		return fallback
	}
	return value
}
