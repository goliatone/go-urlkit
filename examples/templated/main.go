package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/goliatone/go-urlkit"
)

// TemplatedURLExample demonstrates the template-based URL generation features
// of the urlkit library. This includes:
//
// 1. Loading template configurations from JSON
// 2. Programmatic template and variable management
// 3. URL building using template rendering
// 4. Variable inheritance and overriding in nested groups
// 5. Comparison with traditional path concatenation
func main() {
	fmt.Println("=== Templated URL Generation Example ===")
	fmt.Println("This example demonstrates template-based URL generation with variable inheritance")
	fmt.Println()

	// Example 1: Load templated configuration from JSON
	fmt.Println("1. Loading templated configuration from JSON...")
	manager, err := loadTemplatedConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	fmt.Println("   ✓ Configuration loaded successfully")
	fmt.Println()

	// Example 2: Demonstrate i18n URL generation
	fmt.Println("2. Demonstrating internationalized URLs...")
	demonstrateI18nURLs(manager)
	fmt.Println()

	// Example 3: Demonstrate API versioning with templates
	fmt.Println("3. Demonstrating API versioning with templates...")
	demonstrateAPIVersioning(manager)
	fmt.Println()

	// Example 4: Demonstrate CDN region switching
	fmt.Println("4. Demonstrating CDN region switching...")
	demonstrateCDNRegions(manager)
	fmt.Println()

	// Example 5: Programmatic template management
	fmt.Println("5. Demonstrating programmatic template management...")
	demonstrateProgrammaticUsage()
	fmt.Println()

	// Example 6: Template variable inheritance
	fmt.Println("6. Demonstrating template variable inheritance and overrides...")
	demonstrateVariableInheritance()
	fmt.Println()

	fmt.Println("=== End of Templated URL Example ===")
}

// loadTemplatedConfig loads the templated URL configuration from JSON file
func loadTemplatedConfig() (*urlkit.RouteManager, error) {
	// In a real application, you might load this from a file
	configPath := findTemplatedConfigPath()
	if configPath == "" {
		fmt.Printf("   ⚠️  Config file not found, using programmatic configuration\n")
		return createProgrammaticConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config, err := parseTemplatedConfig(data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config JSON: %w", err)
	}

	manager, err := urlkit.NewRouteManagerFromConfig(config)
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func findTemplatedConfigPath() string {
	candidates := []string{
		"examples/templated/templated_urls.json",
		"examples/templated_urls.json",
		"templated_urls.json",
	}

	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			fmt.Printf("   • Loading config from %s\n", candidate)
			return candidate
		}
	}

	return ""
}

func parseTemplatedConfig(data []byte) (urlkit.Config, error) {
	var config urlkit.Config
	if err := json.Unmarshal(data, &config); err == nil && len(config.Groups) > 0 {
		return config, nil
	}

	var wrapped struct {
		URLManager urlkit.Config `json:"url_manager"`
	}
	if err := json.Unmarshal(data, &wrapped); err != nil {
		return urlkit.Config{}, err
	}
	if len(wrapped.URLManager.Groups) == 0 {
		return urlkit.Config{}, fmt.Errorf("no groups found in config")
	}

	return wrapped.URLManager, nil
}

