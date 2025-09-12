// Package securelink provides secure, expiring URL generation and validation using JWT tokens.
// It enables the creation of time-limited, signed URLs that can be safely distributed and validated.
//
// # Key Features
//
//   - Thread-safe, stateless design for concurrent access
//   - Configurable JWT signing algorithms (HS256, HS384, HS512)
//   - Automatic signing key length validation for security
//   - Flexible URL generation (path-based or query parameter)
//   - Support for custom payload data in tokens
//   - Backward compatibility with legacy API
//
// # Basic Usage
//
// The package offers two ways to create a manager:
//
// ## 1. Config Struct (Recommended for new code)
//
//	cfg := securelink.Config{
//		SigningKey:    "a-very-secure-key-of-at-least-32-bytes-long", // Min 32 bytes for HS256
//		Expiration:    1 * time.Hour,                                 // Token lifetime
//		BaseURL:       "https://example.com",                         // Base URL for links
//		QueryKey:      "token",                                       // Query parameter name (if AsQuery=true)
//		Routes:        map[string]string{"activate": "/activate"},    // Route mappings
//		AsQuery:       false,                                         // false=path-based, true=query-based URLs
//		SigningMethod: jwt.SigningMethodHS256,                        // Optional: defaults to HS256
//	}
//	manager, err := securelink.NewManagerFromConfig(cfg)
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Generate secure link with payload
//	payload := securelink.Payload{"user_id": "123", "action": "activate"}
//	link, err := manager.Generate("activate", payload)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Result: https://example.com/activate/eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
//
//	// Validate token and extract payload
//	validatedPayload, err := manager.Validate(token)
//	if err != nil {
//		log.Fatal(err)
//	}
//	userID := validatedPayload["user_id"] // "123"
//
// ## 2. Configurator Interface (Legacy compatibility)
//
//	manager, err := securelink.NewManager(cfg) // where cfg implements Configurator
//
// # Security Best Practices
//
//   - Use strong signing keys with appropriate lengths:
//   - HS256: minimum 32 bytes (256 bits)
//   - HS384: minimum 48 bytes (384 bits)
//   - HS512: minimum 64 bytes (512 bits)
//   - Store signing keys securely (environment variables, key management systems)
//   - Use appropriate token expiration times (shorter for sensitive operations)
//   - Always validate tokens before processing requests
//   - Consider using HS384 or HS512 for higher security requirements
//
// # Migration from Legacy API
//
// The legacy API used a stateful design with method chaining:
//
//	// OLD (deprecated)
//	link, err := manager.WithData("user_id", 123).Generate("activate")
//
// The new API uses a stateless design with payload parameters:
//
//	// NEW (recommended)
//	payload := securelink.Payload{"user_id": 123}
//	link, err := manager.Generate("activate", payload)
//
// The old API is maintained for backward compatibility through the Configurator interface.
package securelink

import (
	"errors"
	"fmt"
	"net/url"
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
	asQuery       bool
	signingMethod jwt.SigningMethod
}

// Configurator holds configuration options for backward compatibility with the legacy API.
// This interface is maintained for existing code that implements these methods.
// For new code, prefer using the Config struct directly with NewManagerFromConfig.
type Configurator interface {
	SigningKey() string        // The secret key used for JWT signing (min 32 bytes for HS256)
	Expiration() time.Duration // How long tokens remain valid
	BaseURL() string           // Base URL for generated links
	QueryKey() string          // Query parameter name when AsQuery() is true
	Routes() map[string]string // Map of route names to URL paths
	AsQuery() bool             // Whether to embed token as query param (true) or path segment (false)
}

