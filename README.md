# go-urlkit

A Go library for URL routing and management with `Express.js` style route templates. Provides a type-safe way to build URLs with parameters and query strings, organized into logical groups.

## Installation

```bash
go get github.com/goliatone/go-urlkit
```

## Features

- `Express.js` style route templates with parameter substitution
- Route organization by groups (frontend, backend, webhooks, etc.)
- Template-based URL generation with variable inheritance
- Fluent builder API for URL construction
- Built in validation for route configuration
- Type safe parameter and query handling
- URL encoding and query string management
- Complete OAuth2 client with state management and encryption

## Quick Start

```go
package main

import (
    "fmt"
    "github.com/goliatone/go-urlkit"
)

func main() {
    // Create a route manager
    rm := urlkit.NewRouteManager()

    // Register route groups
    rm.RegisterGroup("api", "https://api.example.com", map[string]string{
        "user":     "/users/:id",
        "profile":  "/users/:id/profile",
    })

    // Get a group and build URLs
    api := rm.Group("api")

    url, _ := api.Render("user", urlkit.Params{"id": "123"})
    fmt.Println(url) // https://api.example.com/users/123

    // Using the builder pattern
    url = api.Builder("profile").
        WithParam("id", "123").
        WithQuery("tab", "settings").
        MustBuild()
    fmt.Println(url) // https://api.example.com/users/123/profile?tab=settings
}
```

## Core Types

### RouteManager

Central manager for organizing route groups.

```go
rm := urlkit.NewRouteManager()
rm.RegisterGroup("group-name", "base-url", routes)
```

### Group

Container for related routes with a shared base URL.

```go
group := urlkit.NewURIHelper("https://api.example.com", map[string]string{
    "users": "/users/:id",
    "posts": "/posts/:postId",
})
```

### Builder

Fluent API for constructing URLs with parameters and queries.

```go
url := group.Builder("users").
    WithParam("id", "123").
    WithQuery("include", "profile").
    WithQuery("format", "json").
    MustBuild()
```

#### Advanced Builder Helpers

```go
type userParams struct {
    Locale string `json:"locale"`
}

url := group.Builder("user").
    WithParamsMap(map[string]any{"id": 42}).
    WithStruct(userParams{Locale: "fr"}).
    WithQueryValues(map[string][]string{"tag": []string{"new", "sale"}}).
    MustBuild()
// Result: https://api.example.com/users/42/fr?tag=new&tag=sale
```

`WithParamsMap` and `WithStruct` normalize Go values into the builder's
parameter map, merging struct fields by tag (defaulting to JSON semantics).
`WithQueryValues` makes it easy to add multi-value query parameters using the
standard `map[string][]string` shape.

## Usage Examples

### Basic Route Rendering

```go
routes := map[string]string{
    "user":     "/users/:id",
    "userPost": "/users/:userId/posts/:postId",
}

group := urlkit.NewURIHelper("https://api.example.com", routes)

// Simple parameter substitution
url, err := group.Render("user", urlkit.Params{"id": "123"})
// Result: https://api.example.com/users/123

// Multiple parameters
url, err = group.Render("userPost", urlkit.Params{
    "userId": "123",
    "postId": "456",
})
// Result: https://api.example.com/users/123/posts/456
```

### Adding Query Parameters

```go
// Single query parameter
url, err := group.Render("user",
    urlkit.Params{"id": "123"},
    urlkit.Query{"include": "profile"},
)
// Result: https://api.example.com/users/123?include=profile

// Multiple query parameters
url, err := group.Render("user",
    urlkit.Params{"id": "123"},
    urlkit.Query{"include": "profile"},
    urlkit.Query{"format": "json"},
)
// Result: https://api.example.com/users/123?include=profile&format=json
```

### Builder Pattern

```go
builder := group.Builder("userPost")
builder.WithParam("userId", 123)
builder.WithParam("postId", 456)
builder.WithQuery("include", "comments")
builder.WithQuery("page", 2)

url, err := builder.Build()
// Result: https://api.example.com/users/123/posts/456?include=comments&page=2

// Or chain the calls
url = group.Builder("userPost").
    WithParam("userId", 123).
    WithParam("postId", 456).
    WithQuery("include", "comments").
    MustBuild() // Panics on error
```

### Route Manager Resolver

`RouteManager` satisfies the `Resolver` interface so you can build URLs without
holding onto group handles.

