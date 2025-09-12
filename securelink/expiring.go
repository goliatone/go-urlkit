// Package securelink provides secure, expiring URL generation and validation using JWT tokens.
//
// The package offers two ways to create a manager:
//
// 1. Using the Config struct (preferred for new code):
//
//	cfg := securelink.Config{
//		SigningKey:    "a-very-secure-key-of-at-least-32-bytes",
//		Expiration:    1 * time.Hour,
//		BaseURL:       "https://example.com",
//		QueryKey:      "token",
//		Routes:        map[string]string{"activate": "/activate"},
//		AsQuery:       false,
//		SigningMethod: jwt.SigningMethodHS256, // optional, defaults to HS256
//	}
//	manager, err := securelink.NewManagerFromConfig(cfg)
//
// 2. Using the Configurator interface (for backward compatibility):
//
//	manager, err := securelink.NewManager(cfg) // where cfg implements Configurator
package securelink

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	minKeyLengthHS256 = 32 // 256 bits
	minKeyLengthHS384 = 48 // 384 bits
	minKeyLengthHS512 = 64 // 512 bits
)

type manager struct {
	signingKey    string
	expiration    time.Duration
	baseURL       string
	url           *url.URL
	routes        map[string]string
	queryKey      string
	payload       map[string]any
	mx            sync.Mutex
	asQuery       bool
	signingMethod jwt.SigningMethod
}

// Configurator holds configuration options
type Configurator interface {
	SigningKey() string
	Expiration() time.Duration
	BaseURL() string
	QueryKey() string
	Routes() map[string]string
	AsQuery() bool
}

// Config is a struct-based configuration for simplified direct instantiation
type Config struct {
	SigningKey    string
	Expiration    time.Duration
	BaseURL       string
	QueryKey      string
	Routes        map[string]string
	AsQuery       bool
	SigningMethod jwt.SigningMethod
}

// Payload is the output of the link
type Payload map[string]any

// GetString decodes a key
func (p Payload) GetString(key string) (val string, err error) {
	var ok bool
	if val, ok = p[key].(string); !ok {
		return "", fmt.Errorf("error decoding key %s: not found", key)
	}
	return val, nil
}

// Manager secure link manager
type Manager interface {
	Generate(route string) (string, error)
	WithData(key string, val any) Manager
	Validate(token string) (map[string]any, error)
	GetAndValidate(fn func(string) string) (Payload, error)
	GetExpiration() time.Duration
}

// validateSigningKey validates that the signing key meets minimum length requirements for the given algorithm
func validateSigningKey(key string, method jwt.SigningMethod) error {
	keyLength := len(key)
	var minLength int
	var algName string

	switch method {
	case jwt.SigningMethodHS256:
		minLength = minKeyLengthHS256
		algName = "HS256"
	case jwt.SigningMethodHS384:
		minLength = minKeyLengthHS384
		algName = "HS384"
	case jwt.SigningMethodHS512:
		minLength = minKeyLengthHS512
		algName = "HS512"
	default:
		return fmt.Errorf("unsupported signing method: %v", method.Alg())
	}

	if keyLength < minLength {
		return fmt.Errorf("signing key too short for %s algorithm: got %d bytes, need at least %d bytes (%d bits)",
			algName, keyLength, minLength, minLength*8)
	}

	return nil
}

// NewManagerFromConfig returns a manager instance from a Config struct
func NewManagerFromConfig(cfg Config) (Manager, error) {
	u, err := url.Parse(cfg.BaseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid BaseURL configuration: %w", err)
	}

	// Default to HS256 if SigningMethod is not specified
	signingMethod := cfg.SigningMethod
	if signingMethod == nil {
		signingMethod = jwt.SigningMethodHS256
	}

	// Validate signing key length based on the algorithm
	if err := validateSigningKey(cfg.SigningKey, signingMethod); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return &manager{
		url:           u,
		signingKey:    cfg.SigningKey, // Key length validated above
		expiration:    cfg.Expiration,
		baseURL:       cfg.BaseURL,
		routes:        cfg.Routes,
		queryKey:      cfg.QueryKey,
		payload:       nil,
		asQuery:       cfg.AsQuery,
		signingMethod: signingMethod,
	}, nil
}

// NewManager returns a manager instance using the Configurator interface
func NewManager(cfg Configurator) (Manager, error) {
	// Convert Configurator interface to Config struct and delegate to NewManagerFromConfig
	config := Config{
		SigningKey:    cfg.SigningKey(),
		Expiration:    cfg.Expiration(),
		BaseURL:       cfg.BaseURL(),
		QueryKey:      cfg.QueryKey(),
		Routes:        cfg.Routes(),
		AsQuery:       cfg.AsQuery(),
		SigningMethod: nil, // Will default to HS256 in NewManagerFromConfig
	}
	return NewManagerFromConfig(config)
}

func (m *manager) WithData(key string, val any) Manager {
	m.mx.Lock()
	defer m.mx.Unlock()

	if m.payload == nil {
		m.payload = map[string]any{}
	}

	m.payload[key] = val

	return m
}

func (m *manager) Generate(route string) (string, error) {
	token, err := Generate(m.payload, m.signingKey, m.expiration, m.signingMethod)
	if err != nil {
		return "", fmt.Errorf("token generation failed: %w", err)
	}

	m.payload = nil

	var ok bool
	var segment string

	if segment, ok = m.routes[route]; !ok {
		return "", fmt.Errorf("route '%s' not found in configured routes", route)
	}

	var u *url.URL
	if m.asQuery {
		u = m.url.JoinPath(segment)
		u.RawQuery = fmt.Sprintf("%s=%s", m.queryKey, url.QueryEscape(token))
	} else {
		u = m.url.JoinPath(segment, token)
	}

	return u.String(), nil
}

func (m *manager) GetAndValidate(fn func(string) string) (Payload, error) {
	token := fn(m.queryKey)
	return m.Validate(token)
}

func (m *manager) GetExpiration() time.Duration {
	return m.expiration
}

func (m *manager) Validate(token string) (map[string]any, error) {
	return Validate(token, m.signingKey, m.signingMethod)
}

// Generate will return a secure string
func Generate(data map[string]any, signingKey string, expiration time.Duration, signingMethod jwt.SigningMethod) (string, error) {

	claims := jwt.MapClaims{
		"dat": data,
		"iat": jwt.NewNumericDate(time.Now()),
		"exp": jwt.NewNumericDate(
			time.Now().Add(expiration),
		),
	}

	token := jwt.NewWithClaims(signingMethod, claims)

	signedToken, err := token.SignedString([]byte(signingKey))
	if err != nil {
		// Don't expose JWT library internal errors that might leak key information
		return "", errors.New("token signing failed")
	}

	return signedToken, nil
}

// Validate will check the given signed string is valid and return
// the identifier we stored in the token.
func Validate(tokenString, signingKey string, signingMethod jwt.SigningMethod) (map[string]any, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		// Check that the token's signing method matches the expected one
		if token.Method != signingMethod {
			return nil, errors.New("token signing method validation failed")
		}
		return []byte(signingKey), nil
	})

	if err != nil {
		// Don't expose JWT library internal errors that might leak sensitive data
		return nil, errors.New("token validation failed")
	}

	var ok bool
	var claims jwt.MapClaims

	if claims, ok = token.Claims.(jwt.MapClaims); !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	dat, ok := claims["dat"].(map[string]any)
	if !ok {
		return nil, errors.New("token payload extraction failed")
	}

	return dat, nil
}
