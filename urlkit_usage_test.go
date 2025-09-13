package urlkit_test

import (
	"strings"
	"testing"

	"github.com/goliatone/go-urlkit"
)

func TestI18nURLStructureFromSpecification(t *testing.T) {
	// Test the exact i18n URL structure example from the feature specification

	// Create the configuration as shown in the specification
	config := urlkit.Config{
		Groups: []urlkit.GroupConfig{
			{
				Name:        "frontend",
				BaseURL:     "https://www.example.com",
				URLTemplate: "{protocol}://{host}/{locale}/{section}{route_path}",
				TemplateVars: map[string]string{
					"protocol": "https",
					"host":     "www.example.com",
				},
				Paths: map[string]string{},
				Groups: []urlkit.GroupConfig{
					{
						Name: "en",
						TemplateVars: map[string]string{
							"locale": "en-US",
						},
						Groups: []urlkit.GroupConfig{
							{
								Name: "company",
								TemplateVars: map[string]string{
									"section": "about-us",
								},
								Paths: map[string]string{
									"about": "/about",
									"team":  "/team",
								},
							},
							{
								Name: "products",
								TemplateVars: map[string]string{
									"section": "products",
								},
								Paths: map[string]string{
									"catalog": "/catalog",
									"details": "/details/:id",
								},
							},
						},
					},
					{
						Name: "es",
						TemplateVars: map[string]string{
							"locale": "es-ES",
						},
						Groups: []urlkit.GroupConfig{
							{
								Name: "company",
								TemplateVars: map[string]string{
									"section": "nuestra-empresa",
								},
								Paths: map[string]string{
									"about": "/acerca-de",
									"team":  "/equipo",
								},
							},
							{
								Name: "products",
								TemplateVars: map[string]string{
									"section": "productos",
								},
								Paths: map[string]string{
									"catalog": "/catalogo",
									"details": "/detalles/:id",
								},
							},
						},
					},
				},
			},
		},
	}

	manager := urlkit.NewRouteManager(&config)

	// Test English company about page
	enAboutURL, err := manager.Group("frontend").Group("en").Group("company").Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build English about URL: %v", err)
	}
	expected := "https://www.example.com/en-US/about-us/about/"
	if enAboutURL != expected {
		t.Errorf("Expected %q, got %q", expected, enAboutURL)
	}

	// Test Spanish company about page (as shown in specification)
	esAboutURL, err := manager.Group("frontend").Group("es").Group("company").Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish about URL: %v", err)
	}
	expected = "https://www.example.com/es-ES/nuestra-empresa/acerca-de/"
	if esAboutURL != expected {
		t.Errorf("Expected %q, got %q", expected, esAboutURL)
	}

	// Test English products with parameters
	enProductURL, err := manager.Group("frontend").Group("en").Group("products").Builder("details").
		WithParam("id", "laptop-123").
		WithQuery("variant", "pro").
		Build()
	if err != nil {
		t.Fatalf("Failed to build English product URL: %v", err)
	}
	expected = "https://www.example.com/en-US/products/details/laptop-123/?variant=pro"
	if enProductURL != expected {
		t.Errorf("Expected %q, got %q", expected, enProductURL)
	}

	// Test Spanish products
	esProductURL, err := manager.Group("frontend").Group("es").Group("products").Builder("catalog").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish catalog URL: %v", err)
	}
	expected = "https://www.example.com/es-ES/productos/catalogo/"
	if esProductURL != expected {
		t.Errorf("Expected %q, got %q", expected, esProductURL)
	}

	// Test team pages
	enTeamURL, err := manager.Group("frontend").Group("en").Group("company").Builder("team").Build()
	if err != nil {
		t.Fatalf("Failed to build English team URL: %v", err)
	}
	expected = "https://www.example.com/en-US/about-us/team/"
	if enTeamURL != expected {
		t.Errorf("Expected %q, got %q", expected, enTeamURL)
	}

	esTeamURL, err := manager.Group("frontend").Group("es").Group("company").Builder("team").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish team URL: %v", err)
	}
	expected = "https://www.example.com/es-ES/nuestra-empresa/equipo/"
	if esTeamURL != expected {
		t.Errorf("Expected %q, got %q", expected, esTeamURL)
	}
}

