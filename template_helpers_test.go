package urlkit

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/flosch/pongo2/v6"
)

// TestFromPongoValue tests the fromPongoValue function with various pongo2.Value types
func TestFromPongoValue(t *testing.T) {
	tests := []struct {
		name     string
		input    *pongo2.Value
		expected any
	}{
		{
			name:     "nil value",
			input:    nil,
			expected: nil,
		},
		{
			name:     "string value",
			input:    pongo2.AsValue("hello"),
			expected: "hello",
		},
		{
			name:     "empty string value",
			input:    pongo2.AsValue(""),
			expected: "",
		},
		{
			name:     "int value",
			input:    pongo2.AsValue(42),
			expected: 42,
		},
		{
			name:     "zero int value",
			input:    pongo2.AsValue(0),
			expected: 0,
		},
		{
			name:     "negative int value",
			input:    pongo2.AsValue(-123),
			expected: -123,
		},
		{
			name:     "int64 value",
			input:    pongo2.AsValue(int64(9223372036854775807)),
			expected: int(9223372036854775807), // Should be converted to int
		},
		{
			name:     "float64 value",
			input:    pongo2.AsValue(3.14),
			expected: 3.14,
		},
		{
			name:     "zero float value",
			input:    pongo2.AsValue(0.0),
			expected: 0.0,
		},
		{
			name:     "negative float value",
			input:    pongo2.AsValue(-2.71),
			expected: -2.71,
		},
		{
			name:     "bool true value",
			input:    pongo2.AsValue(true),
			expected: true,
		},
		{
			name:     "bool false value",
			input:    pongo2.AsValue(false),
			expected: false,
		},
		{
			name:     "map value",
			input:    pongo2.AsValue(map[string]any{"key1": "value1", "key2": 42}),
			expected: map[string]any{"key1": "value1", "key2": 42},
		},
		{
			name:     "empty map value",
			input:    pongo2.AsValue(map[string]any{}),
			expected: map[string]any{},
		},
		{
			name:     "slice value",
			input:    pongo2.AsValue([]any{"a", "b", "c"}),
			expected: []any{"a", "b", "c"},
		},
		{
			name:     "empty slice value",
			input:    pongo2.AsValue([]any{}),
			expected: []any{},
		},
		{
			name:     "nested map value",
			input:    pongo2.AsValue(map[string]any{"outer": map[string]any{"inner": "value"}}),
			expected: map[string]any{"outer": map[string]any{"inner": "value"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fromPongoValue(tt.input)

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("fromPongoValue() = %v (%T), expected %v (%T)", result, result, tt.expected, tt.expected)
			}
		})
	}
}

// TestFromPongoValueWithNilInterface tests edge case where pongo2.Value.Interface() returns nil
func TestFromPongoValueWithNilInterface(t *testing.T) {
	// Create a pongo2.Value that will have a nil interface
	nilValue := pongo2.AsValue(nil)
	result := fromPongoValue(nilValue)

	if result != nil {
		t.Errorf("fromPongoValue(nil) = %v, expected nil", result)
	}
}

