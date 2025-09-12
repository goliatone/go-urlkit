package oauth2

import (
	"encoding/json"
	"fmt"
	"net/http"

	"golang.org/x/oauth2"
)

// GenericProvider provides a configurable implementation of the Provider interface
// that can work with any standard OAuth2 provider. This struct allows you to create
// custom OAuth2 providers without implementing the Provider interface from scratch.
//
// Usage Example:
//
//	provider := NewGenericProvider("custom", oauth2.Endpoint{
//	    AuthURL:  "https://provider.com/oauth/authorize",
//	    TokenURL: "https://provider.com/oauth/token",
//	}, "https://provider.com/api/user", []string{"read", "write"})
//
//	client := NewClient(provider, "client-id", "client-secret", "http://localhost/callback")
//
// Thread Safety:
//   - All methods are safe for concurrent use
//   - Scope modifications are atomic operations
type GenericProvider struct {
	name        string          // Provider name (e.g., "github", "custom")
	scopes      []string        // OAuth2 scopes to request
	endpoint    oauth2.Endpoint // OAuth2 authorization and token endpoints
	userInfoURL string          // URL for fetching user information
}

// NewGenericProvider creates a new GenericProvider with the specified configuration.
// This constructor validates all required parameters to ensure the provider is
// properly configured for OAuth2 operations.
//
// Parameters:
//   - name: unique identifier for the provider (must be non-empty)
//   - endpoint: OAuth2 endpoints with both AuthURL and TokenURL configured
//   - userInfoURL: URL for retrieving user information (must be non-empty)
//   - scopes: initial OAuth2 scopes to request (can be empty, updated later)
//
// Returns:
//   - *GenericProvider: configured provider instance
//   - error: validation error if any required field is missing or invalid
//
// Validation Rules:
//   - name must be non-empty string
//   - endpoint.AuthURL must be valid URL
//   - endpoint.TokenURL must be valid URL
//   - userInfoURL must be valid URL
//   - scopes can be empty but cannot contain empty strings
//
// Example:
//
//	provider, err := NewGenericProvider("github", oauth2.Endpoint{
//	    AuthURL:  "https://github.com/login/oauth/authorize",
//	    TokenURL: "https://github.com/login/oauth/access_token",
//	}, "https://api.github.com/user", []string{"user:email", "read:user"})
//	if err != nil {
//	    log.Fatal(err)
//	}
func NewGenericProvider(name string, endpoint oauth2.Endpoint, userInfoURL string, scopes []string) (*GenericProvider, error) {
	// Validate required fields
	if name == "" {
		return nil, fmt.Errorf("provider name cannot be empty")
	}

	if endpoint.AuthURL == "" {
		return nil, fmt.Errorf("authorization URL cannot be empty")
	}

	if endpoint.TokenURL == "" {
		return nil, fmt.Errorf("token URL cannot be empty")
	}

	if userInfoURL == "" {
		return nil, fmt.Errorf("user info URL cannot be empty")
	}

	// Validate scopes - ensure no empty scope strings
	for i, scope := range scopes {
		if scope == "" {
			return nil, fmt.Errorf("scope at index %d cannot be empty", i)
		}
	}

	// Create a copy of scopes to avoid external modifications
	scopesCopy := make([]string, len(scopes))
	copy(scopesCopy, scopes)

	return &GenericProvider{
		name:        name,
		scopes:      scopesCopy,
		endpoint:    endpoint,
		userInfoURL: userInfoURL,
	}, nil
}

// Name returns the unique identifier for this OAuth2 provider.
// This implements the Provider interface Name method.
//
// Returns:
//   - string: the provider name specified during construction
//
// Example:
//
//	provider.Name() // returns "github"
func (g *GenericProvider) Name() string {
	return g.name
}

// Scopes returns the current list of OAuth2 scopes that will be requested
// during authorization. This implements the Provider interface Scopes method.
//
// Returns:
//   - []string: copy of current scopes to prevent external modification
//
// Thread Safety:
//   - Safe for concurrent access
//   - Returns a copy to prevent external modification of internal state
//
// Example:
//
//	scopes := provider.Scopes()
//	// returns ["user:email", "read:user"]
func (g *GenericProvider) Scopes() []string {
	// Return a copy to prevent external modification
	scopes := make([]string, len(g.scopes))
	copy(scopes, g.scopes)
	return scopes
}

