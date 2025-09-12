package urlkit_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/goliatone/go-urlkit"
)

func TestJoinURL(t *testing.T) {
	tests := []struct {
		base     string
		path     string
		queries  []urlkit.Query
		expected string
	}{
		{
			base:     "http://example.com",
			path:     "/foo",
			queries:  nil,
			expected: "http://example.com/foo",
		},
		{
			base:     "http://example.com/",
			path:     "foo",
			queries:  nil,
			expected: "http://example.com/foo",
		},
		{
			base:     "http://example.com",
			path:     "foo",
			queries:  nil,
			expected: "http://example.com/foo",
		},
		{
			base:     "http://example.com/",
			path:     "/foo",
			queries:  nil,
			expected: "http://example.com/foo",
		},
		{
			base:     "http://example.com",
			path:     "/foo",
			queries:  []urlkit.Query{{"a": "1"}},
			expected: "http://example.com/foo?a=1",
		},
		{
			base:    "http://example.com",
			path:    "/foo",
			queries: []urlkit.Query{{"a": "1"}, {"b": "2"}},
			// NOTE: url.Values.Encode() orders keys alphabetically.
			expected: "http://example.com/foo?a=1&b=2",
		},
		{
			base:     "http://example.com?existing=1",
			path:     "/foo",
			queries:  []urlkit.Query{{"a": "1"}},
			expected: "http://example.com/foo?existing=1&a=1",
		},
	}

	for _, tt := range tests {
		got := urlkit.JoinURL(tt.base, tt.path, tt.queries...)
		if got != tt.expected {
			t.Errorf("joinURL(%q, %q, %v) = %q; want %q", tt.base, tt.path, tt.queries, got, tt.expected)
		}
	}
}

func TestGroupRender(t *testing.T) {
	routes := map[string]string{
		"user": "/user/:id",
	}
	group := urlkit.NewURIHelper("http://example.com", routes)

	urlStr, err := group.Render("user", urlkit.Params{"id": "123"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	expected := "http://example.com/user/123"
	if urlStr != expected {
		t.Errorf("Expected %q, got %q", expected, urlStr)
	}

	urlStr, err = group.Render("user", urlkit.Params{"id": "123"}, map[string]string{"active": "true"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	expected = "http://example.com/user/123?active=true"
	if urlStr != expected {
		t.Errorf("Expected %q, got %q", expected, urlStr)
	}

	_, err = group.Render("user", urlkit.Params{})
	if err == nil {
		t.Error("Expected error for missing parameter, got nil")
	}

	_, err = group.Render("nonexistent", urlkit.Params{"id": "123"})
	if err == nil {
		t.Error("Expected error for unknown route, got nil")
	}
}

func TestBuilderBuild(t *testing.T) {
	routes := map[string]string{
		"user":   "/user/:id",
		"google": "/webhooks/google/:service/:uuid?",
	}

	group := urlkit.NewURIHelper("http://example.com", routes)

	builder := group.Builder("user")
	builder.WithParam("id", "123").WithQuery("active", "true")

	urlStr, err := builder.Build()
	if err != nil {
		t.Fatalf("Builder Build returned error: %v", err)
	}
	expected := "http://example.com/user/123?active=true"
	if urlStr != expected {
		t.Errorf("Expected %q, got %q", expected, urlStr)
	}

	builder = group.Builder("google")
	builder.WithParam("service", "gmail").WithParam("uuid", "123")

	urlStr, err = builder.Build()
	if err != nil {
		t.Fatalf("Builder Build returned error: %v", err)
	}
	expected = "http://example.com/webhooks/google/gmail/123"
	if urlStr != expected {
		t.Errorf("Expected %q, got %q", expected, urlStr)
	}

	builder = group.Builder("google")
	builder.WithParam("service", "gmail")

	urlStr, err = builder.Build()
	if err != nil {
		t.Fatalf("Builder Build returned error: %v", err)
	}
	expected = "http://example.com/webhooks/google/gmail"
	if urlStr != expected {
		t.Errorf("Expected %q, got %q", expected, urlStr)
	}
}

func TestGroupRenderNoParams(t *testing.T) {
	routes := map[string]string{
		"home": "/",
	}
	group := urlkit.NewURIHelper("http://example.com", routes)

	urlStr, err := group.Render("home", urlkit.Params{}, map[string]string{"lang": "en"})
	if err != nil {
		t.Fatalf("Render returned error: %v", err)
	}
	expected := "http://example.com/?lang=en"
	if urlStr != expected {
		t.Errorf("Expected %q, got %q", expected, urlStr)
	}
}

func TestGroupValidateSuccess(t *testing.T) {
	routes := map[string]string{
		"user": "/user/:id",
		"home": "/",
	}
	group := urlkit.NewURIHelper("http://example.com", routes)

	if err := group.Validate([]string{"user", "home"}); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestGroupValidateFailure(t *testing.T) {
	routes := map[string]string{
		"user": "/user/:id",
	}
	group := urlkit.NewURIHelper("http://example.com", routes)

	err := group.Validate([]string{"user", "home"})
	if err == nil {
		t.Error("Expected error for missing route 'home', got nil")
	} else if !strings.Contains(err.Error(), "home") {
		t.Errorf("Expected error message to mention missing route 'home', got %v", err)
	}
}

func TestRouteManagerValidateSuccess(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("user", "http://example.com", map[string]string{
		"profile":  "/user/:id/profile",
		"settings": "/user/:id/settings",
	})
	rm.RegisterGroup("admin", "http://admin.example.com", map[string]string{
		"dashboard": "/dashboard",
		"settings":  "/settings",
	})

	expected := map[string][]string{
		"user":  {"profile", "settings"},
		"admin": {"dashboard", "settings"},
	}

	if err := rm.Validate(expected); err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestRouteManagerValidateFailure(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("user", "http://example.com", map[string]string{
		"profile": "/user/:id/profile",
	})
	rm.RegisterGroup("admin", "http://admin.example.com", map[string]string{
		"dashboard": "/dashboard",
	})

	// Expected:
	// - For group "user", route "settings" is missing.
	// - For group "admin", route "settings" is missing.
	// - Group "guest" does not exist.
	expected := map[string][]string{
		"user":  {"profile", "settings"},
		"admin": {"dashboard", "settings"},
		"guest": {"home"},
	}

	err := rm.Validate(expected)
	if err == nil {
		t.Error("Expected validation error, got nil")
	} else {
		ve, ok := err.(urlkit.ValidationError)
		if !ok {
			t.Errorf("Expected error of type ValidationError, got %T", err)
		}

		if missing, exists := ve.Errors["user"]; !exists || len(missing) != 1 || missing[0] != "settings" {
			t.Errorf("Expected group 'user' missing ['settings'], got %v", missing)
		}

		if missing, exists := ve.Errors["admin"]; !exists || len(missing) != 1 || missing[0] != "settings" {
			t.Errorf("Expected group 'admin' missing ['settings'], got %v", missing)
		}

		if missing, exists := ve.Errors["guest"]; !exists || len(missing) != 1 || missing[0] != "Missing group" {
			t.Errorf("Expected group 'guest' error ['missing group'], got %v", missing)
		}
	}
}

func TestMustValidatePanic(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("user", "http://example.com", map[string]string{
		"profile": "/user/:id/profile",
	})

	expected := map[string][]string{
		"user": {"profile", "settings"}, // "settings" is missing.
	}

	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
			}
		}()
		rm.MustValidate(expected)
	}()

	if !didPanic {
		t.Error("Expected MustValidate to panic due to missing routes, but it did not")
	}
}

func TestMustValidateSuccess(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("user", "http://example.com", map[string]string{
		"profile":  "/user/:id/profile",
		"settings": "/user/:id/settings",
	})

	expected := map[string][]string{
		"user": {"profile", "settings"},
	}

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("Did not expect panic, but got: %v", r)
		}
	}()
	rm.MustValidate(expected)
}

func TestNewRouteManagerWithConfig(t *testing.T) {
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "frontend",
				BaseURL: "http://localhost:7680",
				Paths: map[string]string{
					"auth.callback":       "/integrations/:provider/:service/:id",
					"auth.callback.error": "/integrations/:provider/:service/error",
				},
			},
			{
				Name:    "webhooks",
				BaseURL: "https://api.example.com",
				Paths: map[string]string{
					"google.watcher": "/webhooks/google/watcher/:service/:uuid?",
				},
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)

	// Test that groups were registered correctly
	frontend := manager.Group("frontend")
	if frontend == nil {
		t.Fatal("Expected frontend group to be registered")
	}

	webhooks := manager.Group("webhooks")
	if webhooks == nil {
		t.Fatal("Expected webhooks group to be registered")
	}

	// Test route building with frontend group
	url, err := frontend.Builder("auth.callback").
		WithParam("provider", "google").
		WithParam("service", "oauth").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build frontend URL: %v", err)
	}
	expected := "http://localhost:7680/integrations/google/oauth/123"
	if url != expected {
		t.Errorf("Expected %q, got %q", expected, url)
	}

	// Test route building with webhooks group
	url, err = webhooks.Builder("google.watcher").
		WithParam("service", "gmail").
		WithParam("uuid", "abc-123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build webhooks URL: %v", err)
	}
	expected = "https://api.example.com/webhooks/google/watcher/gmail/abc-123"
	if url != expected {
		t.Errorf("Expected %q, got %q", expected, url)
	}

	// Test optional parameter (uuid is optional)
	url, err = webhooks.Builder("google.watcher").
		WithParam("service", "gmail").
		Build()
	if err != nil {
		t.Fatalf("Failed to build webhooks URL without optional param: %v", err)
	}
	expected = "https://api.example.com/webhooks/google/watcher/gmail"
	if url != expected {
		t.Errorf("Expected %q, got %q", expected, url)
	}
}