```go
var resolver urlkit.Resolver = rm

url, err := resolver.Resolve("frontend", "login", urlkit.Params{}, urlkit.Query{"lang": "en"})
// Result: https://app.example.com/auth/login?lang=en

url, err = rm.ResolveWith(
    "frontend.profile",
    "details",
    map[string]any{"userId": 123},
    map[string][]string{"tab": []string{"posts", "mentions"}},
)
// Result: https://app.example.com/profile/123?tab=posts&tab=mentions
```

### Route Manager with Multiple Groups

```go
rm := urlkit.NewRouteManager()

// Frontend routes
rm.RegisterGroup("frontend", "https://app.example.com", map[string]string{
    "login":    "/auth/login",
    "dashboard": "/dashboard",
    "profile":   "/profile/:userId",
})

// API routes
rm.RegisterGroup("api", "https://api.example.com/v1", map[string]string{
    "users":  "/users/:id",
    "posts":  "/posts",
})

// Webhook routes
rm.RegisterGroup("webhooks", "https://webhooks.example.com", map[string]string{
    "stripe": "/webhooks/stripe",
    "github": "/webhooks/github/:event",
})

// Build URLs from different groups
frontendURL := rm.Group("frontend").Builder("profile").
    WithParam("userId", "123").
    MustBuild()

apiURL := rm.Group("api").Builder("users").
    WithParam("id", "123").
    WithQuery("include", "posts").
    MustBuild()
```

### Route Validation

```go
rm := urlkit.NewRouteManager()
rm.RegisterGroup("api", "https://api.example.com", map[string]string{
    "users": "/users/:id",
    "posts": "/posts/:id",
})

// Define expected routes per group
expected := map[string][]string{
    "api": {"users", "posts", "comments"}, // "comments" is missing
}

// Validate configuration
if err := rm.Validate(expected); err != nil {
    fmt.Printf("Validation failed: %v\n", err)
    // Output: validation error: group api missing: [comments]
}

// Use MustValidate to panic on validation errors
rm.MustValidate(expected) // Will panic
```

### Optional Parameters

```go
// Routes with optional parameters (using ? suffix)
routes := map[string]string{
    "webhook": "/webhooks/:service/:uuid?",
}

group := urlkit.NewURIHelper("https://api.example.com", routes)

// With optional parameter
url, _ := group.Render("webhook", urlkit.Params{
    "service": "gmail",
    "uuid":    "123",
})
// Result: https://api.example.com/webhooks/gmail/123

// Without optional parameter
url, _ = group.Render("webhook", urlkit.Params{
    "service": "gmail",
})
// Result: https://api.example.com/webhooks/gmail
```

### Template Based URL Generation

The library supports template based URL generation that provides flexible, maintainable URL structures with variable inheritance:

```go
// Create a route manager
rm := urlkit.NewRouteManager()

// Create a group with URL template
app, _ := rm.RegisterGroup("app", "https://app.example.com", map[string]string{
    "dashboard": "/dashboard",
    "profile":   "/profile/:userId",
})

// Set up template with variables
app.SetURLTemplate("https://{tenant}.{domain}{route_path}")
app.SetTemplateVar("domain", "myapp.com")

// Create tenant-specific child groups
acme, _ := app.RegisterNestedGroup("acme", "", map[string]string{
    "dashboard": "/dashboard",
    "settings":  "/settings/:section",
})
acme.SetTemplateVar("tenant", "acme")

// Generate URLs using the template
url, _ := acme.Builder("dashboard").Build()
// Result: https://acme.myapp.com/dashboard

url, _ = acme.Builder("settings").WithParam("section", "billing").Build()
// Result: https://acme.myapp.com/settings/billing
```

#### Internationalization with Templates

```go
// Load i18n configuration from JSON
config, err := loadConfigFromFile("i18n_config.json") // You'll need to implement this helper
if err != nil {
    log.Fatal(err)
}
rm := urlkit.NewRouteManager(&config)

// English URLs
enGroup := rm.Group("frontend.en")
aboutEN, _ := enGroup.Builder("about").Build()
// Result: https://www.example.com/en/about-us

// Spanish URLs
esGroup := rm.Group("frontend.es")
aboutES, _ := esGroup.Builder("about").Build()
// Result: https://www.example.com/es/acerca-de
```

#### Template Features

