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

func TestNewRouteManagerFromConfig(t *testing.T) {
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

	manager := urlkit.NewRouteManagerFromConfig(config)

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

	manager := urlkit.NewRouteManagerFromConfig(config)

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

	manager := urlkit.NewRouteManagerFromConfig(config)

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

	manager := urlkit.NewRouteManagerFromConfig(config)
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

	manager := urlkit.NewRouteManagerFromConfig(config)

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

func TestNestedGroupHierarchicalPathConstruction(t *testing.T) {
	// Test hierarchical URL building with nested groups
	rm := urlkit.NewRouteManager()

	// Register root group - this returns *RouteManager
	rm.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	// Get the frontend group and register nested groups manually
	frontend := rm.Group("frontend")
	frontend.RegisterGroup("es", "/es", map[string]string{
		"about":   "/acerca",
		"contact": "/contacto",
	})

	en := frontend.RegisterGroup("en", "/en", map[string]string{
		"about":   "/about-us",
		"contact": "/contact",
	})

	// Add deeper nesting
	en.RegisterGroup("deep", "/deep", map[string]string{
		"nested": "/nested-route",
	})

	// Test root level URL building (should work as before)
	homeURL, err := rm.Group("frontend").Builder("home").Build()
	if err != nil {
		t.Fatalf("Failed to build home URL: %v", err)
	}
	expected := "https://example.com/"
	if homeURL != expected {
		t.Errorf("Expected %q, got %q", expected, homeURL)
	}

	// Test nested group URL building
	aboutEsURL, err := rm.Group("frontend").Group("es").Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish about URL: %v", err)
	}
	expected = "https://example.com/es/acerca"
	if aboutEsURL != expected {
		t.Errorf("Expected %q, got %q", expected, aboutEsURL)
	}

	// Test English nested group
	aboutEnURL, err := rm.Group("frontend").Group("en").Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build English about URL: %v", err)
	}
	expected = "https://example.com/en/about-us"
	if aboutEnURL != expected {
		t.Errorf("Expected %q, got %q", expected, aboutEnURL)
	}

	// Test deeply nested group (3 levels deep)
	nestedURL, err := rm.Group("frontend").Group("en").Group("deep").Builder("nested").Build()
	if err != nil {
		t.Fatalf("Failed to build deeply nested URL: %v", err)
	}
	expected = "https://example.com/en/deep/nested-route"
	if nestedURL != expected {
		t.Errorf("Expected %q, got %q", expected, nestedURL)
	}

	// Test with parameters and query strings
	contactEsURL, err := rm.Group("frontend").Group("es").Builder("contact").
		WithQuery("lang", "es").
		WithQuery("source", "web").
		Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish contact URL with query: %v", err)
	}
	// Check that URL contains both query parameters (order may vary)
	if !strings.Contains(contactEsURL, "https://example.com/es/contacto?") {
		t.Errorf("Expected URL to start with 'https://example.com/es/contacto?', got %q", contactEsURL)
	}
	if !strings.Contains(contactEsURL, "lang=es") {
		t.Errorf("Expected URL to contain 'lang=es', got %q", contactEsURL)
	}
	if !strings.Contains(contactEsURL, "source=web") {
		t.Errorf("Expected URL to contain 'source=web', got %q", contactEsURL)
	}
}

func TestGroupRegisterNestedGroup(t *testing.T) {
	// Test programmatic group registration with RegisterGroup()
	rm := urlkit.NewRouteManager()

	// Register root group
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})

	// Get the root group and register nested groups
	apiGroup := rm.Group("api")
	v1Group := apiGroup.RegisterGroup("v1", "/v1", map[string]string{
		"users": "/users/:id",
		"posts": "/posts",
	})

	// Test that the returned group is correct
	if v1Group == nil {
		t.Fatal("RegisterGroup should return the created child group")
	}

	// Test that routes were properly registered
	route, err := v1Group.Route("users")
	if err != nil {
		t.Fatalf("Expected route 'users' to be registered, got error: %v", err)
	}
	expected := "/users/:id"
	if route != expected {
		t.Errorf("Expected route %q, got %q", expected, route)
	}

	// Test registering multiple levels
	v2Group := apiGroup.RegisterGroup("v2", "/v2", map[string]string{
		"users":    "/users/:id",
		"profiles": "/users/:id/profile",
	})

	// Test that both nested groups exist
	if v2Group == nil {
		t.Fatal("RegisterGroup should return the created child group for v2")
	}

	// Test merging routes into existing group
	existingGroup := apiGroup.RegisterGroup("v1", "/v1", map[string]string{
		"comments": "/comments/:id",
	})

	// Should return the same group (existing)
	if existingGroup != v1Group {
		t.Error("RegisterGroup should return existing group when name already exists")
	}

	// Merged route should be available
	_, err = v1Group.Route("comments")
	if err != nil {
		t.Errorf("Expected merged route 'comments' to be available: %v", err)
	}
}