// TestParseArgs tests the parseArgs function with various argument combinations
func TestParseArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []*pongo2.Value
		expected    *urlHelperArgs
		expectError bool
		errorMsg    string
	}{
		{
			name:        "insufficient arguments - no args",
			args:        []*pongo2.Value{},
			expected:    nil,
			expectError: true,
			errorMsg:    "at least 2 arguments required",
		},
		{
			name:        "insufficient arguments - only group",
			args:        []*pongo2.Value{pongo2.AsValue("frontend")},
			expected:    nil,
			expectError: true,
			errorMsg:    "at least 2 arguments required",
		},
		{
			name: "minimal valid arguments",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
			},
			expected: &urlHelperArgs{
				Group:  "frontend",
				Route:  "home",
				Params: map[string]any{},
				Query:  map[string]string{},
			},
			expectError: false,
		},
		{
			name: "with params",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
				pongo2.AsValue(map[string]any{"id": 123, "slug": "john"}),
			},
			expected: &urlHelperArgs{
				Group:  "frontend",
				Route:  "user_profile",
				Params: map[string]any{"id": 123, "slug": "john"},
				Query:  map[string]string{},
			},
			expectError: false,
		},
		{
			name: "with params and query",
			args: []*pongo2.Value{
				pongo2.AsValue("api"),
				pongo2.AsValue("users"),
				pongo2.AsValue(map[string]any{"id": 456}),
				pongo2.AsValue(map[string]any{"page": "2", "limit": "10"}),
			},
			expected: &urlHelperArgs{
				Group:  "api",
				Route:  "users",
				Params: map[string]any{"id": 456},
				Query:  map[string]string{"page": "2", "limit": "10"},
			},
			expectError: false,
		},
		{
			name: "query with non-string values (should convert)",
			args: []*pongo2.Value{
				pongo2.AsValue("api"),
				pongo2.AsValue("posts"),
				pongo2.AsValue(map[string]any{}),
				pongo2.AsValue(map[string]any{"page": 2, "active": true, "score": 3.14}),
			},
			expected: &urlHelperArgs{
				Group:  "api",
				Route:  "posts",
				Params: map[string]any{},
				Query:  map[string]string{"page": "2", "active": "true", "score": "3.14"},
			},
			expectError: false,
		},
		{
			name: "with nil params",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
				nil, // nil params
			},
			expected: &urlHelperArgs{
				Group:  "frontend",
				Route:  "home",
				Params: map[string]any{},
				Query:  map[string]string{},
			},
			expectError: false,
		},
		{
			name: "with nil query",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
				pongo2.AsValue(map[string]any{"id": 1}),
				nil, // nil query
			},
			expected: &urlHelperArgs{
				Group:  "frontend",
				Route:  "home",
				Params: map[string]any{"id": 1},
				Query:  map[string]string{},
			},
			expectError: false,
		},
		{
			name: "non-string group",
			args: []*pongo2.Value{
				pongo2.AsValue(123), // non-string group
				pongo2.AsValue("home"),
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "group must be a string",
		},
		{
			name: "non-string route",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue(456), // non-string route
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "route must be a string",
		},
		{
			name: "invalid params type",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
				pongo2.AsValue("not-a-map"), // invalid params
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "params must be a map",
		},
		{
			name: "invalid query type",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
				pongo2.AsValue(map[string]any{}),
				pongo2.AsValue("not-a-map"), // invalid query
			},
			expected:    nil,
			expectError: true,
			errorMsg:    "query must be a map",
		},
		{
			name: "empty params and query maps",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
				pongo2.AsValue(map[string]any{}),
				pongo2.AsValue(map[string]any{}),
			},
			expected: &urlHelperArgs{
				Group:  "frontend",
				Route:  "home",
				Params: map[string]any{},
				Query:  map[string]string{},
			},
			expectError: false,
		},
		{
			name: "query with nil values (should be ignored)",
			args: []*pongo2.Value{
				pongo2.AsValue("api"),
				pongo2.AsValue("posts"),
				pongo2.AsValue(map[string]any{}),
				pongo2.AsValue(map[string]any{"key1": "value1", "key2": nil, "key3": "value3"}),
			},
			expected: &urlHelperArgs{
				Group:  "api",
				Route:  "posts",
				Params: map[string]any{},
				Query:  map[string]string{"key1": "value1", "key3": "value3"},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseArgs(tt.args...)

			if tt.expectError {
				if err == nil {
					t.Errorf("parseArgs() expected error but got none")
					return
				}
				if tt.errorMsg != "" && !containsString(err.Error(), tt.errorMsg) {
					t.Errorf("parseArgs() error = %q, expected to contain %q", err.Error(), tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("parseArgs() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Errorf("parseArgs() returned nil result")
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseArgs() = %+v, expected %+v", result, tt.expected)
			}
		})
	}
}

// TestParseArgsEdgeCases tests edge cases for parseArgs function
func TestParseArgsEdgeCases(t *testing.T) {
	t.Run("complex nested params", func(t *testing.T) {
		args := []*pongo2.Value{
			pongo2.AsValue("api"),
			pongo2.AsValue("complex"),
			pongo2.AsValue(map[string]any{
				"user":     map[string]any{"id": 123, "name": "John"},
				"settings": []any{"setting1", "setting2"},
				"count":    42,
			}),
		}

		result, err := parseArgs(args...)
		if err != nil {
			t.Fatalf("parseArgs() unexpected error = %v", err)
		}

		if result.Group != "api" || result.Route != "complex" {
			t.Errorf("parseArgs() basic fields incorrect: group=%s, route=%s", result.Group, result.Route)
		}

		// Check that complex params are preserved
		if len(result.Params) != 3 {
			t.Errorf("parseArgs() params length = %d, expected 3", len(result.Params))
		}

		// Verify nested structure is preserved
		if userMap, ok := result.Params["user"].(map[string]any); ok {
			if userMap["name"] != "John" {
				t.Errorf("parseArgs() nested user.name = %v, expected 'John'", userMap["name"])
			}
		} else {
			t.Errorf("parseArgs() user param is not a map: %T", result.Params["user"])
		}
	})

	t.Run("extra arguments beyond query", func(t *testing.T) {
		args := []*pongo2.Value{
			pongo2.AsValue("frontend"),
			pongo2.AsValue("home"),
			pongo2.AsValue(map[string]any{"id": 1}),
			pongo2.AsValue(map[string]any{"page": "1"}),
			pongo2.AsValue("extra-arg"), // Extra argument (should be ignored)
		}

		result, err := parseArgs(args...)
		if err != nil {
			t.Fatalf("parseArgs() unexpected error = %v", err)
		}

		// Should still parse correctly, ignoring extra arguments
		if result.Group != "frontend" || result.Route != "home" {
			t.Errorf("parseArgs() basic parsing failed with extra args")
		}
	})
}

// TestArgumentValidationAndTypeConversion tests argument validation and type conversion edge cases
func TestArgumentValidationAndTypeConversion(t *testing.T) {
	t.Run("params with various types", func(t *testing.T) {
		// Test that complex types in params are preserved as-is
		complexParams := map[string]any{
			"string": "value",
			"int":    123,
			"float":  3.14,
			"bool":   true,
			"slice":  []any{1, 2, 3},
			"map":    map[string]any{"nested": "value"},
			"nil":    nil,
		}

		args := []*pongo2.Value{
			pongo2.AsValue("test"),
			pongo2.AsValue("route"),
			pongo2.AsValue(complexParams),
		}

		result, err := parseArgs(args...)
		if err != nil {
			t.Fatalf("parseArgs() unexpected error = %v", err)
		}

		// Params should preserve all types
		if !reflect.DeepEqual(result.Params, complexParams) {
			t.Errorf("parseArgs() params not preserved correctly")
		}
	})

	t.Run("query type conversion", func(t *testing.T) {
		// Test that query values are converted to strings
		queryMap := map[string]any{
			"string":     "value",
			"int":        123,
			"float":      3.14,
			"bool_true":  true,
			"bool_false": false,
			"zero_int":   0,
			"zero_float": 0.0,
			"empty_str":  "",
			"nil_value":  nil, // Should be ignored
		}

		args := []*pongo2.Value{
			pongo2.AsValue("test"),
			pongo2.AsValue("route"),
			pongo2.AsValue(map[string]any{}),
			pongo2.AsValue(queryMap),
		}

		result, err := parseArgs(args...)
		if err != nil {
			t.Fatalf("parseArgs() unexpected error = %v", err)
		}

		expectedQuery := map[string]string{
			"string":     "value",
			"int":        "123",
			"float":      "3.14",
			"bool_true":  "true",
			"bool_false": "false",
			"zero_int":   "0",
			"zero_float": "0",
			"empty_str":  "",
			// nil_value should not be present
		}

		if len(result.Query) != len(expectedQuery) {
			t.Errorf("parseArgs() query length = %d, expected %d", len(result.Query), len(expectedQuery))
		}

		for key, expectedValue := range expectedQuery {
			if actualValue, exists := result.Query[key]; !exists {
				t.Errorf("parseArgs() query missing key %q", key)
			} else if actualValue != expectedValue {
				t.Errorf("parseArgs() query[%q] = %q, expected %q", key, actualValue, expectedValue)
			}
		}

		// Verify nil_value was not included
		if _, exists := result.Query["nil_value"]; exists {
			t.Errorf("parseArgs() query should not include nil_value")
		}
	})
}

// containsString checks if a string contains a substring (helper for tests)
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(substr) > 0 && len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i < len(s)-len(substr)+1; i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}

func TestCurrentRouteIfHelper(t *testing.T) {
	config := DefaultTemplateHelperConfig()
	helperFunc := currentRouteIfHelper(config)

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult string
		expectError    bool
	}{
		{
			name: "route matches - returns true value",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend.home"), // targetRoute
				pongo2.AsValue("frontend.home"), // currentRoute
				pongo2.AsValue("active"),        // valueIfTrue
			},
			expectedResult: "active",
			expectError:    false,
		},
		{
			name: "route doesn't match - returns empty string default",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend.home"),    // targetRoute
				pongo2.AsValue("frontend.profile"), // currentRoute
				pongo2.AsValue("active"),           // valueIfTrue
			},
			expectedResult: "",
			expectError:    false,
		},
		{
			name: "route doesn't match - returns explicit false value",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend.home"),    // targetRoute
				pongo2.AsValue("frontend.profile"), // currentRoute
				pongo2.AsValue("active"),           // valueIfTrue
				pongo2.AsValue("inactive"),         // valueIfFalse
			},
			expectedResult: "inactive",
			expectError:    false,
		},
		{
			name: "insufficient arguments",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend.home"),    // targetRoute
				pongo2.AsValue("frontend.profile"), // currentRoute
				// missing valueIfTrue
			},
			expectedResult: "", // will be error message
			expectError:    true,
		},
		{
			name: "non-string route arguments",
			args: []*pongo2.Value{
				pongo2.AsValue(123),                // targetRoute (non-string)
				pongo2.AsValue("frontend.profile"), // currentRoute
				pongo2.AsValue("active"),           // valueIfTrue
			},
			expectedResult: "", // will be error message
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := helperFunc(tt.args...)

			if err != nil {
				t.Fatalf("Helper function returned error: %v", err)
			}

			if result == nil {
				t.Fatal("Helper function returned nil result")
			}

			resultStr := result.String()

			if tt.expectError {
				// For error cases, we expect the result to contain "#error"
				if resultStr == tt.expectedResult {
					t.Errorf("Expected error message, but got: %s", resultStr)
				}
				// Check that it contains error indicator
				if len(resultStr) == 0 {
					t.Error("Expected error message, but got empty string")
				}
			} else {
				if resultStr != tt.expectedResult {
					t.Errorf("Expected result %q, but got %q", tt.expectedResult, resultStr)
				}
			}
		})
	}
}