// createProgrammaticConfig creates a configuration programmatically as fallback
func createProgrammaticConfig() *urlkit.RouteManager {
	manager := urlkit.NewRouteManager()

	// Create i18n frontend group with template
	manager.RegisterGroup("i18n_frontend", "https://example.com", map[string]string{
		"home":     "/",
		"about":    "/about",
		"contact":  "/contact",
		"products": "/products/:category",
	})
	frontend := manager.Group("i18n_frontend")
	frontend.SetURLTemplate("https://{subdomain}.{domain}/{locale}{route_path}")
	frontend.SetTemplateVar("subdomain", "www")
	frontend.SetTemplateVar("domain", "example.com")

	// Create English localization group
	en := frontend.RegisterGroup("en", "", map[string]string{
		"about":    "/about-us",
		"contact":  "/contact",
		"products": "/products/:category",
	})
	en.SetTemplateVar("locale", "en")

	// Create Spanish localization group
	es := frontend.RegisterGroup("es", "", map[string]string{
		"about":    "/acerca-de",
		"contact":  "/contacto",
		"products": "/productos/:category",
	})
	es.SetTemplateVar("locale", "es")

	// Create API group with version template
	manager.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
		"health": "/health",
	})
	api := manager.Group("api")
	api.SetURLTemplate("{protocol}://{api_host}/{version}{route_path}")
	api.SetTemplateVar("protocol", "https")
	api.SetTemplateVar("api_host", "api.example.com")

	v1 := api.RegisterGroup("v1", "", map[string]string{
		"users": "/users/:id",
		"posts": "/posts",
	})
	v1.SetTemplateVar("version", "v1")

	v2 := api.RegisterGroup("v2", "", map[string]string{
		"users":         "/users/:id",
		"organizations": "/orgs/:orgId",
	})
	v2.SetTemplateVar("version", "v2")

	// Create CDN group with region template
	manager.RegisterGroup("cdn", "https://cdn.example.com", map[string]string{
		"assets": "/assets/:version/:file",
	})
	cdn := manager.Group("cdn")
	cdn.SetURLTemplate("{protocol}://{cdn_region}.{domain}{route_path}")
	cdn.SetTemplateVar("protocol", "https")
	cdn.SetTemplateVar("cdn_region", "us-west-2")
	cdn.SetTemplateVar("domain", "cdn.example.com")

	eu := cdn.RegisterGroup("eu", "", map[string]string{
		"assets": "/assets/:version/:file",
	})
	eu.SetTemplateVar("cdn_region", "eu-west-1")

	asia := cdn.RegisterGroup("asia", "", map[string]string{
		"assets": "/assets/:version/:file",
	})
	asia.SetTemplateVar("cdn_region", "ap-southeast-1")

	return manager
}

// demonstrateI18nURLs shows how templates enable flexible i18n URL structures
func demonstrateI18nURLs(manager *urlkit.RouteManager) {
	fmt.Println("   Building internationalized URLs using templates:")

	// English URLs
	enGroup := getGroup(manager, "i18n_frontend.en")
	if enGroup != nil {
		aboutEN, _ := enGroup.Builder("about").Build()
		productsEN, _ := enGroup.Builder("products").WithParam("category", "electronics").Build()

		fmt.Printf("   • English About:    %s\n", aboutEN)
		fmt.Printf("   • English Products: %s\n", productsEN)
	}

	// Spanish URLs
	esGroup := getGroup(manager, "i18n_frontend.es")
	if esGroup != nil {
		aboutES, _ := esGroup.Builder("about").Build()
		productsES, _ := esGroup.Builder("products").WithParam("category", "electronics").Build()

		fmt.Printf("   • Spanish About:    %s\n", aboutES)
		fmt.Printf("   • Spanish Products: %s\n", productsES)
	}

	fmt.Println("   ✓ Same template generates different localized URLs")
}

// demonstrateAPIVersioning shows API versioning using templates
func demonstrateAPIVersioning(manager *urlkit.RouteManager) {
	fmt.Println("   Building API URLs with version templating:")

	// V1 API URLs
	v1Group := getGroup(manager, "api.v1")
	if v1Group != nil {
		usersV1, _ := v1Group.Builder("users").WithParam("id", "123").Build()
		postsV1, _ := v1Group.Builder("posts").Build()

		fmt.Printf("   • V1 Users API: %s\n", usersV1)
		fmt.Printf("   • V1 Posts API: %s\n", postsV1)
	}

	// V2 API URLs
	v2Group := getGroup(manager, "api.v2")
	if v2Group != nil {
		usersV2, _ := v2Group.Builder("users").WithParam("id", "123").Build()
		orgsV2, _ := v2Group.Builder("organizations").WithParam("orgId", "acme").Build()

		fmt.Printf("   • V2 Users API:  %s\n", usersV2)
		fmt.Printf("   • V2 Orgs API:   %s\n", orgsV2)
	}

	fmt.Println("   ✓ Template enables consistent versioning across API endpoints")
}

// demonstrateCDNRegions shows CDN region switching using templates
func demonstrateCDNRegions(manager *urlkit.RouteManager) {
	fmt.Println("   Building CDN URLs with regional templates:")

	// Default CDN (US West)
	cdnGroup := getGroup(manager, "cdn")
	if cdnGroup != nil {
		assetsUS, _ := cdnGroup.Builder("assets").WithParam("version", "v1.2.3").WithParam("file", "app.js").Build()
		fmt.Printf("   • US CDN Assets: %s\n", assetsUS)
	}

	// EU CDN
	euCDN := getGroup(manager, "cdn.eu")
	if euCDN != nil {
		assetsEU, _ := euCDN.Builder("assets").WithParam("version", "v1.2.3").WithParam("file", "app.js").Build()
		fmt.Printf("   • EU CDN Assets: %s\n", assetsEU)
	}

	// Asia CDN
	asiaCDN := getGroup(manager, "cdn.asia")
	if asiaCDN != nil {
		assetsAsia, _ := asiaCDN.Builder("assets").WithParam("version", "v1.2.3").WithParam("file", "app.js").Build()
		fmt.Printf("   • Asia CDN Assets: %s\n", assetsAsia)
	}

	fmt.Println("   ✓ Same template generates region-specific CDN URLs")
}

