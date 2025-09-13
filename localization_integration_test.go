package urlkit

import (
	"testing"

	"github.com/flosch/pongo2/v6"
)

// TestLocaleDetectionScenarios tests locale detection in different scenarios
func TestLocaleDetectionScenarios(t *testing.T) {
	tests := []struct {
		name           string
		config         *LocaleConfig
		context        map[string]any
		expectedLocale string
		expectedGroup  string
		description    string
	}{
		{
			name: "Context-based detection with primary locale key",
			config: NewMultiStrategyLocaleConfig("en", []string{"en", "es", "fr"}, []LocaleDetectionStrategy{
				LocaleFromContext,
			}),
			context: map[string]any{
				"locale": "es",
			},
			expectedLocale: "es",
			expectedGroup:  "frontend",
			description:    "Should detect 'es' from primary 'locale' key in context",
		},
		{
			name: "Context-based detection with fallback key",
			config: NewMultiStrategyLocaleConfig("en", []string{"en", "es", "fr"}, []LocaleDetectionStrategy{
				LocaleFromContext,
			}),
			context: map[string]any{
				"user_locale": "fr",
			},
			expectedLocale: "fr",
			expectedGroup:  "frontend",
			description:    "Should detect 'fr' from fallback 'user_locale' key",
		},
		{
			name:   "URL-based detection with /locale/ prefix",
			config: NewURLBasedLocaleConfig("en", []string{"en", "es", "fr"}),
			context: map[string]any{
				"url_path": "/es/products/123",
			},
			expectedLocale: "es",
			expectedGroup:  "frontend",
			description:    "Should detect 'es' from URL path /es/products/123",
		},
		{
			name:   "URL-based detection with /locale/{code} pattern",
			config: NewURLBasedLocaleConfig("en", []string{"en", "es", "fr"}),
			context: map[string]any{
				"url_path": "/locale/fr/about",
			},
			expectedLocale: "fr",
			expectedGroup:  "frontend",
			description:    "Should detect 'fr' from URL path /locale/fr/about",
		},
		{
			name:   "Header-based detection with Accept-Language",
			config: NewHeaderBasedLocaleConfig("en", []string{"en", "es", "fr"}),
			context: map[string]any{
				"accept_language": "es-ES,es;q=0.9,en;q=0.8",
			},
			expectedLocale: "es",
			expectedGroup:  "frontend",
			description:    "Should detect 'es' from Accept-Language header",
		},
		{
			name:   "Header-based detection with language prefix fallback",
			config: NewHeaderBasedLocaleConfig("en", []string{"en", "es", "fr"}),
			context: map[string]any{
				"accept_language": "es-MX,de;q=0.9,en;q=0.8",
			},
			expectedLocale: "es",
			expectedGroup:  "frontend",
			description:    "Should detect 'es' from 'es-MX' prefix in Accept-Language",
		},
		{
			name: "Cookie-based detection",
			config: &LocaleConfig{
				DefaultLocale:          "en",
				SupportedLocales:       []string{"en", "es", "fr"},
				DetectionStrategies:    []LocaleDetectionStrategy{LocaleFromCookie},
				EnableLocaleFallback:   true,
				EnableLocaleValidation: true,
			},
			context: map[string]any{
				"cookie_locale": "fr",
			},
			expectedLocale: "fr",
			expectedGroup:  "frontend",
			description:    "Should detect 'fr' from cookie locale",
		},
		{
			name:   "Multi-strategy detection with priority",
			config: NewFullStackLocaleConfig("en", []string{"en", "es", "fr"}),
			context: map[string]any{
				"locale":          "es",     // Highest priority (context)
				"url_path":        "/fr/",   // Lower priority (URL)
				"accept_language": "de;q=1", // Lowest priority (header)
			},
			expectedLocale: "es",
			expectedGroup:  "frontend",
			description:    "Should prioritize context over URL and header",
		},
		{
			name: "Fallback to default when detection fails",
			config: NewMultiStrategyLocaleConfig("en", []string{"en", "es", "fr"}, []LocaleDetectionStrategy{
				LocaleFromContext,
			}),
			context: map[string]any{
				"some_other_key": "value",
			},
			expectedLocale: "en",
			expectedGroup:  "frontend",
			description:    "Should fall back to default 'en' when no locale detected",
		},
		{
			name: "Unsupported locale with fallback enabled",
			config: &LocaleConfig{
				DefaultLocale:          "en",
				SupportedLocales:       []string{"en", "es"},
				DetectionStrategies:    []LocaleDetectionStrategy{LocaleFromContext},
				EnableLocaleFallback:   true,
				EnableLocaleValidation: true,
			},
			context: map[string]any{
				"locale": "de", // Unsupported locale
			},
			expectedLocale: "en",
			expectedGroup:  "frontend",
			description:    "Should fall back to default when detected locale is unsupported",
		},
		{
			name: "Group-specific locale support",
			config: &LocaleConfig{
				DefaultLocale:    "en",
				SupportedLocales: []string{"en", "es", "fr", "de"},
				LocaleGroups: map[string][]string{
					"api":      {"en", "de"},
					"frontend": {"en", "es", "fr"},
				},
				DetectionStrategies:    []LocaleDetectionStrategy{LocaleFromContext},
				EnableLocaleFallback:   true,
				EnableLocaleValidation: true,
			},
			context: map[string]any{
				"locale": "de",
			},
			expectedLocale: "en", // 'de' not supported for 'frontend' group, fallback to 'en'
			expectedGroup:  "frontend",
			description:    "Should respect group-specific locale restrictions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualLocale := tt.config.detectLocale(tt.context, tt.expectedGroup)
			if actualLocale != tt.expectedLocale {
				t.Errorf("Expected locale '%s', got '%s'. %s", tt.expectedLocale, actualLocale, tt.description)
			}
		})
	}
}