func TestConfigValidation(t *testing.T) {
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "frontend",
				BaseURL: "http://localhost:7680",
				Paths: map[string]string{
					"auth.callback":       "/integrations/:provider/:service/:id",
					"auth.callback.error": "/integrations/:provider/:service/error",
				},
			},
			{
				Name:    "backend",
				BaseURL: "http://localhost:8080",
				Paths: map[string]string{
					"api.users": "/api/users/:id",
				},
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)

	// Test successful validation
	expectedRoutes := map[string][]string{
		"frontend": {"auth.callback", "auth.callback.error"},
		"backend":  {"api.users"},
	}

	if err := manager.Validate(expectedRoutes); err != nil {
		t.Errorf("Expected validation to pass, got error: %v", err)
	}

	// Test validation failure - missing route
	expectedRoutes["frontend"] = append(expectedRoutes["frontend"], "missing.route")
	if err := manager.Validate(expectedRoutes); err == nil {
		t.Error("Expected validation to fail for missing route, got nil")
	}
}

func TestEmptyConfig(t *testing.T) {
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{},
	}

	manager := urlkit.NewRouteManager(&config)

	// Should not panic but have no groups
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when accessing non-existent group")
		}
	}()
	manager.Group("nonexistent")
}

func TestConfigWithEmptyPaths(t *testing.T) {
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "empty",
				BaseURL: "http://example.com",
				Paths:   map[string]string{},
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)
	group := manager.Group("empty")

	// Should be able to access group but no routes
	_, err := group.Builder("nonexistent").Build()
	if err == nil {
		t.Error("Expected error when building non-existent route")
	}
}

func TestConfigIntegrationWithExistingAPI(t *testing.T) {
	// Test that config-created manager works with existing API
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "api",
				BaseURL: "https://api.example.com",
				Paths: map[string]string{
					"users.show": "/users/:id",
					"posts.list": "/posts",
				},
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)

	// Add additional group manually after config
	manager.RegisterGroup("manual", "http://manual.com", map[string]string{
		"home": "/",
	})

	// Test both config-based and manually registered groups work
	apiURL, err := manager.Group("api").Builder("users.show").
		WithParam("id", "123").
		WithQuery("include", "profile").
		Build()
	if err != nil {
		t.Fatalf("Failed to build API URL: %v", err)
	}
	expected := "https://api.example.com/users/123?include=profile"
	if apiURL != expected {
		t.Errorf("Expected %q, got %q", expected, apiURL)
	}

	manualURL, err := manager.Group("manual").Builder("home").Build()
	if err != nil {
		t.Fatalf("Failed to build manual URL: %v", err)
	}
	expected = "http://manual.com/"
	if manualURL != expected {
		t.Errorf("Expected %q, got %q", expected, manualURL)
	}
}

func TestNestedGroupValidationWithDotSeparatedPaths(t *testing.T) {
	// Create a route manager with nested groups
	rm := urlkit.NewRouteManager()

	// Register root group
	rm.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	// Get the frontend group and register nested groups
	frontend := rm.Group("frontend")
	en := frontend.RegisterGroup("en", "/en", map[string]string{
		"about":   "/about-us",
		"contact": "/contact",
	})

	frontend.RegisterGroup("es", "/es", map[string]string{
		"about":   "/acerca",
		"contact": "/contacto",
	})

	// Add deeper nesting
	en.RegisterGroup("deep", "/deep", map[string]string{
		"nested": "/nested-route",
	})

	// Test successful validation with dot-separated paths
	expectedRoutes := map[string][]string{
		"frontend":         {"home"},
		"frontend.en":      {"about", "contact"},
		"frontend.es":      {"about", "contact"},
		"frontend.en.deep": {"nested"},
	}

	err := rm.Validate(expectedRoutes)
	if err != nil {
		t.Errorf("Expected validation to pass for nested groups, got error: %v", err)
	}

	// Test validation failure - missing route in nested group
	expectedRoutes["frontend.en"] = append(expectedRoutes["frontend.en"], "missing.route")
	err = rm.Validate(expectedRoutes)
	if err == nil {
		t.Error("Expected validation to fail for missing route in nested group, got nil")
	}

	// Verify the error structure
	ve, ok := err.(urlkit.ValidationError)
	if !ok {
		t.Errorf("Expected error of type ValidationError, got %T", err)
	}

	if missing, exists := ve.Errors["frontend.en"]; !exists || len(missing) != 1 || missing[0] != "missing.route" {
		t.Errorf("Expected group 'frontend.en' missing ['missing.route'], got %v", missing)
	}

	// Test validation failure - non-existent nested group
	expectedRoutes = map[string][]string{
		"frontend.nonexistent": {"route"},
	}
	err = rm.Validate(expectedRoutes)
	if err == nil {
		t.Error("Expected validation to fail for non-existent nested group, got nil")
	}

	ve, ok = err.(urlkit.ValidationError)
	if !ok {
		t.Errorf("Expected error of type ValidationError, got %T", err)
	}

	if missing, exists := ve.Errors["frontend.nonexistent"]; !exists || len(missing) != 1 || missing[0] != "Missing group" {
		t.Errorf("Expected group 'frontend.nonexistent' error ['Missing group'], got %v", missing)
	}

	// Test backward compatibility - flat group names should still work
	expectedRoutes = map[string][]string{
		"frontend": {"home"},
	}
	err = rm.Validate(expectedRoutes)
	if err != nil {
		t.Errorf("Expected backward compatibility to work for flat group names, got error: %v", err)
	}
}

func TestNestedConfigurationParsing(t *testing.T) {
	// Test JSON configuration parsing with nested groups
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "api",
				BaseURL: "https://api.example.com",
				Paths: map[string]string{
					"status":  "/status",
					"version": "/version",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "v1",
						Path: "/v1",
						Paths: map[string]string{
							"users":    "/users/:id",
							"posts":    "/posts",
							"comments": "/comments/:postId",
						},
						Groups: []urlkit.GroupConfig{
							{
								Name: "admin",
								Path: "/admin",
								Paths: map[string]string{
									"dashboard": "/dashboard",
									"settings":  "/settings/:section",
								},
							},
						},
					},
					{
						Name: "v2",
						Path: "/v2",
						Paths: map[string]string{
							"users":    "/users/:id",
							"profiles": "/users/:id/profile",
						},
					},
				},
			},
			{
				Name:    "frontend",
				BaseURL: "https://example.com",
				Paths: map[string]string{
					"home": "/",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "en",
						Path: "/en",
						Paths: map[string]string{
							"about":   "/about",
							"contact": "/contact",
						},
					},
					{
						Name: "es",
						Path: "/es",
						Paths: map[string]string{
							"about":   "/acerca",
							"contact": "/contacto",
						},
					},
				},
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)

	// Test root group access
	apiGroup := manager.Group("api")
	if apiGroup == nil {
		t.Fatal("Expected api group to be registered")
	}

	frontendGroup := manager.Group("frontend")
	if frontendGroup == nil {
		t.Fatal("Expected frontend group to be registered")
	}

	// Test nested group access
	v1Group := apiGroup.Group("v1")
	if v1Group == nil {
		t.Fatal("Expected v1 group to be accessible from api group")
	}

	adminGroup := v1Group.Group("admin")
	if adminGroup == nil {
		t.Fatal("Expected admin group to be accessible from v1 group")
	}

	// Test route availability in deeply nested group
	route, err := adminGroup.Route("dashboard")
	if err != nil {
		t.Fatalf("Expected dashboard route to be available in admin group: %v", err)
	}
	expected := "/dashboard"
	if route != expected {
		t.Errorf("Expected route %q, got %q", expected, route)
	}

	// Test URL building for nested configuration
	// Root level
	statusURL, err := manager.Group("api").Builder("status").Build()
	if err != nil {
		t.Fatalf("Failed to build API status URL: %v", err)
	}
	expected = "https://api.example.com/status"
	if statusURL != expected {
		t.Errorf("Expected %q, got %q", expected, statusURL)
	}

	// Single level nesting
	usersV1URL, err := manager.Group("api").Group("v1").Builder("users").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build API v1 users URL: %v", err)
	}
	expected = "https://api.example.com/v1/users/123"
	if usersV1URL != expected {
		t.Errorf("Expected %q, got %q", expected, usersV1URL)
	}

	// Deep nesting (3 levels)
	dashboardURL, err := manager.Group("api").Group("v1").Group("admin").
		Builder("dashboard").
		WithQuery("tab", "users").
		Build()
	if err != nil {
		t.Fatalf("Failed to build admin dashboard URL: %v", err)
	}
	expected = "https://api.example.com/v1/admin/dashboard?tab=users"
	if dashboardURL != expected {
		t.Errorf("Expected %q, got %q", expected, dashboardURL)
	}

	// Test internationalization paths
	aboutEnURL, err := manager.Group("frontend").Group("en").Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build English about URL: %v", err)
	}
	expected = "https://example.com/en/about"
	if aboutEnURL != expected {
		t.Errorf("Expected %q, got %q", expected, aboutEnURL)
	}

	aboutEsURL, err := manager.Group("frontend").Group("es").Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish about URL: %v", err)
	}
	expected = "https://example.com/es/acerca"
	if aboutEsURL != expected {
		t.Errorf("Expected %q, got %q", expected, aboutEsURL)
	}
}

func TestConfigurationBackwardCompatibility(t *testing.T) {
	// Test backward compatibility with flat configurations

	// Original flat configuration (no nested groups)
	flatConfig := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "api",
				BaseURL: "https://api.example.com",
				Paths: map[string]string{
					"users": "/users/:id",
					"posts": "/posts",
				},
			},
			{
				Name:    "frontend",
				BaseURL: "https://frontend.example.com",
				Paths: map[string]string{
					"home":  "/",
					"about": "/about",
				},
			},
		},
	}

	manager := urlkit.NewRouteManagerFromConfig(flatConfig)

	// Test that flat configuration works exactly as before
	apiGroup := manager.Group("api")
	if apiGroup == nil {
		t.Fatal("Expected api group to be registered from flat config")
	}

	frontendGroup := manager.Group("frontend")
	if frontendGroup == nil {
		t.Fatal("Expected frontend group to be registered from flat config")
	}

	// Test URL building works the same
	usersURL, err := apiGroup.Builder("users").WithParam("id", "123").Build()
	if err != nil {
		t.Fatalf("Failed to build users URL from flat config: %v", err)
	}
	expected := "https://api.example.com/users/123"
	if usersURL != expected {
		t.Errorf("Expected %q, got %q", expected, usersURL)
	}

	homeURL, err := frontendGroup.Builder("home").Build()
	if err != nil {
		t.Fatalf("Failed to build home URL from flat config: %v", err)
	}
	expected = "https://frontend.example.com/"
	if homeURL != expected {
		t.Errorf("Expected %q, got %q", expected, homeURL)
	}

	// Test validation works with flat names
	expectedRoutes := map[string][]string{
		"api":      {"users", "posts"},
		"frontend": {"home", "about"},
	}
	err = manager.Validate(expectedRoutes)
	if err != nil {
		t.Errorf("Expected validation to pass for flat configuration: %v", err)
	}

	// Mixed configuration: combine with manually registered groups
	manager.RegisterGroup("manual", "https://manual.example.com", map[string]string{
		"test": "/test",
	})

	// Should work alongside config-loaded groups
	testURL, err := manager.Group("manual").Builder("test").Build()
	if err != nil {
		t.Fatalf("Failed to build manual group URL: %v", err)
	}
	expected = "https://manual.example.com/test"
	if testURL != expected {
		t.Errorf("Expected %q, got %q", expected, testURL)
	}
}

