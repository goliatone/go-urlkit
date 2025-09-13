package urlkit

import (
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/flosch/pongo2/v6"
)

// TestSecurityURLParameterEscaping tests that URL parameters are properly escaped
func TestSecurityURLParameterEscaping(t *testing.T) {
	tests := []struct {
		name           string
		params         map[string]any
		query          map[string]string
		shouldFail     bool
		mustContain    []string // Strings that must be present in successful URLs
		mustNotContain []string // Dangerous strings that must not be present unencoded
		description    string
	}{
		{
			name: "XSS attempt in path parameter",
			params: map[string]any{
				"id": "<script>alert('xss')</script>",
			},
			shouldFail:     false,
			mustContain:    []string{"https://example.com/items/"},
			mustNotContain: []string{"<script>", "alert('xss')"},
			description:    "Script tags should be URL encoded",
		},
		{
			name: "SQL injection attempt in path parameter",
			params: map[string]any{
				"id": "'; DROP TABLE users; --",
			},
			shouldFail:     false,
			mustContain:    []string{"https://example.com/items/"},
			mustNotContain: []string{"DROP TABLE", "; --"},
			description:    "SQL injection should be URL encoded",
		},
		{
			name: "Special characters in query parameters",
			query: map[string]string{
				"search": "<script>alert('xss')</script>",
				"filter": "'; DROP TABLE users; --",
			},
			shouldFail:     false,
			mustContain:    []string{"https://example.com/search?"},
			mustNotContain: []string{"<script>", "DROP TABLE"},
			description:    "Query parameters should be URL encoded",
		},
		{
			name: "Simple string parameters work",
			params: map[string]any{
				"id": "123",
			},
			shouldFail:     false,
			mustContain:    []string{"https://example.com/items/123"},
			mustNotContain: []string{"#error"},
			description:    "Simple string parameters should work",
		},
	}

	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://example.com", map[string]string{
		"item":   "/items/:id",
		"search": "/search",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []*pongo2.Value
			var routeName string

			if len(tt.params) > 0 {
				routeName = "item"
				args = []*pongo2.Value{
					pongo2.AsValue("test"),
					pongo2.AsValue(routeName),
					pongo2.AsValue(tt.params),
				}
			} else {
				routeName = "search"
				args = []*pongo2.Value{
					pongo2.AsValue("test"),
					pongo2.AsValue(routeName),
					pongo2.AsValue(map[string]any{}),
					pongo2.AsValue(tt.query),
				}
			}

			result, err := urlFunc(args...)

			if tt.shouldFail {
				// We expect this test case to fail (error or #error result)
				if err != nil {
					t.Logf("Expected failure occurred: %v", err)
					return
				}

				resultURL := result.String()
				if strings.HasPrefix(resultURL, "#error") {
					t.Logf("Expected failure occurred: %s", resultURL)
					return
				}

				t.Errorf("Expected failure but got success: %s", resultURL)
				return
			}

			// For success cases, check that we got a valid result
			if err != nil {
				t.Fatalf("Helper function failed unexpectedly: %v", err)
			}

			resultURL := result.String()
			t.Logf("Generated URL: %s", resultURL)

			// Check error URL format
			if strings.HasPrefix(resultURL, "#error") {
				t.Errorf("Unexpected error result: %s", resultURL)
				return
			}

			// Parse the URL to check it's valid
			parsedURL, parseErr := url.Parse(resultURL)
			if parseErr != nil {
				t.Errorf("Generated URL is not valid: %v (URL: %s)", parseErr, resultURL)
				return
			}

			// Check required content is present
			for _, mustContain := range tt.mustContain {
				if !strings.Contains(resultURL, mustContain) {
					t.Errorf("URL must contain '%s' but got: %s", mustContain, resultURL)
				}
			}

			// Check dangerous content is not present unencoded
			for _, mustNotContain := range tt.mustNotContain {
				if strings.Contains(resultURL, mustNotContain) {
					t.Errorf("URL must not contain unencoded '%s' but got: %s", mustNotContain, resultURL)
				}
			}

			// Ensure URL structure is maintained
			if parsedURL.Scheme == "" || parsedURL.Host == "" {
				t.Errorf("URL structure corrupted: %s", resultURL)
			}
		})
	}
}

