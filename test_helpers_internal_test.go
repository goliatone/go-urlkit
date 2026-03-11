package urlkit

import "testing"

func mustManager(t testing.TB, values ...any) *RouteManager {
	t.Helper()
	if len(values) != 2 {
		t.Fatalf("mustManager expected manager and error, got %d values", len(values))
	}
	manager, _ := values[0].(*RouteManager)
	err, _ := values[1].(error)
	if err != nil {
		t.Fatalf("unexpected manager error: %v", err)
	}
	return manager
}

func mustGroup(t testing.TB, values ...any) *Group {
	t.Helper()
	if len(values) != 3 {
		t.Fatalf("mustGroup expected group, result, and error, got %d values", len(values))
	}
	group, _ := values[0].(*Group)
	err, _ := values[2].(error)
	if err != nil {
		t.Fatalf("unexpected group error: %v", err)
	}
	return group
}

func mustManagerFromConfig(t testing.TB, config Configurator, opts ...Option) *RouteManager {
	t.Helper()
	manager, err := NewRouteManagerFromConfig(config, opts...)
	if err != nil {
		t.Fatalf("unexpected manager error: %v", err)
	}
	return manager
}

func mustRegisterGroup(t testing.TB, parent *Group, name, path string, routes map[string]string) *Group {
	t.Helper()
	group, _, err := parent.RegisterGroup(name, path, routes)
	if err != nil {
		t.Fatalf("unexpected group error: %v", err)
	}
	return group
}