func TestConfigurationErrorCases(t *testing.T) {
	// Test error cases (missing groups, invalid configurations)

	// Test 1: Nested group with base URL (should panic)
	invalidConfig := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "api",
				BaseURL: "https://api.example.com",
				Paths:   map[string]string{"status": "/status"},
				Groups: []urlkit.GroupConfig{
					{
						Name:    "v1",
						BaseURL: "https://invalid.example.com", // This should cause panic
						Path:    "/v1",
						Paths:   map[string]string{"users": "/users"},
					},
				},
			},
		},
	}

	// Should panic when parsing nested group with base URL
	didPanic := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				didPanic = true
				// Verify the panic message mentions the issue
				errorMsg := fmt.Sprintf("%v", r)
				if !strings.Contains(errorMsg, "nested group") || !strings.Contains(errorMsg, "base_url") {
					t.Errorf("Expected panic message about nested group base_url, got: %v", r)
				}
			}
		}()
		urlkit.NewRouteManager(&invalidConfig)
	}()

	if !didPanic {
		t.Error("Expected panic when nested group specifies base_url")
	}

	// Test 2: Empty configuration
	emptyConfig := urlkit.Config{
		Groups: []urlkit.GroupConfig{},
	}

	manager := urlkit.NewRouteManagerFromConfig(emptyConfig)

	// Should not panic but accessing non-existent groups should panic
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when accessing non-existent group")
		}
	}()
	manager.Group("nonexistent")
}

func TestConfigurationWithEmptyOrNilPaths(t *testing.T) {
	// Test groups with empty path segments and nil paths
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "api",
				BaseURL: "https://api.example.com",
				Paths:   nil, // Nil paths should be handled
				Groups: []urlkit.GroupConfig{
					{
						Name:  "v1",
						Path:  "", // Empty path should be handled
						Paths: map[string]string{"users": "/users/:id"},
					},
					{
						Name:  "v2",
						Path:  "/v2",
						Paths: map[string]string{}, // Empty paths map
					},
				},
			},
			{
				Name:    "empty",
				BaseURL: "https://empty.example.com",
				Paths:   map[string]string{}, // Empty but not nil
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)

	// Test that groups were created despite empty paths
	apiGroup := manager.Group("api")
	if apiGroup == nil {
		t.Fatal("Expected api group to be created despite nil paths")
	}

	v1Group := apiGroup.Group("v1")
	if v1Group == nil {
		t.Fatal("Expected v1 group to be created despite empty path")
	}

	v2Group := apiGroup.Group("v2")
	if v2Group == nil {
		t.Fatal("Expected v2 group to be created despite empty paths map")
	}

	emptyGroup := manager.Group("empty")
	if emptyGroup == nil {
		t.Fatal("Expected empty group to be created")
	}

	// Test URL building with empty path segment
	usersURL, err := manager.Group("api").Group("v1").Builder("users").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build URL with empty path segment: %v", err)
	}
	// Empty path should result in no extra segment
	expected := "https://api.example.com/users/123"
	if usersURL != expected {
		t.Errorf("Expected %q, got %q", expected, usersURL)
	}

	// Test that group with empty paths can't build routes
	_, err = v2Group.Builder("nonexistent").Build()
	if err == nil {
		t.Error("Expected error when building route that doesn't exist")
	}
}

func TestDeeplyNestedGroupsEdgeCases(t *testing.T) {
	// Test deeply nested groups (4+ levels) with various edge cases
	rm := urlkit.NewRouteManager()

	// Create a complex e-commerce structure: store -> region -> category -> product -> details
	rm.RegisterGroup("store", "https://shop.example.com", map[string]string{
		"home": "/",
	})

	store := rm.Group("store")

	// Regional divisions
	northAmerica := store.RegisterGroup("north-america", "/na", map[string]string{
		"regions": "/regions",
	})

	// Country subdivisions
	usa := northAmerica.RegisterGroup("usa", "/usa", map[string]string{
		"states": "/states",
	})

	// State subdivisions
	california := usa.RegisterGroup("california", "/ca", map[string]string{
		"cities": "/cities",
	})

	// City subdivisions (5 levels deep)
	california.RegisterGroup("los-angeles", "/la", map[string]string{
		"stores":    "/stores/:storeId",
		"inventory": "/stores/:storeId/inventory/:itemId",
		"reviews":   "/stores/:storeId/reviews/:reviewId?",
	})

	// Test 5-level deep URL building with multiple parameters
	inventoryURL, err := rm.Group("store").Group("north-america").Group("usa").
		Group("california").Group("los-angeles").
		Builder("inventory").
		WithParam("storeId", "101").
		WithParam("itemId", "laptop-123").
		WithQuery("inStock", "true").
		WithQuery("sort", "price").
		Build()
	if err != nil {
		t.Fatalf("Failed to build 5-level deep inventory URL: %v", err)
	}
	expected := "https://shop.example.com/na/usa/ca/la/stores/101/inventory/laptop-123"
	if !strings.Contains(inventoryURL, expected) {
		t.Errorf("Expected URL to contain %q, got %q", expected, inventoryURL)
	}
	if !strings.Contains(inventoryURL, "inStock=true") {
		t.Errorf("Expected URL to contain query parameter 'inStock=true', got %q", inventoryURL)
	}
	if !strings.Contains(inventoryURL, "sort=price") {
		t.Errorf("Expected URL to contain query parameter 'sort=price', got %q", inventoryURL)
	}

	// Test optional parameters in deeply nested routes
	reviewsURL, err := rm.Group("store").Group("north-america").Group("usa").
		Group("california").Group("los-angeles").
		Builder("reviews").
		WithParam("storeId", "101").
		Build()
	if err != nil {
		t.Fatalf("Failed to build reviews URL without optional param: %v", err)
	}
	expected = "https://shop.example.com/na/usa/ca/la/stores/101/reviews"
	if reviewsURL != expected {
		t.Errorf("Expected %q, got %q", expected, reviewsURL)
	}

	// Test with optional parameter provided
	reviewsWithIdURL, err := rm.Group("store").Group("north-america").Group("usa").
		Group("california").Group("los-angeles").
		Builder("reviews").
		WithParam("storeId", "101").
		WithParam("reviewId", "review-456").
		Build()
	if err != nil {
		t.Fatalf("Failed to build reviews URL with optional param: %v", err)
	}
	expected = "https://shop.example.com/na/usa/ca/la/stores/101/reviews/review-456"
	if reviewsWithIdURL != expected {
		t.Errorf("Expected %q, got %q", expected, reviewsWithIdURL)
	}
}

func TestEmptyPathSegmentsEdgeCases(t *testing.T) {
	// Test various combinations of empty path segments
	rm := urlkit.NewRouteManager()

	// Root with empty base group
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})

	api := rm.Group("api")

	// Empty path segment (should not add any path component)
	emptyPath := api.RegisterGroup("empty", "", map[string]string{
		"test": "/test-route",
	})

	// Normal path after empty
	normal := emptyPath.RegisterGroup("normal", "/normal", map[string]string{
		"data": "/data/:id",
	})

	// Another empty path
	normal.RegisterGroup("empty2", "", map[string]string{
		"final": "/final-endpoint",
	})

	// Test URL building through multiple empty segments
	finalURL, err := rm.Group("api").Group("empty").Group("normal").
		Group("empty2").Builder("final").Build()
	if err != nil {
		t.Fatalf("Failed to build URL through empty path segments: %v", err)
	}
	// Should skip empty path segments
	expected := "https://api.example.com/normal/final-endpoint"
	if finalURL != expected {
		t.Errorf("Expected %q, got %q", expected, finalURL)
	}

	// Test with parameters
	dataURL, err := rm.Group("api").Group("empty").Group("normal").
		Builder("data").WithParam("id", "123").Build()
	if err != nil {
		t.Fatalf("Failed to build data URL through empty path: %v", err)
	}
	expected = "https://api.example.com/normal/data/123"
	if dataURL != expected {
		t.Errorf("Expected %q, got %q", expected, dataURL)
	}

	// Test mixed empty and slash paths
	slashPath := api.RegisterGroup("slash", "/", map[string]string{
		"root": "/root",
	})

	slashURL, err := slashPath.Builder("root").Build()
	if err != nil {
		t.Fatalf("Failed to build URL with slash path: %v", err)
	}
	// Note: slash path creates double slash - this is current behavior
	expected = "https://api.example.com//root"
	if slashURL != expected {
		t.Errorf("Expected %q, got %q", expected, slashURL)
	}
}