// Config is a struct-based configuration for simplified direct instantiation.
// This is the preferred way to configure the securelink manager for new code.
//
// Example:
//
//	cfg := securelink.Config{
//		SigningKey:    "your-32-byte-or-longer-secret-key-here",
//		Expiration:    1 * time.Hour,
//		BaseURL:       "https://api.example.com",
//		QueryKey:      "token",
//		Routes:        map[string]string{"reset": "/auth/reset", "activate": "/auth/activate"},
//		AsQuery:       false, // Use path-based URLs: /auth/activate/{token}
//		SigningMethod: jwt.SigningMethodHS256, // Optional: defaults to HS256
//	}
type Config struct {
	SigningKey    string            // Secret key for JWT signing (length validated based on algorithm)
	Expiration    time.Duration     // Token lifetime (e.g., 1*time.Hour, 30*time.Minute)
	BaseURL       string            // Base URL for generated links (e.g., "https://api.example.com")
	QueryKey      string            // Query parameter name when AsQuery=true (e.g., "token", "auth")
	Routes        map[string]string // Map of route names to URL paths (e.g., {"reset": "/auth/reset"})
	AsQuery       bool              // false=path URLs (/path/{token}), true=query URLs (/path?key={token})
	SigningMethod jwt.SigningMethod // JWT algorithm (HS256, HS384, HS512). Defaults to HS256 if nil
}

// Payload represents custom data that can be embedded in secure links.
// It is used both for input when generating tokens and output when validating them.
//
// Example usage:
//
//	// Creating payload for token generation
//	payload := securelink.Payload{
//		"user_id":    "12345",
//		"action":     "password_reset",
//		"expires_at": time.Now().Add(1*time.Hour).Unix(),
//	}
//
//	// Using payload from token validation
//	validatedPayload, err := manager.Validate(token)
//	if err == nil {
//		userID := validatedPayload["user_id"].(string)
//	}
type Payload map[string]any

// GetString safely extracts a string value from the payload.
// Returns an error if the key doesn't exist or the value is not a string.
//
// Example:
//
//	payload := securelink.Payload{"user_id": "123", "count": 42}
//	userID, err := payload.GetString("user_id") // Returns "123", nil
//	count, err := payload.GetString("count")    // Returns "", error (not a string)
func (p Payload) GetString(key string) (val string, err error) {
	var ok bool
	if val, ok = p[key].(string); !ok {
		return "", fmt.Errorf("error decoding key %s: not found or not a string", key)
	}
	return val, nil
}

// Manager provides secure link generation and validation capabilities.
// Implementations are thread-safe and can be used concurrently from multiple goroutines.
//
// The Manager interface represents the core functionality for creating time-limited,
// cryptographically signed URLs that can carry custom payload data.
type Manager interface {
	// Generate creates a secure link for the specified route with optional payload data.
	// Multiple payloads can be provided and will be merged (later payloads override earlier ones).
	//
	// Parameters:
	//   route: Must match a key in the Routes configuration map
	//   payloads: Zero or more Payload maps to embed in the token
	//
	// Returns:
	//   string: Complete URL with embedded token
	//   error: Configuration errors, unknown routes, or token generation failures
	//
	// Example:
	//   payload := securelink.Payload{"user_id": "123"}
	//   link, err := manager.Generate("activate", payload)
	//   // Returns: "https://example.com/activate/eyJhbGciOi..."
	Generate(route string, payloads ...Payload) (string, error)

	// Validate verifies a JWT token and returns the embedded payload data.
	// Validates signature, expiration, and token structure.
	//
	// Parameters:
	//   token: JWT token string to validate
	//
	// Returns:
	//   map[string]any: Payload data that was embedded in the token
	//   error: Invalid signature, expired token, or malformed token
	//
	// Example:
	//   payload, err := manager.Validate("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...")
	//   if err == nil {
	//       userID := payload["user_id"].(string)
	//   }
	Validate(token string) (map[string]any, error)

	// GetAndValidate extracts a token using the provided function and validates it.
	// This is a convenience method for web handlers that need to extract tokens
	// from HTTP requests.
	//
	// Parameters:
	//   fn: Function that extracts the token (e.g., from query params, headers, etc.)
	//
	// Returns:
	//   Payload: Validated payload data from the token
	//   error: Token extraction, validation, or parsing errors
	//
	// Example:
	//   payload, err := manager.GetAndValidate(func(key string) string {
	//       return r.URL.Query().Get(key) // Extract from HTTP query params
	//   })
	GetAndValidate(fn func(string) string) (Payload, error)

	// GetExpiration returns the token lifetime configured for this manager.
	// This can be useful for displaying expiration information to users.
	//
	// Returns:
	//   time.Duration: How long tokens remain valid
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

// NewManagerFromConfig creates a new secure link manager from a Config struct.
// This is the recommended constructor for new code as it provides direct configuration
// without requiring interface implementations.
//
// The function performs comprehensive validation:
//   - Validates BaseURL format
//   - Ensures signing key meets minimum length requirements for the chosen algorithm
//   - Sets up proper JWT signing method (defaults to HS256 if not specified)
//
// Parameters:
//
//	cfg: Configuration struct with all required settings
//
// Returns:
//
//	Manager: Thread-safe manager instance ready for use
//	error: Configuration validation errors (invalid URL, weak signing key, etc.)
//
// Example:
//
//	cfg := securelink.Config{
//	    SigningKey: "your-super-secret-key-here-32-bytes",
//	    Expiration: 2 * time.Hour,
//	    BaseURL:    "https://api.myapp.com",
//	    Routes:     map[string]string{"verify": "/auth/verify"},
//	}
//	manager, err := securelink.NewManagerFromConfig(cfg)
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
		asQuery:       cfg.AsQuery,
		signingMethod: signingMethod,
	}, nil
}