func TestCurrentRouteIfHelperIntegration(t *testing.T) {
	// Test with a full RouteManager integration to ensure helpers work properly
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":    "/",
		"profile": "/profile",
		"about":   "/about",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	// Test that current_route_if is available
	currentRouteIfFunc, exists := helpers["current_route_if"]
	if !exists {
		t.Fatal("current_route_if helper not found in helpers map")
	}

	// Cast to the expected function type
	helperFunc, ok := currentRouteIfFunc.(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
	if !ok {
		t.Fatal("current_route_if helper has wrong function signature")
	}

	// Test the helper function with matching routes
	args := []*pongo2.Value{
		pongo2.AsValue("frontend.home"),
		pongo2.AsValue("frontend.home"),
		pongo2.AsValue("nav-active"),
	}

	result, pongoErr := helperFunc(args...)
	if pongoErr != nil {
		t.Fatalf("Helper function returned pongo error: %v", pongoErr)
	}

	if result.String() != "nav-active" {
		t.Errorf("Expected 'nav-active', got '%s'", result.String())
	}
}

func TestTemplateHelperAliases(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	if helpers["URL"] == nil {
		t.Fatal("expected URL alias to be registered")
	}
	if fmt.Sprintf("%p", helpers["URL"]) != fmt.Sprintf("%p", helpers["url"]) {
		t.Error("URL alias should reference the same helper as url")
	}

	if helpers["RoutePath"] == nil {
		t.Fatal("expected RoutePath alias to be registered")
	}
}

func TestNavigationHelper(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":    "/",
		"profile": "/users/:id",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	rawHelper, ok := helpers["Navigation"]
	if !ok {
		t.Fatal("Navigation helper not registered")
	}

	navFunc, ok := rawHelper.(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
	if !ok {
		t.Fatal("Navigation helper has unexpected signature")
	}

	routes := pongo2.AsValue([]any{"home", "profile"})
	params := pongo2.AsValue(map[string]any{
		"profile": map[string]any{"id": 7},
	})

	result, perr := navFunc(pongo2.AsValue("frontend"), routes, params)
	if perr != nil {
		t.Fatalf("Navigation helper returned pongo error: %v", perr)
	}

	nodes, ok := result.Interface().([]NavigationNode)
	if !ok {
		t.Fatalf("expected []NavigationNode, got %T", result.Interface())
	}

	if len(nodes) != 2 {
		t.Fatalf("expected 2 navigation nodes, got %d", len(nodes))
	}

	if nodes[1].URL != "https://example.com/users/7" {
		t.Errorf("expected profile URL to include parameter, got %s", nodes[1].URL)
	}
}

// TestURLHelper tests the url helper function comprehensively
func TestURLHelper(t *testing.T) {
	// Setup test route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":         "/",
		"user_profile": "/users/:id/profile",
		"user_posts":   "/users/:id/posts",
		"search":       "/search",
		"complex":      "/users/:user_id/posts/:post_id/comments/:comment_id",
	})
	manager.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users":      "/users",
		"user_by_id": "/users/:id",
		"posts":      "/posts",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult string
		expectError    bool
	}{
		{
			name: "simple route without parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
			},
			expectedResult: "https://example.com/",
			expectError:    false,
		},
		{
			name: "route with path parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
				pongo2.AsValue(map[string]any{"id": 123}),
			},
			expectedResult: "https://example.com/users/123/profile",
			expectError:    false,
		},
		{
			name: "route with path parameters (string ID)",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
				pongo2.AsValue(map[string]any{"id": "john-doe"}),
			},
			expectedResult: "https://example.com/users/john-doe/profile",
			expectError:    false,
		},
		{
			name: "route with query parameters only",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("search"),
				pongo2.AsValue(map[string]any{}), // empty params
				pongo2.AsValue(map[string]any{"q": "golang", "page": "1"}),
			},
			expectedResult: "https://example.com/search", // Query order is not deterministic
			expectError:    false,
		},
		{
			name: "route with both path and query parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_posts"),
				pongo2.AsValue(map[string]any{"id": 456}),
				pongo2.AsValue(map[string]any{"sort": "date", "limit": "10"}),
			},
			expectedResult: "https://example.com/users/456/posts", // Query order is not deterministic
			expectError:    false,
		},
		{
			name: "complex route with multiple path parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("complex"),
				pongo2.AsValue(map[string]any{
					"user_id":    123,
					"post_id":    456,
					"comment_id": 789,
				}),
			},
			expectedResult: "https://example.com/users/123/posts/456/comments/789",
			expectError:    false,
		},
		{
			name: "different group (API)",
			args: []*pongo2.Value{
				pongo2.AsValue("api"),
				pongo2.AsValue("user_by_id"),
				pongo2.AsValue(map[string]any{"id": 999}),
			},
			expectedResult: "https://api.example.com/users/999",
			expectError:    false,
		},
		{
			name: "non-existent group",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent"),
				pongo2.AsValue("home"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
		{
			name: "non-existent route",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("nonexistent"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
		{
			name: "insufficient arguments",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
		{
			name: "invalid group type",
			args: []*pongo2.Value{
				pongo2.AsValue(123), // non-string
				pongo2.AsValue("home"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
		{
			name: "nil params map",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
				nil, // nil params
			},
			expectedResult: "https://example.com/",
			expectError:    false,
		},
		{
			name: "nil query map",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("search"),
				pongo2.AsValue(map[string]any{}), // empty params
				nil,                              // nil query
			},
			expectedResult: "https://example.com/search",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := urlFunc(tt.args...)

			if err != nil {
				t.Fatalf("URL helper returned pongo error: %v", err)
			}

			if result == nil {
				t.Fatal("URL helper returned nil result")
			}

			resultStr := result.String()

			if tt.expectError {
				// For error cases, we expect the result to contain "#error" or be an error message
				if resultStr == tt.expectedResult {
					t.Errorf("Expected error message, but got exact match: %s", resultStr)
				}
				// Check that it's not a valid URL (should be error message)
				if !containsString(resultStr, "#error") && !containsString(resultStr, "error") {
					t.Errorf("Expected error message, but got: %s", resultStr)
				}
			} else {
				// For query parameter tests, check that the base URL matches and query params exist
				if tt.name == "route with query parameters only" {
					if !containsString(resultStr, "https://example.com/search") {
						t.Errorf("Expected base URL in result %q", resultStr)
					}
					if !containsString(resultStr, "q=golang") || !containsString(resultStr, "page=1") {
						t.Errorf("Expected query parameters in result %q", resultStr)
					}
				} else if tt.name == "route with both path and query parameters" {
					if !containsString(resultStr, "https://example.com/users/456/posts") {
						t.Errorf("Expected base URL with path in result %q", resultStr)
					}
					if !containsString(resultStr, "sort=date") || !containsString(resultStr, "limit=10") {
						t.Errorf("Expected query parameters in result %q", resultStr)
					}
				} else {
					if resultStr != tt.expectedResult {
						t.Errorf("Expected result %q, but got %q", tt.expectedResult, resultStr)
					}
				}
			}
		})
	}
}

// TestRoutePathHelper tests the route_path helper function
func TestRoutePathHelper(t *testing.T) {
	// Setup test route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":         "/",
		"user_profile": "/users/:id/profile",
		"search":       "/search",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	routePathFunc := helpers["route_path"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult string
		expectError    bool
	}{
		{
			name: "simple route path",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
			},
			expectedResult: "/",
			expectError:    false,
		},
		{
			name: "route path with parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
				pongo2.AsValue(map[string]any{"id": 123}),
			},
			expectedResult: "/users/123/profile",
			expectError:    false,
		},
		{
			name: "route path with query parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("search"),
				pongo2.AsValue(map[string]any{}),
				pongo2.AsValue(map[string]any{"q": "test"}),
			},
			expectedResult: "/search?q=test",
			expectError:    false,
		},
		{
			name: "non-existent group",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent"),
				pongo2.AsValue("home"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := routePathFunc(tt.args...)

			if err != nil {
				t.Fatalf("Route path helper returned pongo error: %v", err)
			}

			if result == nil {
				t.Fatal("Route path helper returned nil result")
			}

			resultStr := result.String()

			if tt.expectError {
				if !containsString(resultStr, "#error") && !containsString(resultStr, "error") {
					t.Errorf("Expected error message, but got: %s", resultStr)
				}
			} else {
				if resultStr != tt.expectedResult {
					t.Errorf("Expected result %q, but got %q", tt.expectedResult, resultStr)
				}
			}
		})
	}
}