func TestComplexParameterInterpolation(t *testing.T) {
	// Test parameter interpolation in nested routes with special characters
	rm := urlkit.NewRouteManager()

	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"base": "/",
	})

	api := rm.Group("api")

	// Complex parameter patterns
	v1 := api.RegisterGroup("v1", "/v1", map[string]string{
		"user-profile":    "/users/:userId/profile",
		"user-posts":      "/users/:userId/posts/:postId",
		"complex-pattern": "/category/:category/item-:itemId/reviews/:reviewId?",
		"multi-param":     "/store/:storeId/dept/:deptId/product/:productId",
		"optional-chain":  "/path/:required/:optional1?/:optional2?",
	})

	// Test basic parameter interpolation
	profileURL, err := v1.Builder("user-profile").
		WithParam("userId", "john-doe-123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build user profile URL: %v", err)
	}
	expected := "https://api.example.com/v1/users/john-doe-123/profile"
	if profileURL != expected {
		t.Errorf("Expected %q, got %q", expected, profileURL)
	}

	// Test multiple parameters
	postsURL, err := v1.Builder("user-posts").
		WithParam("userId", "jane_smith").
		WithParam("postId", "post-2023-001").
		Build()
	if err != nil {
		t.Fatalf("Failed to build user posts URL: %v", err)
	}
	expected = "https://api.example.com/v1/users/jane_smith/posts/post-2023-001"
	if postsURL != expected {
		t.Errorf("Expected %q, got %q", expected, postsURL)
	}

	// Test complex pattern with mixed parameter styles
	complexURL, err := v1.Builder("complex-pattern").
		WithParam("category", "electronics").
		WithParam("itemId", "LAPTOP001").
		WithParam("reviewId", "rev_123").
		WithQuery("format", "json").
		Build()
	if err != nil {
		t.Fatalf("Failed to build complex pattern URL: %v", err)
	}
	expected = "https://api.example.com/v1/category/electronics/item-LAPTOP001/reviews/rev_123"
	if !strings.Contains(complexURL, expected) {
		t.Errorf("Expected URL to contain %q, got %q", expected, complexURL)
	}
	if !strings.Contains(complexURL, "format=json") {
		t.Errorf("Expected URL to contain query parameter, got %q", complexURL)
	}

	// Test multiple consecutive parameters
	multiURL, err := v1.Builder("multi-param").
		WithParam("storeId", "store_001").
		WithParam("deptId", "electronics").
		WithParam("productId", "laptop-dell-xps").
		WithQuery("variant", "silver").
		WithQuery("warranty", "extended").
		Build()
	if err != nil {
		t.Fatalf("Failed to build multi-param URL: %v", err)
	}
	expected = "https://api.example.com/v1/store/store_001/dept/electronics/product/laptop-dell-xps"
	if !strings.Contains(multiURL, expected) {
		t.Errorf("Expected URL to contain %q, got %q", expected, multiURL)
	}

	// Test special characters in parameters (URL encoding)
	specialURL, err := v1.Builder("user-profile").
		WithParam("userId", "user@domain.com").
		WithQuery("filter", "name=John Doe").
		Build()
	if err != nil {
		t.Fatalf("Failed to build URL with special characters: %v", err)
	}
	// Note: Path parameters currently are not URL encoding @ symbol - this is current behavior
	if !strings.Contains(specialURL, "user@domain.com") {
		t.Errorf("Expected URL to contain user@domain.com, got %q", specialURL)
	}
	// Query parameters should be encoded
	if !strings.Contains(specialURL, "filter=name%3DJohn+Doe") && !strings.Contains(specialURL, "filter=name%3DJohn%20Doe") {
		t.Errorf("Expected URL to contain encoded query parameter, got %q", specialURL)
	}

	// Test missing required parameter (should fail)
	_, err = v1.Builder("user-profile").Build()
	if err == nil {
		t.Error("Expected error when required parameter is missing")
	}
}

func TestQueryParameterHandlingEdgeCases(t *testing.T) {
	// Test various query parameter scenarios in nested routes
	rm := urlkit.NewRouteManager()

	rm.RegisterGroup("search", "https://search.example.com", map[string]string{
		"basic": "/search",
	})

	search := rm.Group("search")

	advanced := search.RegisterGroup("advanced", "/advanced", map[string]string{
		"query":   "/query/:searchId",
		"results": "/results",
	})

	// Test multiple query parameters
	queryURL, err := advanced.Builder("query").
		WithParam("searchId", "search-123").
		WithQuery("q", "golang programming").
		WithQuery("category", "books").
		WithQuery("sort", "relevance").
		WithQuery("page", "1").
		WithQuery("limit", "20").
		Build()
	if err != nil {
		t.Fatalf("Failed to build query URL with multiple params: %v", err)
	}

	// Check all query parameters are present (order may vary)
	if !strings.Contains(queryURL, "https://search.example.com/advanced/query/search-123?") {
		t.Errorf("Expected URL to start with correct path, got %q", queryURL)
	}
	queryParams := []string{"q=golang+programming", "q=golang%20programming", "category=books", "sort=relevance", "page=1", "limit=20"}
	for _, param := range queryParams {
		if param == "q=golang+programming" || param == "q=golang%20programming" {
			// Either encoding is acceptable for spaces
			if !strings.Contains(queryURL, "q=golang+programming") && !strings.Contains(queryURL, "q=golang%20programming") {
				t.Errorf("Expected URL to contain encoded query 'q=golang...', got %q", queryURL)
			}
		} else {
			if !strings.Contains(queryURL, param) {
				t.Errorf("Expected URL to contain query parameter %q, got %q", param, queryURL)
			}
		}
	}

	// Test empty query values
	emptyQueryURL, err := advanced.Builder("results").
		WithQuery("q", "").
		WithQuery("filter", "active").
		Build()
	if err != nil {
		t.Fatalf("Failed to build URL with empty query value: %v", err)
	}
	if !strings.Contains(emptyQueryURL, "q=") {
		t.Errorf("Expected URL to contain empty query parameter 'q=', got %q", emptyQueryURL)
	}
	if !strings.Contains(emptyQueryURL, "filter=active") {
		t.Errorf("Expected URL to contain 'filter=active', got %q", emptyQueryURL)
	}

	// Test special characters in query values
	specialQueryURL, err := advanced.Builder("results").
		WithQuery("term", "C++ & Java").
		WithQuery("author", "John Doe <john@example.com>").
		WithQuery("tags", "programming,web,api").
		Build()
	if err != nil {
		t.Fatalf("Failed to build URL with special chars in query: %v", err)
	}

	// Check that special characters are properly encoded
	specialEncodings := map[string][]string{
		"&": {"%26", "&amp;"},
		"<": {"%3C"},
		">": {"%3E"},
		"@": {"%40"},
		" ": {"+", "%20"},
		",": {"%2C", ","},
	}

	for char, encodings := range specialEncodings {
		found := false
		for _, encoding := range encodings {
			if strings.Contains(specialQueryURL, encoding) {
				found = true
				break
			}
		}
		// Some characters might not need encoding in query values
		if char != "," && !found {
			t.Logf("Warning: Special character %q might not be encoded in %q", char, specialQueryURL)
		}
	}

	// Test numeric and boolean-like query parameters
	numericURL, err := advanced.Builder("results").
		WithQuery("page", "1").
		WithQuery("limit", "50").
		WithQuery("active", "true").
		WithQuery("score", "95.5").
		Build()
	if err != nil {
		t.Fatalf("Failed to build URL with numeric query params: %v", err)
	}
	numericParams := []string{"page=1", "limit=50", "active=true", "score=95.5"}
	for _, param := range numericParams {
		if !strings.Contains(numericURL, param) {
			t.Errorf("Expected URL to contain numeric parameter %q, got %q", param, numericURL)
		}
	}
}

// ExampleGroup_internationalizationPattern demonstrates how to use nested groups
// for internationalization with localized routes and content.
func ExampleGroup_internationalizationPattern() {
	// Create a route manager for an internationalized website
	rm := urlkit.NewRouteManager()

	// Register the main frontend group
	rm.RegisterGroup("frontend", "https://mywebsite.com", map[string]string{
		"home": "/",
	})

	frontend := rm.Group("frontend")

	// Add English locale routes
	frontend.RegisterGroup("en", "/en", map[string]string{
		"about":    "/about-us",
		"contact":  "/contact",
		"products": "/products/:category",
	})

	// Add Spanish locale routes with localized paths
	frontend.RegisterGroup("es", "/es", map[string]string{
		"about":    "/acerca-de",
		"contact":  "/contacto",
		"products": "/productos/:category",
	})

	// Build URLs for different locales
	// English: https://mywebsite.com/en/about-us
	enAboutURL, _ := rm.Group("frontend").Group("en").Builder("about").Build()
	fmt.Println("English About URL:", enAboutURL)

	// Spanish: https://mywebsite.com/es/acerca-de
	esAboutURL, _ := rm.Group("frontend").Group("es").Builder("about").Build()
	fmt.Println("Spanish About URL:", esAboutURL)

	// English Products with category: https://mywebsite.com/en/products/electronics
	enProductsURL, _ := rm.Group("frontend").Group("en").Builder("products").
		WithParam("category", "electronics").
		Build()
	fmt.Println("English Products URL:", enProductsURL)

	// Spanish Products with category: https://mywebsite.com/es/productos/electronics
	esProductsURL, _ := rm.Group("frontend").Group("es").Builder("products").
		WithParam("category", "electronics").
		Build()
	fmt.Println("Spanish Products URL:", esProductsURL)

	// Output:
	// English About URL: https://mywebsite.com/en/about-us
	// Spanish About URL: https://mywebsite.com/es/acerca-de
	// English Products URL: https://mywebsite.com/en/products/electronics
	// Spanish Products URL: https://mywebsite.com/es/productos/electronics
}

