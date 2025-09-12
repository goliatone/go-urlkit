package oauth2

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"golang.org/x/oauth2"
)

// Client provides a generic OAuth2 client that can work with any Provider implementation.
// It handles the complete OAuth2 authorization code flow with state management and encryption.
//
// The client is generic over type T, allowing you to attach arbitrary user data to the
// OAuth2 state for retrieval after the authorization flow completes.
//
// Usage Example:
//
//	// Create a Google provider
//	provider, err := NewGoogleProvider()
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Create client with user data type
//	type UserContext struct {
//	    UserID   string `json:"user_id"`
//	    ReturnTo string `json:"return_to"`
//	}
//
//	client, err := NewClient[UserContext](provider, "client-id", "client-secret", "http://localhost/callback", "your-encryption-key")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// Generate authorization URL with user context
//	userCtx := UserContext{UserID: "123", ReturnTo: "/dashboard"}
//	authURL, err := client.GenerateURL("random-state", userCtx)
//
//	// Handle callback
//	originalState, userCtx, err := client.ValidateState(encryptedState)
//	token, err := client.Exchange(ctx, code)
//	userInfo, err := client.GetUserInfo(token)
//
// Thread Safety:
//   - All methods are safe for concurrent use
//   - State management is handled by the underlying StateStore implementation
//   - Provider operations are thread-safe as per Provider interface contract
type Client[T any] struct {
	config        *oauth2.Config // OAuth2 configuration for token exchange
	provider      Provider       // Provider implementation for OAuth2 endpoints and user info
	states        StateStore     // State storage for CSRF protection
	encryptionKey string         // Encryption key for state data (24-32 characters)
}

// NewClient creates a new OAuth2 client with the specified provider and configuration.
// The client handles the complete OAuth2 authorization code flow with state management.
//
// Parameters:
//   - provider: Provider implementation defining OAuth2 endpoints and user info URL
//   - clientID: OAuth2 client identifier from your OAuth app registration
//   - clientSecret: OAuth2 client secret from your OAuth app registration
//   - redirectURL: callback URL where the provider will send authorization results
//   - encryptionKey: key for encrypting state data (must be 24-32 characters for AES)
//
// Returns:
//   - *Client[T]: configured OAuth2 client
//   - error: configuration validation errors
//
// Validation Rules:
//   - provider cannot be nil
//   - clientID must be non-empty
//   - clientSecret must be non-empty
//   - redirectURL must be non-empty
//   - encryptionKey must be 24-32 characters (AES key size requirement)
//
// The client automatically configures OAuth2 scopes, endpoints, and user info URL
// from the provider implementation, ensuring consistency across the OAuth2 flow.
//
// Example:
//
//	provider, _ := NewGoogleProvider()
//	client, err := NewClient[MyUserData](
//	    provider,
//	    "your-google-client-id",
//	    "your-google-client-secret",
//	    "https://yourapp.com/oauth/callback",
//	    "your-24-to-32-char-encryption-key",
//	)
//	if err != nil {
//	    log.Fatalf("Failed to create OAuth2 client: %v", err)
//	}
//
// Security Notes:
//   - Store client credentials securely (environment variables, secret management)
//   - Use HTTPS for redirect URLs in production
//   - Generate strong encryption keys and store them securely
//   - Validate redirect URLs match your registered OAuth app configuration
func NewClient[T any](provider Provider, clientID, clientSecret, redirectURL, encryptionKey string) (*Client[T], error) {
	// Validate required parameters
	if provider == nil {
		return nil, fmt.Errorf("provider cannot be nil")
	}

	if clientID == "" {
		return nil, fmt.Errorf("client ID cannot be empty")
	}

	if clientSecret == "" {
		return nil, fmt.Errorf("client secret cannot be empty")
	}

	if redirectURL == "" {
		return nil, fmt.Errorf("redirect URL cannot be empty")
	}

	// Validate encryption key length (AES requirements: 16, 24, or 32 bytes)
	keyLen := len(encryptionKey)
	if keyLen < 24 || keyLen > 32 {
		return nil, fmt.Errorf("encryption key must be 24-32 characters, got %d", keyLen)
	}

	// Create OAuth2 configuration using provider details
	config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       provider.Scopes(),
		Endpoint:     provider.Endpoint(),
	}

	return &Client[T]{
		config:        config,
		provider:      provider,
		states:        NewMemoryStateStore(), // Default to memory store, can be replaced
		encryptionKey: encryptionKey,
	}, nil
}

