# Examples

This directory contains comprehensive examples demonstrating the URL routing and template features of the `go-urlkit` library.

## Example Files

### Configuration Examples

- **[app.json](app.json)** - Original example showing traditional path concatenation with nested groups
- **[templated/templated_urls.json](templated/templated_urls.json)** - Comprehensive template-based configuration example showcasing:
  - Internationalization (i18n) with locale-specific URLs
  - API versioning with consistent patterns
  - CDN region switching
  - Template variable inheritance

### Code Examples

- **[simple_template/main.go](simple_template/main.go)** - **Start here!** A beginner-friendly example showing:
  - Basic template usage and setup
  - Environment-specific URL generation
  - API versioning patterns
  - Dynamic template variable changes

- **[templated/main.go](templated/main.go)** - Advanced template features including:
  - Loading templates from JSON configuration
  - Complex nested group scenarios
  - Variable inheritance and overriding
  - Multi-tenant application patterns
  - Programmatic template management

- **[oauth2/main.go](oauth2/main.go)** - OAuth2 integration example (separate feature)

### OAuth2 Examples

- **[oauth2/](oauth2/)** - Directory containing OAuth2-specific examples

## Quick Start

### 1. Basic Template Usage

```bash
go run examples/simple_template/main.go
```

### 2. Advanced Template Features

```bash
go run examples/templated/main.go
```

### 3. Load Configuration from JSON

```go
config, err := loadConfigFromFile("examples/templated/templated_urls.json") // Helper function needed
if err != nil {
    log.Fatal(err)
}
manager := urlkit.NewRouteManager(&config)
if err != nil {
    log.Fatal(err)
}

// Use the loaded configuration
group := manager.Group("i18n_frontend.en")
url, _ := group.Builder("about").Build()
fmt.Println(url) // https://www.example.com/en/about-us
```

## Template System Overview

The template system allows you to define URL patterns using variables that can be dynamically substituted:

### Template Syntax

```
{protocol}://{subdomain}.{domain}/{locale}{route_path}
```

### Key Concepts

1. **Template Owner**: The group that defines the URL template
2. **Template Variables**: Key-value pairs contributed by groups in the hierarchy  
3. **Variable Inheritance**: Child groups inherit parent variables and can override them
4. **Dynamic Variables**: Automatically provided variables like `route_path` and `base_url`

### Benefits

- **Consistency**: Centralized URL structure management
- **Flexibility**: Easy environment, region, and locale switching
- **Maintainability**: Changes to URL patterns are centralized
- **Internationalization**: Built-in support for localized URL structures

## Configuration Examples

### Simple Template

```json
{
  "name": "api",
  "base_url": "https://api.example.com",
  "url_template": "https://api.{domain}/{version}{route_path}",
  "template_vars": {
    "domain": "example.com"
  },
  "groups": [
    {
      "name": "v1", 
      "template_vars": {
        "version": "v1"
      },
      "routes": {
        "users": "/users/:id"
      }
    }
  ]
}
```

This generates URLs like: `https://api.example.com/v1/users/123`

### Internationalization Template

```json
{
  "name": "frontend",
  "url_template": "https://{subdomain}.{domain}/{locale}{route_path}",
  "template_vars": {
    "subdomain": "www",
    "domain": "example.com"
  },
  "groups": [
    {
      "name": "en",
      "template_vars": { "locale": "en" },
      "routes": { "about": "/about-us" }
    },
    {
      "name": "es", 
      "template_vars": { "locale": "es" },
      "routes": { "about": "/acerca-de" }
    }
  ]
}
```

This generates:
- English: `https://www.example.com/en/about-us`
- Spanish: `https://www.example.com/es/acerca-de`

## Running Examples

All examples are self-contained and can be run directly:

```bash
# Basic template concepts
go run examples/simple_template/main.go

# Advanced template features  
go run examples/templated/main.go

# OAuth2 integration
go run examples/oauth2/main.go
```

The examples will output detailed explanations and demonstrate various URL generation patterns.

## Integration with Your Application

### Loading from JSON Config

```go
config, err := loadConfigFromFile("config/routes.json") // Helper function needed
if err != nil {
    log.Fatal(err)
}
manager := urlkit.NewRouteManager(&config)
if err != nil {
    log.Fatal(err)
}

// Use in HTTP handlers
func handler(w http.ResponseWriter, r *http.Request) {
    userGroup := manager.Group("api.v1")
    userURL, _ := userGroup.Builder("profile").WithParam("id", "123").Build()
    // userURL: https://api.example.com/v1/users/123/profile
}
```

### Programmatic Configuration

```go
manager := urlkit.NewRouteManager()

// Create template-enabled group
app, _ := manager.RegisterGroup("app", "https://app.com", map[string]string{
    "dashboard": "/dashboard",
})
app.SetURLTemplate("https://{tenant}.{domain}{route_path}")
app.SetTemplateVar("domain", "myapp.com")

// Create tenant-specific child groups
acme, _ := app.RegisterNestedGroup("acme", "", map[string]string{
    "dashboard": "/dashboard",
})
acme.SetTemplateVar("tenant", "acme")

// Generate URLs
url, _ := acme.Builder("dashboard").Build()
// url: https://acme.myapp.com/dashboard
```
