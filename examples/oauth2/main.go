package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	"github.com/goliatone/go-urlkit/oauth2"
)

// UserSessionData represents the data we want to preserve during the OAuth2 flow.
// This data will be encrypted and embedded in the OAuth2 state parameter,
// allowing us to retrieve it after the user completes authorization.
type UserSessionData struct {
	UserID    string    `json:"user_id"`   // ID of the user initiating OAuth
	ReturnTo  string    `json:"return_to"` // URL to redirect to after OAuth completion
	Timestamp time.Time `json:"timestamp"` // When the OAuth flow was initiated
	Source    string    `json:"source"`    // Source of the OAuth request (web, mobile, etc.)
}

// OAuth2Example demonstrates the complete OAuth2 authorization code flow
// using the new generic OAuth2 client with Google as the provider.
//
// This example shows:
// 1. Creating a Google OAuth2 provider
// 2. Initializing the OAuth2 client with configuration
// 3. Generating authorization URL with encrypted user data
// 4. Handling the OAuth2 callback
// 5. Validating state and retrieving user data
// 6. Exchanging authorization code for access token
// 7. Fetching user profile information
// 8. Error handling throughout the flow
func main() {
	fmt.Println("=== OAuth2 Client Example ===")
	fmt.Println("This example demonstrates the complete OAuth2 flow using the generic client")
	fmt.Println()

	// Step 1: Create Google OAuth2 provider
	fmt.Println("1. Creating Google OAuth2 provider...")
	provider, err := oauth2.NewGoogleProvider()
	if err != nil {
		log.Fatalf("Failed to create Google provider: %v", err)
	}
	fmt.Printf("   ✓ Provider created: %s\n", provider.Name())
	fmt.Printf("   ✓ Default scopes: %v\n", provider.Scopes())
	fmt.Println()

	// Step 2: Configure OAuth2 client credentials
	// In a real application, these would come from environment variables or secure configuration
	fmt.Println("2. Configuring OAuth2 client...")
	clientID := getEnvOrDefault("GOOGLE_CLIENT_ID", "your-google-client-id")
	clientSecret := getEnvOrDefault("GOOGLE_CLIENT_SECRET", "your-google-client-secret")
	redirectURL := getEnvOrDefault("OAUTH_REDIRECT_URL", "http://localhost:8080/oauth/callback")
	encryptionKey := getEnvOrDefault("OAUTH_ENCRYPTION_KEY", "this-is-a-24-char-key-ok") // 24 characters

	fmt.Printf("   • Client ID: %s\n", maskSecret(clientID))
	fmt.Printf("   • Redirect URL: %s\n", redirectURL)
	fmt.Printf("   • Encryption key length: %d chars\n", len(encryptionKey))

	// Step 3: Create OAuth2 client with user session data type
	fmt.Println("   Creating OAuth2 client...")
	client, err := oauth2.NewClient[UserSessionData](
		provider,
		clientID,
		clientSecret,
		redirectURL,
		encryptionKey,
	)
	if err != nil {
		// Demonstrate error handling for client creation
		handleClientCreationError(err)
		return
	}
	fmt.Printf("   ✓ OAuth2 client created successfully\n")
	fmt.Println()

	// Step 4: Demonstrate extending scopes for additional Google services
	fmt.Println("3. Extending OAuth2 scopes for Gmail and Drive access...")
	oauth2.AddGoogleScopes(provider, []string{"gmail", "drive"})
	fmt.Printf("   ✓ Extended scopes: %v\n", provider.Scopes())
	fmt.Println()

	// Step 5: Generate authorization URL with user session data
	fmt.Println("4. Generating authorization URL...")
	sessionData := UserSessionData{
		UserID:    "user123",
		ReturnTo:  "/dashboard",
		Timestamp: time.Now(),
		Source:    "web-example",
	}

	authURL, err := client.GenerateURL("random-csrf-token", sessionData)
	if err != nil {
		log.Printf("❌ Failed to generate authorization URL: %v", err)
		return
	}

	fmt.Printf("   ✓ Authorization URL generated\n")
	fmt.Printf("   ✓ Session data encrypted in state parameter\n")
	fmt.Printf("   URL: %s\n", authURL)
	fmt.Println()

	// Step 6: Demonstrate callback handling (simulated)
	fmt.Println("5. Simulating OAuth2 callback handling...")
	fmt.Println("   (In a real application, this would be handled by your web server)")

	// Simulate the callback parameters that would come from the OAuth provider
	// In reality, these would be extracted from the HTTP request query parameters
	simulatedState := extractStateFromURL(authURL) // Extract state for simulation
	simulatedCode := "4/mock-authorization-code"   // Mock authorization code

	fmt.Printf("   • Received state: %s...\n", simulatedState[:20]+"***")
	fmt.Printf("   • Received code: %s\n", simulatedCode)
	fmt.Println()

	// Step 7: Validate state and retrieve user session data
	fmt.Println("6. Validating OAuth2 state and retrieving session data...")
	originalState, retrievedSession, err := client.ValidateState(simulatedState)
	if err != nil {
		// Demonstrate different types of state validation errors
		handleStateValidationError(err)
		return
	}

	fmt.Printf("   ✓ State validated successfully\n")
	fmt.Printf("   ✓ Original state: %s\n", originalState)
	fmt.Printf("   ✓ Retrieved session data:\n")
	fmt.Printf("     - User ID: %s\n", retrievedSession.UserID)
	fmt.Printf("     - Return URL: %s\n", retrievedSession.ReturnTo)
	fmt.Printf("     - Timestamp: %s\n", retrievedSession.Timestamp.Format(time.RFC3339))
	fmt.Printf("     - Source: %s\n", retrievedSession.Source)
	fmt.Println()

	// Step 8: Exchange authorization code for access token
	fmt.Println("7. Exchanging authorization code for access token...")
	fmt.Println("   (This step would normally make a real HTTP request to Google)")

	// Note: In this example, we can't actually complete the token exchange without
	// a real authorization code from Google. We'll demonstrate the error handling instead.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	token, err := client.Exchange(ctx, simulatedCode)
	if err != nil {
		// This is expected to fail with our mock code
		fmt.Printf("   ⚠️  Token exchange failed (expected with mock code): %v\n", err)
		fmt.Println("   In a real OAuth2 flow, this would succeed with a valid authorization code")

		// Demonstrate what would happen with a successful token exchange
		demonstrateSuccessfulFlow()
		return
	}

	// This code would execute with a real authorization code
	fmt.Printf("   ✓ Access token obtained\n")
	fmt.Printf("   ✓ Token type: %s\n", token.TokenType)
	fmt.Printf("   ✓ Expires at: %s\n", token.Expiry.Format(time.RFC3339))
	if token.RefreshToken != "" {
		fmt.Printf("   ✓ Refresh token available\n")
	}
	fmt.Println()

	// Step 9: Fetch user information
	fmt.Println("8. Fetching user profile information...")
	userInfo, err := client.GetUserInfo(token)
	if err != nil {
		log.Printf("❌ Failed to fetch user info: %v", err)
		return
	}

	fmt.Printf("   ✓ User information retrieved\n")
	prettyPrintUserInfo(userInfo)
	fmt.Println()

	// Step 10: Complete the OAuth2 flow
	fmt.Println("9. OAuth2 flow completed successfully!")
	fmt.Printf("   → Redirecting user to: %s\n", retrievedSession.ReturnTo)
	fmt.Printf("   → User %s authenticated via %s\n", retrievedSession.UserID, retrievedSession.Source)
	fmt.Println()

	fmt.Println("=== End of OAuth2 Example ===")
}

