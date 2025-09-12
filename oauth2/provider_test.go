package oauth2

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"golang.org/x/oauth2"
)

// TestGenericProvider tests the GenericProvider implementation
func TestGenericProvider(t *testing.T) {
	endpoint := oauth2.Endpoint{
		AuthURL:  "https://example.com/auth",
		TokenURL: "https://example.com/token",
	}
	userInfoURL := "https://example.com/userinfo"
	scopes := []string{"read", "write"}

	provider, err := NewGenericProvider("test", endpoint, userInfoURL, scopes)
	if err != nil {
		t.Fatalf("NewGenericProvider failed: %v", err)
	}

	// Test Name
	if provider.Name() != "test" {
		t.Errorf("Name() = %q, want %q", provider.Name(), "test")
	}

	// Test Endpoint
	if provider.Endpoint() != endpoint {
		t.Errorf("Endpoint() = %v, want %v", provider.Endpoint(), endpoint)
	}

	// Test UserInfoURL
	if provider.UserInfoURL() != userInfoURL {
		t.Errorf("UserInfoURL() = %q, want %q", provider.UserInfoURL(), userInfoURL)
	}

	// Test Scopes
	returnedScopes := provider.Scopes()
	if !reflect.DeepEqual(returnedScopes, scopes) {
		t.Errorf("Scopes() = %v, want %v", returnedScopes, scopes)
	}

	// Test that scopes is a copy (modifying returned slice shouldn't affect internal state)
	returnedScopes[0] = "modified"
	if provider.Scopes()[0] == "modified" {
		t.Error("Scopes() should return a copy, not the internal slice")
	}
}

// TestGenericProviderValidation tests parameter validation in NewGenericProvider
func TestGenericProviderValidation(t *testing.T) {
	validEndpoint := oauth2.Endpoint{
		AuthURL:  "https://example.com/auth",
		TokenURL: "https://example.com/token",
	}

	tests := []struct {
		name          string
		providerName  string
		endpoint      oauth2.Endpoint
		userInfoURL   string
		scopes        []string
		expectError   bool
		errorContains string
	}{
		{
			name:         "valid provider",
			providerName: "valid",
			endpoint:     validEndpoint,
			userInfoURL:  "https://example.com/userinfo",
			scopes:       []string{"read"},
			expectError:  false,
		},
		{
			name:          "empty provider name",
			providerName:  "",
			endpoint:      validEndpoint,
			userInfoURL:   "https://example.com/userinfo",
			scopes:        []string{"read"},
			expectError:   true,
			errorContains: "provider name cannot be empty",
		},
		{
			name:          "empty auth URL",
			providerName:  "test",
			endpoint:      oauth2.Endpoint{TokenURL: "https://example.com/token"},
			userInfoURL:   "https://example.com/userinfo",
			scopes:        []string{"read"},
			expectError:   true,
			errorContains: "authorization URL cannot be empty",
		},
		{
			name:          "empty token URL",
			providerName:  "test",
			endpoint:      oauth2.Endpoint{AuthURL: "https://example.com/auth"},
			userInfoURL:   "https://example.com/userinfo",
			scopes:        []string{"read"},
			expectError:   true,
			errorContains: "token URL cannot be empty",
		},
		{
			name:          "empty user info URL",
			providerName:  "test",
			endpoint:      validEndpoint,
			userInfoURL:   "",
			scopes:        []string{"read"},
			expectError:   true,
			errorContains: "user info URL cannot be empty",
		},
		{
			name:          "empty scope in list",
			providerName:  "test",
			endpoint:      validEndpoint,
			userInfoURL:   "https://example.com/userinfo",
			scopes:        []string{"read", "", "write"},
			expectError:   true,
			errorContains: "scope at index 1 cannot be empty",
		},
		{
			name:         "empty scopes list",
			providerName: "test",
			endpoint:     validEndpoint,
			userInfoURL:  "https://example.com/userinfo",
			scopes:       []string{},
			expectError:  false,
		},
		{
			name:         "nil scopes",
			providerName: "test",
			endpoint:     validEndpoint,
			userInfoURL:  "https://example.com/userinfo",
			scopes:       nil,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider, err := NewGenericProvider(tt.providerName, tt.endpoint, tt.userInfoURL, tt.scopes)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
					t.Errorf("error %q should contain %q", err.Error(), tt.errorContains)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if provider == nil {
					t.Error("provider should not be nil when no error")
				}
			}
		})
	}
}

