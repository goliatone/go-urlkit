# go-urlkit

A Go library for URL routing and management with `Express.js` style route templates. Provides a type-safe way to build URLs with parameters and query strings, organized into logical groups.

## Installation

```bash
go get github.com/goliatone/go-urlkit
```

## Features

- `Express.js` style route templates with parameter substitution
- Route organization by groups (frontend, backend, webhooks, etc.)
- Fluent builder API for URL construction
- Built in validation for route configuration
- Type safe parameter and query handling
- URL encoding and query string management

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

## License

See LICENSE file for details.