// SetStateStore replaces the default StateStore with a custom implementation.
// This allows you to use persistent storage (Redis, database, etc.) instead of
// the default in-memory store.
//
// Parameters:
//   - store: StateStore implementation to use for state management
//
// Usage Example:
//
//	// Use custom Redis-backed state store
//	redisStore := NewRedisStateStore(redisClient)
//	client.SetStateStore(redisStore)
//
//	// Use database-backed state store
//	dbStore := NewDatabaseStateStore(db)
//	client.SetStateStore(dbStore)
//
// Thread Safety:
//   - Safe to call before starting OAuth2 flows
//   - Should not be called concurrently with active OAuth2 flows
//   - New StateStore must be thread-safe for concurrent use
func (c *Client[T]) SetStateStore(store StateStore) {
	c.states = store
}

// GenerateURL creates an OAuth2 authorization URL with encrypted state containing user data.
// This method initiates the OAuth2 flow by generating a URL that redirects users to the
// OAuth2 provider for authentication and authorization.
//
// Parameters:
//   - state: base state string for CSRF protection (if empty, generates UUID)
//   - userData: arbitrary data to encrypt and embed in the state parameter
//
// Returns:
//   - string: authorization URL to redirect the user to
//   - error: state encryption or URL generation errors
//
// Security Features:
//   - Encrypts user data within the state parameter
//   - Stores state for later validation (CSRF protection)
//   - Uses provider-specific OAuth2 endpoints and scopes
//   - Includes offline access and approval prompt for refresh tokens
//
// The generated URL includes:
//   - OAuth2 authorization endpoint from provider
//   - Client ID and redirect URL from configuration
//   - Requested scopes from provider
//   - Encrypted state parameter with user data
//   - Access type and approval prompt for optimal token handling
//
// Example:
//
//	type SessionData struct {
//	    UserID   string `json:"user_id"`
//	    ReturnTo string `json:"return_to"`
//	}
//
//	sessionData := SessionData{
//	    UserID:   "user123",
//	    ReturnTo: "/dashboard",
//	}
//
//	authURL, err := client.GenerateURL("random-csrf-token", sessionData)
//	if err != nil {
//	    log.Printf("Failed to generate auth URL: %v", err)
//	    return err
//	}
//
//	// Redirect user to authURL
//	http.Redirect(w, r, authURL, http.StatusTemporaryRedirect)
//
// Error Conditions:
//   - State encryption failure (invalid encryption key)
//   - JSON serialization failure (invalid user data)
//   - State storage failure (StateStore implementation error)
func (c *Client[T]) GenerateURL(state string, userData T) (string, error) {
	// Generate state if not provided
	if state == "" {
		state = uuid.New().String()
	}

	// Encrypt state with user data
	encryptedState, err := EncryptState([]byte(c.encryptionKey), state, userData)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt state: %w", err)
	}

	// Store encrypted state for later validation
	if !c.states.Store(encryptedState) {
		return "", fmt.Errorf("failed to store state for validation")
	}

	// Build authorization URL with encrypted state
	authURL := c.config.AuthCodeURL(
		encryptedState,
		oauth2.AccessTypeOffline, // Request refresh tokens
		oauth2.ApprovalForce,     // Force approval prompt for consistent UX
	)

	// Clean up URL encoding for better readability
	authURL = strings.ReplaceAll(authURL, "\\u0026", "&")

	return authURL, nil
}

// ValidateState verifies and decrypts an encrypted state parameter from OAuth2 callback.
// This method implements CSRF protection by validating that the state was previously
// generated and stored by this client instance.
//
// Parameters:
//   - encryptedState: encrypted state parameter from OAuth2 callback URL
//
// Returns:
//   - string: original state string that was encrypted
//   - T: decrypted user data of type T
//   - error: validation, decryption, or deserialization errors
//
// Security Features:
//   - Validates state exists in StateStore (prevents CSRF attacks)
//   - Removes state from storage after validation (prevents replay attacks)
//   - Decrypts and validates state data integrity
//   - Returns typed user data for application use
//
// This method should be called when handling the OAuth2 callback to ensure
// the authorization response corresponds to a request initiated by your application.
//
// Example:
//
//	// Handle OAuth2 callback
//	func handleCallback(w http.ResponseWriter, r *http.Request) {
//	    code := r.URL.Query().Get("code")
//	    state := r.URL.Query().Get("state")
//
//	    // Validate state and retrieve user data
//	    originalState, userData, err := client.ValidateState(state)
//	    if err != nil {
//	        if errors.Is(err, ErrStateNotFound) {
//	            http.Error(w, "Invalid or expired authorization request", http.StatusBadRequest)
//	            return
//	        }
//	        http.Error(w, "State validation failed", http.StatusInternalServerError)
//	        return
//	    }
//
//	    // Continue with OAuth2 flow
//	    token, err := client.Exchange(r.Context(), code)
//	    // ... handle token and user data
//	}
//
// Error Conditions:
//   - ErrStateNotFound: state not found or already consumed (potential CSRF attack)
//   - ErrDecryptionFailed: invalid encryption key or corrupted state data
//   - ErrDeserializationFailed: state data doesn't match expected type T
func (c *Client[T]) ValidateState(encryptedState string) (string, T, error) {
	var empty T

	// Validate state exists and remove it (consume-once pattern)
	if !c.states.Validate(encryptedState) {
		return "", empty, ErrStateNotFound
	}

	// Decrypt and deserialize state data
	return DecryptState[T]([]byte(c.encryptionKey), encryptedState)
}

