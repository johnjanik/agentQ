package app

import (
	"fmt"
	"strings"

	"github.com/agentrq/agentrq/backend/internal/service/auth"
)

// validateAuthSecrets fails closed when a required auth secret is missing or
// set to a known shipped default. Both the JWT signing secret and the
// workspace-token key are checked; the latter never passes through the auth
// TokenService, so it would otherwise be unguarded. See SECURITY-REVIEW.md #1.
func validateAuthSecrets(cfg Config) error {
	secrets := []struct {
		name  string
		env   string
		value string
	}{
		{"auth.jwtSecret", "AGENTRQ_AUTH_JWT_SECRET", cfg.Auth.JWTSecret},
		{"auth.workspaceTokenKey", "AGENTRQ_AUTH_WORKSPACE_TOKEN_KEY", cfg.Auth.WorkspaceTokenKey},
	}
	for _, s := range secrets {
		if strings.TrimSpace(s.value) == "" {
			return fmt.Errorf("%s must be set (env %s); run scripts/setup.sh to generate secrets", s.name, s.env)
		}
		if auth.IsInsecureDefaultSecret(s.value) {
			return fmt.Errorf("%s is set to a known insecure default (env %s); run scripts/setup.sh to generate a unique value", s.name, s.env)
		}
	}
	return nil
}
