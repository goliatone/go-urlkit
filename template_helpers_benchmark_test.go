package urlkit

import (
	"testing"

	"github.com/flosch/pongo2/v6"
)

// BenchmarkParseArgs benchmarks the parseArgs function which is critical for all template helpers
func BenchmarkParseArgs(b *testing.B) {
	// Setup test data
	groupArg := pongo2.AsValue("frontend")
	routeArg := pongo2.AsValue("user_profile")
	paramsArg := pongo2.AsValue(map[string]any{
		"id":      123,
		"slug":    "test-user",
		"section": "profile",
	})
	queryArg := pongo2.AsValue(map[string]any{
		"tab":    "settings",
		"page":   "1",
		"format": "json",
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := parseArgs(groupArg, routeArg, paramsArg, queryArg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkParseArgsMinimal benchmarks parseArgs with minimal arguments (group + route only)
func BenchmarkParseArgsMinimal(b *testing.B) {
	groupArg := pongo2.AsValue("frontend")
	routeArg := pongo2.AsValue("user_profile")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := parseArgs(groupArg, routeArg)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFromPongoValue benchmarks the type conversion function for different data types
func BenchmarkFromPongoValue(b *testing.B) {
	tests := []struct {
		name  string
		value *pongo2.Value
	}{
		{"String", pongo2.AsValue("test-string")},
		{"Int", pongo2.AsValue(42)},
		{"Float", pongo2.AsValue(3.14)},
		{"Bool", pongo2.AsValue(true)},
		{"SimpleMap", pongo2.AsValue(map[string]any{"key": "value"})},
		{"ComplexMap", pongo2.AsValue(map[string]any{
			"id":      123,
			"name":    "John Doe",
			"active":  true,
			"score":   98.5,
			"tags":    []string{"user", "admin"},
			"profile": map[string]any{"age": 30, "city": "NYC"},
		})},
		{"Array", pongo2.AsValue([]string{"item1", "item2", "item3"})},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_ = fromPongoValue(tt.value)
			}
		})
	}
}

// BenchmarkUrlHelper benchmarks the core url helper function
func BenchmarkUrlHelper(b *testing.B) {
	// Setup test manager
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()

	b.Run("Legacy", func(b *testing.B) {
		helper := urlHelper(manager, config)

		// Prepare arguments
		groupArg := pongo2.AsValue("frontend")
		routeArg := pongo2.AsValue("user_profile")
		paramsArg := pongo2.AsValue(map[string]any{"id": 123})
		queryArg := pongo2.AsValue(map[string]any{"tab": "profile"})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(groupArg, routeArg, paramsArg, queryArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String() // Force evaluation
		}
	})

	b.Run("WithCache", func(b *testing.B) {
		groupCache := NewGroupCache(manager)
		helper := urlHelperWithCache(groupCache, config)

		// Prepare arguments
		groupArg := pongo2.AsValue("frontend")
		routeArg := pongo2.AsValue("user_profile")
		paramsArg := pongo2.AsValue(map[string]any{"id": 123})
		queryArg := pongo2.AsValue(map[string]any{"tab": "profile"})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(groupArg, routeArg, paramsArg, queryArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String() // Force evaluation
		}
	})
}

// BenchmarkUrlHelperMinimal benchmarks url helper with minimal arguments
func BenchmarkUrlHelperMinimal(b *testing.B) {
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()
	helper := urlHelper(manager, config)

	groupArg := pongo2.AsValue("frontend")
	routeArg := pongo2.AsValue("home")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := helper(groupArg, routeArg)
		if err != nil {
			b.Fatal(err)
		}
		_ = result.String()
	}
}

// BenchmarkHasRouteHelper benchmarks the has_route helper
func BenchmarkHasRouteHelper(b *testing.B) {
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()
	helper := hasRouteHelper(manager, config)

	groupArg := pongo2.AsValue("frontend")
	routeArg := pongo2.AsValue("user_profile")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := helper(groupArg, routeArg)
		if err != nil {
			b.Fatal(err)
		}
		_ = result.Bool()
	}
}