// TestHasRouteHelper tests the has_route helper function
func TestHasRouteHelper(t *testing.T) {
	// Setup test route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":    "/",
		"profile": "/profile",
		"about":   "/about",
	})
	manager.RegisterGroup("admin", "https://admin.example.com", map[string]string{
		"dashboard": "/dashboard",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	hasRouteFunc := helpers["has_route"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult bool
	}{
		{
			name: "existing route",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
			},
			expectedResult: true,
		},
		{
			name: "existing route in different group",
			args: []*pongo2.Value{
				pongo2.AsValue("admin"),
				pongo2.AsValue("dashboard"),
			},
			expectedResult: true,
		},
		{
			name: "non-existent route",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("nonexistent"),
			},
			expectedResult: false,
		},
		{
			name: "non-existent group",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent"),
				pongo2.AsValue("home"),
			},
			expectedResult: false,
		},
		{
			name: "insufficient arguments",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
			},
			expectedResult: false,
		},
		{
			name: "invalid argument types",
			args: []*pongo2.Value{
				pongo2.AsValue(123),
				pongo2.AsValue("home"),
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := hasRouteFunc(tt.args...)

			if err != nil {
				t.Fatalf("Has route helper returned pongo error: %v", err)
			}

			if result == nil {
				t.Fatal("Has route helper returned nil result")
			}

			resultBool := result.Interface().(bool)

			if resultBool != tt.expectedResult {
				t.Errorf("Expected result %v, but got %v", tt.expectedResult, resultBool)
			}
		})
	}
}

// TestRouteTemplateHelper tests the route_template helper function
func TestRouteTemplateHelper(t *testing.T) {
	// Setup test route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":         "/",
		"user_profile": "/users/:id/profile",
		"complex":      "/users/:user_id/posts/:post_id",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	routeTemplateFunc := helpers["route_template"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult string
		expectError    bool
	}{
		{
			name: "simple route template",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
			},
			expectedResult: "/",
			expectError:    false,
		},
		{
			name: "route template with parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
			},
			expectedResult: "/users/:id/profile",
			expectError:    false,
		},
		{
			name: "complex route template",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("complex"),
			},
			expectedResult: "/users/:user_id/posts/:post_id",
			expectError:    false,
		},
		{
			name: "non-existent route",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("nonexistent"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
		{
			name: "non-existent group",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent"),
				pongo2.AsValue("home"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
		{
			name: "insufficient arguments",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := routeTemplateFunc(tt.args...)

			if err != nil {
				t.Fatalf("Route template helper returned pongo error: %v", err)
			}

			if result == nil {
				t.Fatal("Route template helper returned nil result")
			}

			resultStr := result.String()

			if tt.expectError {
				if !containsString(resultStr, "#error") && !containsString(resultStr, "error") {
					t.Errorf("Expected error message, but got: %s", resultStr)
				}
			} else {
				if resultStr != tt.expectedResult {
					t.Errorf("Expected result %q, but got %q", tt.expectedResult, resultStr)
				}
			}
		})
	}
}

// TestRouteVarsHelper tests the route_vars helper function
func TestRouteVarsHelper(t *testing.T) {
	// Setup test route manager with template variables
	manager := NewRouteManager()

	// Create a group with template variables
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
		"user": "/users/:id",
	})
	frontend := manager.Group("frontend")

	// Set some template variables for testing
	frontend.SetTemplateVar("app_version", "1.0.0")
	frontend.SetTemplateVar("api_key", "test-key")
	frontend.SetTemplateVar("domain", "example.com")

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	routeVarsFunc := helpers["route_vars"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult map[string]any
		expectError    bool
	}{
		{
			name: "get template vars for existing group",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
			},
			expectedResult: map[string]any{
				"app_version": "1.0.0",
				"api_key":     "test-key",
				"domain":      "example.com",
			},
			expectError: false,
		},
		{
			name: "non-existent group",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent"),
			},
			expectedResult: nil,
			expectError:    true,
		},
		{
			name:           "insufficient arguments",
			args:           []*pongo2.Value{},
			expectedResult: nil,
			expectError:    true,
		},
		{
			name: "invalid argument type",
			args: []*pongo2.Value{
				pongo2.AsValue(123), // non-string
			},
			expectedResult: nil,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := routeVarsFunc(tt.args...)

			if err != nil {
				t.Fatalf("Route vars helper returned pongo error: %v", err)
			}

			if result == nil {
				t.Fatal("Route vars helper returned nil result")
			}

			if tt.expectError {
				resultStr := result.String()
				if !containsString(resultStr, "#error") && !containsString(resultStr, "error") {
					t.Errorf("Expected error message, but got: %s", resultStr)
				}
			} else {
				resultMap := result.Interface().(map[string]any)
				if !reflect.DeepEqual(resultMap, tt.expectedResult) {
					t.Errorf("Expected result %+v, but got %+v", tt.expectedResult, resultMap)
				}
			}
		})
	}
}

// TestRouteExistsHelper tests the route_exists helper function
func TestRouteExistsHelper(t *testing.T) {
	// Setup test route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})
	manager.RegisterGroup("admin", "https://admin.example.com", map[string]string{
		"dashboard": "/dashboard",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	routeExistsFunc := helpers["route_exists"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult bool
	}{
		{
			name: "existing group",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
			},
			expectedResult: true,
		},
		{
			name: "another existing group",
			args: []*pongo2.Value{
				pongo2.AsValue("admin"),
			},
			expectedResult: true,
		},
		{
			name: "non-existent group",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent"),
			},
			expectedResult: false,
		},
		{
			name:           "insufficient arguments",
			args:           []*pongo2.Value{},
			expectedResult: false,
		},
		{
			name: "invalid argument type",
			args: []*pongo2.Value{
				pongo2.AsValue(123), // non-string
			},
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := routeExistsFunc(tt.args...)

			if err != nil {
				t.Fatalf("Route exists helper returned pongo error: %v", err)
			}

			if result == nil {
				t.Fatal("Route exists helper returned nil result")
			}

			// Handle cases where has_route might return a string error instead of bool
			if result.Interface() == nil {
				t.Errorf("Got nil result, expected %v", tt.expectedResult)
				return
			}

			if resultBool, ok := result.Interface().(bool); ok {
				if resultBool != tt.expectedResult {
					t.Errorf("Expected result %v, but got %v", tt.expectedResult, resultBool)
				}
			} else {
				// If it's not a bool, it should be false for error cases
				if tt.expectedResult != false {
					t.Errorf("Expected bool result %v, but got %T: %v", tt.expectedResult, result.Interface(), result.Interface())
				}
			}
		})
	}
}