func getGroup(manager *urlkit.RouteManager, path string) *urlkit.Group {
	group, err := manager.GetGroup(path)
	if err != nil {
		fmt.Printf("   ⚠️  Group %q not found: %v\n", path, err)
		return nil
	}
	return group
}

// demonstrateProgrammaticUsage shows programmatic template management
func demonstrateProgrammaticUsage() {
	fmt.Println("   Creating and configuring templates programmatically:")

	manager := urlkit.NewRouteManager()

	// Create a multi-tenant application group
	manager.RegisterGroup("app", "https://app.example.com", map[string]string{
		"dashboard": "/dashboard",
		"settings":  "/settings/:section",
		"profile":   "/profile/:userId",
	})
	app := manager.Group("app")

	// Set up tenant-aware URL template
	app.SetURLTemplate("https://{tenant}.{base_domain}{route_path}")
	app.SetTemplateVar("base_domain", "myapp.com")

	fmt.Println("   • Created app group with tenant template")
	fmt.Printf("   • Template: %s\n", "https://{tenant}.{base_domain}{route_path}")

	// Create tenant-specific groups
	tenants := []struct {
		name   string
		tenant string
	}{
		{"acme", "acme"},
		{"widgets", "widgets-co"},
		{"startup", "startup"},
	}

	for _, tenant := range tenants {
		tenantGroup := app.RegisterGroup(tenant.name, "", map[string]string{
			"dashboard": "/dashboard",
			"settings":  "/settings/:section",
		})

		tenantGroup.SetTemplateVar("tenant", tenant.tenant)

		// Build example URLs
		dashboard, _ := tenantGroup.Builder("dashboard").Build()
		settings, _ := tenantGroup.Builder("settings").WithParam("section", "billing").Build()

		fmt.Printf("   • %s Dashboard: %s\n", tenant.name, dashboard)
		fmt.Printf("   • %s Settings:  %s\n", tenant.name, settings)
	}

	fmt.Println("   ✓ Programmatic template management working")
}

// demonstrateVariableInheritance shows how template variables are inherited and overridden
func demonstrateVariableInheritance() {
	fmt.Println("   Demonstrating template variable inheritance:")

	manager := urlkit.NewRouteManager()

	// Root group with base template and variables
	manager.RegisterGroup("platform", "https://platform.example.com", map[string]string{
		"home": "/",
		"docs": "/docs/:page",
	})
	root := manager.Group("platform")
	root.SetURLTemplate("{protocol}://{service}.{domain}/{environment}/{region}{route_path}")
	root.SetTemplateVar("protocol", "https")
	root.SetTemplateVar("domain", "example.com")
	root.SetTemplateVar("service", "platform")

	fmt.Println("   • Root template variables: protocol, domain, service")

	// Production environment group
	prod := root.RegisterGroup("prod", "", map[string]string{
		"api":    "/api/:version",
		"health": "/health",
	})
	prod.SetTemplateVar("environment", "prod")
	prod.SetTemplateVar("region", "us-east-1")

	fmt.Println("   • Production adds: environment, region")

	// Staging overrides some variables
	staging := root.RegisterGroup("staging", "", map[string]string{
		"api":    "/api/:version",
		"health": "/health",
	})
	staging.SetTemplateVar("environment", "staging")
	staging.SetTemplateVar("region", "us-west-2")
	staging.SetTemplateVar("service", "staging-platform") // Override parent

	fmt.Println("   • Staging adds environment, region and overrides service")

	// Build URLs to show inheritance
	prodAPI, _ := prod.Builder("api").WithParam("version", "v1").Build()
	stagingAPI, _ := staging.Builder("api").WithParam("version", "v1").Build()

	fmt.Printf("   • Production API:  %s\n", prodAPI)
	fmt.Printf("   • Staging API:     %s\n", stagingAPI)

	fmt.Println("   ✓ Child groups inherit parent variables and can override them")
}