func TestURLSegmentReorderingThroughTemplateChanges(t *testing.T) {
	// Test URL segment reordering by changing only the template, as shown in specification

	rm := urlkit.NewRouteManager()

	// Set up the structure
	rm.RegisterGroup("frontend", "https://www.example.com", map[string]string{})

	frontend := rm.Group("frontend")
	frontend.SetTemplateVar("protocol", "https")
	frontend.SetTemplateVar("host", "www.example.com")

	// Create English group
	en := frontend.RegisterGroup("en", "", map[string]string{})
	en.SetTemplateVar("locale", "en-US")

	// Create company section
	company := en.RegisterGroup("company", "", map[string]string{
		"about": "/about",
	})
	company.SetTemplateVar("section", "about-us")

	// Create Spanish group
	es := frontend.RegisterGroup("es", "", map[string]string{})
	es.SetTemplateVar("locale", "es-ES")

	// Create Spanish company section
	companyEs := es.RegisterGroup("company", "", map[string]string{
		"about": "/acerca-de",
	})
	companyEs.SetTemplateVar("section", "nuestra-empresa")

	// Test 1: Original template order - locale/section
	frontend.SetURLTemplate("{protocol}://{host}/{locale}/{section}{route_path}")

	enAboutURL, err := company.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build English about URL: %v", err)
	}
	expected := "https://www.example.com/en-US/about-us/about/"
	if enAboutURL != expected {
		t.Errorf("Original template - Expected %q, got %q", expected, enAboutURL)
	}

	esAboutURL, err := companyEs.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build Spanish about URL: %v", err)
	}
	expected = "https://www.example.com/es-ES/nuestra-empresa/acerca-de/"
	if esAboutURL != expected {
		t.Errorf("Original template - Expected %q, got %q", expected, esAboutURL)
	}

	// Test 2: Reordered template - section/locale (as shown in specification)
	frontend.SetURLTemplate("{protocol}://{host}/{section}/{locale}{route_path}")

	enAboutURLReordered, err := company.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build reordered English about URL: %v", err)
	}
	expected = "https://www.example.com/about-us/en-US/about/"
	if enAboutURLReordered != expected {
		t.Errorf("Reordered template - Expected %q, got %q", expected, enAboutURLReordered)
	}

	esAboutURLReordered, err := companyEs.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build reordered Spanish about URL: %v", err)
	}
	expected = "https://www.example.com/nuestra-empresa/es-ES/acerca-de/"
	if esAboutURLReordered != expected {
		t.Errorf("Reordered template - Expected %q, got %q", expected, esAboutURLReordered)
	}

	// Test 3: Different template structure - no protocol variable, different order
	frontend.SetURLTemplate("{host}/{section}_{locale}{route_path}")

	enAboutURLAlt, err := company.Builder("about").Build()
	if err != nil {
		t.Fatalf("Failed to build alternative English about URL: %v", err)
	}
	expected = "www.example.com/about-us_en-US/about/"
	if enAboutURLAlt != expected {
		t.Errorf("Alternative template - Expected %q, got %q", expected, enAboutURLAlt)
	}

	// Test 4: Add query parameters to make sure they still work
	enAboutWithQuery, err := company.Builder("about").
		WithQuery("ref", "homepage").
		WithQuery("utm_source", "nav").
		Build()
	if err != nil {
		t.Fatalf("Failed to build URL with query: %v", err)
	}
	// Check URL base and path first
	expectedBase := "www.example.com/about-us_en-US/about/?"
	if !strings.HasPrefix(enAboutWithQuery, expectedBase) {
		t.Errorf("Template with query - Expected URL to start with %q, got %q", expectedBase, enAboutWithQuery)
	}
	// Check that both query parameters are present (order may vary due to map iteration)
	if !strings.Contains(enAboutWithQuery, "ref=homepage") {
		t.Errorf("Template with query - Expected URL to contain 'ref=homepage', got %q", enAboutWithQuery)
	}
	if !strings.Contains(enAboutWithQuery, "utm_source=nav") {
		t.Errorf("Template with query - Expected URL to contain 'utm_source=nav', got %q", enAboutWithQuery)
	}
}