// TestURLAbsHelper tests the url_abs helper function
func TestURLAbsHelper(t *testing.T) {
	// Setup test route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":         "/",
		"user_profile": "/users/:id/profile",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlAbsFunc := helpers["url_abs"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	tests := []struct {
		name           string
		args           []*pongo2.Value
		expectedResult string
		expectError    bool
	}{
		{
			name: "simple absolute URL",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("home"),
			},
			expectedResult: "https://example.com/",
			expectError:    false,
		},
		{
			name: "absolute URL with parameters",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
				pongo2.AsValue(map[string]any{"id": 123}),
			},
			expectedResult: "https://example.com/users/123/profile",
			expectError:    false,
		},
		{
			name: "non-existent group",
			args: []*pongo2.Value{
				pongo2.AsValue("nonexistent"),
				pongo2.AsValue("home"),
			},
			expectedResult: "", // Will be error message
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := urlAbsFunc(tt.args...)

			if err != nil {
				t.Fatalf("URL abs helper returned pongo error: %v", err)
			}

			if result == nil {
				t.Fatal("URL abs helper returned nil result")
			}

			resultStr := result.String()

			if tt.expectError {
				if !containsString(resultStr, "#error") && !containsString(resultStr, "error") {
					t.Errorf("Expected error message, but got: %s", resultStr)
				}
			} else {
				if resultStr != tt.expectedResult {
					t.Errorf("Expected result %q, but got %q", tt.expectedResult, resultStr)
				}
			}
		})
	}
}

// TestTemplateRenderingModes tests both path concatenation and template rendering modes
func TestTemplateRenderingModes(t *testing.T) {
	t.Run("Path Concatenation Mode (default)", func(t *testing.T) {
		// Setup route manager with standard base URL + routes
		manager := NewRouteManager()
		manager.RegisterGroup("frontend", "https://example.com", map[string]string{
			"home":         "/",
			"user_profile": "/users/:id/profile",
		})

		config := DefaultTemplateHelperConfig()
		helpers := TemplateHelpers(manager, config)
		urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test simple concatenation
		result, err := urlFunc(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("user_profile"),
			pongo2.AsValue(map[string]any{"id": 123}),
		)

		if err != nil {
			t.Fatalf("URL helper returned error: %v", err)
		}

		expected := "https://example.com/users/123/profile"
		if result.String() != expected {
			t.Errorf("Path concatenation mode: expected %q, got %q", expected, result.String())
		}
	})

	t.Run("Template Rendering Mode", func(t *testing.T) {
		// Setup route manager with URL template
		manager := NewRouteManager()
		manager.RegisterGroup("frontend", "", map[string]string{
			"user_profile": "/users/:id/profile",
			"api_call":     "/api/v1/:resource",
		})
		frontend := manager.Group("frontend")

		// Set URL template with variables
		frontend.SetURLTemplate("{protocol}://{host}{route_path}")
		frontend.SetTemplateVar("protocol", "https")
		frontend.SetTemplateVar("host", "api.example.com")

		config := DefaultTemplateHelperConfig()
		helpers := TemplateHelpers(manager, config)
		urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test template rendering
		result, err := urlFunc(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("user_profile"),
			pongo2.AsValue(map[string]any{"id": 456}),
		)

		if err != nil {
			t.Fatalf("URL helper returned error: %v", err)
		}

		expected := "https://api.example.com/users/456/profile"
		resultStr := result.String()
		// Handle potential trailing slash variations
		if resultStr != expected && resultStr != expected+"/" {
			t.Errorf("Template rendering mode: expected %q (or with trailing slash), got %q", expected, resultStr)
		}
	})

	t.Run("Template Rendering with Query Parameters", func(t *testing.T) {
		// Test template rendering with query parameters
		manager := NewRouteManager()
		manager.RegisterGroup("api", "", map[string]string{
			"search": "/search",
		})
		api := manager.Group("api")

		api.SetURLTemplate("{protocol}://{host}/api/{version}{route_path}")
		api.SetTemplateVar("protocol", "https")
		api.SetTemplateVar("host", "search.example.com")
		api.SetTemplateVar("version", "v2")

		config := DefaultTemplateHelperConfig()
		helpers := TemplateHelpers(manager, config)
		urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		result, err := urlFunc(
			pongo2.AsValue("api"),
			pongo2.AsValue("search"),
			pongo2.AsValue(map[string]any{}), // no path params
			pongo2.AsValue(map[string]any{"q": "golang", "limit": "20"}),
		)

		if err != nil {
			t.Fatalf("URL helper returned error: %v", err)
		}

		resultStr := result.String()
		// Check that it contains the template-rendered base and query params
		if !containsString(resultStr, "https://search.example.com/api/v2/search") {
			t.Errorf("Template rendering with query: expected template base, got %q", resultStr)
		}
		if !containsString(resultStr, "q=golang") || !containsString(resultStr, "limit=20") {
			t.Errorf("Template rendering with query: missing query params, got %q", resultStr)
		}
	})
}

// TestCoreHelperErrorCases tests comprehensive error scenarios for core helpers
func TestCoreHelperErrorCases(t *testing.T) {
	// Setup minimal route manager
	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://test.com", map[string]string{
		"valid": "/valid",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	testCases := []struct {
		helperName  string
		args        []*pongo2.Value
		expectError bool
	}{
		// URL helper error cases
		{"url", []*pongo2.Value{}, true},                                                                           // no args
		{"url", []*pongo2.Value{pongo2.AsValue("test")}, true},                                                     // insufficient args
		{"url", []*pongo2.Value{pongo2.AsValue(123), pongo2.AsValue("valid")}, true},                               // invalid group type
		{"url", []*pongo2.Value{pongo2.AsValue("test"), pongo2.AsValue(456)}, true},                                // invalid route type
		{"url", []*pongo2.Value{pongo2.AsValue("nonexistent"), pongo2.AsValue("valid")}, true},                     // missing group
		{"url", []*pongo2.Value{pongo2.AsValue("test"), pongo2.AsValue("nonexistent")}, true},                      // missing route
		{"url", []*pongo2.Value{pongo2.AsValue("test"), pongo2.AsValue("valid"), pongo2.AsValue("invalid")}, true}, // invalid params type

		// Route path helper error cases
		{"route_path", []*pongo2.Value{}, true},                                                       // no args
		{"route_path", []*pongo2.Value{pongo2.AsValue("nonexistent"), pongo2.AsValue("valid")}, true}, // missing group

		// Route template helper error cases
		{"route_template", []*pongo2.Value{}, true},                                                       // no args
		{"route_template", []*pongo2.Value{pongo2.AsValue("test")}, true},                                 // insufficient args
		{"route_template", []*pongo2.Value{pongo2.AsValue(123), pongo2.AsValue("valid")}, true},           // invalid group type
		{"route_template", []*pongo2.Value{pongo2.AsValue("test"), pongo2.AsValue(456)}, true},            // invalid route type
		{"route_template", []*pongo2.Value{pongo2.AsValue("nonexistent"), pongo2.AsValue("valid")}, true}, // missing group
		{"route_template", []*pongo2.Value{pongo2.AsValue("test"), pongo2.AsValue("nonexistent")}, true},  // missing route

		// Route vars helper error cases
		{"route_vars", []*pongo2.Value{}, true},                              // no args
		{"route_vars", []*pongo2.Value{pongo2.AsValue(123)}, true},           // invalid group type
		{"route_vars", []*pongo2.Value{pongo2.AsValue("nonexistent")}, true}, // missing group

		// URL abs helper error cases
		{"url_abs", []*pongo2.Value{}, true}, // no args
		{"url_abs", []*pongo2.Value{pongo2.AsValue("nonexistent"), pongo2.AsValue("valid")}, true}, // missing group
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%s_error_case", tc.helperName), func(t *testing.T) {
			helperFunc, exists := helpers[tc.helperName]
			if !exists {
				t.Fatalf("Helper %s not found", tc.helperName)
			}

			fn := helperFunc.(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
			result, err := fn(tc.args...)

			if err != nil {
				t.Fatalf("Helper %s returned pongo error: %v", tc.helperName, err)
			}

			if result == nil {
				t.Fatalf("Helper %s returned nil result", tc.helperName)
			}

			if tc.expectError {
				resultStr := result.String()
				// Should be an error message (contains #error or is clearly an error)
				if !containsString(resultStr, "#error") && !containsString(resultStr, "error") && len(resultStr) > 0 {
					// For some helpers like has_route, route_exists, the error is returning false instead of error string
					if tc.helperName == "has_route" || tc.helperName == "route_exists" {
						// These return boolean false for errors, which is acceptable
						return
					}
					t.Errorf("Helper %s expected error message, but got: %s", tc.helperName, resultStr)
				}
			}
		})
	}
}