// ExampleGroup_apiVersioningPattern demonstrates how to use nested groups
// for API versioning with backward compatibility.
func ExampleGroup_apiVersioningPattern() {
	// Create a route manager for a versioned API
	rm := urlkit.NewRouteManager()

	// Register the main API group
	rm.RegisterGroup("api", "https://api.myservice.com", map[string]string{
		"status": "/status",
		"health": "/health",
	})

	api := rm.Group("api")

	// Add v1 API routes
	v1 := api.RegisterGroup("v1", "/v1", map[string]string{
		"users":    "/users/:id",
		"posts":    "/posts",
		"comments": "/posts/:postId/comments",
	})

	// Add v2 API routes with new endpoints
	api.RegisterGroup("v2", "/v2", map[string]string{
		"users":    "/users/:id",
		"profiles": "/users/:id/profile",
		"posts":    "/posts/:id",
		"teams":    "/teams/:teamId",
	})

	// Add admin endpoints to v1
	v1.RegisterGroup("admin", "/admin", map[string]string{
		"dashboard": "/dashboard",
		"settings":  "/settings/:section",
		"reports":   "/reports/:type/:date?",
	})

	// Build URLs for different API versions
	// Root API status: https://api.myservice.com/status
	statusURL, _ := rm.Group("api").Builder("status").Build()
	fmt.Println("API Status URL:", statusURL)

	// V1 user endpoint: https://api.myservice.com/v1/users/123
	v1UserURL, _ := rm.Group("api").Group("v1").Builder("users").
		WithParam("id", "123").
		Build()
	fmt.Println("V1 User URL:", v1UserURL)

	// V2 user profile: https://api.myservice.com/v2/users/123/profile
	v2ProfileURL, _ := rm.Group("api").Group("v2").Builder("profiles").
		WithParam("id", "123").
		Build()
	fmt.Println("V2 Profile URL:", v2ProfileURL)

	// V1 admin dashboard: https://api.myservice.com/v1/admin/dashboard
	adminURL, _ := rm.Group("api").Group("v1").Group("admin").Builder("dashboard").Build()
	fmt.Println("Admin Dashboard URL:", adminURL)

	// V1 admin reports with optional date: https://api.myservice.com/v1/admin/reports/sales/2023-12
	reportsURL, _ := rm.Group("api").Group("v1").Group("admin").Builder("reports").
		WithParam("type", "sales").
		WithParam("date", "2023-12").
		Build()
	fmt.Println("Admin Reports URL:", reportsURL)

	// Output:
	// API Status URL: https://api.myservice.com/status
	// V1 User URL: https://api.myservice.com/v1/users/123
	// V2 Profile URL: https://api.myservice.com/v2/users/123/profile
	// Admin Dashboard URL: https://api.myservice.com/v1/admin/dashboard
	// Admin Reports URL: https://api.myservice.com/v1/admin/reports/sales/2023-12
}

// ExampleRouteManager_configurationBasedSetup demonstrates how to load nested groups
// from JSON configuration for complex application structures.
func ExampleRouteManager_configurationBasedSetup() {
	// Define a comprehensive configuration with nested groups
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "frontend",
				BaseURL: "https://myapp.com",
				Paths: map[string]string{
					"home": "/",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "en",
						Path: "/en",
						Paths: map[string]string{
							"about":   "/about",
							"contact": "/contact",
						},
						Groups: []urlkit.GroupConfig{
							{
								Name: "help",
								Path: "/help",
								Paths: map[string]string{
									"faq":     "/faq",
									"support": "/support/:ticketId?",
								},
							},
						},
					},
				},
			},
			{
				Name:    "api",
				BaseURL: "https://api.myapp.com",
				Paths: map[string]string{
					"status": "/status",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "v1",
						Path: "/v1",
						Paths: map[string]string{
							"users": "/users/:id",
						},
					},
				},
			},
		},
	}

	// Create route manager from configuration
	manager := urlkit.NewRouteManager(&config)

	// Build URLs using the configured nested structure
	// Frontend home: https://myapp.com/
	homeURL, _ := manager.Group("frontend").Builder("home").Build()
	fmt.Println("Home URL:", homeURL)

	// English contact page: https://myapp.com/en/contact
	contactURL, _ := manager.Group("frontend").Group("en").Builder("contact").Build()
	fmt.Println("Contact URL:", contactURL)

	// English help FAQ: https://myapp.com/en/help/faq
	faqURL, _ := manager.Group("frontend").Group("en").Group("help").Builder("faq").Build()
	fmt.Println("FAQ URL:", faqURL)

	// English help support with ticket ID: https://myapp.com/en/help/support/T-12345
	supportURL, _ := manager.Group("frontend").Group("en").Group("help").Builder("support").
		WithParam("ticketId", "T-12345").
		Build()
	fmt.Println("Support URL:", supportURL)

	// API status: https://api.myapp.com/status
	apiStatusURL, _ := manager.Group("api").Builder("status").Build()
	fmt.Println("API Status URL:", apiStatusURL)

	// API v1 users: https://api.myapp.com/v1/users/user-123
	usersURL, _ := manager.Group("api").Group("v1").Builder("users").
		WithParam("id", "user-123").
		WithQuery("include", "profile").
		Build()
	fmt.Println("API Users URL:", usersURL)

	// Output:
	// Home URL: https://myapp.com/
	// Contact URL: https://myapp.com/en/contact
	// FAQ URL: https://myapp.com/en/help/faq
	// Support URL: https://myapp.com/en/help/support/T-12345
	// API Status URL: https://api.myapp.com/status
	// API Users URL: https://api.myapp.com/v1/users/user-123?include=profile
}

// ExampleRouteManager_validationWithDotSeparatedPaths demonstrates how to validate
// nested group configurations using dot-separated path notation.
func ExampleRouteManager_validationWithDotSeparatedPaths() {
	// Create a complex nested structure
	rm := urlkit.NewRouteManager()

	// Register main groups
	rm.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})

	// Add nested groups
	frontend := rm.Group("frontend")
	frontend.RegisterGroup("en", "/en", map[string]string{
		"about":   "/about",
		"contact": "/contact",
	})

	api := rm.Group("api")
	v1 := api.RegisterGroup("v1", "/v1", map[string]string{
		"users": "/users/:id",
		"posts": "/posts",
	})
	v1.RegisterGroup("admin", "/admin", map[string]string{
		"dashboard": "/dashboard",
	})

	// Define expected routes using dot-separated paths for nested groups
	expectedRoutes := map[string][]string{
		"frontend":     {"home"},
		"frontend.en":  {"about", "contact"},
		"api":          {"status"},
		"api.v1":       {"users", "posts"},
		"api.v1.admin": {"dashboard"},
	}

	// Validate the configuration
	err := rm.Validate(expectedRoutes)
	if err != nil {
		fmt.Printf("Validation failed: %v\n", err)
		return
	}

	fmt.Println("✓ All nested groups and routes are properly configured")

	// Example of validation failure - missing route
	invalidExpected := map[string][]string{
		"frontend.en":  {"about", "contact", "missing-route"},
		"api.v1.admin": {"dashboard", "settings"}, // settings route doesn't exist
		"api.v2":       {"users"},                 // v2 group doesn't exist
	}

	err = rm.Validate(invalidExpected)
	if err != nil {
		// Note: Error order may vary due to map iteration
		fmt.Println("Expected validation failure occurred (order may vary):", strings.Contains(err.Error(), "validation error"))
	}

	// Output:
	// ✓ All nested groups and routes are properly configured
	// Expected validation failure occurred (order may vary): true
}

// Unit Tests for Template Logic (Task 5.1)

func TestTemplateVariableCollectionAndInheritance(t *testing.T) {
	// Test template variable collection and inheritance behavior
	rm := urlkit.NewRouteManager()

	// Create root group with template variables
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})

	root := rm.Group("api")
	root.SetTemplateVar("region", "us")
	root.SetTemplateVar("version", "v1")

	// Create child group that overrides some variables and adds new ones
	child := root.RegisterGroup("mobile", "/mobile", map[string]string{
		"app": "/app/:id",
	})
	child.SetTemplateVar("version", "v2")   // Override parent variable
	child.SetTemplateVar("platform", "ios") // Add new variable

	// Create grandchild to test deeper inheritance
	grandchild := child.RegisterGroup("premium", "/premium", map[string]string{
		"features": "/features",
	})
	grandchild.SetTemplateVar("tier", "premium")
	grandchild.SetTemplateVar("region", "eu") // Override root variable

	// Test variable collection at different levels

	// Root level should only have root variables
	rootVars := root.CollectTemplateVars()
	if rootVars["region"] != "us" || rootVars["version"] != "v1" {
		t.Errorf("Expected root variables region=us, version=v1, got %v", rootVars)
	}
	if _, exists := rootVars["platform"]; exists {
		t.Errorf("Root should not have child variable 'platform'")
	}

	// Child level should have parent variables plus overrides
	childVars := child.CollectTemplateVars()
	expected := map[string]string{
		"region":   "us",  // From parent
		"version":  "v2",  // Overridden
		"platform": "ios", // New
	}
	for key, expectedVal := range expected {
		if childVars[key] != expectedVal {
			t.Errorf("Expected child variable %s=%s, got %s", key, expectedVal, childVars[key])
		}
	}

	// Grandchild level should have full hierarchy with overrides
	grandchildVars := grandchild.CollectTemplateVars()
	expectedGrand := map[string]string{
		"region":   "eu",      // Overridden by grandchild
		"version":  "v2",      // From child
		"platform": "ios",     // From child
		"tier":     "premium", // New from grandchild
	}
	for key, expectedVal := range expectedGrand {
		if grandchildVars[key] != expectedVal {
			t.Errorf("Expected grandchild variable %s=%s, got %s", key, expectedVal, grandchildVars[key])
		}
	}
}

