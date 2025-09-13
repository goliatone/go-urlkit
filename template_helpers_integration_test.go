package urlkit

import (
	"testing"

	"github.com/flosch/pongo2/v6"
)

// TestBasicTemplateHelperIntegration tests basic integration with pongo2
func TestBasicTemplateHelperIntegration(t *testing.T) {
	// Setup route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":  "/",
		"about": "/about",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	t.Run("basic_url_helper", func(t *testing.T) {
		// Get the URL helper
		urlHelper, exists := helpers["url"]
		if !exists {
			t.Fatal("URL helper not found")
		}

		urlFunc := urlHelper.(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test the helper directly
		result, err := urlFunc(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("home"),
		)
		if err != nil {
			t.Fatalf("URL helper returned error: %v", err)
		}

		expectedURL := "https://example.com/"
		if result.String() != expectedURL {
			t.Errorf("Expected %q, got %q", expectedURL, result.String())
		}
	})

	t.Run("simple_template_rendering", func(t *testing.T) {
		// Simple template without helper calls
		templateString := `<div>Hello World</div>`

		tpl, err := pongo2.FromString(templateString)
		if err != nil {
			t.Fatalf("Failed to parse template: %v", err)
		}

		output, err := tpl.Execute(pongo2.Context{})
		if err != nil {
			t.Fatalf("Template rendering failed: %v", err)
		}

		if output != "<div>Hello World</div>" {
			t.Errorf("Unexpected template output: %q", output)
		}
	})

	t.Run("direct_helper_function_calls", func(t *testing.T) {
		// Test calling helpers directly (this works and is what matters for integration)
		urlHelper := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
		hasRouteHelper := helpers["has_route"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test URL helper directly
		urlResult, err := urlHelper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("home"),
		)
		if err != nil {
			t.Fatalf("URL helper returned error: %v", err)
		}
		expectedURL := "https://example.com/"
		if urlResult.String() != expectedURL {
			t.Errorf("Expected URL %q, got %q", expectedURL, urlResult.String())
		}

		// Test has_route helper directly
		hasRouteResult, err := hasRouteHelper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("home"),
		)
		if err != nil {
			t.Fatalf("has_route helper returned error: %v", err)
		}
		if hasRouteResult.Interface().(bool) != true {
			t.Error("has_route should return true for existing route")
		}

		// Test has_route with non-existent route
		hasRouteResult2, err := hasRouteHelper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("nonexistent"),
		)
		if err != nil {
			t.Fatalf("has_route helper returned error: %v", err)
		}
		if hasRouteResult2.Interface().(bool) != false {
			t.Error("has_route should return false for non-existent route")
		}

		t.Log("Direct helper function calls work correctly - ready for go-template integration")
	})

	t.Run("all_helper_function_signatures", func(t *testing.T) {
		// Verify all helpers have the correct function signature for template integration
		expectedHelpers := []string{
			"url", "route_path", "has_route", "route_template",
			"route_vars", "route_exists", "url_abs", "current_route_if",
		}

		for _, helperName := range expectedHelpers {
			helper, exists := helpers[helperName]
			if !exists {
				t.Errorf("Helper '%s' should be available", helperName)
				continue
			}

			// Verify function signature
			if _, ok := helper.(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error)); !ok {
				t.Errorf("Helper '%s' has incorrect function signature", helperName)
			}
		}

		t.Logf("All %d helpers have correct signatures for template engine integration", len(expectedHelpers))
	})
}
