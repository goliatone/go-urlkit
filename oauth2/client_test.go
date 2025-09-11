package oauth2

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

// TestUserData represents test data for OAuth2 state encryption
type TestUserData struct {
	UserID   string `json:"user_id"`
	ReturnTo string `json:"return_to"`
	Source   string `json:"source"`
}

// TestNewClient tests the Client constructor
func TestNewClient(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	tests := []struct {
		name          string
		provider      Provider
		clientID      string
		clientSecret  string
		redirectURL   string
		encryptionKey string
		expectError   bool
		errorContains string
	}{
		{
			name:          "valid client",
			provider:      provider,
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "this-is-a-24-char-key-ok",
			expectError:   false,
		},
		{
			name:          "nil provider",
			provider:      nil,
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "this-is-a-24-char-key-ok",
			expectError:   true,
			errorContains: "provider cannot be nil",
		},
		{
			name:          "empty client ID",
			provider:      provider,
			clientID:      "",
			clientSecret:  "test-client-secret",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "this-is-a-24-char-key-ok",
			expectError:   true,
			errorContains: "client ID cannot be empty",
		},
		{
			name:          "empty client secret",
			provider:      provider,
			clientID:      "test-client-id",
			clientSecret:  "",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "this-is-a-24-char-key-ok",
			expectError:   true,
			errorContains: "client secret cannot be empty",
		},
		{
			name:          "empty redirect URL",
			provider:      provider,
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			redirectURL:   "",
			encryptionKey: "this-is-a-24-char-key-ok",
			expectError:   true,
			errorContains: "redirect URL cannot be empty",
		},
		{
			name:          "encryption key too short",
			provider:      provider,
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "short",
			expectError:   true,
			errorContains: "encryption key must be 24-32 characters",
		},
		{
			name:          "encryption key too long",
			provider:      provider,
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "this-encryption-key-is-way-too-long-for-aes",
			expectError:   true,
			errorContains: "encryption key must be 24-32 characters",
		},
		{
			name:          "valid 24-char key",
			provider:      provider,
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "this-is-a-24-char-key-ok", // exactly 24 chars
			expectError:   false,
		},
		{
			name:          "valid 32-char key",
			provider:      provider,
			clientID:      "test-client-id",
			clientSecret:  "test-client-secret",
			redirectURL:   "http://localhost:8080/callback",
			encryptionKey: "this-is-a-32-character-key-ok!!", // exactly 32 chars
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient[TestUserData](
				tt.provider,
				tt.clientID,
				tt.clientSecret,
				tt.redirectURL,
				tt.encryptionKey,
			)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errorContains)
				}
				if client != nil {
					t.Error("client should be nil when error occurs")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if client == nil {
					t.Error("client should not be nil when no error")
				}
			}
		})
	}
}

// TestClientGenerateURL tests URL generation with state encryption
func TestClientGenerateURL(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	userData := TestUserData{
		UserID:   "test-user-123",
		ReturnTo: "/dashboard",
		Source:   "web",
	}

	// Test with provided state
	authURL, err := client.GenerateURL("test-state", userData)
	if err != nil {
		t.Fatalf("GenerateURL failed: %v", err)
	}

	// Verify URL is not empty
	if authURL == "" {
		t.Error("Generated auth URL should not be empty")
	}

	// Verify URL contains expected components
	if !strings.Contains(authURL, "accounts.google.com") {
		t.Error("Auth URL should contain Google's auth endpoint")
	}
	if !strings.Contains(authURL, "test-client-id") {
		t.Error("Auth URL should contain client ID")
	}
	if !strings.Contains(authURL, "localhost") {
		t.Error("Auth URL should contain redirect URL (localhost)")
	}
	if !strings.Contains(authURL, "state=") {
		t.Error("Auth URL should contain state parameter")
	}

	// Test with empty state (should generate one)
	authURL2, err := client.GenerateURL("", userData)
	if err != nil {
		t.Fatalf("GenerateURL with empty state failed: %v", err)
	}

	if authURL2 == "" {
		t.Error("Generated auth URL should not be empty")
	}

	// URLs should be different due to different states
	if authURL == authURL2 {
		t.Error("URLs with different states should be different")
	}
}