// TestGenericProviderSetScopes tests the SetScopes method
func TestGenericProviderSetScopes(t *testing.T) {
	provider, err := NewGenericProvider(
		"test",
		oauth2.Endpoint{AuthURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
		"https://example.com/userinfo",
		[]string{"initial"},
	)
	if err != nil {
		t.Fatalf("NewGenericProvider failed: %v", err)
	}

	// Test setting new scopes
	newScopes := []string{"read", "write", "admin"}
	provider.SetScopes(newScopes)

	returnedScopes := provider.Scopes()
	if !reflect.DeepEqual(returnedScopes, newScopes) {
		t.Errorf("After SetScopes, Scopes() = %v, want %v", returnedScopes, newScopes)
	}

	// Test that modifying the input slice doesn't affect internal state
	newScopes[0] = "modified"
	if provider.Scopes()[0] == "modified" {
		t.Error("SetScopes should create internal copy, modification of input should not affect provider")
	}

	// Test setting scopes with empty strings (should be filtered out)
	scopesWithEmpty := []string{"valid1", "", "valid2", "", "valid3"}
	provider.SetScopes(scopesWithEmpty)

	expectedFiltered := []string{"valid1", "valid2", "valid3"}
	returnedScopes = provider.Scopes()
	if !reflect.DeepEqual(returnedScopes, expectedFiltered) {
		t.Errorf("SetScopes should filter empty strings: got %v, want %v", returnedScopes, expectedFiltered)
	}

	// Test setting empty scopes
	provider.SetScopes([]string{})
	if len(provider.Scopes()) != 0 {
		t.Error("SetScopes with empty slice should result in empty scopes")
	}

	// Test setting nil scopes
	provider.SetScopes(nil)
	if len(provider.Scopes()) != 0 {
		t.Error("SetScopes with nil should result in empty scopes")
	}
}

// TestGenericProviderGetUserInfo tests the GetUserInfo method
func TestGenericProviderGetUserInfo(t *testing.T) {
	// Create test server
	expectedUserInfo := map[string]any{
		"id":    "12345",
		"email": "test@example.com",
		"name":  "Test User",
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check for Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			http.Error(w, "Missing authorization", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expectedUserInfo)
	}))
	defer server.Close()

	provider, err := NewGenericProvider(
		"test",
		oauth2.Endpoint{AuthURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
		server.URL,
		[]string{"profile"},
	)
	if err != nil {
		t.Fatalf("NewGenericProvider failed: %v", err)
	}

	// Create HTTP client with auth header
	client := &http.Client{
		Transport: &addAuthHeaderTransport{
			base: http.DefaultTransport,
			auth: "Bearer test-token",
		},
	}

	// Test successful user info retrieval
	userInfo, err := provider.GetUserInfo(client)
	if err != nil {
		t.Fatalf("GetUserInfo failed: %v", err)
	}

	if !reflect.DeepEqual(userInfo, expectedUserInfo) {
		t.Errorf("GetUserInfo returned %v, want %v", userInfo, expectedUserInfo)
	}
}