// TestSecurityTemplateInjectionPrevention tests prevention of template injection
func TestSecurityTemplateInjectionPrevention(t *testing.T) {
	tests := []struct {
		name         string
		groupName    string
		routeName    string
		params       map[string]any
		expectError  bool
		description  string
	}{
		{
			name:        "Template injection in group name",
			groupName:   "{{ malicious }}",
			routeName:   "test",
			params:      map[string]any{},
			expectError: true, // Group not found
			description: "Group names with template syntax should be treated as literals",
		},
		{
			name:        "Template injection in route name",
			groupName:   "test",
			routeName:   "{{ malicious }}",
			params:      map[string]any{},
			expectError: true, // Route not found
			description: "Route names with template syntax should be treated as literals",
		},
		{
			name:      "Template injection in parameter values",
			groupName: "test",
			routeName: "item",
			params: map[string]any{
				"id": "{{ system.secrets }}",
			},
			expectError: false, // Should encode the template syntax
			description: "Parameter values with template syntax should be URL encoded",
		},
		{
			name:      "Normal parameters work",
			groupName: "test",
			routeName: "item", 
			params: map[string]any{
				"id": "123",
			},
			expectError: false,
			description: "Normal parameters should work correctly",
		},
	}

	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://example.com", map[string]string{
		"item": "/items/:id",
		"list": "/items",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []*pongo2.Value{
				pongo2.AsValue(tt.groupName),
				pongo2.AsValue(tt.routeName),
				pongo2.AsValue(tt.params),
			}

			result, err := urlFunc(args...)

			if tt.expectError {
				// We expect either an error or #error result
				if err != nil {
					t.Logf("Expected error occurred: %v", err)
					return
				}

				if result != nil {
					resultURL := result.String()
					if strings.HasPrefix(resultURL, "#error") {
						t.Logf("Expected error occurred: %s", resultURL)
						return
					}
				}

				t.Errorf("Expected error but got success")
				return
			}

			// For success cases
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			resultURL := result.String()
			t.Logf("Generated URL: %s", resultURL)

			if strings.HasPrefix(resultURL, "#error") {
				t.Errorf("Unexpected error result: %s", resultURL)
			}

			// Ensure template syntax is properly encoded in URLs
			if strings.Contains(tt.params["id"].(string), "{{") {
				// Template braces should be encoded in URL parameters  
				if strings.Contains(resultURL, "{{") && !strings.Contains(resultURL, "%7B%7B") {
					t.Errorf("Template syntax not properly encoded: %s", resultURL)
				}
			}
		})
	}
}

// TestSecurityErrorMessageSafety tests that error messages don't leak sensitive information
func TestSecurityErrorMessageSafety(t *testing.T) {
	// Create manager with some sensitive route names
	manager := NewRouteManager()
	manager.RegisterGroup("admin_secrets", "https://internal.company.com", map[string]string{
		"database_config": "/admin/db/config/:secret_key",
		"api_keys":        "/admin/api/keys/:api_token",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name              string
		args              []*pongo2.Value
		shouldError       bool
		forbiddenContent  []string
		description       string
	}{
		{
			name: "Non-existent group error",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent_group"),
				pongo2.AsValue("some_route"),
			},
			shouldError:      true,
			forbiddenContent: []string{"admin_secrets", "database_config", "secret_key", "internal.company.com"},
			description:      "Error for non-existent group should not reveal other group names",
		},
		{
			name: "Non-existent route error",
			args: []*pongo2.Value{
				pongo2.AsValue("admin_secrets"),
				pongo2.AsValue("nonexistent_route"),
			},
			shouldError:      true,
			forbiddenContent: []string{"database_config", "api_keys", "secret_key", "api_token"},
			description:      "Error for non-existent route should not reveal other route names",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := urlFunc(tt.args...)

			var errorMessage string
			gotError := false

			if err != nil {
				errorMessage = err.Error()
				gotError = true
			} else if result != nil {
				resultStr := result.String()
				if strings.HasPrefix(resultStr, "#error") {
					errorMessage = resultStr
					gotError = true
				}
			}

			if tt.shouldError && !gotError {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.shouldError && gotError {
				t.Errorf("Unexpected error: %s", errorMessage)
				return
			}

			if gotError {
				t.Logf("Error message: %s", errorMessage)

				// Check that forbidden content is not present
				for _, forbidden := range tt.forbiddenContent {
					if strings.Contains(errorMessage, forbidden) {
						t.Errorf("Error message should not contain sensitive info '%s': %s", forbidden, errorMessage)
					}
				}

				// Ensure error message is reasonably short (no info dumping)
				if len(errorMessage) > 200 {
					t.Errorf("Error message too verbose, may leak information: %s", errorMessage)
				}
			}
		})
	}
}