// TestHierarchicalLocaleGroups tests hierarchical locale groups
func TestHierarchicalLocaleGroups(t *testing.T) {
	// Set up URLKit with hierarchical locale structure
	manager := NewRouteManager()

	// Register base groups
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":    "/",
		"about":   "/about",
		"contact": "/contact",
	})

	// Register hierarchical locale groups: frontend.en, frontend.es, frontend.fr
	manager.RegisterGroup("frontend.en", "https://example.com", map[string]string{
		"home":    "/en/",
		"about":   "/en/about-us",
		"contact": "/en/contact-us",
	})

	manager.RegisterGroup("frontend.es", "https://example.com", map[string]string{
		"home":    "/es/",
		"about":   "/es/acerca-de",
		"contact": "/es/contacto",
	})

	manager.RegisterGroup("frontend.fr", "https://example.com", map[string]string{
		"home":    "/fr/",
		"about":   "/fr/a-propos",
		"contact": "/fr/contact",
	})

	// Set up locale configuration for hierarchical structure
	localeConfig := &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          []string{"en", "es", "fr"},
		EnableHierarchicalLocales: true,
		EnableLocaleFallback:      true,
		EnableLocaleValidation:    true,
	}

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpersWithLocale(manager, config, localeConfig)

	tests := []struct {
		name           string
		helperName     string
		args           []*pongo2.Value
		expectedResult string
		description    string
	}{
		{
			name:       "url_locale with English locale",
			helperName: "url_locale",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("about"),
				pongo2.AsValue("en"),
			},
			expectedResult: "https://example.com/en/about-us",
			description:    "Should generate URL for hierarchical en locale group",
		},
		{
			name:       "url_locale with Spanish locale",
			helperName: "url_locale",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("about"),
				pongo2.AsValue("es"),
			},
			expectedResult: "https://example.com/es/acerca-de",
			description:    "Should generate URL for hierarchical es locale group",
		},
		{
			name:       "url_locale with French locale",
			helperName: "url_locale",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("contact"),
				pongo2.AsValue("fr"),
			},
			expectedResult: "https://example.com/fr/contact",
			description:    "Should generate URL for hierarchical fr locale group",
		},
		{
			name:       "url_locale with parameters",
			helperName: "url_locale",
			args: []*pongo2.Value{
				pongo2.AsValue("frontend"),
				pongo2.AsValue("about"),
				pongo2.AsValue("es"),
				pongo2.AsValue(map[string]any{"section": "team"}),
			},
			expectedResult: "https://example.com/es/acerca-de",
			description:    "Should handle parameters in hierarchical locale groups",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			helperFunc, exists := helpers[tt.helperName]
			if !exists {
				t.Fatalf("Helper '%s' not found", tt.helperName)
			}

			templateHelper, ok := helperFunc.(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
			if !ok {
				t.Fatalf("Helper '%s' has wrong signature", tt.helperName)
			}

			result, err := templateHelper(tt.args...)
			if err != nil {
				t.Fatalf("Helper '%s' returned error: %v", tt.helperName, err)
			}

			actualResult := result.String()
			if actualResult != tt.expectedResult {
				t.Errorf("Expected '%s', got '%s'. %s", tt.expectedResult, actualResult, tt.description)
			}
		})
	}
}

