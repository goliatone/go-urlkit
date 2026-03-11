package urlkit_test

import (
	"errors"
	"testing"

	urlkit "github.com/goliatone/go-urlkit"
)

func TestRouteMutationDefaultPolicyIsErrorAndCollectsConflicts(t *testing.T) {
	rm := urlkit.NewRouteManager()
	if _, _, err := rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users":   "/users/:id",
		"status":  "/status",
		"metrics": "/metrics",
	}); err != nil {
		t.Fatalf("RegisterGroup failed: %v", err)
	}

	_, result, err := rm.AddRoutes("api", map[string]string{
		"users":   "/v2/users/:id",
		"status":  "/v2/status",
		"reports": "/reports",
	})
	if err == nil {
		t.Fatal("expected conflict error, got nil")
	}

	var conflicts urlkit.RouteConflictErrors
	if !errors.As(err, &conflicts) {
		t.Fatalf("expected RouteConflictErrors, got %T", err)
	}
	if len(conflicts.Conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d", len(conflicts.Conflicts))
	}
	if len(result.Added) != 0 {
		t.Fatalf("expected no committed additions on conflict error, got %+v", result.Added)
	}
	if len(result.Conflicts) != 2 {
		t.Fatalf("expected conflict details in result, got %+v", result.Conflicts)
	}
	if _, err := rm.RouteTemplate("api", "reports"); !errors.Is(err, urlkit.ErrRouteNotFound) {
		t.Fatalf("expected reports route to be absent after conflict failure, got %v", err)
	}
}

func TestRouteMutationPoliciesSkipAndReplace(t *testing.T) {
	skipManager := urlkit.NewRouteManager(urlkit.WithConflictPolicy(urlkit.RouteConflictPolicySkip))
	if _, _, err := skipManager.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users": "/users/:id",
	}); err != nil {
		t.Fatalf("skip manager RegisterGroup failed: %v", err)
	}

	_, skipResult, err := skipManager.AddRoutes("api", map[string]string{
		"users":  "/v2/users/:id",
		"health": "/health",
	})
	if err != nil {
		t.Fatalf("skip manager AddRoutes failed: %v", err)
	}
	if len(skipResult.Skipped) != 1 || skipResult.Skipped[0] != "users" {
		t.Fatalf("expected users to be skipped, got %+v", skipResult.Skipped)
	}
	template, err := skipManager.RouteTemplate("api", "users")
	if err != nil {
		t.Fatalf("skip manager RouteTemplate failed: %v", err)
	}
	if template != "/users/:id" {
		t.Fatalf("expected original users template to remain, got %q", template)
	}

	replaceManager := urlkit.NewRouteManager(urlkit.WithConflictPolicy(urlkit.RouteConflictPolicyReplace))
	if _, _, err := replaceManager.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users": "/users/:id",
	}); err != nil {
		t.Fatalf("replace manager RegisterGroup failed: %v", err)
	}

	_, replaceResult, err := replaceManager.AddRoutes("api", map[string]string{
		"users": "/v2/users/:id",
	})
	if err != nil {
		t.Fatalf("replace manager AddRoutes failed: %v", err)
	}
	if len(replaceResult.Replaced) != 1 || replaceResult.Replaced[0] != "users" {
		t.Fatalf("expected users to be replaced, got %+v", replaceResult.Replaced)
	}
	template, err = replaceManager.RouteTemplate("api", "users")
	if err != nil {
		t.Fatalf("replace manager RouteTemplate failed: %v", err)
	}
	if template != "/v2/users/:id" {
		t.Fatalf("expected replaced users template, got %q", template)
	}
}