// TestCurrentRouteIfHelperContextualScenarios tests the current_route_if helper
// with various route matching scenarios that simulate real-world usage patterns
func TestCurrentRouteIfHelperContextualScenarios(t *testing.T) {
	config := DefaultTemplateHelperConfig()
	helperFunc := currentRouteIfHelper(config)

	tests := []struct {
		name           string
		targetRoute    string
		currentRoute   string
		valueIfTrue    any
		valueIfFalse   any
		expectedResult string
		description    string
	}{
		{
			name:           "exact route match",
			targetRoute:    "frontend.home",
			currentRoute:   "frontend.home",
			valueIfTrue:    "active",
			valueIfFalse:   nil,
			expectedResult: "active",
			description:    "Navigation link should be active when on the exact route",
		},
		{
			name:           "different routes in same group",
			targetRoute:    "frontend.home",
			currentRoute:   "frontend.profile",
			valueIfTrue:    "active",
			valueIfFalse:   "inactive",
			expectedResult: "inactive",
			description:    "Navigation link should be inactive when on different route",
		},
		{
			name:           "different route groups",
			targetRoute:    "frontend.dashboard",
			currentRoute:   "admin.dashboard",
			valueIfTrue:    "nav-active",
			valueIfFalse:   "nav-inactive",
			expectedResult: "nav-inactive",
			description:    "Should distinguish between routes in different groups",
		},
		{
			name:           "complex route names with dots",
			targetRoute:    "api.v2.users.list",
			currentRoute:   "api.v2.users.list",
			valueIfTrue:    "current-page",
			valueIfFalse:   nil,
			expectedResult: "current-page",
			description:    "Should handle complex hierarchical route names",
		},
		{
			name:           "numeric values",
			targetRoute:    "frontend.posts",
			currentRoute:   "frontend.posts",
			valueIfTrue:    1,
			valueIfFalse:   0,
			expectedResult: "1",
			description:    "Should handle numeric true/false values",
		},
		{
			name:           "boolean values",
			targetRoute:    "admin.users",
			currentRoute:   "admin.settings",
			valueIfTrue:    true,
			valueIfFalse:   false,
			expectedResult: "False",
			description:    "Should handle boolean true/false values",
		},
		{
			name:           "empty string current route",
			targetRoute:    "frontend.home",
			currentRoute:   "",
			valueIfTrue:    "active",
			valueIfFalse:   "no-route",
			expectedResult: "no-route",
			description:    "Should handle empty current route gracefully",
		},
		{
			name:           "mixed case routes",
			targetRoute:    "Frontend.Home",
			currentRoute:   "frontend.home",
			valueIfTrue:    "active",
			valueIfFalse:   "inactive",
			expectedResult: "inactive",
			description:    "Route matching should be case-sensitive",
		},
		{
			name:           "partial route matching",
			targetRoute:    "frontend.user",
			currentRoute:   "frontend.user.profile",
			valueIfTrue:    "parent-active",
			valueIfFalse:   "not-parent",
			expectedResult: "not-parent",
			description:    "Should not match partial route names",
		},
		{
			name:           "whitespace in routes",
			targetRoute:    "frontend.home",
			currentRoute:   " frontend.home ",
			valueIfTrue:    "active",
			valueIfFalse:   "inactive",
			expectedResult: "inactive",
			description:    "Should not trim whitespace - exact matching required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var args []*pongo2.Value
			args = append(args, pongo2.AsValue(tt.targetRoute))
			args = append(args, pongo2.AsValue(tt.currentRoute))
			args = append(args, pongo2.AsValue(tt.valueIfTrue))

			if tt.valueIfFalse != nil {
				args = append(args, pongo2.AsValue(tt.valueIfFalse))
			}

			result, err := helperFunc(args...)

			if err != nil {
				t.Fatalf("Helper function returned error: %v", err)
			}

			if result == nil {
				t.Fatal("Helper function returned nil result")
			}

			resultStr := result.String()
			if resultStr != tt.expectedResult {
				t.Errorf("Test '%s' failed:\n  Description: %s\n  Expected: %q\n  Got: %q",
					tt.name, tt.description, tt.expectedResult, resultStr)
			}
		})
	}
}

// TestCurrentRouteIfHelperWithNavigationContexts tests realistic navigation scenarios
func TestCurrentRouteIfHelperWithNavigationContexts(t *testing.T) {
	config := DefaultTemplateHelperConfig()
	helperFunc := currentRouteIfHelper(config)

	// Simulate a typical website navigation context
	navigationTests := []struct {
		currentRoute string
		navLinks     map[string]string // route -> expected CSS class
		description  string
	}{
		{
			currentRoute: "frontend.home",
			navLinks: map[string]string{
				"frontend.home":    "nav-item active",
				"frontend.about":   "nav-item",
				"frontend.contact": "nav-item",
				"admin.dashboard":  "nav-item",
			},
			description: "Home page navigation state",
		},
		{
			currentRoute: "admin.dashboard",
			navLinks: map[string]string{
				"frontend.home":   "nav-item",
				"frontend.about":  "nav-item",
				"admin.dashboard": "nav-item admin-active",
				"admin.users":     "nav-item",
			},
			description: "Admin dashboard navigation state",
		},
		{
			currentRoute: "api.v1.users",
			navLinks: map[string]string{
				"api.v1.users":      "api-nav current",
				"api.v1.posts":      "api-nav",
				"api.v2.users":      "api-nav",
				"frontend.api_docs": "nav-item",
			},
			description: "API section navigation state",
		},
	}

	for _, navTest := range navigationTests {
		t.Run(navTest.description, func(t *testing.T) {
			for targetRoute, expectedClass := range navTest.navLinks {
				args := []*pongo2.Value{
					pongo2.AsValue(targetRoute),
					pongo2.AsValue(navTest.currentRoute),
					pongo2.AsValue(expectedClass),
					pongo2.AsValue("nav-item"), // default inactive class
				}

				result, err := helperFunc(args...)
				if err != nil {
					t.Fatalf("Helper function returned error for route %s: %v", targetRoute, err)
				}

				resultStr := result.String()
				if targetRoute == navTest.currentRoute {
					if resultStr != expectedClass {
						t.Errorf("Route %s should be active with class %q, got %q",
							targetRoute, expectedClass, resultStr)
					}
				} else {
					if resultStr != "nav-item" {
						t.Errorf("Route %s should be inactive with class 'nav-item', got %q",
							targetRoute, resultStr)
					}
				}
			}
		})
	}
}

