package urlkit

import (
	"maps"
	"slices"
	"testing"
)

func BenchmarkNavigationRendering(b *testing.B) {
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	frontend := manager.Group("frontend")

	locales := []string{"en", "es", "fr"}
	sections := []struct {
		name   string
		path   string
		routes map[string]string
	}{
		{
			name: "marketing",
			path: "/marketing",
			routes: map[string]string{
				"landing": "/landing",
				"pricing": "/pricing",
				"events":  "/events/:slug",
			},
		},
		{
			name: "support",
			path: "/support",
			routes: map[string]string{
				"docs":    "/docs/:category/:article",
				"faq":     "/faq",
				"contact": "/contact",
			},
		},
		{
			name: "account",
			path: "/account",
			routes: map[string]string{
				"profile": "/profile",
				"orders":  "/orders/:id",
				"billing": "/billing",
			},
		},
	}

	type navTarget struct {
		group  *Group
		routes []string
		params func(route string) Params
	}

	targets := make([]navTarget, 0, len(locales)*len(sections))

	for _, locale := range locales {
		localeGroup := frontend.RegisterGroup(locale, "/"+locale, map[string]string{})
		localeGroup.SetTemplateVar("locale", locale)

		for _, section := range sections {
			sectionName := section.name
			sectionGroup := localeGroup.RegisterGroup(sectionName, section.path, section.routes)

			routeNames := slices.Sorted(maps.Keys(section.routes))

			localeCopy := locale
			sectionCopy := sectionName
			paramsFn := func(route string) Params {
				return Params{
					"slug":    route,
					"locale":  localeCopy,
					"section": sectionCopy,
				}
			}

			targets = append(targets, navTarget{
				group:  sectionGroup,
				routes: routeNames,
				params: paramsFn,
			})
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for _, target := range targets {
			nodes, err := target.group.Navigation(target.routes, target.params)
			if err != nil {
				b.Fatalf("navigation build failed: %v", err)
			}

			if len(nodes) != len(target.routes) {
				b.Fatalf("unexpected node count: got %d, want %d", len(nodes), len(target.routes))
			}
		}
	}
}