// getEnvOrDefault retrieves an environment variable or returns a default value
func getEnvOrDefault(envVar, defaultValue string) string {
	if value := os.Getenv(envVar); value != "" {
		return value
	}
	return defaultValue
}

// maskSecret masks sensitive information for display
func maskSecret(secret string) string {
	if len(secret) <= 8 {
		return "***"
	}
	return secret[:4] + "***" + secret[len(secret)-4:]
}

// extractStateFromURL extracts the state parameter from an OAuth2 authorization URL
// This is a helper function for the example - in real apps, you'd get this from the callback
func extractStateFromURL(authURL string) string {
	u, err := url.Parse(authURL)
	if err != nil {
		return ""
	}
	return u.Query().Get("state")
}

// handleClientCreationError demonstrates different client creation error scenarios
func handleClientCreationError(err error) {
	fmt.Printf("❌ Failed to create OAuth2 client: %v\n", err)
	fmt.Println()
	fmt.Println("Common client creation errors:")
	fmt.Println("• Empty client ID or secret - ensure OAuth app is properly configured")
	fmt.Println("• Invalid redirect URL - must match OAuth app registration")
	fmt.Println("• Invalid encryption key - must be 24-32 characters for AES encryption")
	fmt.Println("• Nil provider - ensure provider was created successfully")
	fmt.Println()
}