// BenchmarkRouteTemplateHelper benchmarks the route_template helper
func BenchmarkRouteTemplateHelper(b *testing.B) {
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()
	helper := routeTemplateHelper(manager, config)

	groupArg := pongo2.AsValue("frontend")
	routeArg := pongo2.AsValue("user_profile")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := helper(groupArg, routeArg)
		if err != nil {
			b.Fatal(err)
		}
		_ = result.String()
	}
}

// BenchmarkLocalizationHelpersPerformance benchmarks the localization helper functions
func BenchmarkLocalizationHelpersPerformance(b *testing.B) {
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()
	localeConfig := &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          []string{"en", "es", "fr", "de"},
		LocaleGroups:              map[string][]string{"frontend": {"en", "es", "fr"}},
		EnableLocaleFallback:      true,
		EnableHierarchicalLocales: true,
		EnableLocaleValidation:    true,
		DetectionStrategies:       []LocaleDetectionStrategy{LocaleFromContext},
	}

	b.Run("url_i18n", func(b *testing.B) {
		helper := urlI18nHelper(manager, config, localeConfig)
		groupArg := pongo2.AsValue("frontend")
		routeArg := pongo2.AsValue("user_profile")
		paramsArg := pongo2.AsValue(map[string]any{"id": 123})
		contextArg := pongo2.AsValue(map[string]any{"locale": "es"})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(groupArg, routeArg, paramsArg, pongo2.AsValue(map[string]any{}), contextArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String()
		}
	})

	b.Run("url_locale", func(b *testing.B) {
		helper := urlLocaleHelper(manager, config, localeConfig)
		groupArg := pongo2.AsValue("frontend")
		routeArg := pongo2.AsValue("user_profile")
		localeArg := pongo2.AsValue("es")
		paramsArg := pongo2.AsValue(map[string]any{"id": 123})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(groupArg, routeArg, localeArg, paramsArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String()
		}
	})

	b.Run("url_all_locales", func(b *testing.B) {
		helper := urlAllLocalesHelper(manager, config, localeConfig)
		groupArg := pongo2.AsValue("frontend")
		routeArg := pongo2.AsValue("user_profile")
		paramsArg := pongo2.AsValue(map[string]any{"id": 123})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(groupArg, routeArg, paramsArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.Interface()
		}
	})

	b.Run("has_locale", func(b *testing.B) {
		helper := hasLocaleHelper(manager, config, localeConfig)
		groupArg := pongo2.AsValue("frontend")
		localeArg := pongo2.AsValue("es")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(groupArg, localeArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.Bool()
		}
	})

	b.Run("current_locale", func(b *testing.B) {
		helper := currentLocaleHelper(config, localeConfig)
		contextArg := pongo2.AsValue(map[string]any{"locale": "es"})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(contextArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String()
		}
	})
}

// BenchmarkMemoryAllocations specifically focuses on memory allocation patterns
func BenchmarkMemoryAllocations(b *testing.B) {
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()
	helper := urlHelper(manager, config)

	b.Run("ReusableArgs", func(b *testing.B) {
		// Pre-allocate arguments to measure just the helper execution
		groupArg := pongo2.AsValue("frontend")
		routeArg := pongo2.AsValue("user_profile")
		paramsArg := pongo2.AsValue(map[string]any{"id": 123})

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := helper(groupArg, routeArg, paramsArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String()
		}
	})

	b.Run("FreshArgs", func(b *testing.B) {
		// Create fresh arguments each time to measure total allocation cost
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			groupArg := pongo2.AsValue("frontend")
			routeArg := pongo2.AsValue("user_profile")
			paramsArg := pongo2.AsValue(map[string]any{"id": 123})

			result, err := helper(groupArg, routeArg, paramsArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String()
		}
	})
}

