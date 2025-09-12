package oauth2

import (
	"net/http"

	"golang.org/x/oauth2"
)

// Provider defines the interface that all OAuth2 providers must implement.
// This interface abstracts the provider-specific details and allows the Client
// to work with any OAuth2 provider implementation.
//
// Usage Example:
//
//	provider := NewGoogleProvider()
//	provider.SetScopes([]string{"profile", "email", "https://www.googleapis.com/auth/drive"})
//
//	client := NewClient(provider, "your-client-id", "your-client-secret", "http://localhost/callback")
//	authURL := client.GenerateURL("random-state", userData)
//
// Implementation Notes:
//   - Providers should validate their configuration during construction
//   - Thread safety is the responsibility of individual implementations
//   - Providers should handle HTTP errors gracefully in GetUserInfo
//   - Scopes can be modified after creation to support different access levels
type Provider interface {
	// Name returns the unique identifier for this OAuth2 provider.
	// This is typically used for logging, debugging, and provider selection.
	// Common names include "google", "github", "facebook", etc.
	//
	// Returns:
	//   - string: lowercase provider name (e.g., "google")
	//
	// Example:
	//   provider.Name() // returns "google"
	Name() string

	// Scopes returns the current list of OAuth2 scopes that will be requested
	// during authorization. Scopes define the level of access your application
	// requires from the user's account.
	//
	// Returns:
	//   - []string: slice of scope strings as defined by the provider
	//
	// Example:
	//   scopes := provider.Scopes()
	//   // returns ["https://www.googleapis.com/auth/userinfo.profile", "https://www.googleapis.com/auth/userinfo.email"]
	Scopes() []string

	// Endpoint returns the OAuth2 endpoint configuration for this provider.
	// This includes the authorization URL and token exchange URL needed
	// for the OAuth2 flow.
	//
	// Returns:
	//   - oauth2.Endpoint: provider-specific endpoint configuration
	//
	// Example:
	//   endpoint := provider.Endpoint()
	//   // endpoint.AuthURL = "https://accounts.google.com/o/oauth2/auth"
	//   // endpoint.TokenURL = "https://oauth2.googleapis.com/token"
	Endpoint() oauth2.Endpoint

	// UserInfoURL returns the URL endpoint for retrieving user information
	// after successful OAuth2 authentication. This URL is used with an
	// authenticated HTTP client to fetch user profile data.
	//
	// Returns:
	//   - string: fully qualified URL for user info endpoint
	//
	// Example:
	//   url := provider.UserInfoURL()
	//   // returns "https://www.googleapis.com/oauth2/v3/userinfo"
	UserInfoURL() string

	// SetScopes updates the OAuth2 scopes that will be requested during
	// authorization. This allows dynamic scope configuration based on
	// application requirements or user preferences.
	//
	// Parameters:
	//   - scopes: slice of scope strings to request
	//
	// Usage Notes:
	//   - Scopes should be valid for the provider
	//   - Changes take effect on next authorization URL generation
	//   - Invalid scopes may cause authorization failures
	//
	// Example:
	//   provider.SetScopes([]string{
	//       "https://www.googleapis.com/auth/userinfo.profile",
	//       "https://www.googleapis.com/auth/userinfo.email",
	//       "https://www.googleapis.com/auth/drive.readonly",
	//   })
	SetScopes(scopes []string)

	// GetUserInfo fetches user profile information using an authenticated HTTP client.
	// The client must be configured with a valid OAuth2 token that includes
	// appropriate scopes for accessing user information.
	//
	// Parameters:
	//   - client: HTTP client configured with OAuth2 token
	//
	// Returns:
	//   - map[string]any: user profile data as key-value pairs
	//   - error: any error during HTTP request or response parsing
	//
	// Common user info fields include:
	//   - "id": unique user identifier
	//   - "email": user's email address
	//   - "name": user's display name
	//   - "picture": profile picture URL
	//
	// Error Conditions:
	//   - Network errors during HTTP request
	//   - Invalid or expired OAuth2 token
	//   - Insufficient scopes for user info access
	//   - Provider API errors or rate limiting
	//
	// Example:
	//   config := &oauth2.Config{...}
	//   token := &oauth2.Token{...}
	//   client := config.Client(context.Background(), token)
	//
	//   userInfo, err := provider.GetUserInfo(client)
	//   if err != nil {
	//       log.Printf("Failed to get user info: %v", err)
	//       return err
	//   }
	//
	//   email := userInfo["email"].(string)
	//   name := userInfo["name"].(string)
	GetUserInfo(client *http.Client) (map[string]any, error)
}