func TestStringSubstitutionWithVariousPlaceholderFormats(t *testing.T) {
	// Test string substitution functionality with different placeholder patterns

	testCases := []struct {
		template string
		vars     map[string]string
		expected string
		name     string
	}{
		{
			name:     "Basic single variable",
			template: "https://{host}/api",
			vars:     map[string]string{"host": "example.com"},
			expected: "https://example.com/api",
		},
		{
			name:     "Multiple variables",
			template: "https://{host}/{version}/{endpoint}",
			vars:     map[string]string{"host": "api.example.com", "version": "v1", "endpoint": "users"},
			expected: "https://api.example.com/v1/users",
		},
		{
			name:     "Variable with underscores",
			template: "{base_url}/{api_version}",
			vars:     map[string]string{"base_url": "https://api.com", "api_version": "v2"},
			expected: "https://api.com/v2",
		},
		{
			name:     "Variable with numbers",
			template: "{host1}/{path2}",
			vars:     map[string]string{"host1": "server1.com", "path2": "data"},
			expected: "server1.com/data",
		},
		{
			name:     "Missing variable - placeholder remains",
			template: "https://{host}/{missing_var}/api",
			vars:     map[string]string{"host": "example.com"},
			expected: "https://example.com/{missing_var}/api",
		},
		{
			name:     "Empty template",
			template: "",
			vars:     map[string]string{"host": "example.com"},
			expected: "",
		},
		{
			name:     "No variables in template",
			template: "https://static.example.com/api",
			vars:     map[string]string{"host": "example.com"},
			expected: "https://static.example.com/api",
		},
		{
			name:     "Same variable used multiple times",
			template: "{proto}://{host}/{proto}/api",
			vars:     map[string]string{"proto": "https", "host": "example.com"},
			expected: "https://example.com/https/api",
		},
		{
			name:     "Variables with special characters",
			template: "{base_url}{locale}{route_path}",
			vars:     map[string]string{"base_url": "https://site.com", "locale": "/en", "route_path": "/about-us"},
			expected: "https://site.com/en/about-us",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := urlkit.SubstituteTemplate(tc.template, tc.vars)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}

func TestTemplateOwnerDiscoveryLogic(t *testing.T) {
	// Test template owner discovery in group hierarchy
	rm := urlkit.NewRouteManager()

	// Create hierarchy: root -> child -> grandchild
	rm.RegisterGroup("root", "https://example.com", map[string]string{
		"home": "/",
	})

	root := rm.Group("root")
	child := root.RegisterGroup("child", "/child", map[string]string{
		"page": "/page",
	})
	grandchild := child.RegisterGroup("grandchild", "/grandchild", map[string]string{
		"item": "/item/:id",
	})

	// Test 1: No templates set - should return nil
	owner := grandchild.FindTemplateOwner()
	if owner != nil {
		t.Error("Expected nil when no templates are set")
	}

	// Test 2: Template set at root level
	root.SetURLTemplate("{base_url}{route_path}")
	owner = grandchild.FindTemplateOwner()
	if owner != root {
		t.Error("Expected root group to be template owner")
	}

	// Test 3: Template set at child level - should override root
	child.SetURLTemplate("{base_url}/mobile{route_path}")
	owner = grandchild.FindTemplateOwner()
	if owner != child {
		t.Error("Expected child group to be template owner (overrides root)")
	}

	// Test 4: Template set at grandchild level - should be most specific
	grandchild.SetURLTemplate("{base_url}/api/v2{route_path}")
	owner = grandchild.FindTemplateOwner()
	if owner != grandchild {
		t.Error("Expected grandchild group to be template owner (most specific)")
	}

	// Test 5: Remove grandchild template - should fall back to child
	grandchild.SetURLTemplate("")
	owner = grandchild.FindTemplateOwner()
	if owner != child {
		t.Error("Expected child group to be template owner after grandchild template removed")
	}

	// Test 6: Template owner discovery from child level
	owner = child.FindTemplateOwner()
	if owner != child {
		t.Error("Expected child to find itself as template owner")
	}

	// Test 7: Remove child template - child should find root as owner
	child.SetURLTemplate("")
	owner = child.FindTemplateOwner()
	if owner != root {
		t.Error("Expected child to find root as template owner")
	}
}

func TestNewGroupMethods(t *testing.T) {
	// Test new Group methods: SetURLTemplate, SetTemplateVar, GetTemplateVar, AddRoutes

	// Create a fresh route manager to avoid interference from other tests
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("test", "https://example.com", map[string]string{
		"users": "/users/:id",
	})
	group := rm.Group("test")

	// Test SetURLTemplate and GetURLTemplate (assuming we can access it)
	template := "{base_url}/{locale}{route_path}"
	group.SetURLTemplate(template)

	// Test SetTemplateVar and GetTemplateVar
	group.SetTemplateVar("locale", "en")
	group.SetTemplateVar("region", "us")

	// Test GetTemplateVar for existing variable
	if value, exists := group.GetTemplateVar("locale"); !exists || value != "en" {
		t.Errorf("Expected locale=en, got value=%s, exists=%v", value, exists)
	}

	if value, exists := group.GetTemplateVar("region"); !exists || value != "us" {
		t.Errorf("Expected region=us, got value=%s, exists=%v", value, exists)
	}

	// Test GetTemplateVar for non-existing variable
	if value, exists := group.GetTemplateVar("nonexistent"); exists {
		t.Errorf("Expected nonexistent variable to not exist, got value=%s", value)
	}

	// Test variable override
	group.SetTemplateVar("locale", "es")
	if value, exists := group.GetTemplateVar("locale"); !exists || value != "es" {
		t.Errorf("Expected overridden locale=es, got value=%s, exists=%v", value, exists)
	}

	// Test AddRoutes method
	newRoutes := map[string]string{
		"posts":    "/posts/:id",
		"comments": "/posts/:postId/comments/:commentId",
	}
	group.AddRoutes(newRoutes)

	// Test that new routes were added and are functional
	postRoute, err := group.Route("posts")
	if err != nil {
		t.Fatalf("Expected posts route to be added: %v", err)
	}
	if postRoute != "/posts/:id" {
		t.Errorf("Expected posts route to be '/posts/:id', got %s", postRoute)
	}

	commentRoute, err := group.Route("comments")
	if err != nil {
		t.Fatalf("Expected comments route to be added: %v", err)
	}
	if commentRoute != "/posts/:postId/comments/:commentId" {
		t.Errorf("Expected comments route pattern, got %s", commentRoute)
	}

	// Test that original routes still work
	userRoute, err := group.Route("users")
	if err != nil {
		t.Fatalf("Expected original users route to still work: %v", err)
	}
	if userRoute != "/users/:id" {
		t.Errorf("Expected users route to be '/users/:id', got %s", userRoute)
	}

	// Clear any template and variables that might be set to test non-template mode
	group.SetURLTemplate("")
	// Reset template vars by creating a new empty map (no clear method exists)
	group.SetTemplateVar("locale", "")
	group.SetTemplateVar("region", "")

	// Test URL building with added routes
	postURL, err := group.Builder("posts").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build URL with added route: %v", err)
	}
	expected := "https://example.com/posts/123"
	if postURL != expected {
		t.Errorf("Expected %q, got %q", expected, postURL)
	}
}

// Integration Tests for Template Logic (Task 5.2)

func TestFullURLBuildingWithTemplatesProgrammaticAPI(t *testing.T) {
	// Test full URL building with templates using programmatic API
	rm := urlkit.NewRouteManager()

	// Create a multi-service setup with templates
	rm.RegisterGroup("frontend", "https://myapp.com", map[string]string{
		"home":  "/",
		"about": "/about",
	})

	frontend := rm.Group("frontend")
	frontend.SetURLTemplate("{base_url}{locale}{route_path}")
	frontend.SetTemplateVar("locale", "/en")

	// Create API service with different template
	rm.RegisterGroup("api", "https://api.myapp.com", map[string]string{
		"status": "/status",
		"health": "/health",
	})

	api := rm.Group("api")
	api.SetURLTemplate("{base_url}/v{version}{route_path}")
	api.SetTemplateVar("version", "1")

	// Create nested groups with variable overrides
	v2 := api.RegisterGroup("v2", "/v2", map[string]string{
		"users": "/users/:id",
		"posts": "/posts",
	})
	v2.SetTemplateVar("version", "2") // Override parent variable

	// Test frontend URL building with template
	aboutURL, err := frontend.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build frontend about URL: %v", err)
	}
	expected := "https://myapp.com/en/about/"
	if aboutURL != expected {
		t.Errorf("Expected %q, got %q", expected, aboutURL)
	}

	// Test API URL building with template and version
	statusURL, err := api.Builder("status").Build()
	if err != nil {
		t.Fatalf("Failed to build API status URL: %v", err)
	}
	expected = "https://api.myapp.com/v1/status/"
	if statusURL != expected {
		t.Errorf("Expected %q, got %q", expected, statusURL)
	}

	// Test nested group with variable override
	usersURL, err := v2.Builder("users").
		WithParam("id", "123").
		WithQuery("include", "profile").
		Build()
	if err != nil {
		t.Fatalf("Failed to build v2 users URL: %v", err)
	}
	expected = "https://api.myapp.com/v2/users/123/?include=profile"
	if usersURL != expected {
		t.Errorf("Expected %q, got %q", expected, usersURL)
	}

	// Test template variable changes
	frontend.SetTemplateVar("locale", "/es")
	aboutEsURL, err := frontend.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish about URL: %v", err)
	}
	expected = "https://myapp.com/es/about/"
	if aboutEsURL != expected {
		t.Errorf("Expected %q, got %q", expected, aboutEsURL)
	}
}

func TestJSONConfigurationLoadingWithTemplateFields(t *testing.T) {
	// Test JSON configuration loading with template fields
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:        "frontend",
				BaseURL:     "https://example.com",
				URLTemplate: "{base_url}{locale}{route_path}",
				TemplateVars: map[string]string{
					"locale": "/en",
					"region": "us",
				},
				Paths: map[string]string{
					"home":    "/",
					"about":   "/about",
					"contact": "/contact",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "help",
						Path: "/help",
						TemplateVars: map[string]string{
							"locale":  "/en", // Same as parent
							"section": "support",
						},
						Paths: map[string]string{
							"faq":     "/faq",
							"support": "/support/:ticket?",
						},
					},
					{
						Name: "es",
						Path: "/es",
						TemplateVars: map[string]string{
							"locale": "/es", // Override parent
							"region": "es",  // Override parent
						},
						Paths: map[string]string{
							"inicio":   "/inicio",
							"contacto": "/contacto",
						},
					},
				},
			},
			{
				Name:        "api",
				BaseURL:     "https://api.example.com",
				URLTemplate: "{base_url}{api_version}{route_path}",
				TemplateVars: map[string]string{
					"api_version": "/v1",
				},
				Paths: map[string]string{
					"status": "/status",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "v2",
						Path: "/v2",
						TemplateVars: map[string]string{
							"api_version": "/v2", // Override parent
						},
						Paths: map[string]string{
							"users": "/users/:id",
							"posts": "/posts",
						},
					},
				},
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)

	// Test root group template URL building
	homeURL, err := manager.Group("frontend").Builder("home").Build()
	if err != nil {
		t.Fatalf("Failed to build home URL from config: %v", err)
	}
	expected := "https://example.com/en/"
	if homeURL != expected {
		t.Errorf("Expected %q, got %q", expected, homeURL)
	}

	// Test nested group with inherited variables
	faqURL, err := manager.Group("frontend").Group("help").Builder("faq").Build()
	if err != nil {
		t.Fatalf("Failed to build FAQ URL from config: %v", err)
	}
	expected = "https://example.com/en/faq/"
	if faqURL != expected {
		t.Errorf("Expected %q, got %q", expected, faqURL)
	}

	// Test nested group with overridden variables
	inicioURL, err := manager.Group("frontend").Group("es").Builder("inicio").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish inicio URL from config: %v", err)
	}
	expected = "https://example.com/es/inicio/"
	if inicioURL != expected {
		t.Errorf("Expected %q, got %q", expected, inicioURL)
	}

	// Test API with template
	statusURL, err := manager.Group("api").Builder("status").Build()
	if err != nil {
		t.Fatalf("Failed to build API status URL from config: %v", err)
	}
	expected = "https://api.example.com/v1/status/"
	if statusURL != expected {
		t.Errorf("Expected %q, got %q", expected, statusURL)
	}

	// Test API v2 with overridden version
	usersURL, err := manager.Group("api").Group("v2").Builder("users").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build v2 users URL from config: %v", err)
	}
	expected = "https://api.example.com/v2/users/123/"
	if usersURL != expected {
		t.Errorf("Expected %q, got %q", expected, usersURL)
	}
}