// TestSecurityMaliciousInputHandling tests various malicious input scenarios 
func TestSecurityMaliciousInputHandling(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://example.com", map[string]string{
		"user": "/users/:id",
		"file": "/files/:path",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name        string
		params      map[string]any
		expectError bool
		description string
	}{
		{
			name: "Simple string parameter",
			params: map[string]any{
				"id": "123",
			},
			expectError: false,
			description: "Normal string should work",
		},
		{
			name: "Buffer overflow attempt",
			params: map[string]any{
				"id": strings.Repeat("A", 1000),
			},
			expectError: false, // Should handle gracefully by encoding
			description: "Large input should not cause buffer overflow",
		},
		{
			name: "Format string attack",
			params: map[string]any{
				"id": "%s%s%s%s%n",
			},
			expectError: false, // Should be treated as literal string
			description: "Format string specifiers should be literal",
		},
		{
			name: "Integer parameter",
			params: map[string]any{
				"id": 123,
			},
			expectError: false, // Should convert to string
			description: "Integer parameters should be handled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := []*pongo2.Value{
				pongo2.AsValue("test"),
				pongo2.AsValue("user"),
				pongo2.AsValue(tt.params),
			}

			result, err := urlFunc(args...)

			if tt.expectError {
				if err == nil && (result == nil || !strings.HasPrefix(result.String(), "#error")) {
					t.Errorf("Expected error for malicious input, but got success")
				} else {
					t.Logf("Expected error occurred")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error for input handling: %v", err)
				} else if result != nil {
					resultURL := result.String()
					if strings.HasPrefix(resultURL, "#error") {
						t.Errorf("Unexpected error result: %s", resultURL)
					} else {
						t.Logf("Successfully handled input: %s", resultURL)
						
						// Basic sanity checks for successful URLs
						if !strings.HasPrefix(resultURL, "https://") {
							t.Errorf("Invalid URL format: %s", resultURL)
						}
					}
				}
			}
		})
	}
}

// TestSecurityConcurrentAccess tests thread safety under concurrent access
func TestSecurityConcurrentAccess(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://example.com", map[string]string{
		"item": "/items/:id",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	// Launch multiple goroutines with various inputs
	numWorkers := 10
	numOps := 100

	done := make(chan bool, numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Worker %d panicked: %v", workerID, r)
				}
				done <- true
			}()

			for j := 0; j < numOps; j++ {
				// Mix of valid and potentially problematic inputs
				inputs := []string{
					fmt.Sprintf("user_%d_%d", workerID, j),
					"<script>alert('test')</script>",
					strings.Repeat("A", 100),
					"normal_value",
				}

				params := map[string]any{"id": inputs[j%len(inputs)]}
				args := []*pongo2.Value{
					pongo2.AsValue("test"),
					pongo2.AsValue("item"),
					pongo2.AsValue(params),
				}

				result, err := urlFunc(args...)
				if err == nil && result != nil {
					// Just ensure we get some result without panicking
					_ = result.String()
				}
			}
		}(i)
	}

	// Wait for all workers to complete with timeout
	completed := 0
	timeout := time.After(10 * time.Second)
	
	for completed < numWorkers {
		select {
		case <-done:
			completed++
		case <-timeout:
			t.Fatalf("Concurrent test timed out, possible deadlock")
		}
	}
}