// TestClientValidateState tests state validation and decryption
func TestClientValidateState(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	originalState := "test-state-123"
	userData := TestUserData{
		UserID:   "test-user-456",
		ReturnTo: "/profile",
		Source:   "mobile",
	}

	// Generate URL to create encrypted state
	authURL, err := client.GenerateURL(originalState, userData)
	if err != nil {
		t.Fatalf("GenerateURL failed: %v", err)
	}

	// Extract encrypted state from URL
	encryptedState := extractStateFromAuthURL(authURL)
	if encryptedState == "" {
		t.Fatal("Could not extract encrypted state from auth URL")
	}

	// Test successful validation
	decryptedState, decryptedData, err := client.ValidateState(encryptedState)
	if err != nil {
		t.Fatalf("ValidateState failed: %v", err)
	}

	// Verify decrypted state matches original
	if decryptedState != originalState {
		t.Errorf("Decrypted state %q does not match original %q", decryptedState, originalState)
	}

	// Verify decrypted data matches original
	if !reflect.DeepEqual(decryptedData, userData) {
		t.Errorf("Decrypted data %+v does not match original %+v", decryptedData, userData)
	}

	// Test second validation (should fail - consume-once pattern)
	_, _, err = client.ValidateState(encryptedState)
	if err == nil {
		t.Error("Second validation should fail")
	}
	if !errors.Is(err, ErrStateNotFound) {
		t.Errorf("Second validation should return ErrStateNotFound, got %v", err)
	}
}

