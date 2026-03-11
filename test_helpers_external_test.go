package urlkit_test

import (
	"testing"

	urlkit "github.com/goliatone/go-urlkit"
)

func mustManager(t testing.TB, values ...any) *urlkit.RouteManager {
	t.Helper()
	if len(values) != 2 {
		t.Fatalf("mustManager expected manager and error, got %d values", len(values))
	}
	manager, _ := values[0].(*urlkit.RouteManager)
	err, _ := values[1].(error)
	if err != nil {
		t.Fatalf("unexpected manager error: %v", err)
	}
	return manager
}

func mustGroup(t testing.TB, values ...any) *urlkit.Group {
	t.Helper()
	if len(values) != 3 {
		t.Fatalf("mustGroup expected group, result, and error, got %d values", len(values))
	}
	group, _ := values[0].(*urlkit.Group)
	err, _ := values[2].(error)
	if err != nil {
		t.Fatalf("unexpected group error: %v", err)
	}
	return group
}

func mustManagerFromConfig(t testing.TB, config urlkit.Configurator, opts ...urlkit.Option) *urlkit.RouteManager {
	t.Helper()
	manager, err := urlkit.NewRouteManagerFromConfig(config, opts...)
	if err != nil {
		t.Fatalf("unexpected manager error: %v", err)
	}
	return manager
}

func mustRegisterGroup(t testing.TB, parent *urlkit.Group, name, path string, routes map[string]string) *urlkit.Group {
	t.Helper()
	group, _, err := parent.RegisterGroup(name, path, routes)
	if err != nil {
		t.Fatalf("unexpected group error: %v", err)
	}
	return group
}