// TestLanguageSwitcherImplementation tests language switcher implementation
func TestLanguageSwitcherImplementation(t *testing.T) {
	// Set up URLKit with multiple locale groups
	manager := NewRouteManager()

	// Register hierarchical locale groups for language switcher
	locales := []string{"en", "es", "fr", "de"}
	for _, locale := range locales {
		groupName := "frontend." + locale
		routes := map[string]string{
			"home":     "/" + locale + "/",
			"products": "/" + locale + "/products/:category?",
			"about":    "/" + locale + "/about",
		}
		manager.RegisterGroup(groupName, "https://example.com", routes)
	}

	// Set up locale configuration
	localeConfig := &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          locales,
		DetectionStrategies:       []LocaleDetectionStrategy{LocaleFromContext}, // Add detection strategy
		EnableHierarchicalLocales: true,
		EnableLocaleFallback:      true,
		EnableLocaleValidation:    true,
	}

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpersWithLocale(manager, config, localeConfig)

	t.Run("url_all_locales for language switcher", func(t *testing.T) {
		helper := helpers["url_all_locales"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Generate URLs for all locales for the products page
		result, err := helper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("products"),
			pongo2.AsValue(map[string]any{"category": "electronics"}),
		)
		if err != nil {
			t.Fatalf("url_all_locales returned error: %v", err)
		}

		// Convert result to slice
		localeInfos, ok := result.Interface().([]LocaleInfo)
		if !ok {
			t.Fatalf("Expected []LocaleInfo, got %T", result.Interface())
		}

		// Verify we have URLs for all locales
		if len(localeInfos) != len(locales) {
			t.Errorf("Expected %d locale URLs, got %d", len(locales), len(localeInfos))
		}

		// Verify each locale has correct URL
		expectedURLs := map[string]string{
			"en": "https://example.com/en/products/electronics",
			"es": "https://example.com/es/products/electronics",
			"fr": "https://example.com/fr/products/electronics",
			"de": "https://example.com/de/products/electronics",
		}

		for _, localeInfo := range localeInfos {
			expectedURL, exists := expectedURLs[localeInfo.Locale]
			if !exists {
				t.Errorf("Unexpected locale in results: %s", localeInfo.Locale)
				continue
			}

			if localeInfo.URL != expectedURL {
				t.Errorf("For locale %s, expected URL '%s', got '%s'", localeInfo.Locale, expectedURL, localeInfo.URL)
			}
		}
	})

	t.Run("url_all_locales with query parameters", func(t *testing.T) {
		helper := helpers["url_all_locales"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Generate URLs with query parameters
		result, err := helper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("products"),
			pongo2.AsValue(map[string]any{"category": "books"}),
			pongo2.AsValue(map[string]any{"sort": "price", "page": "2"}),
		)
		if err != nil {
			t.Fatalf("url_all_locales with query returned error: %v", err)
		}

		localeInfos, ok := result.Interface().([]LocaleInfo)
		if !ok {
			t.Fatalf("Expected []LocaleInfo, got %T", result.Interface())
		}

		// Verify URLs contain query parameters
		for _, localeInfo := range localeInfos {
			expectedPrefix := "https://example.com/" + localeInfo.Locale + "/products/books"
			if !contains(localeInfo.URL, expectedPrefix) {
				t.Errorf("URL '%s' doesn't contain expected prefix '%s'", localeInfo.URL, expectedPrefix)
			}

			// Verify query parameters are present
			if !contains(localeInfo.URL, "sort=price") || !contains(localeInfo.URL, "page=2") {
				t.Errorf("URL '%s' missing expected query parameters", localeInfo.URL)
			}
		}
	})

	t.Run("current_locale helper", func(t *testing.T) {
		helper := helpers["current_locale"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test with context containing locale
		context := map[string]any{
			"locale": "es",
		}

		result, err := helper(pongo2.AsValue(context))
		if err != nil {
			t.Fatalf("current_locale returned error: %v", err)
		}

		currentLocale := result.String()
		if currentLocale != "es" {
			t.Errorf("Expected current locale 'es', got '%s'", currentLocale)
		}
	})
}

