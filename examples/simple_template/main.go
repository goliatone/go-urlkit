package main

import (
	"fmt"

	"github.com/goliatone/go-urlkit"
)

// SimpleTemplateExample demonstrates the core templating concepts in a minimal example.
// This is perfect for getting started with template-based URL generation.
func main() {
	fmt.Println("=== Simple Template Example ===")
	fmt.Println()

	// Create a route manager
	manager := urlkit.NewRouteManager()

	// Example 1: Basic template usage
	fmt.Println("1. Basic Template Usage")
	fmt.Println("   Creating a group with URL template...")

	// Create a group with a simple template
	manager.RegisterGroup("blog", "https://example.com", map[string]string{
		"home":  "/",
		"post":  "/post/:slug",
		"about": "/about",
	})
	blog := manager.Group("blog")

	// Set up template - this replaces the traditional baseURL + path concatenation
	blog.SetURLTemplate("https://{subdomain}.{domain}{route_path}")
	blog.SetTemplateVar("subdomain", "blog")
	blog.SetTemplateVar("domain", "example.com")

	// Build URLs using the template
	home, _ := blog.Builder("home").Build()
	post, _ := blog.Builder("post").WithParam("slug", "hello-world").Build()
	about, _ := blog.Builder("about").Build()

	fmt.Printf("   • Home:  %s\n", home)
	fmt.Printf("   • Post:  %s\n", post)
	fmt.Printf("   • About: %s\n", about)
	fmt.Println()

	// Example 2: Environment-based templates
	fmt.Println("2. Environment-based Templates")
	fmt.Println("   Creating environment-specific child groups...")

	// Create development environment
	dev := blog.RegisterGroup("dev", "", map[string]string{
		"home": "/",
		"post": "/post/:slug",
	})
	dev.SetTemplateVar("subdomain", "dev-blog") // Override parent's subdomain

	// Create staging environment
	staging := blog.RegisterGroup("staging", "", map[string]string{
		"home": "/",
		"post": "/post/:slug",
	})
	staging.SetTemplateVar("subdomain", "staging-blog") // Override parent's subdomain

	// Build URLs for different environments
	devHome, _ := dev.Builder("home").Build()
	stagingHome, _ := staging.Builder("home").Build()
	devPost, _ := dev.Builder("post").WithParam("slug", "test-post").Build()

	fmt.Printf("   • Production: %s\n", home)
	fmt.Printf("   • Development: %s\n", devHome)
	fmt.Printf("   • Staging: %s\n", stagingHome)
	fmt.Printf("   • Dev Post: %s\n", devPost)
	fmt.Println()

	// Example 3: API versioning with templates
	fmt.Println("3. API Versioning with Templates")

	manager.RegisterGroup("api", "https://api.example.com", map[string]string{
		"status": "/status",
		"health": "/health",
	})
	api := manager.Group("api")

	// Template that includes version in the URL
	api.SetURLTemplate("https://api.{domain}/{version}{route_path}")
	api.SetTemplateVar("domain", "example.com")

	// Create v1 API
	v1 := api.RegisterGroup("v1", "", map[string]string{
		"users":  "/users/:id",
		"posts":  "/posts",
		"search": "/search",
	})
	v1.SetTemplateVar("version", "v1")

	// Create v2 API
	v2 := api.RegisterGroup("v2", "", map[string]string{
		"users":    "/users/:id",
		"profiles": "/users/:id/profile",
		"search":   "/search/advanced",
	})
	v2.SetTemplateVar("version", "v2")

	// Build API URLs
	v1Users, _ := v1.Builder("users").WithParam("id", "123").Build()
	v2Users, _ := v2.Builder("users").WithParam("id", "123").Build()
	v2Profiles, _ := v2.Builder("profiles").WithParam("id", "123").Build()

	fmt.Printf("   • V1 Users: %s\n", v1Users)
	fmt.Printf("   • V2 Users: %s\n", v2Users)
	fmt.Printf("   • V2 Profiles: %s\n", v2Profiles)
	fmt.Println()

	// Example 4: Dynamic template variables
	fmt.Println("4. Dynamic Template Variables")
	fmt.Println("   Changing template variables at runtime...")

	// Create a CDN group
	manager.RegisterGroup("cdn", "https://cdn.example.com", map[string]string{
		"assets": "/assets/:version/:file",
		"images": "/images/:size/:filename",
	})
	cdn := manager.Group("cdn")

	cdn.SetURLTemplate("https://{region}.{service}.{domain}{route_path}")
	cdn.SetTemplateVar("service", "cdn")
	cdn.SetTemplateVar("domain", "example.com")
	cdn.SetTemplateVar("region", "us-east-1") // Default region

	// Build URL with default region
	assetsDefault, _ := cdn.Builder("assets").
		WithParam("version", "v1.0.0").
		WithParam("file", "app.js").
		Build()

	fmt.Printf("   • Default Region: %s\n", assetsDefault)

	// Change region dynamically
	cdn.SetTemplateVar("region", "eu-west-1")
	assetsEU, _ := cdn.Builder("assets").
		WithParam("version", "v1.0.0").
		WithParam("file", "app.js").
		Build()

	fmt.Printf("   • EU Region: %s\n", assetsEU)

	// Change region again
	cdn.SetTemplateVar("region", "ap-southeast-1")
	assetsAsia, _ := cdn.Builder("assets").
		WithParam("version", "v1.0.0").
		WithParam("file", "app.js").
		Build()

	fmt.Printf("   • Asia Region: %s\n", assetsAsia)
	fmt.Println()

	fmt.Println("✓ Template-based URL generation provides flexible, maintainable URL structures!")
	fmt.Println()

	// Show the key benefits
	fmt.Println("Key Benefits of Template-based URLs:")
	fmt.Println("• Consistent URL patterns across your application")
	fmt.Println("• Easy environment and region switching")
	fmt.Println("• Flexible internationalization support")
	fmt.Println("• Centralized URL structure management")
	fmt.Println("• Variable inheritance and override capabilities")
	fmt.Println()

	fmt.Println("=== End of Simple Template Example ===")
}