func TestProtocolHostPathVariableCombinations(t *testing.T) {
	// Test various combinations of protocol, host, and path variables

	rm := urlkit.NewRouteManager()

	// Test 1: Full URL template with all components
	rm.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users":  "/users/:id",
		"status": "/status",
	})

	api := rm.Group("api")
	api.SetURLTemplate("{protocol}://{subdomain}.{domain}/{env}/{version}{route_path}")
	api.SetTemplateVar("protocol", "https")
	api.SetTemplateVar("subdomain", "api")
	api.SetTemplateVar("domain", "example.com")
	api.SetTemplateVar("env", "prod")
	api.SetTemplateVar("version", "v1")

	userURL, err := api.Builder("users").
		WithParam("id", "123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build user URL: %v", err)
	}
	expected := "https://api.example.com/prod/v1/users/123/"
	if userURL != expected {
		t.Errorf("Full template - Expected %q, got %q", expected, userURL)
	}

	// Test 2: Environment override in child group
	staging := api.RegisterGroup("staging", "", map[string]string{
		"debug": "/debug",
		"logs":  "/logs/:service",
	})
	staging.SetTemplateVar("env", "staging")
	staging.SetTemplateVar("subdomain", "staging-api")

	debugURL, err := staging.Builder("debug").Build()
	if err != nil {
		t.Fatalf("Failed to build debug URL: %v", err)
	}
	expected = "https://staging-api.example.com/staging/v1/debug/"
	if debugURL != expected {
		t.Errorf("Environment override - Expected %q, got %q", expected, debugURL)
	}

	// Test 3: Protocol switching
	insecure := api.RegisterGroup("insecure", "", map[string]string{
		"health": "/health",
	})
	insecure.SetTemplateVar("protocol", "http")
	insecure.SetTemplateVar("subdomain", "internal")

	healthURL, err := insecure.Builder("health").Build()
	if err != nil {
		t.Fatalf("Failed to build health URL: %v", err)
	}
	expected = "http://internal.example.com/prod/v1/health/"
	if healthURL != expected {
		t.Errorf("Protocol switch - Expected %q, got %q", expected, healthURL)
	}

	// Test 4: Custom port and path prefix
	rm.RegisterGroup("custom", "https://custom.example.com", map[string]string{
		"webhooks": "/webhooks/:event",
	})

	custom := rm.Group("custom")
	custom.SetURLTemplate("{protocol}://{host}:{port}/{prefix}{route_path}")
	custom.SetTemplateVar("protocol", "https")
	custom.SetTemplateVar("host", "webhooks.example.com")
	custom.SetTemplateVar("port", "8443")
	custom.SetTemplateVar("prefix", "/api/v1")

	webhookURL, err := custom.Builder("webhooks").
		WithParam("event", "user.created").
		WithQuery("signature", "abc123").
		Build()
	if err != nil {
		t.Fatalf("Failed to build webhook URL: %v", err)
	}
	expected = "https://webhooks.example.com:8443//api/v1/webhooks/user.created/?signature=abc123"
	if webhookURL != expected {
		t.Errorf("Custom port and prefix - Expected %q, got %q", expected, webhookURL)
	}

	// Test 5: Dynamic host based on region
	rm.RegisterGroup("cdn", "https://cdn.example.com", map[string]string{
		"assets": "/assets/:file",
		"images": "/images/:image",
	})

	cdn := rm.Group("cdn")
	cdn.SetURLTemplate("{protocol}://{region}.{service}.{domain}{route_path}")
	cdn.SetTemplateVar("protocol", "https")
	cdn.SetTemplateVar("service", "cdn")
	cdn.SetTemplateVar("domain", "example.com")
	cdn.SetTemplateVar("region", "us-west")

	// US West assets
	assetURL, err := cdn.Builder("assets").
		WithParam("file", "app.js").
		Build()
	if err != nil {
		t.Fatalf("Failed to build asset URL: %v", err)
	}
	expected = "https://us-west.cdn.example.com/assets/app.js/"
	if assetURL != expected {
		t.Errorf("Regional CDN - Expected %q, got %q", expected, assetURL)
	}

	// Change region dynamically
	cdn.SetTemplateVar("region", "eu-central")

	assetEUURL, err := cdn.Builder("assets").
		WithParam("file", "styles.css").
		Build()
	if err != nil {
		t.Fatalf("Failed to build EU asset URL: %v", err)
	}
	expected = "https://eu-central.cdn.example.com/assets/styles.css/"
	if assetEUURL != expected {
		t.Errorf("EU Regional CDN - Expected %q, got %q", expected, assetEUURL)
	}

	// Test 6: Base URL override
	rm.RegisterGroup("override", "https://original.example.com", map[string]string{
		"data": "/data/:id",
	})

	override := rm.Group("override")
	override.SetURLTemplate("{base_url}/custom{route_path}")

	// Should use base_url from the original registration, not template vars
	dataURL, err := override.Builder("data").
		WithParam("id", "test").
		Build()
	if err != nil {
		t.Fatalf("Failed to build data URL: %v", err)
	}
	expected = "https://original.example.com/custom/data/test/"
	if dataURL != expected {
		t.Errorf("Base URL usage - Expected %q, got %q", expected, dataURL)
	}
}
