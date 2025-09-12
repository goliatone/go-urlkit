package securelink

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestNewManagerFromConfig(t *testing.T) {
	cfg := Config{
		SigningKey: "a-very-secure-key-of-at-least-32-bytes",
		Expiration: 1 * time.Hour,
		BaseURL:    "https://example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"activate": "/activate"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManagerFromConfig returned nil manager")
	}

	// Test that the manager has the correct expiration
	if manager.GetExpiration() != cfg.Expiration {
		t.Errorf("Expected expiration %v, got %v", cfg.Expiration, manager.GetExpiration())
	}
}

func TestNewManagerFromConfigWithSigningMethod(t *testing.T) {
	cfg := Config{
		SigningKey:    "a-very-secure-key-of-at-least-48-bytes-for-HS384-algorithm",
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS384,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManagerFromConfig returned nil manager")
	}
}

func TestNewManagerFromConfigDefaultSigningMethod(t *testing.T) {
	cfg := Config{
		SigningKey: "a-very-secure-key-of-at-least-32-bytes",
		Expiration: 1 * time.Hour,
		BaseURL:    "https://example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"activate": "/activate"},
		AsQuery:    false,
		// SigningMethod not set, should default to HS256
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManagerFromConfig returned nil manager")
	}
}

// Test that the old NewManager function still works (backward compatibility)
type testConfigurator struct {
	signingKey string
	expiration time.Duration
	baseURL    string
	queryKey   string
	routes     map[string]string
	asQuery    bool
}

func (tc *testConfigurator) SigningKey() string        { return tc.signingKey }
func (tc *testConfigurator) Expiration() time.Duration { return tc.expiration }
func (tc *testConfigurator) BaseURL() string           { return tc.baseURL }
func (tc *testConfigurator) QueryKey() string          { return tc.queryKey }
func (tc *testConfigurator) Routes() map[string]string { return tc.routes }
func (tc *testConfigurator) AsQuery() bool             { return tc.asQuery }

func TestNewManagerBackwardCompatibility(t *testing.T) {
	cfg := &testConfigurator{
		signingKey: "a-very-secure-key-of-at-least-32-bytes",
		expiration: 1 * time.Hour,
		baseURL:    "https://example.com",
		queryKey:   "token",
		routes:     map[string]string{"activate": "/activate"},
		asQuery:    false,
	}

	manager, err := NewManager(cfg)
	if err != nil {
		t.Fatalf("NewManager failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewManager returned nil manager")
	}

	// Test that the manager has the correct expiration
	if manager.GetExpiration() != cfg.Expiration() {
		t.Errorf("Expected expiration %v, got %v", cfg.Expiration(), manager.GetExpiration())
	}
}

// Test default HS256 behavior when SigningMethod is nil
func TestDefaultSigningMethodHS256(t *testing.T) {
	cfg := Config{
		SigningKey: "a-very-secure-key-of-at-least-32-bytes",
		Expiration: 1 * time.Hour,
		BaseURL:    "https://example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"activate": "/activate"},
		AsQuery:    false,
		// SigningMethod is nil, should default to HS256
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Test token generation and validation
	manager = manager.WithData("user_id", "123")
	link, err := manager.Generate("activate")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if link == "" {
		t.Fatal("Generated link is empty")
	}

	// Extract token from the link and validate it
	// For this test, the token is part of the URL path
	// URL format: https://example.com/activate/{token}
	expectedPrefix := "https://example.com/activate/"
	if !strings.HasPrefix(link, expectedPrefix) {
		t.Fatalf("Link doesn't have expected prefix. Got: %s", link)
	}

	token := strings.TrimPrefix(link, expectedPrefix)
	payload, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if payload["user_id"] != "123" {
		t.Errorf("Expected user_id 123, got %v", payload["user_id"])
	}
}

// Test custom signing methods (HS384, HS512)
func TestCustomSigningMethods(t *testing.T) {
	testCases := []struct {
		name          string
		signingMethod jwt.SigningMethod
		keyLength     int
	}{
		{"HS384", jwt.SigningMethodHS384, 48},
		{"HS512", jwt.SigningMethodHS512, 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate a key of appropriate length
			signingKey := strings.Repeat("a", tc.keyLength)

			cfg := Config{
				SigningKey:    signingKey,
				Expiration:    1 * time.Hour,
				BaseURL:       "https://example.com",
				QueryKey:      "token",
				Routes:        map[string]string{"activate": "/activate"},
				AsQuery:       false,
				SigningMethod: tc.signingMethod,
			}

			manager, err := NewManagerFromConfig(cfg)
			if err != nil {
				t.Fatalf("NewManagerFromConfig failed: %v", err)
			}

			// Test token generation and validation
			manager = manager.WithData("user_id", "456")
			link, err := manager.Generate("activate")
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if link == "" {
				t.Fatal("Generated link is empty")
			}

			// Extract token and validate
			expectedPrefix := "https://example.com/activate/"
			if !strings.HasPrefix(link, expectedPrefix) {
				t.Fatalf("Link doesn't have expected prefix. Got: %s", link)
			}

			token := strings.TrimPrefix(link, expectedPrefix)
			payload, err := manager.Validate(token)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			if payload["user_id"] != "456" {
				t.Errorf("Expected user_id 456, got %v", payload["user_id"])
			}
		})
	}
}