// BenchmarkSafeTemplateHelper benchmarks the panic recovery wrapper
func BenchmarkSafeTemplateHelper(b *testing.B) {
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()

	baseHelper := urlHelper(manager, config)
	safeHelper := safeTemplateHelper("url", config, baseHelper)

	groupArg := pongo2.AsValue("frontend")
	routeArg := pongo2.AsValue("user_profile")

	b.Run("WithSafety", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := safeHelper(groupArg, routeArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String()
		}
	})

	b.Run("WithoutSafety", func(b *testing.B) {
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			result, err := baseHelper(groupArg, routeArg)
			if err != nil {
				b.Fatal(err)
			}
			_ = result.String()
		}
	})
}

// setupBenchmarkManager creates a RouteManager for benchmarking
func setupBenchmarkManager() *RouteManager {
	config := &Config{
		Groups: []GroupConfig{
			{
				Name:    "frontend",
				BaseURL: "https://example.com",
				Routes: map[string]string{
					"home":         "/",
					"about":        "/about",
					"contact":      "/contact",
					"user_profile": "/users/:id/profile",
					"user_posts":   "/users/:id/posts",
					"product":      "/products/:slug",
					"category":     "/categories/:category/:subcategory?",
				},
			},
			{
				Name:    "api",
				BaseURL: "https://api.example.com",
				Routes: map[string]string{
					"users":      "/v1/users/:id?",
					"posts":      "/v1/posts",
					"health":     "/health",
					"metrics":    "/metrics",
					"user_posts": "/v1/users/:id/posts",
					"search":     "/v1/search",
				},
			},
			{
				Name:    "admin",
				BaseURL: "https://admin.example.com",
				Routes: map[string]string{
					"dashboard": "/dashboard",
					"users":     "/users",
					"reports":   "/reports/:type",
					"settings":  "/settings/:section?",
				},
			},
		},
	}

	manager := NewRouteManager(config)
	return manager
}

// BenchmarkComplexScenarios tests realistic usage patterns
func BenchmarkComplexScenarios(b *testing.B) {
	manager := setupBenchmarkManager()
	config := DefaultTemplateHelperConfig()
	localeConfig := &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          []string{"en", "es", "fr", "de", "it", "pt"},
		LocaleGroups:              map[string][]string{"frontend": {"en", "es", "fr"}},
		EnableLocaleFallback:      true,
		EnableHierarchicalLocales: true,
		EnableLocaleValidation:    true,
		DetectionStrategies:       []LocaleDetectionStrategy{LocaleFromContext, LocaleFromURL, LocaleFromCookie},
	}

	helpers := TemplateHelpersWithLocale(manager, config, localeConfig)

	// Simulate common template rendering scenarios
	scenarios := []struct {
		name   string
		helper string
		args   []*pongo2.Value
	}{
		{
			name:   "SimpleURL",
			helper: "url",
			args:   []*pongo2.Value{pongo2.AsValue("frontend"), pongo2.AsValue("home")},
		},
		{
			name:   "URLWithParams",
			helper: "url",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
				pongo2.AsValue(map[string]any{"id": 123}),
			},
		},
		{
			name:   "URLWithParamsAndQuery",
			helper: "url",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_posts"),
				pongo2.AsValue(map[string]any{"id": 123}),
				pongo2.AsValue(map[string]any{"page": "1", "sort": "date", "format": "json"}),
			},
		},
		{
			name:   "RouteValidation",
			helper: "has_route",
			args:   []*pongo2.Value{pongo2.AsValue("frontend"), pongo2.AsValue("user_profile")},
		},
		{
			name:   "LocalizedURL",
			helper: "url_locale",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("user_profile"),
				pongo2.AsValue("es"),
				pongo2.AsValue(map[string]any{"id": 123}),
			},
		},
	}

	for _, scenario := range scenarios {
		b.Run(scenario.name, func(b *testing.B) {
			helperFunc := helpers[scenario.helper].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				result, err := helperFunc(scenario.args...)
				if err != nil {
					b.Fatal(err)
				}
				_ = result.Interface() // Force evaluation
			}
		})
	}
}