// TestSEOHrefLangGeneration tests SEO hreflang tag generation
func TestSEOHrefLangGeneration(t *testing.T) {
	// Set up URLKit with SEO friendly locale structure
	manager := NewRouteManager()

	// Register locale groups with SEO friendly URLs
	seoLocales := map[string]map[string]string{
		"en": {
			"home":     "/",
			"products": "/products/:slug",
			"blog":     "/blog/:post_id",
		},
		"es": {
			"home":     "/es/",
			"products": "/es/productos/:slug",
			"blog":     "/es/blog/:post_id",
		},
		"fr": {
			"home":     "/fr/",
			"products": "/fr/produits/:slug",
			"blog":     "/fr/blog/:post_id",
		},
		"de": {
			"home":     "/de/",
			"products": "/de/produkte/:slug",
			"blog":     "/de/blog/:post_id",
		},
	}

	for locale, routes := range seoLocales {
		groupName := "frontend." + locale
		manager.RegisterGroup(groupName, "https://example.com", routes)
	}

	// Set up locale configuration for SEO
	localeConfig := &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          []string{"en", "es", "fr", "de"},
		EnableHierarchicalLocales: true,
		EnableLocaleFallback:      true,
		EnableLocaleValidation:    true,
	}

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpersWithLocale(manager, config, localeConfig)

	t.Run("hreflang URLs for blog post", func(t *testing.T) {
		helper := helpers["url_all_locales"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Generate hreflang URLs for a specific blog post
		result, err := helper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("blog"),
			pongo2.AsValue(map[string]any{"post_id": "my-awesome-post"}),
		)
		if err != nil {
			t.Fatalf("url_all_locales for hreflang returned error: %v", err)
		}

		localeInfos, ok := result.Interface().([]LocaleInfo)
		if !ok {
			t.Fatalf("Expected []LocaleInfo, got %T", result.Interface())
		}

		// Expected hreflang URLs
		expectedHrefLangs := map[string]string{
			"en": "https://example.com/blog/my-awesome-post",
			"es": "https://example.com/es/blog/my-awesome-post",
			"fr": "https://example.com/fr/blog/my-awesome-post",
			"de": "https://example.com/de/blog/my-awesome-post",
		}

		// Verify each locale has correct hreflang URL
		for _, localeInfo := range localeInfos {
			expectedURL, exists := expectedHrefLangs[localeInfo.Locale]
			if !exists {
				t.Errorf("Unexpected locale in hreflang results: %s", localeInfo.Locale)
				continue
			}

			if localeInfo.URL != expectedURL {
				t.Errorf("For hreflang %s, expected URL '%s', got '%s'", localeInfo.Locale, expectedURL, localeInfo.URL)
			}

			// Verify structure for HTML generation
			if localeInfo.Locale == "" || localeInfo.URL == "" {
				t.Errorf("LocaleInfo has empty fields: %+v", localeInfo)
			}
		}

		t.Logf("Generated %d hreflang URLs for SEO", len(localeInfos))
		for _, info := range localeInfos {
			t.Logf("  <link rel=\"alternate\" hreflang=\"%s\" href=\"%s\">", info.Locale, info.URL)
		}
	})

	t.Run("hreflang URLs for product pages", func(t *testing.T) {
		helper := helpers["url_all_locales"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Generate hreflang URLs for a product page with localized slugs
		result, err := helper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("products"),
			pongo2.AsValue(map[string]any{"slug": "gaming-laptop"}),
		)
		if err != nil {
			t.Fatalf("url_all_locales for product hreflang returned error: %v", err)
		}

		localeInfos, ok := result.Interface().([]LocaleInfo)
		if !ok {
			t.Fatalf("Expected []LocaleInfo, got %T", result.Interface())
		}

		// Verify we have the correct number of locales
		if len(localeInfos) != 4 {
			t.Errorf("Expected 4 hreflang URLs, got %d", len(localeInfos))
		}

		// Verify each URL structure for different locales
		localeURLs := make(map[string]string)
		for _, info := range localeInfos {
			localeURLs[info.Locale] = info.URL
		}

		// Check specific locale URL patterns
		if url, exists := localeURLs["en"]; exists {
			if url != "https://example.com/products/gaming-laptop" {
				t.Errorf("English URL incorrect: %s", url)
			}
		}

		if url, exists := localeURLs["es"]; exists {
			if url != "https://example.com/es/productos/gaming-laptop" {
				t.Errorf("Spanish URL incorrect: %s", url)
			}
		}

		if url, exists := localeURLs["fr"]; exists {
			if url != "https://example.com/fr/produits/gaming-laptop" {
				t.Errorf("French URL incorrect: %s", url)
			}
		}

		if url, exists := localeURLs["de"]; exists {
			if url != "https://example.com/de/produkte/gaming-laptop" {
				t.Errorf("German URL incorrect: %s", url)
			}
		}
	})

	t.Run("has_locale validation for SEO", func(t *testing.T) {
		helper := helpers["has_locale"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test locale availability for SEO content
		tests := []struct {
			group    string
			locale   string
			expected bool
		}{
			{"frontend", "en", true},
			{"frontend", "es", true},
			{"frontend", "fr", true},
			{"frontend", "de", true},
			{"frontend", "zh", false}, // Unsupported locale
		}

		for _, test := range tests {
			result, err := helper(pongo2.AsValue(test.group), pongo2.AsValue(test.locale))
			if err != nil {
				t.Fatalf("has_locale returned error for %s/%s: %v", test.group, test.locale, err)
			}

			hasLocale, ok := result.Interface().(bool)
			if !ok {
				t.Fatalf("Expected bool, got %T", result.Interface())
			}

			if hasLocale != test.expected {
				t.Errorf("For group '%s' and locale '%s', expected %v, got %v", test.group, test.locale, test.expected, hasLocale)
			}
		}
	})
}