// TestMiddlewareContextInjection tests the concept of middleware based context injection
// This simulates how context data would be injected into templates by middleware
func TestMiddlewareContextInjection(t *testing.T) {
	// Test different context key naming conventions that middleware might use
	contextScenarios := []struct {
		name        string
		description string
		contextKeys struct {
			routeName string
			params    string
			query     string
		}
	}{
		{
			name:        "standard_keys",
			description: "Standard context key names commonly used",
			contextKeys: struct {
				routeName string
				params    string
				query     string
			}{
				routeName: "current_route_name",
				params:    "current_params",
				query:     "current_query",
			},
		},
		{
			name:        "short_keys",
			description: "Short context key names for minimal overhead",
			contextKeys: struct {
				routeName string
				params    string
				query     string
			}{
				routeName: "route",
				params:    "params",
				query:     "query",
			},
		},
		{
			name:        "explicit_keys",
			description: "Explicit descriptive context key names",
			contextKeys: struct {
				routeName string
				params    string
				query     string
			}{
				routeName: "active_route_name",
				params:    "route_parameters",
				query:     "query_parameters",
			},
		},
	}

	for _, scenario := range contextScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Simulate template data that would be injected by middleware
			templateData := map[string]any{
				scenario.contextKeys.routeName: "frontend.user.profile",
				scenario.contextKeys.params: map[string]any{
					"user_id": "123",
					"section": "personal",
				},
				scenario.contextKeys.query: map[string]string{
					"tab":  "settings",
					"edit": "true",
				},
				"user": map[string]any{
					"id":   123,
					"name": "John Doe",
				},
			}

			// Test that template data contains expected keys
			currentRoute, routeExists := templateData[scenario.contextKeys.routeName]
			if !routeExists {
				t.Errorf("Template data should contain key %q", scenario.contextKeys.routeName)
			}
			if currentRoute != "frontend.user.profile" {
				t.Errorf("Current route should be 'frontend.user.profile', got %q", currentRoute)
			}

			currentParams, paramsExist := templateData[scenario.contextKeys.params]
			if !paramsExist {
				t.Errorf("Template data should contain key %q", scenario.contextKeys.params)
			}
			if paramsMap, ok := currentParams.(map[string]any); ok {
				if paramsMap["user_id"] != "123" {
					t.Errorf("Expected user_id '123', got %q", paramsMap["user_id"])
				}
			} else {
				t.Errorf("Current params should be a map, got %T", currentParams)
			}

			currentQuery, queryExists := templateData[scenario.contextKeys.query]
			if !queryExists {
				t.Errorf("Template data should contain key %q", scenario.contextKeys.query)
			}
			if queryMap, ok := currentQuery.(map[string]string); ok {
				if queryMap["tab"] != "settings" {
					t.Errorf("Expected tab 'settings', got %q", queryMap["tab"])
				}
			} else {
				t.Errorf("Current query should be a string map, got %T", currentQuery)
			}

			// Test that other template data is preserved
			user, userExists := templateData["user"]
			if !userExists {
				t.Error("Template data should preserve non-context data")
			}
			if userMap, ok := user.(map[string]any); ok {
				if userMap["name"] != "John Doe" {
					t.Errorf("User data should be preserved: expected 'John Doe', got %q", userMap["name"])
				}
			}

			// Test that the current_route_if helper works with this context data
			config := DefaultTemplateHelperConfig()
			helperFunc := currentRouteIfHelper(config)

			currentRouteStr := currentRoute.(string)
			result, err := helperFunc(
				pongo2.AsValue(currentRouteStr),
				pongo2.AsValue(currentRouteStr),
				pongo2.AsValue("test-active"),
			)
			if err != nil {
				t.Fatalf("current_route_if helper failed with context data: %v", err)
			}
			if result.String() != "test-active" {
				t.Errorf("current_route_if helper should work with context data")
			}
		})
	}
}

// TestTemplateRenderingWithInjectedContext tests how template helpers work
// with context data that would be injected by middleware
func TestTemplateRenderingWithInjectedContext(t *testing.T) {
	// Setup route manager for testing
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":         "/",
		"user_profile": "/users/:id/profile",
		"user_posts":   "/users/:id/posts",
		"search":       "/search",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	// Get helper functions
	urlFunc := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
	currentRouteIfFunc := helpers["current_route_if"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	// Use standard context key names for testing
	const (
		currentRouteKey  = "current_route_name"
		currentParamsKey = "current_params"
		currentQueryKey  = "current_query"
	)

	// Test scenarios simulating real template rendering contexts
	testScenarios := []struct {
		name        string
		contextData map[string]any
		testCases   []contextualTestCase
		description string
	}{
		{
			name: "user profile page context",
			contextData: map[string]any{
				currentRouteKey: "frontend.user_profile",
				currentParamsKey: map[string]any{
					"id": "123",
				},
				currentQueryKey: map[string]string{
					"tab": "profile",
				},
				"user": map[string]any{
					"id":   123,
					"name": "John Doe",
				},
			},
			testCases: []contextualTestCase{
				{
					description: "generate URL for current user's posts",
					testFunc: func(t *testing.T, data map[string]any) {
						userID := data["user"].(map[string]any)["id"]
						result, err := urlFunc(
							pongo2.AsValue("frontend"),
							pongo2.AsValue("user_posts"),
							pongo2.AsValue(map[string]any{"id": userID}),
						)
						if err != nil {
							t.Fatalf("URL helper error: %v", err)
						}
						expected := "https://example.com/users/123/posts"
						if result.String() != expected {
							t.Errorf("Expected %q, got %q", expected, result.String())
						}
					},
				},
				{
					description: "check if current route is active in navigation",
					testFunc: func(t *testing.T, data map[string]any) {
						currentRoute := data[currentRouteKey].(string)
						result, err := currentRouteIfFunc(
							pongo2.AsValue("frontend.user_profile"),
							pongo2.AsValue(currentRoute),
							pongo2.AsValue("nav-active"),
							pongo2.AsValue("nav-inactive"),
						)
						if err != nil {
							t.Fatalf("current_route_if helper error: %v", err)
						}
						if result.String() != "nav-active" {
							t.Errorf("Expected 'nav-active', got %q", result.String())
						}
					},
				},
			},
			description: "User viewing their profile page",
		},
		{
			name: "search page context",
			contextData: map[string]any{
				currentRouteKey:  "frontend.search",
				currentParamsKey: map[string]any{},
				currentQueryKey: map[string]string{
					"q":    "golang",
					"page": "2",
				},
			},
			testCases: []contextualTestCase{
				{
					description: "rebuild search URL with modified query",
					testFunc: func(t *testing.T, data map[string]any) {
						currentQuery := data[currentQueryKey].(map[string]string)
						// Create modified query for next page
						newQuery := make(map[string]string)
						for k, v := range currentQuery {
							newQuery[k] = v
						}
						newQuery["page"] = "3"

						result, err := urlFunc(
							pongo2.AsValue("frontend"),
							pongo2.AsValue("search"),
							pongo2.AsValue(map[string]any{}),
							pongo2.AsValue(newQuery),
						)
						if err != nil {
							t.Fatalf("URL helper error: %v", err)
						}
						resultStr := result.String()
						if !containsString(resultStr, "q=golang") {
							t.Errorf("Expected q=golang in result: %s", resultStr)
						}
						if !containsString(resultStr, "page=3") {
							t.Errorf("Expected page=3 in result: %s", resultStr)
						}
					},
				},
			},
			description: "User on search results page",
		},
		{
			name: "home page context",
			contextData: map[string]any{
				currentRouteKey:  "frontend.home",
				currentParamsKey: map[string]any{},
				currentQueryKey:  map[string]string{},
			},
			testCases: []contextualTestCase{
				{
					description: "verify navigation state on home page",
					testFunc: func(t *testing.T, data map[string]any) {
						currentRoute := data[currentRouteKey].(string)

						// Test home link is active
						result, err := currentRouteIfFunc(
							pongo2.AsValue("frontend.home"),
							pongo2.AsValue(currentRoute),
							pongo2.AsValue("active"),
						)
						if err != nil {
							t.Fatalf("current_route_if helper error: %v", err)
						}
						if result.String() != "active" {
							t.Errorf("Home link should be active, got %q", result.String())
						}

						// Test other links are not active
						result, err = currentRouteIfFunc(
							pongo2.AsValue("frontend.user_profile"),
							pongo2.AsValue(currentRoute),
							pongo2.AsValue("active"),
							pongo2.AsValue("inactive"),
						)
						if err != nil {
							t.Fatalf("current_route_if helper error: %v", err)
						}
						if result.String() != "inactive" {
							t.Errorf("Profile link should be inactive, got %q", result.String())
						}
					},
				},
			},
			description: "User on home page",
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			for i, testCase := range scenario.testCases {
				t.Run(fmt.Sprintf("case_%d_%s", i+1, testCase.description), func(t *testing.T) {
					testCase.testFunc(t, scenario.contextData)
				})
			}
		})
	}
}

