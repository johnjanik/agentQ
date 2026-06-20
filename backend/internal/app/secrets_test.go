package app

import "testing"

// newConfig builds a Config with the two guarded auth secrets set to the given
// values; everything else stays zero, which is fine for validateAuthSecrets.
func newConfig(jwtSecret, workspaceTokenKey string) Config {
	var cfg Config
	cfg.Auth.JWTSecret = jwtSecret
	cfg.Auth.WorkspaceTokenKey = workspaceTokenKey
	return cfg
}

func TestValidateAuthSecrets(t *testing.T) {
	cases := []struct {
		name              string
		jwtSecret         string
		workspaceTokenKey string
		wantErr           bool
	}{
		{"both valid", "unique-jwt-secret", "unique-token-key", false},
		{"jwt empty", "", "unique-token-key", true},
		{"jwt whitespace only", "   ", "unique-token-key", true},
		{"workspace key empty", "unique-jwt-secret", "", true},
		{"jwt default", "agentrq-secret-change-me", "unique-token-key", true},
		{"workspace key default", "unique-jwt-secret", "agentrq-token-key-change-me-32by", true},
		{"both default", "agentrq-secret-change-me", "agentrq-token-key-change-me-32by", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateAuthSecrets(newConfig(tc.jwtSecret, tc.workspaceTokenKey))
			if tc.wantErr && err == nil {
				t.Errorf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