// Test token generation and validation with different algorithms
func TestSigningMethodMismatchValidation(t *testing.T) {
	signingKey := strings.Repeat("a", 64) // Long enough for all methods

	// Create manager with HS256
	cfg256 := Config{
		SigningKey:    signingKey,
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager256, err := NewManagerFromConfig(cfg256)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Generate token with HS256
	manager256 = manager256.WithData("user_id", "789")
	link, err := manager256.Generate("activate")
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Extract token
	token := strings.TrimPrefix(link, "https://example.com/activate/")

	// Create manager with HS384 (different algorithm)
	cfg384 := Config{
		SigningKey:    signingKey,
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS384,
	}

	manager384, err := NewManagerFromConfig(cfg384)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Try to validate HS256 token with HS384 manager (should fail)
	_, err = manager384.Validate(token)
	if err == nil {
		t.Fatal("Expected validation to fail with different signing method, but it succeeded")
	}

	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("Expected 'token validation failed' error, got: %v", err)
	}
}

// Test validation rejects keys that are too short
func TestValidationRejectsShortKeys(t *testing.T) {
	testCases := []struct {
		name          string
		signingMethod jwt.SigningMethod
		keyLength     int
		expectedError string
	}{
		{"HS256 short key", jwt.SigningMethodHS256, 16, "signing key too short for HS256 algorithm"},
		{"HS384 short key", jwt.SigningMethodHS384, 32, "signing key too short for HS384 algorithm"},
		{"HS512 short key", jwt.SigningMethodHS512, 48, "signing key too short for HS512 algorithm"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			shortKey := strings.Repeat("a", tc.keyLength)

			cfg := Config{
				SigningKey:    shortKey,
				Expiration:    1 * time.Hour,
				BaseURL:       "https://example.com",
				QueryKey:      "token",
				Routes:        map[string]string{"activate": "/activate"},
				AsQuery:       false,
				SigningMethod: tc.signingMethod,
			}

			_, err := NewManagerFromConfig(cfg)
			if err == nil {
				t.Fatal("Expected NewManagerFromConfig to fail with short key, but it succeeded")
			}

			if !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Expected error to contain '%s', got: %v", tc.expectedError, err)
			}

			// Check that the error message includes bit information
			if !strings.Contains(err.Error(), "bits") {
				t.Errorf("Expected error message to include bit information, got: %v", err)
			}
		})
	}
}

// Test validation accepts keys of appropriate length
func TestValidationAcceptsValidKeys(t *testing.T) {
	testCases := []struct {
		name          string
		signingMethod jwt.SigningMethod
		keyLength     int
	}{
		{"HS256 exact minimum", jwt.SigningMethodHS256, 32},
		{"HS256 longer key", jwt.SigningMethodHS256, 64},
		{"HS384 exact minimum", jwt.SigningMethodHS384, 48},
		{"HS384 longer key", jwt.SigningMethodHS384, 64},
		{"HS512 exact minimum", jwt.SigningMethodHS512, 64},
		{"HS512 longer key", jwt.SigningMethodHS512, 128},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validKey := strings.Repeat("a", tc.keyLength)

			cfg := Config{
				SigningKey:    validKey,
				Expiration:    1 * time.Hour,
				BaseURL:       "https://example.com",
				QueryKey:      "token",
				Routes:        map[string]string{"activate": "/activate"},
				AsQuery:       false,
				SigningMethod: tc.signingMethod,
			}

			manager, err := NewManagerFromConfig(cfg)
			if err != nil {
				t.Fatalf("NewManagerFromConfig failed with valid key: %v", err)
			}

			if manager == nil {
				t.Fatal("NewManagerFromConfig returned nil manager with valid key")
			}
		})
	}
}

// Test different key lengths for different algorithms
func TestDifferentKeyLengthsForAlgorithms(t *testing.T) {
	// Test that a key valid for HS256 might not be valid for HS384/HS512
	hs256Key := strings.Repeat("a", 32) // Valid for HS256, too short for HS384/HS512

	// Should work for HS256
	cfgHS256 := Config{
		SigningKey:    hs256Key,
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	_, err := NewManagerFromConfig(cfgHS256)
	if err != nil {
		t.Fatalf("HS256 should accept 32-byte key: %v", err)
	}

	// Should fail for HS384
	cfgHS384 := Config{
		SigningKey:    hs256Key,
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS384,
	}

	_, err = NewManagerFromConfig(cfgHS384)
	if err == nil {
		t.Fatal("HS384 should reject 32-byte key")
	}

	// Should fail for HS512
	cfgHS512 := Config{
		SigningKey:    hs256Key,
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS512,
	}

	_, err = NewManagerFromConfig(cfgHS512)
	if err == nil {
		t.Fatal("HS512 should reject 32-byte key")
	}
}

// Test error messages are descriptive and helpful
func TestErrorMessagesAreDescriptive(t *testing.T) {
	shortKey := strings.Repeat("a", 16) // Too short for any algorithm

	cfg := Config{
		SigningKey:    shortKey,
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	_, err := NewManagerFromConfig(cfg)
	if err == nil {
		t.Fatal("Expected error with short key")
	}

	errorMsg := err.Error()

	// Check that error message contains all expected information
	expectedParts := []string{
		"signing key too short",
		"HS256",
		"got 16 bytes",
		"need at least 32 bytes",
		"256 bits",
	}

	for _, part := range expectedParts {
		if !strings.Contains(errorMsg, part) {
			t.Errorf("Error message should contain '%s'. Got: %s", part, errorMsg)
		}
	}
}

// Test that backward compatibility through NewManager also validates keys
func TestBackwardCompatibilityValidatesKeys(t *testing.T) {
	cfg := &testConfigurator{
		signingKey: "short", // Too short (5 bytes, need 32 for HS256)
		expiration: 1 * time.Hour,
		baseURL:    "https://example.com",
		queryKey:   "token",
		routes:     map[string]string{"activate": "/activate"},
		asQuery:    false,
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Fatal("Expected NewManager to fail with short key")
	}

	if !strings.Contains(err.Error(), "signing key too short") {
		t.Errorf("Expected error about short key, got: %v", err)
	}
}