// handleStateValidationError demonstrates different state validation error scenarios
func handleStateValidationError(err error) {
	fmt.Printf("❌ State validation failed: %v\n", err)
	fmt.Println()

	if errors.Is(err, oauth2.ErrStateNotFound) {
		fmt.Println("State validation error analysis:")
		fmt.Println("• ErrStateNotFound - possible CSRF attack or expired state")
		fmt.Println("• State may have been already consumed (replay attack prevention)")
		fmt.Println("• State may have expired or been cleared from storage")
		fmt.Println("• User may have taken too long to complete OAuth flow")
	} else if errors.Is(err, oauth2.ErrDecryptionFailed) {
		fmt.Println("State decryption error analysis:")
		fmt.Println("• ErrDecryptionFailed - encryption key mismatch or corrupted data")
		fmt.Println("• Encryption key may have changed between request and callback")
		fmt.Println("• State parameter may have been tampered with")
	} else if errors.Is(err, oauth2.ErrDeserializationFailed) {
		fmt.Println("State deserialization error analysis:")
		fmt.Println("• ErrDeserializationFailed - incompatible user data type")
		fmt.Println("• User data structure may have changed")
		fmt.Println("• JSON parsing failed for encrypted session data")
	}
	fmt.Println()
}

// demonstrateSuccessfulFlow shows what would happen in a successful OAuth2 flow
func demonstrateSuccessfulFlow() {
	fmt.Println()
	fmt.Println("=== Successful OAuth2 Flow Demonstration ===")
	fmt.Println("With a real authorization code from Google, the flow would continue as:")
	fmt.Println()

	fmt.Println("7. Token Exchange Success:")
	fmt.Println("   ✓ Authorization code exchanged for access token")
	fmt.Println("   ✓ Access token: ya29.a0AfB_byD... (shortened)")
	fmt.Println("   ✓ Refresh token: 1//0GWVh8ZjVbT... (shortened)")
	fmt.Println("   ✓ Token expires: 2024-01-01T12:00:00Z")
	fmt.Println("   ✓ Token type: Bearer")
	fmt.Println()

	fmt.Println("8. User Info Retrieval Success:")
	fmt.Println("   ✓ GET https://www.googleapis.com/oauth2/v3/userinfo")
	fmt.Println("   ✓ User profile data retrieved:")

	// Example user info response from Google
	exampleUserInfo := map[string]any{
		"id":             "123456789012345678901",
		"email":          "user@example.com",
		"verified_email": true,
		"name":           "John Doe",
		"given_name":     "John",
		"family_name":    "Doe",
		"picture":        "https://lh3.googleusercontent.com/a/example",
		"locale":         "en",
	}

	prettyPrintUserInfo(exampleUserInfo)
	fmt.Println()

	fmt.Println("9. OAuth2 Flow Completion:")
	fmt.Println("   ✓ User successfully authenticated")
	fmt.Println("   ✓ Session data retrieved: user123 → /dashboard")
	fmt.Println("   ✓ Access token stored securely for API calls")
	fmt.Println("   ✓ User redirected to dashboard")
	fmt.Println()

	fmt.Println("=== OAuth2 Integration Tips ===")
	fmt.Println("• Store tokens securely (encrypted database, secure storage)")
	fmt.Println("• Implement token refresh logic for expired access tokens")
	fmt.Println("• Use HTTPS for all OAuth2 endpoints in production")
	fmt.Println("• Implement proper session management after OAuth2 completion")
	fmt.Println("• Add logging and monitoring for OAuth2 flows")
	fmt.Println("• Consider implementing OAuth2 scopes based on user permissions")
	fmt.Println()
}

// prettyPrintUserInfo formats and displays user information in a readable format
func prettyPrintUserInfo(userInfo map[string]any) {
	fmt.Println("     User Information:")

	// Print common fields in a structured way
	commonFields := []string{"id", "email", "name", "given_name", "family_name", "picture", "locale", "verified_email"}

	for _, field := range commonFields {
		if value, exists := userInfo[field]; exists {
			fmt.Printf("     • %s: %v\n", field, value)
		}
	}

	// Print any additional fields not in the common list
	fmt.Printf("     Additional fields: ")
	additionalFields := []string{}
	for key := range userInfo {
		isCommon := false
		for _, common := range commonFields {
			if key == common {
				isCommon = true
				break
			}
		}
		if !isCommon {
			additionalFields = append(additionalFields, key)
		}
	}

	if len(additionalFields) > 0 {
		fmt.Printf("%v\n", additionalFields)
	} else {
		fmt.Printf("none\n")
	}

	// Show the raw JSON for developers
	if jsonData, err := json.MarshalIndent(userInfo, "     ", "  "); err == nil {
		fmt.Printf("     Raw JSON response:\n%s\n", string(jsonData))
	}
}