// TestGenericProviderGetUserInfoErrors tests error conditions in GetUserInfo
func TestGenericProviderGetUserInfoErrors(t *testing.T) {
	tests := []struct {
		name          string
		serverHandler http.HandlerFunc
		expectError   bool
		errorContains string
	}{
		{
			name: "HTTP 401 error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
			},
			expectError:   true,
			errorContains: "user info request failed with status 401",
		},
		{
			name: "HTTP 403 error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Forbidden", http.StatusForbidden)
			},
			expectError:   true,
			errorContains: "user info request failed with status 403",
		},
		{
			name: "HTTP 500 error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			},
			expectError:   true,
			errorContains: "user info request failed with status 500",
		},
		{
			name: "Invalid JSON response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte("invalid json {"))
			},
			expectError:   true,
			errorContains: "failed to decode user info JSON",
		},
		{
			name: "Empty response",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.Write([]byte(""))
			},
			expectError:   true,
			errorContains: "failed to decode user info JSON",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			provider, err := NewGenericProvider(
				"test",
				oauth2.Endpoint{AuthURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
				server.URL,
				[]string{"profile"},
			)
			if err != nil {
				t.Fatalf("NewGenericProvider failed: %v", err)
			}

			client := &http.Client{}
			_, err = provider.GetUserInfo(client)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorContains != "" && !containsString(err.Error(), tt.errorContains) {
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

// TestGenericProviderNetworkError tests network-related errors
func TestGenericProviderNetworkError(t *testing.T) {
	provider, err := NewGenericProvider(
		"test",
		oauth2.Endpoint{AuthURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
		"http://nonexistent.localhost:99999/userinfo",
		[]string{"profile"},
	)
	if err != nil {
		t.Fatalf("NewGenericProvider failed: %v", err)
	}

	client := &http.Client{}
	_, err = provider.GetUserInfo(client)

	if err == nil {
		t.Error("expected network error but got none")
	}

	if !containsString(err.Error(), "failed to fetch user info") {
		t.Errorf("error should contain 'failed to fetch user info': %v", err)
	}
}

// TestGoogleProvider tests the pre-configured Google provider
func TestGoogleProvider(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	// Test provider name
	if provider.Name() != "google" {
		t.Errorf("Google provider name = %q, want %q", provider.Name(), "google")
	}

	// Test default scopes
	expectedScopes := GoogleDefaultScopes
	if !reflect.DeepEqual(provider.Scopes(), expectedScopes) {
		t.Errorf("Google provider default scopes = %v, want %v", provider.Scopes(), expectedScopes)
	}

	// Test user info URL
	expectedUserInfoURL := "https://www.googleapis.com/oauth2/v3/userinfo"
	if provider.UserInfoURL() != expectedUserInfoURL {
		t.Errorf("Google provider user info URL = %q, want %q", provider.UserInfoURL(), expectedUserInfoURL)
	}

	// Test endpoints (should use google.Endpoint)
	endpoint := provider.Endpoint()
	if endpoint.AuthURL == "" || endpoint.TokenURL == "" {
		t.Error("Google provider should have non-empty auth and token URLs")
	}
}

// TestGoogleProviderWithScopes tests the Google provider with custom scopes
func TestGoogleProviderWithScopes(t *testing.T) {
	customScopes := []string{"https://www.googleapis.com/auth/userinfo.email", "https://www.googleapis.com/auth/drive.readonly"}

	provider, err := NewGoogleProviderWithScopes(customScopes)
	if err != nil {
		t.Fatalf("NewGoogleProviderWithScopes failed: %v", err)
	}

	if provider.Name() != "google" {
		t.Errorf("Google provider name = %q, want %q", provider.Name(), "google")
	}

	if !reflect.DeepEqual(provider.Scopes(), customScopes) {
		t.Errorf("Google provider custom scopes = %v, want %v", provider.Scopes(), customScopes)
	}
}

// TestAddGoogleScopes tests the AddGoogleScopes helper function
func TestAddGoogleScopes(t *testing.T) {
	provider, err := NewGoogleProvider()
	if err != nil {
		t.Fatalf("NewGoogleProvider failed: %v", err)
	}

	initialScopes := provider.Scopes()

	// Add Gmail and Drive scopes
	AddGoogleScopes(provider, []string{"gmail", "drive"})

	newScopes := provider.Scopes()

	// Should contain all initial scopes
	for _, scope := range initialScopes {
		if !containsScope(newScopes, scope) {
			t.Errorf("Missing initial scope after AddGoogleScopes: %s", scope)
		}
	}

	// Should contain Gmail scopes
	for _, scope := range GoogleExtendedScopes["gmail"] {
		if !containsScope(newScopes, scope) {
			t.Errorf("Missing Gmail scope after AddGoogleScopes: %s", scope)
		}
	}

	// Should contain Drive scopes
	for _, scope := range GoogleExtendedScopes["drive"] {
		if !containsScope(newScopes, scope) {
			t.Errorf("Missing Drive scope after AddGoogleScopes: %s", scope)
		}
	}

	// Test with unknown service (should be ignored)
	scopesBefore := len(provider.Scopes())
	AddGoogleScopes(provider, []string{"unknown-service"})
	scopesAfter := len(provider.Scopes())

	if scopesBefore != scopesAfter {
		t.Error("AddGoogleScopes should ignore unknown services")
	}

	// Test deduplication
	AddGoogleScopes(provider, []string{"gmail"}) // Add Gmail again
	finalScopes := provider.Scopes()

	// Count Gmail scopes
	gmailScopeCount := 0
	for _, scope := range finalScopes {
		for _, gmailScope := range GoogleExtendedScopes["gmail"] {
			if scope == gmailScope {
				gmailScopeCount++
			}
		}
	}

	if gmailScopeCount != len(GoogleExtendedScopes["gmail"]) {
		t.Error("AddGoogleScopes should deduplicate scopes")
	}
}

// Helper functions

// addAuthHeaderTransport is a test helper that adds an Authorization header
type addAuthHeaderTransport struct {
	base http.RoundTripper
	auth string
}

func (t *addAuthHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", t.auth)
	return t.base.RoundTrip(req)
}

// containsString checks if a string contains a substring
func containsString(s, substr string) bool {
	return len(substr) == 0 || len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}()))
}

// containsScope checks if a scope exists in a slice of scopes
func containsScope(scopes []string, scope string) bool {
	for _, s := range scopes {
		if s == scope {
			return true
		}
	}
	return false
}

// BenchmarkGenericProviderScopes benchmarks the Scopes method
func BenchmarkGenericProviderScopes(b *testing.B) {
	provider, err := NewGenericProvider(
		"test",
		oauth2.Endpoint{AuthURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
		"https://example.com/userinfo",
		[]string{"scope1", "scope2", "scope3", "scope4", "scope5"},
	)
	if err != nil {
		b.Fatalf("NewGenericProvider failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = provider.Scopes()
	}
}

// BenchmarkGenericProviderSetScopes benchmarks the SetScopes method
func BenchmarkGenericProviderSetScopes(b *testing.B) {
	provider, err := NewGenericProvider(
		"test",
		oauth2.Endpoint{AuthURL: "https://example.com/auth", TokenURL: "https://example.com/token"},
		"https://example.com/userinfo",
		[]string{"initial"},
	)
	if err != nil {
		b.Fatalf("NewGenericProvider failed: %v", err)
	}

	scopes := []string{"scope1", "scope2", "scope3", "scope4", "scope5"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider.SetScopes(scopes)
	}
}