// TestLocalizationIntegrationWithTemplateVariables tests localization with URLKit's template variables
func TestLocalizationIntegrationWithTemplateVariables(t *testing.T) {
	// Set up URLKit with template variables for localization
	manager := NewRouteManager()

	// Use template variables for locale-specific configuration
	config := &Config{
		Groups: []GroupConfig{
			{
				Name:        "frontend.en",
				BaseURL:     "https://example.com",
				URLTemplate: "{protocol}://{host}/{locale_prefix}{route_path}",
				TemplateVars: map[string]string{
					"protocol":      "https",
					"host":          "example.com",
					"locale_prefix": "",
				},
				Paths: map[string]string{
					"home":    "/",
					"about":   "/about-us",
					"contact": "/contact-us",
				},
			},
			{
				Name:        "frontend.es",
				BaseURL:     "https://example.com",
				URLTemplate: "{protocol}://{host}/{locale_prefix}{route_path}",
				TemplateVars: map[string]string{
					"protocol":      "https",
					"host":          "example.com",
					"locale_prefix": "es/",
				},
				Paths: map[string]string{
					"home":    "/",
					"about":   "/acerca-de",
					"contact": "/contacto",
				},
			},
		},
	}

	manager = NewRouteManager(config)

	// Set up localization config
	localeConfig := &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          []string{"en", "es"},
		EnableHierarchicalLocales: true,
		EnableLocaleFallback:      true,
		EnableLocaleValidation:    true,
	}

	helperConfig := DefaultTemplateHelperConfig()
	helpers := TemplateHelpersWithLocale(manager, helperConfig, localeConfig)

	t.Run("url_locale with template variables", func(t *testing.T) {
		helper := helpers["url_locale"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test English URL (no locale prefix)
		result, err := helper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("about"),
			pongo2.AsValue("en"),
		)
		if err != nil {
			t.Fatalf("url_locale for English returned error: %v", err)
		}

		enURL := result.String()
		expectedEnURL := "https://example.com//about-us/" // Template has double slash due to empty locale_prefix
		if enURL != expectedEnURL {
			t.Errorf("English URL: expected '%s', got '%s'", expectedEnURL, enURL)
		}

		// Test Spanish URL (with locale prefix)
		result, err = helper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("about"),
			pongo2.AsValue("es"),
		)
		if err != nil {
			t.Fatalf("url_locale for Spanish returned error: %v", err)
		}

		esURL := result.String()
		expectedEsURL := "https://example.com/es//acerca-de/" // Template has double slash pattern
		if esURL != expectedEsURL {
			t.Errorf("Spanish URL: expected '%s', got '%s'", expectedEsURL, esURL)
		}
	})
}