// TestSecurityURLStructureValidation tests that generated URLs maintain proper structure
func TestSecurityURLStructureValidation(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://example.com", map[string]string{
		"item":   "/items/:id",
		"search": "/search",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	maliciousInputs := []struct {
		name   string
		params map[string]any
		query  map[string]string
	}{
		{
			name: "URL scheme injection",
			params: map[string]any{
				"id": "javascript:alert('xss')",
			},
		},
		{
			name: "Protocol relative URL",
			params: map[string]any{
				"id": "//evil.com/malicious",
			},
		},
		{
			name: "Fragment injection",
			params: map[string]any{
				"id": "normal#<script>alert('xss')</script>",
			},
		},
		{
			name: "Query injection via path param",
			params: map[string]any{
				"id": "123?malicious=param",
			},
		},
		{
			name: "Query injection via query param",
			query: map[string]string{
				"search": "term&malicious=param",
			},
		},
	}

	for _, tt := range maliciousInputs {
		t.Run(tt.name, func(t *testing.T) {
			var args []*pongo2.Value
			var routeName string

			if len(tt.params) > 0 {
				routeName = "item"
				args = []*pongo2.Value{
					pongo2.AsValue("test"),
					pongo2.AsValue(routeName),
					pongo2.AsValue(tt.params),
				}
			} else {
				routeName = "search"
				args = []*pongo2.Value{
					pongo2.AsValue("test"),
					pongo2.AsValue(routeName),
					pongo2.AsValue(map[string]any{}),
					pongo2.AsValue(tt.query),
				}
			}

			result, err := urlFunc(args...)
			if err != nil {
				t.Logf("Function returned error (acceptable): %v", err)
				return
			}

			resultURL := result.String()
			t.Logf("Generated URL: %s", resultURL)

			if strings.HasPrefix(resultURL, "#error") {
				t.Logf("Function returned error URL (acceptable): %s", resultURL)
				return
			}

			// Parse URL to ensure it's structurally valid
			parsedURL, parseErr := url.Parse(resultURL)
			if parseErr != nil {
				t.Errorf("Generated invalid URL: %s (error: %v)", resultURL, parseErr)
				return
			}

			// Ensure proper URL structure is maintained
			if parsedURL.Scheme != "https" {
				t.Errorf("URL scheme changed unexpectedly: %s", resultURL)
			}

			if parsedURL.Host != "example.com" {
				t.Errorf("URL host changed unexpectedly: %s", resultURL)
			}

			// Check for dangerous fragments or schemes (should be encoded)
			if strings.Contains(resultURL, "javascript:") && !strings.Contains(resultURL, "%") {
				t.Errorf("Dangerous JavaScript URL not encoded: %s", resultURL)
			}

			if strings.Contains(resultURL, "//evil.com") && !strings.Contains(resultURL, "%") {
				t.Errorf("URL redirect attack not encoded: %s", resultURL)
			}
		})
	}
}

// BenchmarkSecurityParameterEscaping benchmarks parameter escaping performance
func BenchmarkSecurityParameterEscaping(b *testing.B) {
	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://example.com", map[string]string{
		"item": "/items/:id",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	// Test with realistic but potentially malicious input
	params := map[string]any{
		"id": "<script>alert('test')</script>",
	}

	args := []*pongo2.Value{
		pongo2.AsValue("test"),
		pongo2.AsValue("item"),
		pongo2.AsValue(params),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		result, err := urlFunc(args...)
		if err != nil {
			b.Fatalf("Function failed: %v", err)
		}
		_ = result.String()
	}
}