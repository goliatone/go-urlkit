package urlkit_test

import (
	"strings"
	"testing"

	"github.com/goliatone/go-urlkit"
)

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
	// Check URL base and path first
	expectedBase := "https://shop.example.com/us/payment/checkout/billing?"
	if !strings.HasPrefix(checkoutURL, expectedBase) {
		t.Errorf("Expected URL to start with %q, got %q", expectedBase, checkoutURL)
	}
	// Check that both query parameters are present (order may vary due to map iteration)
	if !strings.Contains(checkoutURL, "return=cart") {
		t.Errorf("Expected URL to contain 'return=cart', got %q", checkoutURL)
	}
	if !strings.Contains(checkoutURL, "method=credit") {
		t.Errorf("Expected URL to contain 'method=credit', got %q", checkoutURL)
	}
}