// TestClientValidateStateErrors tests error conditions in state validation
func TestClientValidateStateErrors(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	tests := []struct {
		name           string
		encryptedState string
		expectError    error
	}{
		{
			name:           "nonexistent state",
			encryptedState: "nonexistent-encrypted-state",
			expectError:    ErrStateNotFound,
		},
		{
			name:           "empty state",
			encryptedState: "",
			expectError:    ErrStateNotFound,
		},
		{
			name:           "invalid encrypted data",
			encryptedState: "invalid-base64-data-!@#$%",
			expectError:    ErrStateNotFound, // Will fail validation first
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := client.ValidateState(tt.encryptedState)
			if err == nil {
				t.Error("expected error but got none")
			}
			if !errors.Is(err, tt.expectError) {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

// TestClientExchange tests token exchange
func TestClientExchange(t *testing.T) {
	// Create mock token server
	expectedToken := &oauth2.Token{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Check for required parameters
		if err := r.ParseForm(); err != nil {
			http.Error(w, "Invalid form", http.StatusBadRequest)
			return
		}

		code := r.FormValue("code")
		if code != "test-auth-code" {
			http.Error(w, "Invalid code", http.StatusBadRequest)
			return
		}

		// Return mock token response
		response := map[string]interface{}{
			"access_token":  expectedToken.AccessToken,
			"refresh_token": expectedToken.RefreshToken,
			"token_type":    expectedToken.TokenType,
			"expires_in":    3600,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Create provider with mock server endpoint
	provider, err := NewGenericProvider(
		"test",
		oauth2.Endpoint{
			AuthURL:  server.URL + "/auth",
			TokenURL: server.URL,
		},
		server.URL+"/userinfo",
		[]string{"profile"},
	)
	if err != nil {
		t.Fatalf("NewGenericProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Test successful token exchange
	ctx := context.Background()
	token, err := client.Exchange(ctx, "test-auth-code")
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}

	if token.AccessToken != expectedToken.AccessToken {
		t.Errorf("Access token mismatch: got %q, want %q", token.AccessToken, expectedToken.AccessToken)
	}
	if token.RefreshToken != expectedToken.RefreshToken {
		t.Errorf("Refresh token mismatch: got %q, want %q", token.RefreshToken, expectedToken.RefreshToken)
	}
	if token.TokenType != expectedToken.TokenType {
		t.Errorf("Token type mismatch: got %q, want %q", token.TokenType, expectedToken.TokenType)
	}
}

// TestClientExchangeErrors tests error conditions in token exchange
func TestClientExchangeErrors(t *testing.T) {
	tests := []struct {
		name          string
		serverHandler http.HandlerFunc
		authCode      string
		expectError   bool
		errorContains string
	}{
		{
			name: "invalid authorization code",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "invalid_grant", http.StatusBadRequest)
			},
			authCode:      "invalid-code",
			expectError:   true,
			errorContains: "OAuth2 token exchange failed",
		},
		{
			name: "server error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			},
			authCode:      "test-code",
			expectError:   true,
			errorContains: "OAuth2 token exchange failed",
		},
		{
			name: "network timeout",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second * 10) // Longer than context timeout
			},
			authCode:      "test-code",
			expectError:   true,
			errorContains: "OAuth2 token exchange failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			provider, err := NewGenericProvider(
				"test",
				oauth2.Endpoint{
					AuthURL:  server.URL + "/auth",
					TokenURL: server.URL,
				},
				server.URL+"/userinfo",
				[]string{"profile"},
			)
			if err != nil {
				t.Fatalf("NewGenericProvider failed: %v", err)
			}

			client, err := NewClient[TestUserData](
				provider,
				"test-client-id",
				"test-client-secret",
				"http://localhost:8080/callback",
				"this-is-a-24-char-key-ok",
			)
			if err != nil {
				t.Fatalf("NewClient failed: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()

			_, err = client.Exchange(ctx, tt.authCode)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

// TestClientGetUserInfo tests user info retrieval
func TestClientGetUserInfo(t *testing.T) {
	expectedUserInfo := map[string]interface{}{
		"id":    "user123",
		"email": "test@example.com",
		"name":  "Test User",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Authorization header
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, "Invalid authorization", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedUserInfo)
	}))
	defer server.Close()

	provider, err := NewGenericProvider(
		"test",
		oauth2.Endpoint{
			AuthURL:  server.URL + "/auth",
			TokenURL: server.URL + "/token",
		},
		server.URL,
		[]string{"profile"},
	)
	if err != nil {
		t.Fatalf("NewGenericProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	token := &oauth2.Token{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
	}

	userInfo, err := client.GetUserInfo(token)
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}

	if !reflect.DeepEqual(userInfo, expectedUserInfo) {
		t.Errorf("User info mismatch: got %v, want %v", userInfo, expectedUserInfo)
	}
}

// TestClientSetStateStore tests custom state store functionality
func TestClientSetStateStore(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Create custom state store
	customStore := NewMemoryStateStore()
	client.SetStateStore(customStore)

	userData := TestUserData{UserID: "test", ReturnTo: "/test", Source: "test"}

	// Generate URL (should use custom store)
	authURL, err := client.GenerateURL("test-state", userData)
	if err != nil {
		t.Fatalf("GenerateURL failed: %v", err)
	}

	encryptedState := extractStateFromAuthURL(authURL)
	if encryptedState == "" {
		t.Fatal("Could not extract encrypted state")
	}

	// Validate state (should work with custom store)
	_, _, err = client.ValidateState(encryptedState)
	if err != nil {
		t.Fatalf("ValidateState failed: %v", err)
	}
}

// TestClientConcurrency tests concurrent client operations
func TestClientConcurrency(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"test-client-id",
		"test-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	const numGoroutines = 50
	const operationsPerGoroutine = 20

	var wg sync.WaitGroup
	errors := make(chan error, numGoroutines*operationsPerGoroutine)

	// Concurrent GenerateURL operations
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < operationsPerGoroutine; j++ {
				userData := TestUserData{
					UserID:   fmt.Sprintf("user-%d-%d", routineID, j),
					ReturnTo: fmt.Sprintf("/page-%d-%d", routineID, j),
					Source:   "concurrent-test",
				}
				state := fmt.Sprintf("state-%d-%d", routineID, j)

				_, err := client.GenerateURL(state, userData)
				if err != nil {
					errors <- fmt.Errorf("GenerateURL failed in goroutine %d, operation %d: %v", routineID, j, err)
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Error(err)
	}
}

// TestClientIntegration tests the complete OAuth2 flow
func TestClientIntegration(t *testing.T) {
	// This test simulates the complete OAuth2 flow without external dependencies

	expectedUserInfo := map[string]interface{}{
		"id":    "integration-user-123",
		"email": "integration@example.com",
		"name":  "Integration Test User",
	}

	// Create mock OAuth2 server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/token":
			// Token exchange endpoint
			if err := r.ParseForm(); err != nil {
				http.Error(w, "Invalid form", http.StatusBadRequest)
				return
			}

			response := map[string]interface{}{
				"access_token":  "integration-access-token",
				"refresh_token": "integration-refresh-token",
				"token_type":    "Bearer",
				"expires_in":    3600,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)

		case "/userinfo":
			// User info endpoint
			auth := r.Header.Get("Authorization")
			if !strings.HasPrefix(auth, "Bearer ") {
				http.Error(w, "Invalid authorization", http.StatusUnauthorized)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedUserInfo)

		default:
			http.Error(w, "Not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create provider with mock server
	provider, err := NewGenericProvider(
		"integration-test",
		oauth2.Endpoint{
			AuthURL:  server.URL + "/auth",
			TokenURL: server.URL + "/token",
		},
		server.URL+"/userinfo",
		[]string{"profile", "email"},
	)
	if err != nil {
		t.Fatalf("NewGenericProvider failed: %v", err)
	}

	// Create client
	client, err := NewClient[TestUserData](
		provider,
		"integration-client-id",
		"integration-client-secret",
		"http://localhost:8080/oauth/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		t.Fatalf("NewClient failed: %v", err)
	}

	// Step 1: Generate authorization URL
	userData := TestUserData{
		UserID:   "integration-user",
		ReturnTo: "/integration-dashboard",
		Source:   "integration-test",
	}

	authURL, err := client.GenerateURL("integration-state", userData)
	if err != nil {
		t.Fatalf("GenerateURL failed: %v", err)
	}

	// Verify auth URL structure
	if !strings.Contains(authURL, "integration-client-id") {
		t.Error("Auth URL should contain client ID")
	}

	// Step 2: Extract and validate state (simulating OAuth callback)
	encryptedState := extractStateFromAuthURL(authURL)
	if encryptedState == "" {
		t.Fatal("Could not extract encrypted state from auth URL")
	}

	decryptedState, decryptedData, err := client.ValidateState(encryptedState)
	if err != nil {
		t.Fatalf("ValidateState failed: %v", err)
	}

	if decryptedState != "integration-state" {
		t.Errorf("State mismatch: got %q, want %q", decryptedState, "integration-state")
	}

	if !reflect.DeepEqual(decryptedData, userData) {
		t.Errorf("User data mismatch: got %+v, want %+v", decryptedData, userData)
	}

	// Step 3: Exchange authorization code for token
	ctx := context.Background()
	token, err := client.Exchange(ctx, "integration-auth-code")
	if err != nil {
		t.Fatalf("Exchange failed: %v", err)
	}

	if token.AccessToken != "integration-access-token" {
		t.Errorf("Access token mismatch: got %q, want %q", token.AccessToken, "integration-access-token")
	}

	// Step 4: Get user info
	userInfo, err := client.GetUserInfo(token)
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}

	if !reflect.DeepEqual(userInfo, expectedUserInfo) {
		t.Errorf("User info mismatch: got %v, want %v", userInfo, expectedUserInfo)
	}

	// Integration test complete!
	t.Log("Complete OAuth2 flow integration test passed")
}

// Helper functions

// extractStateFromAuthURL extracts the state parameter from an OAuth2 authorization URL
func extractStateFromAuthURL(authURL string) string {
	parsedURL, err := url.Parse(authURL)
	if err != nil {
		return ""
	}

	return parsedURL.Query().Get("state")
}

// Benchmark tests

// BenchmarkClientGenerateURL benchmarks URL generation
func BenchmarkClientGenerateURL(b *testing.B) {
	provider, err := NewGoogleProvider()
	if err != nil {
		b.Fatalf("NewGoogleProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"bench-client-id",
		"bench-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		b.Fatalf("NewClient failed: %v", err)
	}

	userData := TestUserData{
		UserID:   "bench-user",
		ReturnTo: "/bench-dashboard",
		Source:   "benchmark",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := fmt.Sprintf("bench-state-%d", i)
		_, err := client.GenerateURL(state, userData)
		if err != nil {
			b.Fatalf("GenerateURL failed: %v", err)
		}
	}
}

// BenchmarkClientValidateState benchmarks state validation
func BenchmarkClientValidateState(b *testing.B) {
	provider, err := NewGoogleProvider()
	if err != nil {
		b.Fatalf("NewGoogleProvider failed: %v", err)
	}

	client, err := NewClient[TestUserData](
		provider,
		"bench-client-id",
		"bench-client-secret",
		"http://localhost:8080/callback",
		"this-is-a-24-char-key-ok",
	)
	if err != nil {
		b.Fatalf("NewClient failed: %v", err)
	}

	userData := TestUserData{
		UserID:   "bench-user",
		ReturnTo: "/bench-dashboard",
		Source:   "benchmark",
	}

	// Pre-generate encrypted states
	encryptedStates := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		state := fmt.Sprintf("bench-state-%d", i)
		authURL, err := client.GenerateURL(state, userData)
		if err != nil {
			b.Fatalf("GenerateURL failed: %v", err)
		}
		encryptedStates[i] = extractStateFromAuthURL(authURL)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := client.ValidateState(encryptedStates[i])
		if err != nil {
			b.Fatalf("ValidateState failed: %v", err)
		}
	}
}