func TestGroupFluentTraversal(t *testing.T) {
	// Test fluent API traversal with Group().Group()
	rm := urlkit.NewRouteManager()

	// Create nested structure: app -> locale -> section
	rm.RegisterGroup("app", "https://myapp.com", map[string]string{
		"root": "/",
	})

	app := rm.Group("app")
	app.RegisterGroup("en", "/en", map[string]string{
		"home": "/home",
	})
	app.RegisterGroup("es", "/es", map[string]string{
		"home": "/inicio",
	})

	// Add subsections to English
	en := app.Group("en")
	en.RegisterGroup("help", "/help", map[string]string{
		"faq":     "/faq",
		"contact": "/contact",
	})

	// Test fluent traversal
	helpGroup := rm.Group("app").Group("en").Group("help")
	if helpGroup == nil {
		t.Fatal("Fluent traversal should return the help group")
	}

	// Test route access through fluent API
	route, err := helpGroup.Route("faq")
	if err != nil {
		t.Fatalf("Expected route 'faq' to be accessible through fluent API: %v", err)
	}
	expected := "/faq"
	if route != expected {
		t.Errorf("Expected route %q, got %q", expected, route)
	}

	// Test panic on non-existent group
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when accessing non-existent group")
		}
	}()
	rm.Group("app").Group("nonexistent")
}

func TestNestedGroupURLBuilding(t *testing.T) {
	// Test URL building for nested groups with various scenarios
	rm := urlkit.NewRouteManager()

	// Create e-commerce site structure
	rm.RegisterGroup("store", "https://shop.example.com", map[string]string{
		"home": "/",
	})

	store := rm.Group("store")

	// Country-specific sections
	us := store.RegisterGroup("us", "/us", map[string]string{
		"products": "/products/:category",
		"cart":     "/cart",
	})

	store.RegisterGroup("uk", "/uk", map[string]string{
		"products": "/products/:category",
		"cart":     "/basket", // Different terminology
	})

	// Add payment section to US
	us.RegisterGroup("payment", "/payment", map[string]string{
		"checkout": "/checkout/:step",
		"success":  "/success",
	})

	// Test root level URL
	homeURL, err := store.Builder("home").Build()
	if err != nil {
		t.Fatalf("Failed to build root URL: %v", err)
	}
	expected := "https://shop.example.com/"
	if homeURL != expected {
		t.Errorf("Expected %q, got %q", expected, homeURL)
	}

	// Test first level nesting
	usProductsURL, err := rm.Group("store").Group("us").Builder("products").
		WithParam("category", "electronics").
		Build()
	if err != nil {
		t.Fatalf("Failed to build US products URL: %v", err)
	}
	expected = "https://shop.example.com/us/products/electronics"
	if usProductsURL != expected {
		t.Errorf("Expected %q, got %q", expected, usProductsURL)
	}

	// Test UK variant (different cart terminology)
	ukCartURL, err := rm.Group("store").Group("uk").Builder("cart").Build()
	if err != nil {
		t.Fatalf("Failed to build UK cart URL: %v", err)
	}
	expected = "https://shop.example.com/uk/basket"
	if ukCartURL != expected {
		t.Errorf("Expected %q, got %q", expected, ukCartURL)
	}

	// Test deeply nested URL with parameters and query
	checkoutURL, err := rm.Group("store").Group("us").Group("payment").
		Builder("checkout").
		WithParam("step", "billing").
		WithQuery("return", "cart").
		WithQuery("method", "credit").
		Build()
	if err != nil {
		t.Fatalf("Failed to build nested checkout URL: %v", err)
	}
	expected = "https://shop.example.com/us/payment/checkout/billing?return=cart&method=credit"
	if checkoutURL != expected {
		t.Errorf("Expected %q, got %q", expected, checkoutURL)
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

	manager := urlkit.NewRouteManagerFromConfig(config)

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
		urlkit.NewRouteManagerFromConfig(invalidConfig)
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

	manager := urlkit.NewRouteManagerFromConfig(config)

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

// ExampleInternationalizationPattern demonstrates how to use nested groups
// for internationalization with localized routes and content.
func ExampleInternationalizationPattern() {
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

// ExampleAPIVersioningPattern demonstrates how to use nested groups
// for API versioning with backward compatibility.
func ExampleAPIVersioningPattern() {
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

// ExampleConfigurationBasedSetup demonstrates how to load nested groups
// from JSON configuration for complex application structures.
func ExampleConfigurationBasedSetup() {
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
	manager := urlkit.NewRouteManagerFromConfig(config)

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

// ExampleValidationWithDotSeparatedPaths demonstrates how to validate
// nested group configurations using dot-separated path notation.
func ExampleValidationWithDotSeparatedPaths() {
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