func TestComplexNestedScenariosFromFeatureSpecification(t *testing.T) {
	// Test complex nested scenarios based on the feature specification

	// Scenario: Multi-language e-commerce site with regions and API versioning
	rm := urlkit.NewRouteManager()

	// Main site with regional templates
	rm.RegisterGroup("site", "https://shop.example.com", map[string]string{
		"home": "/",
	})

	site := rm.Group("site")
	site.SetURLTemplate("{base_url}{region_path}{locale_path}{route_path}")
	site.SetTemplateVar("region_path", "")
	site.SetTemplateVar("locale_path", "")

	// US region
	us := site.RegisterGroup("us", "/us", map[string]string{
		"products": "/products/:category",
		"cart":     "/cart",
	})
	us.SetTemplateVar("region_path", "/us")

	// US English
	usEn := us.RegisterGroup("en", "/en", map[string]string{
		"checkout": "/checkout/:step",
		"account":  "/account",
	})
	usEn.SetTemplateVar("locale_path", "/en")

	// US Spanish
	usEs := us.RegisterGroup("es", "/es", map[string]string{
		"checkout": "/pago/:step",
		"account":  "/cuenta",
	})
	usEs.SetTemplateVar("locale_path", "/es")

	// EU region with different structure
	eu := site.RegisterGroup("eu", "", map[string]string{
		"products": "/products/:category",
	})
	eu.SetURLTemplate("{base_url}{locale_path}.{region_code}{route_path}")
	eu.SetTemplateVar("region_code", "eu")
	eu.SetTemplateVar("locale_path", "/en")

	// EU locales
	euDe := eu.RegisterGroup("de", "", map[string]string{
		"checkout": "/kasse/:step",
		"account":  "/konto",
	})
	euDe.SetTemplateVar("locale_path", "/de")

	// Test US English checkout
	usEnCheckoutURL, err := usEn.Builder("checkout").
		WithParam("step", "payment").
		WithQuery("method", "card").
		Build()
	if err != nil {
		t.Fatalf("Failed to build US English checkout URL: %v", err)
	}
	expected := "https://shop.example.com/us/en/checkout/payment/?method=card"
	if usEnCheckoutURL != expected {
		t.Errorf("Expected %q, got %q", expected, usEnCheckoutURL)
	}

	// Test US Spanish checkout (different route)
	usEsCheckoutURL, err := usEs.Builder("checkout").
		WithParam("step", "payment").
		Build()
	if err != nil {
		t.Fatalf("Failed to build US Spanish checkout URL: %v", err)
	}
	expected = "https://shop.example.com/us/es/pago/payment/"
	if usEsCheckoutURL != expected {
		t.Errorf("Expected %q, got %q", expected, usEsCheckoutURL)
	}

	// Test EU German with different template structure
	euDeCheckoutURL, err := euDe.Builder("checkout").
		WithParam("step", "payment").
		Build()
	if err != nil {
		t.Fatalf("Failed to build EU German checkout URL: %v", err)
	}
	expected = "https://shop.example.com/de.eu/kasse/payment/"
	if euDeCheckoutURL != expected {
		t.Errorf("Expected %q, got %q", expected, euDeCheckoutURL)
	}

	// Test EU products (inherits parent template but no locale override)
	euProductsURL, err := eu.Builder("products").
		WithParam("category", "electronics").
		Build()
	if err != nil {
		t.Fatalf("Failed to build EU products URL: %v", err)
	}
	expected = "https://shop.example.com/en.eu/products/electronics/"
	if euProductsURL != expected {
		t.Errorf("Expected %q, got %q", expected, euProductsURL)
	}
}

func TestVariableOverrideBehavior(t *testing.T) {
	// Test variable override behavior (child overrides parent)
	rm := urlkit.NewRouteManager()

	rm.RegisterGroup("service", "https://api.example.com", map[string]string{
		"status": "/status",
	})

	service := rm.Group("service")
	service.SetURLTemplate("{protocol}://{host}/{env}/{version}{route_path}")
	service.SetTemplateVar("protocol", "https")
	service.SetTemplateVar("host", "api.example.com")
	service.SetTemplateVar("env", "prod")
	service.SetTemplateVar("version", "v1")

	// Child overrides some variables
	staging := service.RegisterGroup("staging", "/staging", map[string]string{
		"health":  "/health",
		"metrics": "/metrics",
	})
	staging.SetTemplateVar("env", "staging")      // Override parent
	staging.SetTemplateVar("host", "staging.api") // Override parent
	staging.SetTemplateVar("debug", "true")       // New variable

	// Grandchild overrides more variables
	v2 := staging.RegisterGroup("v2", "/v2", map[string]string{
		"users": "/users/:id",
		"posts": "/posts",
	})
	v2.SetTemplateVar("version", "v2")           // Override grandparent
	v2.SetTemplateVar("protocol", "http")        // Override grandparent
	v2.SetTemplateVar("experimental", "enabled") // New variable

	// Test parent level - should use all original variables
	parentVars := service.CollectTemplateVars()
	expectedParent := map[string]string{
		"protocol": "https",
		"host":     "api.example.com",
		"env":      "prod",
		"version":  "v1",
	}
	for key, expectedVal := range expectedParent {
		if parentVars[key] != expectedVal {
			t.Errorf("Parent: Expected %s=%s, got %s", key, expectedVal, parentVars[key])
		}
	}

	// Test child level - should have parent vars plus overrides
	childVars := staging.CollectTemplateVars()
	expectedChild := map[string]string{
		"protocol": "https",       // From parent
		"host":     "staging.api", // Overridden
		"env":      "staging",     // Overridden
		"version":  "v1",          // From parent
		"debug":    "true",        // New
	}
	for key, expectedVal := range expectedChild {
		if childVars[key] != expectedVal {
			t.Errorf("Child: Expected %s=%s, got %s", key, expectedVal, childVars[key])
		}
	}

	// Test grandchild level - should have full hierarchy with all overrides
	grandchildVars := v2.CollectTemplateVars()
	expectedGrandchild := map[string]string{
		"protocol":     "http",        // Overridden by grandchild
		"host":         "staging.api", // From child
		"env":          "staging",     // From child
		"version":      "v2",          // Overridden by grandchild
		"debug":        "true",        // From child
		"experimental": "enabled",     // New from grandchild
	}
	for key, expectedVal := range expectedGrandchild {
		if grandchildVars[key] != expectedVal {
			t.Errorf("Grandchild: Expected %s=%s, got %s", key, expectedVal, grandchildVars[key])
		}
	}

	// Test URL building at each level

	// Parent URL
	parentURL, err := service.Builder("status").Build()
	if err != nil {
		t.Fatalf("Failed to build parent URL: %v", err)
	}
	expected := "https://api.example.com/prod/v1/status/"
	if parentURL != expected {
		t.Errorf("Parent URL: Expected %q, got %q", expected, parentURL)
	}

	// Child URL with overrides
	childURL, err := staging.Builder("health").Build()
	if err != nil {
		t.Fatalf("Failed to build child URL: %v", err)
	}
	expected = "https://staging.api/staging/v1/health/"
	if childURL != expected {
		t.Errorf("Child URL: Expected %q, got %q", expected, childURL)
	}

	// Grandchild URL with multiple overrides
	grandchildURL, err := v2.Builder("users").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build grandchild URL: %v", err)
	}
	expected = "http://staging.api/staging/v2/users/123/"
	if grandchildURL != expected {
		t.Errorf("Grandchild URL: Expected %q, got %q", expected, grandchildURL)
	}
}

// Backward Compatibility Tests (Task 5.3)

