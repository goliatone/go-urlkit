package oauth2

import (
	"golang.org/x/oauth2/google"
)

// Default Google OAuth2 scopes for profile and email access.
// These scopes provide basic user identification and email information.
var (
	// GoogleDefaultScopes are the basic scopes required for user identification.
	// These scopes allow access to:
	//   - User's basic profile information (ID, name, profile picture)
	//   - User's email address (primary email)
	GoogleDefaultScopes = []string{
		"https://www.googleapis.com/auth/userinfo.profile",
		"https://www.googleapis.com/auth/userinfo.email",
	}

	// GoogleExtendedScopes provides access to additional Google services.
	// These can be added to the default scopes for enhanced functionality.
	GoogleExtendedScopes = map[string][]string{
		"gmail": {
			"https://www.googleapis.com/auth/gmail.readonly",
			"https://www.googleapis.com/auth/gmail.send",
			"https://www.googleapis.com/auth/gmail.compose",
		},
		"drive": {
			"https://www.googleapis.com/auth/drive.readonly",
			"https://www.googleapis.com/auth/drive.file",
		},
		"calendar": {
			"https://www.googleapis.com/auth/calendar.readonly",
			"https://www.googleapis.com/auth/calendar.events",
		},
		"docs": {
			"https://www.googleapis.com/auth/documents.readonly",
			"https://www.googleapis.com/auth/documents",
		},
		"sheets": {
			"https://www.googleapis.com/auth/spreadsheets.readonly",
			"https://www.googleapis.com/auth/spreadsheets",
		},
	}
)

// NewGoogleProvider creates a pre-configured GenericProvider for Google OAuth2.
// This function provides a convenient way to create a Google OAuth2 provider with
// standard configuration and default scopes for profile and email access.
//
// The provider is configured with:
//   - Provider name: "google"
//   - Google's OAuth2 endpoints (from golang.org/x/oauth2/google package)
//   - User info URL: https://www.googleapis.com/oauth2/v3/userinfo
//   - Default scopes: profile and email access
//
// Additional scopes can be added using the SetScopes method or by using
// NewGoogleProviderWithScopes for more control over initial scope configuration.
//
// Returns:
//   - *GenericProvider: configured Google OAuth2 provider
//   - error: should never return error under normal conditions (all parameters are predefined)
//
// Usage Example:
//
//	// Basic Google provider with profile and email access
//	provider, err := NewGoogleProvider()
//	if err != nil {
//	    log.Fatal(err) // Should not happen with valid predefined configuration
//	}
//
//	// Create OAuth2 client
//	client := NewClient(provider, "your-client-id", "your-client-secret", "http://localhost/callback")
//
//	// Generate authorization URL
//	authURL, err := client.GenerateURL("random-state", userData)
//
// Extended Usage:
//
//	// Add additional scopes after creation
//	provider.SetScopes(append(provider.Scopes(), GoogleExtendedScopes["gmail"]...))
//
// Thread Safety:
//   - Safe for concurrent use after creation
//   - Scope modifications are thread-safe
func NewGoogleProvider() (*GenericProvider, error) {
	return NewGenericProvider(
		"google",
		google.Endpoint,
		"https://www.googleapis.com/oauth2/v3/userinfo",
		GoogleDefaultScopes,
	)
}

// NewGoogleProviderWithScopes creates a Google OAuth2 provider with custom scopes.
// This function allows you to specify exactly which scopes should be requested
// during OAuth2 authorization, providing more control than the default provider.
//
// Parameters:
//   - scopes: slice of OAuth2 scope strings to request from Google
//
// Common scope combinations:
//   - Basic: GoogleDefaultScopes (profile + email)
//   - Gmail: append(GoogleDefaultScopes, GoogleExtendedScopes["gmail"]...)
//   - Drive: append(GoogleDefaultScopes, GoogleExtendedScopes["drive"]...)
//   - Multiple: append(GoogleDefaultScopes, GoogleExtendedScopes["gmail"]..., GoogleExtendedScopes["drive"]...)
//
// Returns:
//   - *GenericProvider: configured Google OAuth2 provider with specified scopes
//   - error: validation error if any scope is invalid (empty string)
//
// Usage Examples:
//
//	// Provider with Gmail access
//	gmailScopes := append(GoogleDefaultScopes, GoogleExtendedScopes["gmail"]...)
//	provider, err := NewGoogleProviderWithScopes(gmailScopes)
//
//	// Provider with multiple services
//	allScopes := append(GoogleDefaultScopes,
//	    append(GoogleExtendedScopes["gmail"], GoogleExtendedScopes["drive"]...)...)
//	provider, err := NewGoogleProviderWithScopes(allScopes)
//
//	// Custom scopes
//	customScopes := []string{
//	    "https://www.googleapis.com/auth/userinfo.email",
//	    "https://www.googleapis.com/auth/drive.readonly",
//	}
//	provider, err := NewGoogleProviderWithScopes(customScopes)
//
// Validation:
//   - Empty scopes slice is allowed (can be set later)
//   - Empty scope strings are rejected with validation error
//   - Invalid scope URLs are accepted (validation happens at Google's end)
func NewGoogleProviderWithScopes(scopes []string) (*GenericProvider, error) {
	return NewGenericProvider(
		"google",
		google.Endpoint,
		"https://www.googleapis.com/oauth2/v3/userinfo",
		scopes,
	)
}

// AddGoogleScopes is a convenience function that adds predefined Google service scopes
// to an existing provider. This function helps avoid manual scope string management
// when extending a Google provider with additional service access.
//
// Parameters:
//   - provider: existing Google provider (should be created with NewGoogleProvider)
//   - services: slice of service names to add scopes for
//
// Supported service names:
//   - "gmail": Gmail read, send, and compose access
//   - "drive": Google Drive read and file access
//   - "calendar": Google Calendar read and event access
//   - "docs": Google Docs read and write access
//   - "sheets": Google Sheets read and write access
//
// The function preserves existing scopes and adds new ones without duplication.
//
// Usage Examples:
//
//	// Add Gmail and Drive access to existing provider
//	provider, _ := NewGoogleProvider()
//	AddGoogleScopes(provider, []string{"gmail", "drive"})
//
//	// Add all available services
//	AddGoogleScopes(provider, []string{"gmail", "drive", "calendar", "docs", "sheets"})
//
// Error Handling:
//   - Unknown service names are silently ignored
//   - Function never panics, maintains provider stability
//   - Invalid service names don't affect existing scopes
//
// Thread Safety:
//   - Safe for concurrent use
//   - Uses provider's thread-safe SetScopes method
func AddGoogleScopes(provider *GenericProvider, services []string) {
	currentScopes := provider.Scopes()

	// Collect all new scopes to add
	var newScopes []string
	for _, service := range services {
		if scopes, exists := GoogleExtendedScopes[service]; exists {
			newScopes = append(newScopes, scopes...)
		}
	}

	// Combine current and new scopes, removing duplicates
	allScopes := append(currentScopes, newScopes...)

	// Remove duplicates by using a map
	scopeSet := make(map[string]bool)
	var uniqueScopes []string

	for _, scope := range allScopes {
		if !scopeSet[scope] {
			scopeSet[scope] = true
			uniqueScopes = append(uniqueScopes, scope)
		}
	}

	provider.SetScopes(uniqueScopes)
}