// NewManager creates a manager instance using the Configurator interface.
// This constructor is maintained for backward compatibility with existing code
// that implements the Configurator interface.
//
// For new code, prefer NewManagerFromConfig which provides direct configuration
// without requiring interface implementations.
//
// Parameters:
//
//	cfg: Any type that implements the Configurator interface
//
// Returns:
//
//	Manager: Thread-safe manager instance
//	error: Configuration validation errors
//
// Example:
//
//	type MyConfig struct { /* implement Configurator methods */ }
//	config := &MyConfig{...}
//	manager, err := securelink.NewManager(config)
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

func (m *manager) Generate(route string, payloads ...Payload) (string, error) {
	// Merge all payloads into one map
	var combinedPayload map[string]any
	if len(payloads) > 0 {
		combinedPayload = make(map[string]any)
		for _, payload := range payloads {
			for k, v := range payload {
				combinedPayload[k] = v
			}
		}
	}

	token, err := Generate(combinedPayload, m.signingKey, m.expiration, m.signingMethod)
	if err != nil {
		return "", fmt.Errorf("token generation failed: %w", err)
	}

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

// Generate creates a JWT token containing the provided data with the specified expiration.
// This is a low-level function used internally by the Manager. For most use cases,
// prefer using Manager.Generate() which handles URL construction automatically.
//
// The function creates a JWT token with:
//   - "dat" claim containing the provided data
//   - "iat" (issued at) claim with current timestamp
//   - "exp" (expiration) claim with expiration timestamp
//
// Parameters:
//
//	data: Custom data to embed in the token (can be nil for empty payload)
//	signingKey: Secret key for JWT signing (length must match algorithm requirements)
//	expiration: How long the token should remain valid
//	signingMethod: JWT signing algorithm (HS256, HS384, or HS512)
//
// Returns:
//
//	string: Signed JWT token
//	error: Token generation failures (invalid key, signing errors)
//
// Security note: This function does not validate key length. Use NewManagerFromConfig
// for automatic key validation.
func Generate(data map[string]any, signingKey string, expiration time.Duration, signingMethod jwt.SigningMethod) (string, error) {
	// Ensure data is not nil to prevent validation issues
	if data == nil {
		data = make(map[string]any)
	}

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

// Validate verifies a JWT token and extracts the embedded data payload.
// This is a low-level function used internally by the Manager. For most use cases,
// prefer using Manager.Validate() which uses the configured signing method automatically.
//
// The function performs comprehensive validation:
//   - Verifies JWT signature using the provided key and method
//   - Checks token expiration (rejects expired tokens)
//   - Validates token structure and claims
//   - Extracts the "dat" claim containing custom payload data
//
// Parameters:
//
//	tokenString: JWT token to validate
//	signingKey: Secret key used for signature verification (must match generation key)
//	signingMethod: Expected JWT algorithm (must match the algorithm used for generation)
//
// Returns:
//
//	map[string]any: Custom data that was embedded in the token during generation
//	error: Invalid signature, expired token, algorithm mismatch, or malformed token
//
// Security notes:
//   - Tokens signed with different algorithms are rejected
//   - Expired tokens are automatically rejected
//   - Error messages are generic to prevent information leakage
func Validate(tokenString, signingKey string, signingMethod jwt.SigningMethod) (map[string]any, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
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