func TestRouteManagerFreezeBlocksMutationsAndAllowsReads(t *testing.T) {
	rm := urlkit.NewRouteManager()
	api, _, err := rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})
	if err != nil {
		t.Fatalf("RegisterGroup failed: %v", err)
	}

	rm.Freeze()
	if !rm.Frozen() {
		t.Fatal("expected manager to be frozen")
	}

	if _, _, err := rm.RegisterGroup("admin", "https://admin.example.com", nil); err == nil {
		t.Fatal("expected frozen RegisterGroup to fail")
	}
	if _, _, err := rm.AddRoutes("api", map[string]string{"users": "/users/:id"}); err == nil {
		t.Fatal("expected frozen AddRoutes to fail")
	}
	if err := api.SetTemplateVar("version", "v1"); err == nil {
		t.Fatal("expected frozen SetTemplateVar to fail")
	}
	if _, err := rm.EnsureGroup("api.v1"); err == nil {
		t.Fatal("expected frozen EnsureGroup create path to fail")
	}
	if _, err := rm.EnsureGroup("api"); err != nil {
		t.Fatalf("expected frozen EnsureGroup existing path to succeed: %v", err)
	}
	if path, err := rm.RoutePath("api", "status"); err != nil || path != "/status" {
		t.Fatalf("expected RoutePath to keep working after freeze, got %q err=%v", path, err)
	}
}

func TestRouteManagerManifestAndDiff(t *testing.T) {
	before := urlkit.NewRouteManager()
	root, _, err := before.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	})
	if err != nil {
		t.Fatalf("before RegisterGroup failed: %v", err)
	}
	mustRegisterGroup(t, root, "v1", "/v1", map[string]string{
		"users": "/users/:id",
	})

	after := urlkit.NewRouteManager(urlkit.WithConflictPolicy(urlkit.RouteConflictPolicyReplace))
	rootAfter, _, err := after.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/health",
	})
	if err != nil {
		t.Fatalf("after RegisterGroup failed: %v", err)
	}
	v1After := mustRegisterGroup(t, rootAfter, "v1", "/v1", map[string]string{
		"users": "/users/:id",
	})
	if _, err := v1After.AddRoutes(map[string]string{"reports": "/reports"}); err != nil {
		t.Fatalf("after AddRoutes failed: %v", err)
	}

	manifest := after.Manifest()
	if len(manifest) != 3 {
		t.Fatalf("expected 3 manifest entries, got %d", len(manifest))
	}
	if manifest[0].GroupFQN != "api" || manifest[0].RouteKey != "status" {
		t.Fatalf("expected first manifest entry api/status, got %+v", manifest[0])
	}
	if manifest[1].GroupFQN != "api.v1" || manifest[1].RouteKey != "reports" {
		t.Fatalf("expected second manifest entry api.v1/reports, got %+v", manifest[1])
	}
	if manifest[2].FullPathTemplate != "/v1/users/:id" {
		t.Fatalf("expected full path template /v1/users/:id, got %q", manifest[2].FullPathTemplate)
	}

	diff := urlkit.DiffRouteManifest(before.Manifest(), manifest)
	if len(diff.Added) != 1 || diff.Added[0].RouteKey != "reports" {
		t.Fatalf("expected reports to be added, got %+v", diff.Added)
	}
	if len(diff.Changed) != 1 || diff.Changed[0].Before.RouteTemplate != "/status" || diff.Changed[0].After.RouteTemplate != "/health" {
		t.Fatalf("expected status route change, got %+v", diff.Changed)
	}
	if len(diff.Removed) != 0 {
		t.Fatalf("expected no removed routes, got %+v", diff.Removed)
	}
}

func TestRouteManagerRejectsDottedRootRegistration(t *testing.T) {
	rm := urlkit.NewRouteManager()
	if _, _, err := rm.RegisterGroup("frontend.en", "https://example.com", map[string]string{
		"home": "/",
	}); err == nil {
		t.Fatal("expected dotted root registration to fail")
	}
}

func TestRouteManagerRejectsConflictingRootBaseURL(t *testing.T) {
	rm := urlkit.NewRouteManager(urlkit.WithConflictPolicy(urlkit.RouteConflictPolicyReplace))
	if _, _, err := rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
	}); err != nil {
		t.Fatalf("initial RegisterGroup failed: %v", err)
	}

	_, _, err := rm.RegisterGroup("api", "https://admin.example.com", map[string]string{
		"health": "/health",
	})
	if err == nil {
		t.Fatal("expected root base_url conflict")
	}

	var conflict urlkit.RootGroupConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("expected RootGroupConflictError, got %T", err)
	}
	if conflict.ExistingBaseURL != "https://api.example.com" || conflict.IncomingBaseURL != "https://admin.example.com" {
		t.Fatalf("unexpected base_url conflict payload: %+v", conflict)
	}
}