// TestLocalizationErrorHandling tests error handling in localization helpers
func TestLocalizationErrorHandling(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home": "/",
	})

	// Set up restrictive locale config for error testing
	localeConfig := &LocaleConfig{
		DefaultLocale:    "en",
		SupportedLocales: []string{"en"},
		LocaleGroups: map[string][]string{
			"frontend": {"en"}, // Only English supported for frontend
		},
		EnableLocaleFallback:   false, // Disable fallback for testing
		EnableLocaleValidation: true,
	}

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpersWithLocale(manager, config, localeConfig)

	t.Run("url_locale with unsupported locale", func(t *testing.T) {
		helper := helpers["url_locale"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		result, err := helper(
			pongo2.AsValue("frontend"),
			pongo2.AsValue("home"),
			pongo2.AsValue("fr"), // Unsupported locale
		)
		if err != nil {
			t.Fatalf("url_locale should not return pongo2.Error: %v", err)
		}

		resultStr := result.String()
		if !contains(resultStr, "#error") {
			t.Errorf("Expected error URL containing '#error', got: %s", resultStr)
		}

		if !contains(resultStr, "unsupported_locale") {
			t.Errorf("Expected error URL containing 'unsupported_locale', got: %s", resultStr)
		}
	})

	t.Run("has_locale with invalid arguments", func(t *testing.T) {
		helper := helpers["has_locale"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

		// Test with insufficient arguments
		result, err := helper(pongo2.AsValue("frontend"))
		if err != nil {
			t.Fatalf("has_locale should not return pongo2.Error: %v", err)
		}

		hasLocale, ok := result.Interface().(bool)
		if !ok {
			t.Fatalf("Expected bool, got %T", result.Interface())
		}

		if hasLocale {
			t.Errorf("Expected false for insufficient arguments, got true")
		}
	})
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				indexOf(s, substr) >= 0))
}

