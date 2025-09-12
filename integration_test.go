package urlkit

import (
	"testing"
	"time"

	"github.com/goliatone/go-urlkit/securelink"
)

// TestSecurelinkIntegration verifies that the refactored securelink package
// integrates properly with the main urlkit package and works correctly.
func TestSecurelinkIntegration(t *testing.T) {
	// Test that securelink can be imported and used without issues
	cfg := securelink.Config{
		SigningKey: "a-very-secure-key-of-at-least-32-bytes-for-integration-testing",
		Expiration: 1 * time.Hour,
		BaseURL:    "https://integration-test.example.com",
		QueryKey:   "token",
		Routes: map[string]string{
			"activate": "/auth/activate",
			"reset":    "/auth/reset",
		},
		AsQuery: false,
	}

	manager, err := securelink.NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create securelink manager: %v", err)
	}

	// Test basic functionality
	payload := securelink.Payload{
		"user_id": "integration-test-user",
		"action":  "account_activation",
	}

	link, err := manager.Generate("activate", payload)
	if err != nil {
		t.Fatalf("Failed to generate secure link: %v", err)
	}

	if link == "" {
		t.Fatal("Generated link is empty")
	}

	// Verify the link has the expected structure
	expectedPrefix := "https://integration-test.example.com/auth/activate/"
	if len(link) <= len(expectedPrefix) {
		t.Fatalf("Generated link appears too short: %s", link)
	}

	// Extract token and validate it
	token := link[len(expectedPrefix):]
	validatedPayload, err := manager.Validate(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	// Verify payload data
	if validatedPayload["user_id"] != "integration-test-user" {
		t.Errorf("Expected user_id 'integration-test-user', got %v", validatedPayload["user_id"])
	}

	if validatedPayload["action"] != "account_activation" {
		t.Errorf("Expected action 'account_activation', got %v", validatedPayload["action"])
	}

	t.Log("Securelink integration test passed successfully")
}

// TestSecurelinkBackwardCompatibility verifies that the legacy Configurator
// interface still works correctly after refactoring.
func TestSecurelinkBackwardCompatibility(t *testing.T) {
	// Create a test configurator implementation
	cfg := &testConfig{
		signingKey: "legacy-api-test-key-32-bytes-long",
		expiration: 30 * time.Minute,
		baseURL:    "https://legacy-test.example.com",
		queryKey:   "auth",
		routes:     map[string]string{"verify": "/legacy/verify"},
		asQuery:    true, // Test query-based URLs
	}

	manager, err := securelink.NewManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("Failed to create manager with legacy API: %v", err)
	}

	// Test generation and validation
	payload := securelink.Payload{"legacy": "test"}
	link, err := manager.Generate("verify", payload)
	if err != nil {
		t.Fatalf("Failed to generate link with legacy API: %v", err)
	}

	// Should be query-based URL
	expectedContains := "https://legacy-test.example.com/legacy/verify?auth="
	if len(link) <= len(expectedContains) || link[:len(expectedContains)] != expectedContains {
		t.Fatalf("Expected query-based URL, got: %s", link)
	}

	t.Log("Securelink backward compatibility test passed successfully")
}

// testConfig implements the securelink.Configurator interface for testing
type testConfig struct {
	signingKey string
	expiration time.Duration
	baseURL    string
	queryKey   string
	routes     map[string]string
	asQuery    bool
}

func (c *testConfig) GetSigningKey() string        { return c.signingKey }
func (c *testConfig) GetExpiration() time.Duration { return c.expiration }
func (c *testConfig) GetBaseURL() string           { return c.baseURL }
func (c *testConfig) GetQueryKey() string          { return c.queryKey }
func (c *testConfig) GetRoutes() map[string]string { return c.routes }
func (c *testConfig) GetAsQuery() bool             { return c.asQuery }