// contextualTestCase represents a test case within a specific context scenario
type contextualTestCase struct {
	description string
	testFunc    func(t *testing.T, contextData map[string]any)
}

// TestMockMiddlewareContextData tests various mock context data scenarios
// that simulate different middleware implementations
func TestMockMiddlewareContextData(t *testing.T) {
	// Use standard context key names for consistent testing
	const (
		currentRouteKey  = "current_route_name"
		currentParamsKey = "current_params"
		currentQueryKey  = "current_query"
	)

	mockScenarios := []struct {
		name        string
		contextData map[string]any
		validates   func(t *testing.T, data map[string]any)
		description string
	}{
		{
			name: "basic web request context",
			contextData: map[string]any{
				currentRouteKey:  "frontend.home",
				currentParamsKey: map[string]any{},
				currentQueryKey:  map[string]string{},
				"request_method": "GET",
				"request_path":   "/",
			},
			validates: func(t *testing.T, data map[string]any) {
				if data[currentRouteKey] != "frontend.home" {
					t.Error("Current route should be frontend.home")
				}
				if data["request_method"] != "GET" {
					t.Error("Request method should be preserved")
				}
			},
			description: "Basic GET request to home page",
		},
		{
			name: "authenticated user context",
			contextData: map[string]any{
				currentRouteKey:  "frontend.user_dashboard",
				currentParamsKey: map[string]any{"user_id": "456"},
				currentQueryKey:  map[string]string{"view": "summary"},
				"user": map[string]any{
					"id":          456,
					"name":        "Jane Smith",
					"is_admin":    true,
					"permissions": []string{"read", "write", "delete"},
				},
				"session_id": "sess_abc123",
				"csrf_token": "csrf_xyz789",
			},
			validates: func(t *testing.T, data map[string]any) {
				params := data[currentParamsKey].(map[string]any)
				if params["user_id"] != "456" {
					t.Error("User ID should be preserved in params")
				}

				user := data["user"].(map[string]any)
				if user["is_admin"] != true {
					t.Error("User admin status should be preserved")
				}

				permissions := user["permissions"].([]string)
				if len(permissions) != 3 {
					t.Error("User permissions should be preserved")
				}
			},
			description: "Authenticated admin user viewing dashboard",
		},
		{
			name: "API request context",
			contextData: map[string]any{
				currentRouteKey:  "api.v1.users.list",
				currentParamsKey: map[string]any{},
				currentQueryKey: map[string]string{
					"page":   "5",
					"limit":  "20",
					"sort":   "created_at",
					"order":  "desc",
					"filter": "active",
				},
				"api_version":   "v1",
				"rate_limit":    map[string]any{"remaining": 95, "reset": 1609459200},
				"request_id":    "req_123456789",
				"accept_format": "json",
			},
			validates: func(t *testing.T, data map[string]any) {
				query := data[currentQueryKey].(map[string]string)
				if query["page"] != "5" {
					t.Error("Pagination should be preserved")
				}
				if query["sort"] != "created_at" {
					t.Error("Sort parameters should be preserved")
				}

				if data["api_version"] != "v1" {
					t.Error("API version should be preserved")
				}

				rateLimit := data["rate_limit"].(map[string]any)
				if rateLimit["remaining"] != 95 {
					t.Error("Rate limit data should be preserved")
				}
			},
			description: "API request with pagination and filtering",
		},
		{
			name: "multilingual site context",
			contextData: map[string]any{
				currentRouteKey:      "frontend.about",
				currentParamsKey:     map[string]any{},
				currentQueryKey:      map[string]string{},
				"locale":             "es",
				"available_locales":  []string{"en", "es", "fr", "de"},
				"language_direction": "ltr",
				"currency":           "EUR",
				"timezone":           "Europe/Madrid",
			},
			validates: func(t *testing.T, data map[string]any) {
				if data["locale"] != "es" {
					t.Error("Current locale should be preserved")
				}

				locales := data["available_locales"].([]string)
				if len(locales) != 4 {
					t.Error("Available locales should be preserved")
				}

				if data["currency"] != "EUR" {
					t.Error("Localization data should be preserved")
				}
			},
			description: "Spanish user viewing about page",
		},
		{
			name: "error page context",
			contextData: map[string]any{
				currentRouteKey:  "frontend.error",
				currentParamsKey: map[string]any{"code": "404"},
				currentQueryKey:  map[string]string{},
				"error": map[string]any{
					"code":          404,
					"message":       "Page not found",
					"original_path": "/users/999/profile",
					"timestamp":     "2023-01-01T12:00:00Z",
				},
				"referrer":   "https://example.com/users",
				"user_agent": "Mozilla/5.0...",
			},
			validates: func(t *testing.T, data map[string]any) {
				params := data[currentParamsKey].(map[string]any)
				if params["code"] != "404" {
					t.Error("Error code should be preserved in params")
				}

				errorData := data["error"].(map[string]any)
				if errorData["code"] != 404 {
					t.Error("Error details should be preserved")
				}
				if errorData["original_path"] != "/users/999/profile" {
					t.Error("Original path should be preserved for debugging")
				}
			},
			description: "404 error page with debugging context",
		},
	}

	for _, scenario := range mockScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Validate that all expected context keys are present
			if _, exists := scenario.contextData[currentRouteKey]; !exists {
				t.Errorf("Mock context should include %s", currentRouteKey)
			}
			if _, exists := scenario.contextData[currentParamsKey]; !exists {
				t.Errorf("Mock context should include %s", currentParamsKey)
			}
			if _, exists := scenario.contextData[currentQueryKey]; !exists {
				t.Errorf("Mock context should include %s", currentQueryKey)
			}

			// Run custom validations
			scenario.validates(t, scenario.contextData)

			// Verify that context data can be used with current_route_if helper
			currentRoute := scenario.contextData[currentRouteKey].(string)
			config := DefaultTemplateHelperConfig()
			helperFunc := currentRouteIfHelper(config)

			result, err := helperFunc(
				pongo2.AsValue(currentRoute),
				pongo2.AsValue(currentRoute),
				pongo2.AsValue("test-active"),
			)
			if err != nil {
				t.Fatalf("current_route_if helper failed with mock data: %v", err)
			}
			if result.String() != "test-active" {
				t.Errorf("Mock context should work with current_route_if helper")
			}
		})
	}
}
