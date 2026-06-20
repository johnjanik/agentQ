package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const ClaimsContextKey = "mcp_claims"

// InsecureDefaultSecrets are placeholder secret values that shipped as config
// defaults historically. The app must refuse to boot when any auth secret is
// set to one of these — see SECURITY-REVIEW.md finding #1.
var InsecureDefaultSecrets = []string{
	"agentrq-secret-change-me",         // former AGENTRQ_AUTH_JWT_SECRET default
	"agentrq-token-key-change-me-32by", // former AGENTRQ_AUTH_WORKSPACE_TOKEN_KEY default
}

// IsInsecureDefaultSecret reports whether s matches a known shipped default
// secret. Empty values are handled separately by callers.
func IsInsecureDefaultSecret(s string) bool {
	for _, d := range InsecureDefaultSecrets {
		if s == d {
			return true
		}
	}
	return false
}

type TokenConfig struct {
	JWTSecret string `yaml:"jwt_secret"`
}

type Claims struct {
	jwt.RegisteredClaims
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
	Picture string `json:"picture,omitempty"`
}

type TokenService interface {
	CreateToken(userID, email, name, picture string) (string, error)
	CreateMCPToken(userID, workspaceID, tokenType string) (string, error)
	CreateOAuthCodeToken(userID, workspaceID string) (string, error)
	ValidateToken(tokenStr string) (*Claims, error)
}

type tokenService struct {
	secret []byte
}

func NewTokenService(cfg TokenConfig) TokenService {
	if cfg.JWTSecret == "" {
		// Critical: fallback to an empty secret is not allowed.
		// In production, the app should fail to start if JWTSecret is missing.
		panic("situational security: JWT secret is required but not provided in configuration")
	}
	if IsInsecureDefaultSecret(cfg.JWTSecret) {
		// A known placeholder default is as dangerous as an empty secret: it is
		// public knowledge, so any token could be forged. Refuse to boot.
		panic("situational security: JWT secret is set to a known insecure default; set AGENTRQ_AUTH_JWT_SECRET to a unique value (see scripts/setup.sh)")
	}
	return &tokenService{
		secret: []byte(cfg.JWTSecret),
	}
}

func (s *tokenService) CreateToken(userID, email, name, picture string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
		Email:   email,
		Name:    name,
		Picture: picture,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *tokenService) CreateMCPToken(userID, workspaceID, tokenType string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(365 * 24 * time.Hour)), // 365 days
		},
	}

	if workspaceID != "" {
		claims.Audience = jwt.ClaimStrings{workspaceID}
	}
	if tokenType != "" {
		claims.Audience = append(claims.Audience, tokenType)
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *tokenService) CreateOAuthCodeToken(userID, workspaceID string) (string, error) {
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Minute)), // 2 minutes short lived
		},
	}

	if workspaceID != "" {
		claims.Audience = jwt.ClaimStrings{workspaceID}
	}
	claims.Audience = append(claims.Audience, "authorization_code")

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

func (s *tokenService) ValidateToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method to prevent algorithm confusion attacks
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return s.secret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("invalid token")
}