- **Variable Inheritance**: Child groups inherit parent variables and can override them
- **Dynamic Variables**: Automatically provided variables like `route_path` and `base_url`
- **Flexible Patterns**: Support for protocol, subdomain, path, and query customization
- **JSON Configuration**: Load complex template configurations from JSON files

See [examples/](examples/) for comprehensive template usage examples.

## Template Helpers

The library provides template helper functions that integrate seamlessly with Go template engines like [go-template](https://github.com/goliatone/go-template), enabling clean URL generation directly in templates.

### Basic Setup

```go
import (
    "github.com/goliatone/go-template"
    "github.com/goliatone/go-urlkit"
)

func main() {
    // Setup URLKit with routes
    manager := urlkit.NewRouteManager()
    manager.RegisterGroup("frontend", "https://example.com", map[string]string{
        "home":         "/",
        "user_profile": "/users/:id/profile",
        "contact":      "/contact",
    })

    // Create template renderer with URLKit helpers
    renderer, err := template.NewRenderer(
        template.WithBaseDir("./templates"),
        template.WithGlobalData(urlkit.TemplateHelpers(manager, nil)),
    )
    if err != nil {
        panic(err)
    }
}
```

### Template Helper Functions

#### `url(group, route, [params], [query])`
Generate complete URLs with optional path parameters and query strings:

```html
<!-- Simple URL -->
<a href="{{ url('frontend', 'home') }}">Home</a>
<!-- Result: https://example.com/ -->

<!-- With path parameters -->
<a href="{{ url('frontend', 'user_profile', {'id': user.id}) }}">Profile</a>
<!-- Result: https://example.com/users/123/profile -->

<!-- With query strings -->
<a href="{{ url('api', 'users', {}, {'page': 1, 'limit': 10}) }}">Users API</a>
<!-- Result: https://api.example.com/users?page=1&limit=10 -->

<!-- With both parameters and query strings -->
<a href="{{ url('api', 'user_posts', {'id': 1}, {'sort': 'date'}) }}">Posts</a>
<!-- Result: https://api.example.com/users/1/posts?sort=date -->
```

The helpers are also registered using PascalCase aliases (`URL`, `RoutePath`, `Navigation`, etc.) for teams that prefer Go-style naming in templates.

#### `route_path(group, route, [params], [query])`
Generate path and query portion only (for JavaScript/AJAX usage):

```html
<script>
const apiPath = "{{ route_path('api', 'users', {'id': 123}, {'format': 'json'}) }}";
// Result: /api/users/123?format=json
</script>
```

#### `has_route(group, route)`
Check if a route exists for conditional rendering:

```html
{% if has_route('admin', 'dashboard') %}
    <a href="{{ url('admin', 'dashboard') }}">Admin Panel</a>
{% endif %}
```

#### `route_template(group, route)`
Get the raw route template for debugging:

```html
<!-- Shows: "/users/:id/profile" -->
<div>Route template: {{ route_template('frontend', 'user_profile') }}</div>
```

#### `navigation(group, routes, [params])`
Return structured navigation nodes ready for rendering menus. The optional params map lets you supply per-route parameters.

```html
{% set main_nav = Navigation('frontend', ['home', 'profile'], {
    'profile': {'id': current_user.id},
}) %}

<nav>
  {% for item in main_nav %}
    <a href="{{ item.url }}" class="{% if item.full_route == current_route_name %}active{% endif %}">
      {{ item.route|title }}
    </a>
  {% endfor %}
</nav>
```

### Contextual Features

Template helpers support contextual features like navigation active states and URL rebuilding by accessing template variables. These context variables are typically provided by middleware that injects routing information into your template data.

#### Context Variables

Your application should provide the following template variables:

- `current_route_name`: Current route identifier (e.g., "frontend.user_profile")
- `current_params`: Current URL path parameters (e.g., {"id": "123"})
- `current_query`: Current query parameters (e.g., {"page": "2"})

#### Integration with go-router

If you're using [go-router](https://github.com/goliatone/go-router), the `routecontext` middleware automatically provides these variables. The middleware can be configured to export them either as top-level template variables or nested under a configurable map key:

```go
// Default configuration exports as top-level variables:
// current_route_name, current_params, current_query
middleware.RouteContext(middleware.RouteContextConfig{
    ExportAsMap: false, // Export as individual template variables
})

// Or export under a template_context map:
middleware.RouteContext(middleware.RouteContextConfig{
    ExportAsMap: true,
    MapKey: "template_context", // Variables nested under {{ template_context.current_route_name }}
})
```

#### Navigation Active States

```html
<nav>
    <a href="{{ url('frontend', 'home') }}"
       class="{{ current_route_if('frontend.home', current_route_name, 'active') }}">
        Home
    </a>
    <a href="{{ url('frontend', 'profile') }}"
       class="{{ current_route_if('frontend.profile', current_route_name, 'active', 'inactive') }}">
        Profile
    </a>
</nav>
```

#### URL Rebuilding with Modified Query Parameters

```html
<!-- Current URL: /products?category=electronics&page=2 -->
<thead>
    <tr>
        <th>
            <a href="{{ url('frontend', current_route_name, current_params, current_query|merge({'sort': 'name'})) }}">
                Product Name
            </a>
        </th>
    </tr>
</thead>
<!-- Result: /products?category=electronics&page=2&sort=name -->
```

#### Pagination Controls

```html
<div class="pagination">
    <a href="{{ url('frontend', current_route_name, current_params, current_query|merge({'page': current_query.page|int - 1})) }}">
        Previous
    </a>
    <a href="{{ url('frontend', current_route_name, current_params, current_query|merge({'page': current_query.page|int + 1})) }}">
        Next
    </a>
</div>
```

#### Current Route Detection

```html
<!-- Access current route directly -->
<body data-route="{{ current_route_name }}">

<!-- Conditional content based on current route -->
{% if current_route_name == 'frontend.user_profile' %}
    <div class="profile-actions">
        <a href="{{ url('frontend', 'edit_profile', current_params) }}">Edit Profile</a>
    </div>
{% endif %}
```

### Template Data Integration

When rendering templates, merge the contextual data with your template variables:

```go
func renderTemplate(w http.ResponseWriter, r *http.Request, templateName string, data map[string]any) {
    // Add contextual data from your routing system
    data["current_route_name"] = "frontend.user_profile"
    data["current_params"] = map[string]any{"id": "123"}
    data["current_query"] = map[string]string{"tab": "profile"}

    // Render template
    renderer.ExecuteTemplate(w, templateName, data)
}
```

The template helpers automatically handle parameter conversion, error handling, and URL encoding, providing a robust foundation for template-based URL generation.

### URL Joining Utility

The package also provides a standalone URL joining function:

```go
// Basic path joining
url := urlkit.JoinURL("https://api.example.com", "/users")
// Result: https://api.example.com/users

// With query parameters
url = urlkit.JoinURL("https://api.example.com", "/users",
    urlkit.Query{"page": "1"},
    urlkit.Query{"limit": "10"},
)
// Result: https://api.example.com/users?page=1&limit=10

// Preserves existing query parameters
url = urlkit.JoinURL("https://api.example.com?existing=1", "/users",
    urlkit.Query{"new": "2"},
)
// Result: https://api.example.com/users?existing=1&new=2
```

## OAuth2 Integration

The library includes a complete OAuth2 client with state management, encryption, and support for multiple providers. It provides a secure, type safe way to implement OAuth2 authorization flows.

### Features

- **Generic Provider Interface**: Support for any OAuth2 provider (Google, GitHub, Facebook, etc.)
- **State Management**: Automatic CSRF protection with encrypted state parameters
- **Type-Safe User Data**: Attach custom data structures to the OAuth2 flow
- **Built-in Encryption**: AES encryption for sensitive state data
- **Thread-Safe**: Concurrent operation support
- **Comprehensive Error Handling**: Detailed error types for different failure scenarios

### Quick Start

```go
import "github.com/goliatone/go-urlkit/oauth2"

// Define custom user data for the OAuth2 flow
type UserContext struct {
    UserID   string `json:"user_id"`
    ReturnTo string `json:"return_to"`
}

// Create a Google OAuth2 provider
provider, err := oauth2.NewGoogleProvider()
if err != nil {
    log.Fatal(err)
}

// Create OAuth2 client with user context type
client, err := oauth2.NewClient[UserContext](
    provider,
    "your-google-client-id",
    "your-google-client-secret",
    "http://localhost:8080/oauth/callback",
    "your-24-character-encrypt-key",
)
if err != nil {
    log.Fatal(err)
}

// Generate authorization URL with user context
userCtx := UserContext{
    UserID:   "user123",
    ReturnTo: "/dashboard",
}
authURL, err := client.GenerateURL("random-csrf-token", userCtx)
if err != nil {
    log.Fatal(err)
}

// Redirect user to authURL...

// Handle OAuth2 callback
func handleCallback(w http.ResponseWriter, r *http.Request) {
    code := r.URL.Query().Get("code")
    state := r.URL.Query().Get("state")

    // Validate state and retrieve user context
    originalState, userCtx, err := client.ValidateState(state)
    if err != nil {
        http.Error(w, "Invalid state", http.StatusBadRequest)
        return
    }

    // Exchange code for access token
    token, err := client.Exchange(r.Context(), code)
    if err != nil {
        http.Error(w, "Token exchange failed", http.StatusInternalServerError)
        return
    }

    // Get user information
    userInfo, err := client.GetUserInfo(token)
    if err != nil {
        http.Error(w, "Failed to get user info", http.StatusInternalServerError)
        return
    }

    // Use userCtx.ReturnTo to redirect user back to their original destination
    // userInfo contains the OAuth2 user profile data
}
```

### Extending OAuth2 Scopes

```go
// Add additional scopes for Google services
oauth2.AddGoogleScopes(provider, []string{"gmail", "drive"})

// Or set custom scopes
provider.SetScopes([]string{
    "https://www.googleapis.com/auth/userinfo.profile",
    "https://www.googleapis.com/auth/userinfo.email",
    "https://www.googleapis.com/auth/gmail.readonly",
    "https://www.googleapis.com/auth/drive.file",
})
```

### Custom OAuth2 Providers

```go
// Implement the Provider interface for custom OAuth2 providers
type CustomProvider struct {
    name      string
    config    *oauth2.Config
    scopes    []string
    userURL   string
}

func (p *CustomProvider) Name() string { return p.name }
func (p *CustomProvider) Config() *oauth2.Config { return p.config }
func (p *CustomProvider) Scopes() []string { return p.scopes }
func (p *CustomProvider) SetScopes(scopes []string) { p.scopes = scopes }
func (p *CustomProvider) GetUserInfo(token *oauth2.Token) (map[string]any, error) {
    // Implementation for fetching user info from your provider
}

// Use your custom provider
provider := &CustomProvider{
    name: "mycorp",
    config: &oauth2.Config{
        ClientID:     "your-client-id",
        ClientSecret: "your-client-secret",
        Endpoint: oauth2.Endpoint{
            AuthURL:  "https://auth.mycorp.com/oauth/authorize",
            TokenURL: "https://auth.mycorp.com/oauth/token",
        },
    },
    userURL: "https://api.mycorp.com/user",
}

client, err := oauth2.NewClient[UserContext](provider, ...)
```

### OAuth2 Error Handling

```go
// State validation errors
_, userCtx, err := client.ValidateState(state)
if err != nil {
    switch {
    case errors.Is(err, oauth2.ErrStateNotFound):
        // CSRF attack or expired state
    case errors.Is(err, oauth2.ErrDecryptionFailed):
        // Encryption key mismatch or corrupted data
    case errors.Is(err, oauth2.ErrDeserializationFailed):
        // JSON parsing failed for user data
    }
}
```

### OAuth2 Examples

See [examples/oauth2_example.go](examples/oauth2_example.go) and [examples/oauth2/](examples/oauth2/) for comprehensive OAuth2 integration examples including:

- Complete authorization flow implementation
- State management and validation
- Error handling for different scenarios
- Multi provider support
- User data preservation across the OAuth2 flow

## Error Handling

The library provides specific error types for different failure scenarios:

### ValidationError

Returned when route validation fails:

```go
if err := rm.Validate(expected); err != nil {
    if ve, ok := err.(urlkit.ValidationError); ok {
        for group, missing := range ve.Errors {
            fmt.Printf("Group %s is missing routes: %v\n", group, missing)
        }
    }
}
```

### GroupValidationError

Returned when a specific group fails validation:

```go
if err := group.Validate([]string{"users", "posts"}); err != nil {
    if gve, ok := err.(urlkit.GroupValidationError); ok {
        fmt.Printf("Missing routes: %v\n", gve.MissingRoutes)
    }
}
```

## Testing

Run tests with:

```bash
./taskfile dev:test
```

Run tests with coverage:

```bash
./taskfile dev:cover
```

## Requirements

- Go 1.23.4 or later
- github.com/soongo/path-to-regexp v1.6.4
- golang.org/x/oauth2 (for OAuth2 functionality)
- github.com/google/uuid (for OAuth2 functionality)

## License

See LICENSE file for details.
