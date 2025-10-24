package urlkit_test

import (
	"errors"
	"strings"
	"testing"

	urlkit "github.com/goliatone/go-urlkit"
)

func TestEnsureGroupCreatesHierarchyAndDefaults(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	blog, err := rm.EnsureGroup("frontend.en.blog")
	if err != nil {
		t.Fatalf("EnsureGroup failed: %v", err)
	}
	if blog == nil {
		t.Fatal("EnsureGroup returned nil group")
	}

	if got := blog.FullName(); got != "frontend.en.blog" {
		t.Fatalf("expected full name frontend.en.blog, got %s", got)
	}

	if rm.Group("frontend.en.blog") != blog {
		t.Fatal("Group lookup should return ensured group instance")
	}

	if _, err := rm.AddRoutes("frontend.en.blog", map[string]string{
		"article": "/:slug",
	}); err != nil {
		t.Fatalf("AddRoutes failed: %v", err)
	}

	url, err := rm.Group("frontend.en.blog").
		Builder("article").
		WithParam("slug", "launch").
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := "https://example.com/en/blog/launch"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestEnsureGroupSupportsCustomPaths(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	if _, err := rm.EnsureGroup("frontend.marketing:/mkt.landing"); err != nil {
		t.Fatalf("EnsureGroup with custom path failed: %v", err)
	}

	if _, err := rm.AddRoutes("frontend.marketing.landing", map[string]string{
		"promo": "/:slug",
	}); err != nil {
		t.Fatalf("AddRoutes failed: %v", err)
	}

	url, err := rm.Group("frontend.marketing.landing").
		Builder("promo").
		WithParam("slug", "fall-sale").
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	expected := "https://example.com/mkt/landing/fall-sale"
	if url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestRouteManagerAddRoutesRedefinesTemplates(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})

	if _, err := rm.AddRoutes("api", map[string]string{
		"users": "/users/:id",
	}); err != nil {
		t.Fatalf("initial AddRoutes failed: %v", err)
	}

	url, err := rm.Group("api").
		Builder("users").
		WithParam("id", 1).
		Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if expected := "https://api.example.com/users/1"; url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}

	// Redefine the users route with a new template
	if _, err := rm.AddRoutes("api", map[string]string{
		"users": "/v2/users/:id",
	}); err != nil {
		t.Fatalf("redefinition AddRoutes failed: %v", err)
	}

	url, err = rm.Group("api").
		Builder("users").
		WithParam("id", 42).
		Build()
	if err != nil {
		t.Fatalf("Build failed after redefinition: %v", err)
	}
	if expected := "https://api.example.com/v2/users/42"; url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestAddRoutesRespectsTemplateOwner(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	frontend := rm.Group("frontend")
	frontend.SetURLTemplate("{base_url}{locale}{route_path}")
	frontend.SetTemplateVar("locale", "/en")

	en, err := rm.EnsureGroup("frontend.en")
	if err != nil {
		t.Fatalf("EnsureGroup failed: %v", err)
	}
	en.SetTemplateVar("locale", "/es")

	if _, err := rm.AddRoutes("frontend.en", map[string]string{
		"about": "/about",
	}); err != nil {
		t.Fatalf("AddRoutes failed: %v", err)
	}

	url, err := rm.Group("frontend.en").Builder("about").Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if expected := "https://example.com/es/about/"; url != expected {
		t.Fatalf("expected %s, got %s", expected, url)
	}
}

func TestHelpfulErrorsForUnknownGroupsAndRoutes(t *testing.T) {
	rm := urlkit.NewRouteManager()
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})

	if _, err := rm.GetGroup("missing"); !errors.Is(err, urlkit.ErrGroupNotFound) {
		t.Fatalf("expected ErrGroupNotFound, got %v", err)
	}

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic from Group on missing path")
			}
		}()

		rm.Group("missing")
	}()

	if err := func() error {
		_, routeErr := rm.Group("api").Route("missing")
		return routeErr
	}(); err == nil {
		t.Fatal("expected route lookup to fail")
	} else {
		if !errors.Is(err, urlkit.ErrRouteNotFound) {
			t.Fatalf("expected ErrRouteNotFound, got %v", err)
		}
		if !strings.Contains(err.Error(), "api") {
			t.Fatalf("expected error message to include group path, got %q", err.Error())
		}
	}

	if _, err := rm.Group("api").Render("missing", nil); !errors.Is(err, urlkit.ErrRouteNotFound) {
		t.Fatalf("expected ErrRouteNotFound from Render, got %v", err)
	}

	if _, err := rm.Group("api").Builder("missing").Build(); err == nil {
		t.Fatal("expected builder to fail for missing route")
	} else if !errors.Is(err, urlkit.ErrRouteNotFound) {
		t.Fatalf("expected ErrRouteNotFound from builder, got %v", err)
	}

}
