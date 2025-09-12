package securelink

import (
	"strings"
	"sync"
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
	payload := Payload{"user_id": "123"}
	link, err := manager.Generate("activate", payload)
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
	validatedPayload, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	if validatedPayload["user_id"] != "123" {
		t.Errorf("Expected user_id 123, got %v", validatedPayload["user_id"])
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
			payload := Payload{"user_id": "456"}
			link, err := manager.Generate("activate", payload)
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
			validatedPayload, err := manager.Validate(token)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			if validatedPayload["user_id"] != "456" {
				t.Errorf("Expected user_id 456, got %v", validatedPayload["user_id"])
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
	payload := Payload{"user_id": "789"}
	link, err := manager256.Generate("activate", payload)
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

// Test error messages don't contain sensitive information
func TestErrorMessagesDontContainSensitiveInfo(t *testing.T) {
	signingKey := strings.Repeat("s", 64) // Valid length key with identifiable pattern

	cfg := Config{
		SigningKey:    signingKey,
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Test with invalid token
	_, err = manager.Validate("invalid.jwt.token")
	if err == nil {
		t.Fatal("Expected validation to fail with invalid token")
	}

	errorMsg := err.Error()

	// Ensure error message doesn't contain the signing key
	if strings.Contains(errorMsg, signingKey) {
		t.Errorf("Error message contains signing key: %s", errorMsg)
	}

	// Ensure error message doesn't contain signing key patterns
	if strings.Contains(errorMsg, "ssss") {
		t.Errorf("Error message contains signing key pattern: %s", errorMsg)
	}

	// Should be a generic message
	if !strings.Contains(errorMsg, "token validation failed") {
		t.Errorf("Expected generic error message, got: %s", errorMsg)
	}
}

// Test error contexts are helpful for debugging
func TestErrorContextsAreHelpful(t *testing.T) {
	testCases := []struct {
		name           string
		cfg            Config
		expectedErrors []string
	}{
		{
			name: "Invalid BaseURL",
			cfg: Config{
				SigningKey: strings.Repeat("a", 32),
				Expiration: 1 * time.Hour,
				BaseURL:    "://invalid-url",
				QueryKey:   "token",
				Routes:     map[string]string{"activate": "/activate"},
				AsQuery:    false,
			},
			expectedErrors: []string{"invalid BaseURL configuration"},
		},
		{
			name: "Short signing key",
			cfg: Config{
				SigningKey: "short",
				Expiration: 1 * time.Hour,
				BaseURL:    "https://example.com",
				QueryKey:   "token",
				Routes:     map[string]string{"activate": "/activate"},
				AsQuery:    false,
			},
			expectedErrors: []string{"configuration validation failed", "signing key too short"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewManagerFromConfig(tc.cfg)
			if err == nil {
				t.Fatal("Expected error but got none")
			}

			errorMsg := err.Error()
			for _, expectedError := range tc.expectedErrors {
				if !strings.Contains(errorMsg, expectedError) {
					t.Errorf("Expected error to contain '%s', got: %s", expectedError, errorMsg)
				}
			}
		})
	}
}

// Test route error provides helpful context
func TestRouteErrorContext(t *testing.T) {
	cfg := Config{
		SigningKey:    strings.Repeat("a", 32),
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Try to generate with unknown route
	_, err = manager.Generate("unknown")
	if err == nil {
		t.Fatal("Expected error for unknown route")
	}

	errorMsg := err.Error()

	// Should contain the route name and helpful context
	if !strings.Contains(errorMsg, "unknown") {
		t.Errorf("Error should mention the route name, got: %s", errorMsg)
	}

	if !strings.Contains(errorMsg, "not found in configured routes") {
		t.Errorf("Error should mention configured routes, got: %s", errorMsg)
	}
}

// Test JWT generation error handling
func TestJWTGenerationErrorHandling(t *testing.T) {
	// This is a more complex test - we would need to simulate JWT signing failure
	// For now, we'll verify the error message structure
	cfg := Config{
		SigningKey:    strings.Repeat("a", 32),
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Test normal generation works (no error case to test signing failure easily)
	payload := Payload{"test": "value"}
	link, err := manager.Generate("activate", payload)
	if err != nil {
		// If there's an error, verify it has proper context
		errorMsg := err.Error()
		if strings.Contains(errorMsg, "token generation failed") {
			// This is expected structure
			return
		}
		t.Fatalf("Unexpected error format: %v", err)
	}

	// Verify successful generation
	if link == "" {
		t.Fatal("Generated link is empty")
	}
}

// Test new Generate method with no payload
func TestGenerateWithNoPayload(t *testing.T) {
	cfg := Config{
		SigningKey:    strings.Repeat("a", 32),
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Generate with no payload
	link, err := manager.Generate("activate")
	if err != nil {
		t.Fatalf("Generate with no payload failed: %v", err)
	}

	if link == "" {
		t.Fatal("Generated link is empty")
	}

	// Extract and validate token
	expectedPrefix := "https://example.com/activate/"
	if !strings.HasPrefix(link, expectedPrefix) {
		t.Fatalf("Link doesn't have expected prefix. Got: %s", link)
	}

	token := strings.TrimPrefix(link, expectedPrefix)
	validatedPayload, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Should have empty payload
	if len(validatedPayload) != 0 {
		t.Errorf("Expected empty payload, got %v", validatedPayload)
	}
}

// Test new Generate method with single payload
func TestGenerateWithSinglePayload(t *testing.T) {
	cfg := Config{
		SigningKey:    strings.Repeat("a", 32),
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Generate with single payload
	payload := Payload{"user_id": "123", "role": "admin"}
	link, err := manager.Generate("activate", payload)
	if err != nil {
		t.Fatalf("Generate with single payload failed: %v", err)
	}

	if link == "" {
		t.Fatal("Generated link is empty")
	}

	// Extract and validate token
	expectedPrefix := "https://example.com/activate/"
	if !strings.HasPrefix(link, expectedPrefix) {
		t.Fatalf("Link doesn't have expected prefix. Got: %s", link)
	}

	token := strings.TrimPrefix(link, expectedPrefix)
	validatedPayload, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Check payload contents
	if validatedPayload["user_id"] != "123" {
		t.Errorf("Expected user_id 123, got %v", validatedPayload["user_id"])
	}
	if validatedPayload["role"] != "admin" {
		t.Errorf("Expected role admin, got %v", validatedPayload["role"])
	}
}

// Test new Generate method with multiple payloads (should merge)
func TestGenerateWithMultiplePayloads(t *testing.T) {
	cfg := Config{
		SigningKey:    strings.Repeat("a", 32),
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	// Generate with multiple payloads
	payload1 := Payload{"user_id": "123", "role": "admin"}
	payload2 := Payload{"session": "xyz", "timestamp": "2023-01-01"}
	payload3 := Payload{"role": "superadmin"} // Should override payload1's role

	link, err := manager.Generate("activate", payload1, payload2, payload3)
	if err != nil {
		t.Fatalf("Generate with multiple payloads failed: %v", err)
	}

	if link == "" {
		t.Fatal("Generated link is empty")
	}

	// Extract and validate token
	expectedPrefix := "https://example.com/activate/"
	if !strings.HasPrefix(link, expectedPrefix) {
		t.Fatalf("Link doesn't have expected prefix. Got: %s", link)
	}

	token := strings.TrimPrefix(link, expectedPrefix)
	validatedPayload, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}

	// Check merged payload contents
	if validatedPayload["user_id"] != "123" {
		t.Errorf("Expected user_id 123, got %v", validatedPayload["user_id"])
	}
	if validatedPayload["session"] != "xyz" {
		t.Errorf("Expected session xyz, got %v", validatedPayload["session"])
	}
	if validatedPayload["timestamp"] != "2023-01-01" {
		t.Errorf("Expected timestamp 2023-01-01, got %v", validatedPayload["timestamp"])
	}
	// Last payload should override
	if validatedPayload["role"] != "superadmin" {
		t.Errorf("Expected role superadmin (overridden), got %v", validatedPayload["role"])
	}
}

// Test thread safety by calling Generate from multiple goroutines
func TestThreadSafetyGenerate(t *testing.T) {
	cfg := Config{
		SigningKey:    strings.Repeat("a", 32),
		Expiration:    1 * time.Hour,
		BaseURL:       "https://example.com",
		QueryKey:      "token",
		Routes:        map[string]string{"activate": "/activate"},
		AsQuery:       false,
		SigningMethod: jwt.SigningMethodHS256,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewManagerFromConfig failed: %v", err)
	}

	const numGoroutines = 50
	const numGenerationsPerRoutine = 10

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*numGenerationsPerRoutine)
	links := make(chan string, numGoroutines*numGenerationsPerRoutine)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numGenerationsPerRoutine; j++ {
				payload := Payload{
					"user_id":    routineID,
					"generation": j,
					"timestamp":  time.Now().Format(time.RFC3339),
				}

				link, err := manager.Generate("activate", payload)
				if err != nil {
					errors <- err
					return
				}
				links <- link
			}
		}(i)
	}

	wg.Wait()
	close(errors)
	close(links)

	// Check for any errors
	if len(errors) > 0 {
		err := <-errors
		t.Fatalf("Concurrent generation failed: %v", err)
	}

	// Verify all links were generated
	linkCount := 0
	for range links {
		linkCount++
	}

	expectedLinks := numGoroutines * numGenerationsPerRoutine
	if linkCount != expectedLinks {
		t.Errorf("Expected %d links, got %d", expectedLinks, linkCount)
	}
}

// Test internal Generate function with different signing methods
func TestInternalGenerateWithDifferentSigningMethods(t *testing.T) {
	testCases := []struct {
		name          string
		signingMethod jwt.SigningMethod
		keyLength     int
	}{
		{"HS256", jwt.SigningMethodHS256, 32},
		{"HS384", jwt.SigningMethodHS384, 48},
		{"HS512", jwt.SigningMethodHS512, 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Generate a key of appropriate length
			signingKey := strings.Repeat("a", tc.keyLength)
			testData := map[string]any{
				"user_id": "123",
				"role":    "admin",
			}
			expiration := 1 * time.Hour

			// Test internal Generate function directly
			token, err := Generate(testData, signingKey, expiration, tc.signingMethod)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if token == "" {
				t.Fatal("Generated token is empty")
			}

			// Validate the token using the same signing method
			validatedPayload, err := Validate(token, signingKey, tc.signingMethod)
			if err != nil {
				t.Fatalf("Validate failed: %v", err)
			}

			// Check payload contents
			if validatedPayload["user_id"] != "123" {
				t.Errorf("Expected user_id 123, got %v", validatedPayload["user_id"])
			}
			if validatedPayload["role"] != "admin" {
				t.Errorf("Expected role admin, got %v", validatedPayload["role"])
			}
		})
	}
}

// Test that internal Generate function rejects mismatched signing methods during validation
func TestInternalGenerateSigningMethodValidation(t *testing.T) {
	signingKey := strings.Repeat("a", 64) // Long enough for all methods
	testData := map[string]any{"user_id": "123"}
	expiration := 1 * time.Hour

	// Generate token with HS256
	token, err := Generate(testData, signingKey, expiration, jwt.SigningMethodHS256)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// Try to validate with HS384 (should fail)
	_, err = Validate(token, signingKey, jwt.SigningMethodHS384)
	if err == nil {
		t.Fatal("Expected validation to fail with different signing method, but it succeeded")
	}

	if !strings.Contains(err.Error(), "token validation failed") {
		t.Errorf("Expected 'token validation failed' error, got: %v", err)
	}
}

// Test manager integration passes signing method correctly
func TestManagerIntegrationPassesSigningMethodCorrectly(t *testing.T) {
	// Test that the manager correctly passes the configured signing method to internal Generate function
	testCases := []struct {
		name          string
		signingMethod jwt.SigningMethod
		keyLength     int
	}{
		{"HS256", jwt.SigningMethodHS256, 32},
		{"HS384", jwt.SigningMethodHS384, 48},
		{"HS512", jwt.SigningMethodHS512, 64},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
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

			// Generate token through manager
			payload := Payload{"user_id": "456"}
			link, err := manager.Generate("activate", payload)
			if err != nil {
				t.Fatalf("Manager Generate failed: %v", err)
			}

			// Extract token from link
			expectedPrefix := "https://example.com/activate/"
			if !strings.HasPrefix(link, expectedPrefix) {
				t.Fatalf("Link doesn't have expected prefix. Got: %s", link)
			}

			token := strings.TrimPrefix(link, expectedPrefix)

			// Validate directly using the expected signing method
			validatedPayload, err := Validate(token, signingKey, tc.signingMethod)
			if err != nil {
				t.Fatalf("Direct validation failed: %v", err)
			}

			if validatedPayload["user_id"] != "456" {
				t.Errorf("Expected user_id 456, got %v", validatedPayload["user_id"])
			}

			// Also validate through manager (should use same signing method)
			managerValidatedPayload, err := manager.Validate(token)
			if err != nil {
				t.Fatalf("Manager validation failed: %v", err)
			}

			if managerValidatedPayload["user_id"] != "456" {
				t.Errorf("Expected user_id 456 from manager validation, got %v", managerValidatedPayload["user_id"])
			}
		})
	}
}