// Helper function to find substring index
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// TestLocalizationConfigValidation tests locale configuration validation
func TestLocalizationConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      *LocaleConfig
		expectError bool
		description string
	}{
		{
			name: "Valid configuration",
			config: &LocaleConfig{
				DefaultLocale:    "en",
				SupportedLocales: []string{"en", "es", "fr"},
				LocaleGroups: map[string][]string{
					"frontend": {"en", "es"},
				},
			},
			expectError: false,
			description: "Should accept valid configuration",
		},
		{
			name: "Empty default locale",
			config: &LocaleConfig{
				DefaultLocale:    "",
				SupportedLocales: []string{"en", "es"},
			},
			expectError: true,
			description: "Should reject empty default locale",
		},
		{
			name: "Empty supported locales",
			config: &LocaleConfig{
				DefaultLocale:    "en",
				SupportedLocales: []string{},
			},
			expectError: true,
			description: "Should reject empty supported locales",
		},
		{
			name: "Default locale not in supported",
			config: &LocaleConfig{
				DefaultLocale:    "de",
				SupportedLocales: []string{"en", "es", "fr"},
			},
			expectError: true,
			description: "Should reject default locale not in supported list",
		},
		{
			name: "Group locale not in global supported",
			config: &LocaleConfig{
				DefaultLocale:    "en",
				SupportedLocales: []string{"en", "es"},
				LocaleGroups: map[string][]string{
					"frontend": {"en", "fr"}, // 'fr' not in global supported
				},
			},
			expectError: true,
			description: "Should reject group locale not in global supported locales",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ValidateLocaleConfig()

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none. %s", tt.description)
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v. %s", err, tt.description)
			}
		})
	}
}

// TestLocalizationIntegrationBenchmarks benchmarks localization helper performance
func BenchmarkLocalizationHelpers(b *testing.B) {
	// Set up URLKit with multiple locale groups
	manager := NewRouteManager()
	locales := []string{"en", "es", "fr", "de", "it", "pt", "ru", "zh", "ja", "ko"}

	for _, locale := range locales {
		groupName := "frontend." + locale
		routes := map[string]string{
			"home":     "/" + locale + "/",
			"products": "/" + locale + "/products/:id",
			"about":    "/" + locale + "/about",
			"contact":  "/" + locale + "/contact",
		}
		manager.RegisterGroup(groupName, "https://example.com", routes)
	}

	localeConfig := &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          locales,
		EnableHierarchicalLocales: true,
		EnableLocaleFallback:      true,
		EnableLocaleValidation:    true,
	}

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpersWithLocale(manager, config, localeConfig)

	b.Run("url_locale", func(b *testing.B) {
		helper := helpers["url_locale"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
		args := []*pongo2.Value{
			pongo2.AsValue("frontend"),
			pongo2.AsValue("products"),
			pongo2.AsValue("es"),
			pongo2.AsValue(map[string]any{"id": "123"}),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := helper(args...)
			if err != nil {
				b.Fatalf("Helper error: %v", err)
			}
		}
	})

	b.Run("url_all_locales", func(b *testing.B) {
		helper := helpers["url_all_locales"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))
		args := []*pongo2.Value{
			pongo2.AsValue("frontend"),
			pongo2.AsValue("products"),
			pongo2.AsValue(map[string]any{"id": "456"}),
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := helper(args...)
			if err != nil {
				b.Fatalf("Helper error: %v", err)
			}
		}
	})

	b.Run("locale_detection", func(b *testing.B) {
		context := map[string]any{
			"locale":          "es",
			"url_path":        "/fr/products",
			"accept_language": "de,en;q=0.9",
			"cookie_locale":   "it",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = localeConfig.detectLocale(context, "frontend")
		}
	})
}