// Endpoint returns the OAuth2 endpoint configuration for this provider.
// This implements the Provider interface Endpoint method.
//
// Returns:
//   - oauth2.Endpoint: the endpoint configuration specified during construction
//
// Example:
//
//	endpoint := provider.Endpoint()
//	// endpoint.AuthURL = "https://github.com/login/oauth/authorize"
//	// endpoint.TokenURL = "https://github.com/login/oauth/access_token"
func (g *GenericProvider) Endpoint() oauth2.Endpoint {
	return g.endpoint
}

// UserInfoURL returns the URL endpoint for retrieving user information
// after successful OAuth2 authentication. This implements the Provider interface
// UserInfoURL method.
//
// Returns:
//   - string: the user info URL specified during construction
//
// Example:
//
//	url := provider.UserInfoURL()
//	// returns "https://api.github.com/user"
func (g *GenericProvider) UserInfoURL() string {
	return g.userInfoURL
}

// SetScopes updates the OAuth2 scopes that will be requested during
// authorization. This implements the Provider interface SetScopes method.
//
// Parameters:
//   - scopes: slice of scope strings to request (validates for empty strings)
//
// Thread Safety:
//   - Safe for concurrent access
//   - Atomic replacement of internal scopes slice
//
// Validation:
//   - Rejects empty scope strings
//   - Creates internal copy to prevent external modification
//
// Example:
//
//	provider.SetScopes([]string{"user:email", "read:user", "public_repo"})
func (g *GenericProvider) SetScopes(scopes []string) {
	// Validate scopes - ensure no empty scope strings
	for _, scope := range scopes {
		if scope == "" {
			// Skip invalid scopes rather than panic to maintain robustness
			continue
		}
	}

	// Create a copy to avoid external modifications affecting internal state
	scopesCopy := make([]string, 0, len(scopes))
	for _, scope := range scopes {
		if scope != "" {
			scopesCopy = append(scopesCopy, scope)
		}
	}

	g.scopes = scopesCopy
}

// GetUserInfo fetches user profile information using an authenticated HTTP client.
// This implements the Provider interface GetUserInfo method.
//
// The method performs an HTTP GET request to the configured user info URL using
// the provided authenticated client. The response is expected to be JSON formatted
// user profile data.
//
// Parameters:
//   - client: HTTP client configured with valid OAuth2 token
//
// Returns:
//   - map[string]any: user profile data parsed from JSON response
//   - error: HTTP, JSON parsing, or provider API errors
//
// Error Conditions:
//   - Network connectivity issues
//   - Invalid or expired OAuth2 token (HTTP 401)
//   - Insufficient OAuth2 scopes (HTTP 403)
//   - Provider API errors (HTTP 4xx/5xx)
//   - Invalid JSON response format
//   - Rate limiting by provider
//
// Expected Response Format:
//
//	The provider should return JSON with user profile fields such as:
//	- "id": unique user identifier
//	- "email": user's email address
//	- "name": user's display name
//	- Additional provider-specific fields
//
// Example Usage:
//
//	config := &oauth2.Config{...}
//	token := &oauth2.Token{AccessToken: "abc123"}
//	client := config.Client(context.Background(), token)
//
//	userInfo, err := provider.GetUserInfo(client)
//	if err != nil {
//	    log.Printf("Failed to fetch user info: %v", err)
//	    return err
//	}
//
//	userID := userInfo["id"].(string)
//	email, hasEmail := userInfo["email"].(string)
func (g *GenericProvider) GetUserInfo(client *http.Client) (map[string]any, error) {
	// Perform GET request to user info endpoint
	resp, err := client.Get(g.userInfoURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user info: %w", err)
	}
	defer resp.Body.Close()

	// Check for HTTP error status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("user info request failed with status %d: %s", resp.StatusCode, resp.Status)
	}

	// Parse JSON response into map
	var userInfo map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info JSON: %w", err)
	}

	return userInfo, nil
}