// Exchange trades an authorization code for OAuth2 access and refresh tokens.
// This method completes the OAuth2 authorization code flow by exchanging the
// authorization code received in the callback for usable access tokens.
//
// Parameters:
//   - ctx: context for the HTTP request (timeout, cancellation, etc.)
//   - code: authorization code from OAuth2 callback URL
//
// Returns:
//   - *oauth2.Token: access token, refresh token, and token metadata
//   - error: network, authentication, or OAuth2 protocol errors
//
// The returned token contains:
//   - AccessToken: token for making authenticated API requests
//   - RefreshToken: token for obtaining new access tokens (if available)
//   - Expiry: when the access token expires
//   - TokenType: typically "Bearer" for OAuth2
//
// Token Usage:
//   - Use AccessToken in Authorization header: "Bearer <access_token>"
//   - Store RefreshToken securely for obtaining new access tokens
//   - Check Expiry before using token, refresh if expired
//   - Use oauth2.Config.Client(ctx, token) for authenticated HTTP client
//
// Example:
//
//	// Exchange authorization code for token
//	token, err := client.Exchange(ctx, authCode)
//	if err != nil {
//	    log.Printf("Token exchange failed: %v", err)
//	    return err
//	}
//
//	// Create authenticated HTTP client
//	httpClient := client.config.Client(ctx, token)
//
//	// Make authenticated requests
//	userInfo, err := client.GetUserInfo(token)
//
//	// Store tokens securely for later use
//	err = storeTokens(userID, token.AccessToken, token.RefreshToken)
//
// Error Conditions:
//   - Invalid or expired authorization code
//   - Network connectivity issues
//   - OAuth2 provider errors (invalid_grant, etc.)
//   - Client authentication failures
func (c *Client[T]) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	token, err := c.config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("OAuth2 token exchange failed: %w", err)
	}
	return token, nil
}

// GetUserInfo retrieves user profile information using an OAuth2 access token.
// This method uses the provider's user info endpoint to fetch authenticated
// user data after successful OAuth2 token exchange.
//
// Parameters:
//   - token: OAuth2 token with sufficient scopes for user info access
//
// Returns:
//   - map[string]any: user profile data as key-value pairs
//   - error: network, authentication, or parsing errors
//
// Common user info fields (provider-dependent):
//   - "id": unique user identifier
//   - "email": user's email address
//   - "name": user's display name
//   - "picture": profile picture URL
//   - Provider-specific fields (avatar_url, login, etc.)
//
// The method automatically:
//   - Creates authenticated HTTP client from token
//   - Calls provider-specific user info endpoint
//   - Handles HTTP errors and JSON parsing
//   - Returns structured user data
//
// Example:
//
//	// Get user info after successful token exchange
//	token, err := client.Exchange(ctx, code)
//	if err != nil {
//	    return err
//	}
//
//	userInfo, err := client.GetUserInfo(token)
//	if err != nil {
//	    log.Printf("Failed to get user info: %v", err)
//	    return err
//	}
//
//	// Extract user details
//	userID, ok := userInfo["id"].(string)
//	if !ok {
//	    return fmt.Errorf("user ID not found in response")
//	}
//
//	email, hasEmail := userInfo["email"].(string)
//	name, hasName := userInfo["name"].(string)
//
//	// Create or update user in your system
//	err = createOrUpdateUser(userID, email, name, userInfo)
//
// Error Conditions:
//   - Token expired or invalid (HTTP 401)
//   - Insufficient scopes for user info access (HTTP 403)
//   - Network connectivity issues
//   - Provider API errors or rate limiting
//   - Invalid JSON response format
//
// Scope Requirements:
//   - Most providers require "profile" scope for basic user info
//   - Email access typically requires "email" or "userinfo.email" scope
//   - Check provider documentation for specific scope requirements
func (c *Client[T]) GetUserInfo(token *oauth2.Token) (map[string]any, error) {
	// Create authenticated HTTP client
	httpClient := c.config.Client(context.Background(), token)

	// Use provider's GetUserInfo method
	return c.provider.GetUserInfo(httpClient)
}