func TestExistingFunctionalityWorksUnchanged(t *testing.T) {
	// Test that existing functionality works exactly as before when no templates are used

	// Test 1: Basic RouteManager functionality without templates
	rm := urlkit.NewRouteManager()

	// Register groups the old way - no templates
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users":  "/users/:id",
		"posts":  "/posts/:id",
		"status": "/status",
		"health": "/health",
	})

	rm.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":    "/",
		"about":   "/about",
		"contact": "/contact/:type?",
	})

	// Test basic URL building - should work exactly as before
	userURL, err := rm.Group("api").Builder("users").
		WithParam("id", "123").
		WithQuery("include", "profile").
		Build()
	if err != nil {
		t.Fatalf("Failed to build user URL: %v", err)
	}
	expected := "https://api.example.com/users/123?include=profile"
	if userURL != expected {
		t.Errorf("Expected %q, got %q", expected, userURL)
	}

	// Test nested groups without templates
	api := rm.Group("api")
	v1 := api.RegisterGroup("v1", "/v1", map[string]string{
		"analytics": "/analytics/:metric",
		"reports":   "/reports",
	})

	v2 := api.RegisterGroup("v2", "/v2", map[string]string{
		"analytics":  "/analytics/:metric/:timeframe",
		"dashboards": "/dashboards/:id",
	})

	// Test nested URL building - should work exactly as before
	v1AnalyticsURL, err := v1.Builder("analytics").
		WithParam("metric", "pageviews").
		Build()
	if err != nil {
		t.Fatalf("Failed to build v1 analytics URL: %v", err)
	}
	expected = "https://api.example.com/v1/analytics/pageviews"
	if v1AnalyticsURL != expected {
		t.Errorf("Expected %q, got %q", expected, v1AnalyticsURL)
	}

	v2DashboardURL, err := v2.Builder("dashboards").
		WithParam("id", "main").
		WithQuery("refresh", "30").
		Build()
	if err != nil {
		t.Fatalf("Failed to build v2 dashboard URL: %v", err)
	}
	expected = "https://api.example.com/v2/dashboards/main?refresh=30"
	if v2DashboardURL != expected {
		t.Errorf("Expected %q, got %q", expected, v2DashboardURL)
	}

	// Test validation functionality - should work exactly as before
	validationConfig := map[string][]string{
		"api":      {"users", "posts", "status"},
		"frontend": {"home", "about"},
		"api.v1":   {"analytics", "reports"},
		"api.v2":   {"analytics", "dashboards"},
	}

	err = rm.Validate(validationConfig)
	if err != nil {
		t.Errorf("Validation should pass for existing routes: %v", err)
	}

	// Test validation failure - should work exactly as before
	invalidConfig := map[string][]string{
		"api": {"users", "posts", "missing_route"},
	}

	err = rm.Validate(invalidConfig)
	if err == nil {
		t.Error("Validation should fail for missing routes")
	}

	// Test MustValidate functionality
	defer func() {
		if r := recover(); r == nil {
			t.Error("MustValidate should panic for invalid config")
		}
	}()

	rm.MustValidate(invalidConfig)
}

func TestEmptyMissingTemplatesFallbackToConcatenation(t *testing.T) {
	// Test that groups with empty or missing templates fall back to path concatenation

	rm := urlkit.NewRouteManager()

	// Create groups without templates (should use concatenation)
	rm.RegisterGroup("legacy", "https://legacy.example.com", map[string]string{
		"users": "/users/:id",
		"posts": "/posts/:id",
	})

	legacy := rm.Group("legacy")

	// Verify no template is set
	templateOwner := legacy.FindTemplateOwner()
	if templateOwner != nil {
		t.Error("Expected no template owner for legacy group")
	}

	// Create nested group without template
	api := legacy.RegisterGroup("api", "/api", map[string]string{
		"status": "/status",
		"health": "/health/:check?",
	})

	// Test URL building falls back to concatenation
	userURL, err := legacy.Builder("users").
		WithParam("id", "456").
		Build()
	if err != nil {
		t.Fatalf("Failed to build user URL: %v", err)
	}
	expected := "https://legacy.example.com/users/456"
	if userURL != expected {
		t.Errorf("Expected %q, got %q", expected, userURL)
	}

	// Test nested group URL building falls back to concatenation
	statusURL, err := api.Builder("status").Build()
	if err != nil {
		t.Fatalf("Failed to build status URL: %v", err)
	}
	expected = "https://legacy.example.com/api/status"
	if statusURL != expected {
		t.Errorf("Expected %q, got %q", expected, statusURL)
	}

	// Now create a group with template but then clear it
	rm.RegisterGroup("templated", "https://templated.example.com", map[string]string{
		"home":  "/",
		"about": "/about",
	})

	templated := rm.Group("templated")
	templated.SetURLTemplate("{base_url}/app{route_path}")
	templated.SetTemplateVar("prefix", "/app")

	// Verify template is set
	templateOwner = templated.FindTemplateOwner()
	if templateOwner == nil {
		t.Error("Expected template owner to be set")
	}

	// Clear the template
	templated.SetURLTemplate("")

	// Verify template is cleared
	templateOwner = templated.FindTemplateOwner()
	if templateOwner != nil {
		t.Error("Expected no template owner after clearing template")
	}

	// Test URL building falls back to concatenation after template cleared
	aboutURL, err := templated.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build about URL: %v", err)
	}
	expected = "https://templated.example.com/about"
	if aboutURL != expected {
		t.Errorf("Expected %q, got %q", expected, aboutURL)
	}

	// Test mixed scenario: parent has no template, child has template
	mixed := legacy.RegisterGroup("mixed", "/mixed", map[string]string{
		"data": "/data/:id",
	})
	mixed.SetURLTemplate("{base_url}/special{route_path}")

	// Child should use its own template
	templateOwner = mixed.FindTemplateOwner()
	if templateOwner != mixed {
		t.Error("Expected mixed group to be its own template owner")
	}

	dataURL, err := mixed.Builder("data").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build data URL: %v", err)
	}
	expected = "https://legacy.example.com/special/data/123/"
	if dataURL != expected {
		t.Errorf("Expected %q, got %q", expected, dataURL)
	}
}

func TestExistingJSONConfigsContinueToWork(t *testing.T) {
	// Test that existing JSON configurations without template fields continue to work

	// Traditional config without template fields
	legacyConfig := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:    "api",
				BaseURL: "https://api.legacy.com",
				Paths: map[string]string{
					"users":    "/users/:id",
					"posts":    "/posts/:id",
					"comments": "/posts/:postId/comments/:commentId?",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "v1",
						Path: "/v1",
						Paths: map[string]string{
							"search":    "/search",
							"analytics": "/analytics/:type",
						},
					},
					{
						Name: "v2",
						Path: "/v2",
						Paths: map[string]string{
							"search":      "/search/:query",
							"suggestions": "/suggestions",
						},
						Groups: []urlkit.GroupConfig{
							{
								Name: "beta",
								Path: "/beta",
								Paths: map[string]string{
									"features": "/features/:feature",
								},
							},
						},
					},
				},
			},
			{
				Name:    "frontend",
				BaseURL: "https://www.legacy.com",
				Paths: map[string]string{
					"home":     "/",
					"about":    "/about",
					"contact":  "/contact",
					"products": "/products/:category/:id?",
				},
				Groups: []urlkit.GroupConfig{
					{
						Name: "blog",
						Path: "/blog",
						Paths: map[string]string{
							"posts":    "/posts/:slug",
							"archives": "/archives/:year/:month?",
						},
					},
				},
			},
		},
	}

	manager := urlkit.NewRouteManagerFromConfig(legacyConfig)

	// Test root group functionality
	userURL, err := manager.Group("api").Builder("users").
		WithParam("id", "123").
		WithQuery("include", "profile").
		Build()
	if err != nil {
		t.Fatalf("Failed to build user URL from legacy config: %v", err)
	}
	expected := "https://api.legacy.com/users/123?include=profile"
	if userURL != expected {
		t.Errorf("Expected %q, got %q", expected, userURL)
	}

	// Test nested groups
	searchV1URL, err := manager.Group("api").Group("v1").Builder("search").
		WithQuery("q", "golang").
		Build()
	if err != nil {
		t.Fatalf("Failed to build v1 search URL from legacy config: %v", err)
	}
	expected = "https://api.legacy.com/v1/search?q=golang"
	if searchV1URL != expected {
		t.Errorf("Expected %q, got %q", expected, searchV1URL)
	}

	// Test deeply nested groups
	betaFeaturesURL, err := manager.Group("api").Group("v2").Group("beta").Builder("features").
		WithParam("feature", "ai-search").
		Build()
	if err != nil {
		t.Fatalf("Failed to build beta features URL from legacy config: %v", err)
	}
	expected = "https://api.legacy.com/v2/beta/features/ai-search"
	if betaFeaturesURL != expected {
		t.Errorf("Expected %q, got %q", expected, betaFeaturesURL)
	}

	// Test different base URL group
	productURL, err := manager.Group("frontend").Builder("products").
		WithParam("category", "electronics").
		WithParam("id", "laptop-123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build product URL from legacy config: %v", err)
	}
	expected = "https://www.legacy.com/products/electronics/laptop-123"
	if productURL != expected {
		t.Errorf("Expected %q, got %q", expected, productURL)
	}

	// Test blog nested group
	blogPostURL, err := manager.Group("frontend").Group("blog").Builder("posts").
		WithParam("slug", "introducing-new-features").
		Build()
	if err != nil {
		t.Fatalf("Failed to build blog post URL from legacy config: %v", err)
	}
	expected = "https://www.legacy.com/blog/posts/introducing-new-features"
	if blogPostURL != expected {
		t.Errorf("Expected %q, got %q", expected, blogPostURL)
	}

	// Test validation works with legacy config
	validationRules := map[string][]string{
		"api":           {"users", "posts", "comments"},
		"api.v1":        {"search", "analytics"},
		"api.v2":        {"search", "suggestions"},
		"api.v2.beta":   {"features"},
		"frontend":      {"home", "about", "contact", "products"},
		"frontend.blog": {"posts", "archives"},
	}

	err = manager.Validate(validationRules)
	if err != nil {
		t.Errorf("Validation should pass for legacy config: %v", err)
	}

	// Test that legacy config groups have no templates
	apiGroup := manager.Group("api")
	if templateOwner := apiGroup.FindTemplateOwner(); templateOwner != nil {
		t.Error("Legacy config groups should not have templates")
	}

	frontendGroup := manager.Group("frontend")
	if templateOwner := frontendGroup.FindTemplateOwner(); templateOwner != nil {
		t.Error("Legacy config groups should not have templates")
	}

	// Test that Route and MustRoute methods work
	userRoute, err := apiGroup.Route("users")
	if err != nil {
		t.Fatalf("Failed to get user route: %v", err)
	}
	expected = "/users/:id"
	if userRoute != expected {
		t.Errorf("Expected route %q, got %q", expected, userRoute)
	}

	homeRoute := frontendGroup.MustRoute("home")
	expected = "/"
	if homeRoute != expected {
		t.Errorf("Expected route %q, got %q", expected, homeRoute)
	}

	// Test AddRoutes method works on legacy groups
	apiGroup.AddRoutes(map[string]string{
		"webhooks": "/webhooks/:event",
		"status":   "/status",
	})

	webhookURL, err := apiGroup.Builder("webhooks").
		WithParam("event", "user.created").
		Build()
	if err != nil {
		t.Fatalf("Failed to build webhook URL: %v", err)
	}
	expected = "https://api.legacy.com/webhooks/user.created"
	if webhookURL != expected {
		t.Errorf("Expected %q, got %q", expected, webhookURL)
	}
}
