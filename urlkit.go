package urlkit

import (
	"fmt"
	"maps"
	"net/url"
	"strings"

	ptre "github.com/soongo/path-to-regexp"
)

type Params map[string]any
type Query map[string]string

type ValidationError struct {
	Errors map[string][]string
}

func (v ValidationError) Error() string {
	var parts []string
	for group, missing := range v.Errors {
		parts = append(parts, fmt.Sprintf("group %s missing: %v", group, missing))
	}
	return "validation error: " + strings.Join(parts, ";")
}

type GroupValidationError struct {
	MissingRoutes []string
}

func (g GroupValidationError) Error() string {
	return fmt.Sprintf("missing routes: %v", g.MissingRoutes)
}

type RouteManager struct {
	groups map[string]*Group
}

type Config struct {
	Groups []GroupConfig `json:"groups"`
}

// GroupConfig defines the configuration structure for a group when loading from JSON.
// It supports both traditional path concatenation and template-based URL generation.
type GroupConfig struct {
	Name    string            `json:"name"`
	BaseURL string            `json:"base_url,omitempty"`
	Path    string            `json:"path,omitempty"`
	Paths   map[string]string `json:"paths"`
	Groups  []GroupConfig     `json:"groups,omitempty"`

	// Template Configuration Fields

	// URLTemplate defines the URL structure using placeholder syntax.
	// Example: "{protocol}://{host}/{locale}/{section}{route_path}"
	// When set, this group becomes a template owner and uses template rendering
	// instead of simple path concatenation. Template variables are substituted
	// using {variable_name} syntax.
	URLTemplate string `json:"url_template,omitempty"`

	// TemplateVars contains key-value pairs that this group contributes to template rendering.
	// Child groups can override parent variables, following a precedence rule where
	// child variables take priority over parent variables.
	// Special variables:
	//   - base_url: Automatically set to the group's base URL
	//   - route_path: Automatically set to the compiled route path with parameters
	TemplateVars map[string]string `json:"template_vars,omitempty"`
}

// Configurator defines the interface for route manager configuration.
// This interface follows the Config Getters pattern and allows for flexible
// configuration implementations that can be generated automatically.
type Configurator interface {
	GetGroups() []GroupConfig
}

// GetGroups implements the Configurator interface for the Config struct.
func (c Config) GetGroups() []GroupConfig {
	return c.Groups
}

// NewRouteManagerFromConfig creates a new RouteManager from a Configurator.
// This follows the configurator pattern used throughout the application.
func NewRouteManagerFromConfig(config Configurator) *RouteManager {
	cfg := &Config{
		Groups: config.GetGroups(),
	}
	return NewRouteManager(cfg)
}

// parseNestedGroups recursively processes nested groups in the configuration
func (m *RouteManager) parseNestedGroups(config GroupConfig, parent *Group) {
	for _, childConfig := range config.Groups {
		childGroup := parent.RegisterGroup(childConfig.Name, childConfig.Path, childConfig.Paths)

		if childConfig.BaseURL != "" {
			panic(fmt.Errorf("nested group %s cannot specify base_url, only root groups can have base URLs", childConfig.Name))
		}

		// Set URL template if provided
		if childConfig.URLTemplate != "" {
			childGroup.SetURLTemplate(childConfig.URLTemplate)
		}

		// Set template variables if provided
		for key, value := range childConfig.TemplateVars {
			childGroup.SetTemplateVar(key, value)
		}

		m.parseNestedGroups(childConfig, childGroup)
	}
}

func NewRouteManager(config ...*Config) *RouteManager {
	manager := &RouteManager{
		groups: map[string]*Group{},
	}

	// If config is provided, process it
	if len(config) > 0 && config[0] != nil {
		for _, groupConfig := range config[0].Groups {
			manager.RegisterGroup(groupConfig.Name, groupConfig.BaseURL, groupConfig.Paths)
			rootGroup := manager.Group(groupConfig.Name)

			// Set URL template if provided
			if groupConfig.URLTemplate != "" {
				rootGroup.SetURLTemplate(groupConfig.URLTemplate)
			}

			// Set template variables if provided
			for key, value := range groupConfig.TemplateVars {
				rootGroup.SetTemplateVar(key, value)
			}

			manager.parseNestedGroups(groupConfig, rootGroup)
		}
	}

	return manager
}

func (m *RouteManager) RegisterGroup(name, baseURL string, routes map[string]string) *RouteManager {
	if group, exists := m.groups[name]; exists {
		maps.Copy(m.groups[name].routes, routes)
		for route, tpl := range routes {
			group.compiledRoutes[route] = ptre.MustCompile(tpl, &ptre.Options{
				Encode: func(uri string, token any) string {
					return url.PathEscape(uri)
				},
			})
		}
	} else {
		m.groups[name] = NewURIHelper(baseURL, routes)
	}
	return m
}

// MustValidate calls Validate and panics if validation errors are found.
func (m *RouteManager) MustValidate(groups map[string][]string) *RouteManager {
	if err := m.Validate(groups); err != nil {
		panic(err)
	}
	return m
}

// Validate iterates over the given groups and their expected routes,
// calling each group's Validate method. It returns a ValidationError
// if any group is missing routes or if a group is entirely missing.
// Supports dot-separated group paths (e.g., "frontend.en.deep") for nested groups.
func (m *RouteManager) Validate(groups map[string][]string) error {
	validation := make(map[string][]string)
	failed := false
	for name, routes := range groups {
		var group *Group

		// Check if this is a dot-separated path for nested groups
		if strings.Contains(name, ".") {
			group = m.findGroupByPath(name)
		} else {
			// Backward compatibility: check root-level groups first
			var ok bool
			group, ok = m.groups[name]
			if !ok {
				group = nil
			}
		}

		if group == nil {
			validation[name] = []string{"Missing group"}
			failed = true
			continue
		}

		if err := group.Validate(routes); err != nil {
			failed = true
			if g, ok := err.(GroupValidationError); ok {
				validation[name] = g.MissingRoutes
			} else {
				validation[name] = []string{err.Error()}
			}
		}
	}

	if failed {
		return ValidationError{Errors: validation}
	}

	return nil
}

func (m *RouteManager) Group(name string) *Group {
	group, exists := m.groups[name]
	if !exists {
		panic(fmt.Errorf("group %s not found", name))
	}
	return group
}

// findGroupByPath traverses the group hierarchy using dot-separated paths
// to find the target group. Returns nil if the group is not found.
func (m *RouteManager) findGroupByPath(path string) *Group {
	if path == "" {
		return nil
	}

	// Split the path by dots to get individual group names
	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil
	}

	// Start with the root group
	rootGroup, exists := m.groups[parts[0]]
	if !exists {
		return nil
	}

	// If there's only one part, return the root group
	if len(parts) == 1 {
		return rootGroup
	}

	// Traverse the hierarchy for nested groups
	currentGroup := rootGroup
	for i := 1; i < len(parts); i++ {
		childGroup, exists := currentGroup.children[parts[i]]
		if !exists {
			return nil
		}
		currentGroup = childGroup
	}

	return currentGroup
}

// Group represents a collection of routes with optional templating capabilities.
// Groups can be organized in a hierarchy where child groups inherit and can override
// template variables from their parents.
//
// Template System:
// Groups support two URL generation modes:
// 1. Path Concatenation (default): URLs are built by concatenating baseURL + group paths + route
// 2. Template Rendering: URLs are built using a template string with variable substitution
//
// Template Variable Precedence (highest to lowest priority):
// - Built in variables (base_url, route_path)
// - Current group's template variables
// - Parent group's template variables (recursively up the hierarchy)
//
// Supported Template Syntax:
// - {variable_name}: Substitutes the variable with its value
// - {base_url}: Automatically available, contains the root group's base URL
// - {route_path}: Automatically available, contains the compiled route with parameters
type Group struct {
	baseURL        string
	routes         map[string]string
	compiledRoutes map[string]func(any) (string, error)
	name           string            // The name of this group relative to its parent
	path           string            // The path prefix for this group (e.g., "/en", "/v1")
	parent         *Group            // Pointer to parent group (nil for root groups)
	children       map[string]*Group // Map of child groups
	urlTemplate    string            // URL template string (e.g., "{base_url}/{locale}{route_path}")
	templateVars   map[string]string // Key-value pairs provided by this group
}

func NewURIHelper(baseURL string, routes map[string]string) *Group {
	compiled := make(map[string]func(any) (string, error), len(routes))

	for route, tpl := range routes {
		compiled[route] = ptre.MustCompile(tpl, &ptre.Options{
			Encode: func(uri string, token any) string {
				return url.PathEscape(uri)
			},
		})
	}

	return &Group{
		baseURL:        baseURL,
		routes:         routes,
		compiledRoutes: compiled,
		name:           "",
		path:           "",
		parent:         nil,
		children:       make(map[string]*Group),
		urlTemplate:    "",
		templateVars:   make(map[string]string),
	}
}

// Validate checks whether the group contains all expected routes.
// It returns a GroupValidationError if any routes are missing.
func (u *Group) Validate(routes []string) error {
	var missing []string
	for _, name := range routes {
		if _, ok := u.routes[name]; !ok {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		return GroupValidationError{MissingRoutes: missing}
	}

	return nil
}

func (u *Group) Render(routeName string, params Params, queries ...Query) (string, error) {
	compiled, ok := u.compiledRoutes[routeName]
	if !ok {
		return "", fmt.Errorf("route %s not found", routeName)
	}

	// Check if template rendering mode is available
	templateOwner := u.FindTemplateOwner()
	if templateOwner != nil {
		// Use template rendering mode
		return u.renderTemplatedURL(routeName, params, queries...)
	}

	// Fall back to existing path concatenation mode
	routePath, err := compiled(params)
	if err != nil {
		return "", fmt.Errorf("failed to build route: %s", err)
	}

	fullPath := u.getFullPath() + routePath

	rootGroup := u.getRootGroup()
	baseURL := rootGroup.baseURL

	return JoinURL(baseURL, fullPath, queries...), nil
}

func (u *Group) Route(routeName string) (string, error) {
	route, ok := u.routes[routeName]
	if !ok {
		return "", fmt.Errorf("route %s not found", routeName)
	}
	return route, nil
}

func (u *Group) MustRoute(routeName string) string {
	r, err := u.Route(routeName)
	if err != nil {
		panic(err)
	}
	return r
}

func (u *Group) Builder(routeName string) *Builder {
	return &Builder{
		helper:    u,
		routeName: routeName,
		params:    make(Params),
		query:     make(Query),
	}
}

// Group returns a child group by name for fluent API traversal.
// It panics if the child group is not found.
func (u *Group) Group(name string) *Group {
	group, exists := u.children[name]
	if !exists {
		panic(fmt.Errorf("group %s not found", name))
	}
	return group
}

// getFullPath builds the full path by traversing up the parent chain.
// It accumulates path segments from child to root, excluding the route itself.
func (u *Group) getFullPath() string {
	if u.parent == nil {
		// Root group - return empty string as base URL handles the prefix
		return ""
	}

	parentPath := u.parent.getFullPath()
	return parentPath + u.path
}

// getRootGroup finds and returns the root group by traversing up the parent chain.
func (u *Group) getRootGroup() *Group {
	if u.parent == nil {
		return u
	}
	return u.parent.getRootGroup()
}

// RegisterGroup creates and registers a new child group under the current group.
// If a child group with the same name already exists, it merges the routes.
func (u *Group) RegisterGroup(name, path string, routes map[string]string) *Group {
	if existingGroup, exists := u.children[name]; exists {
		// Merge routes into existing child group
		maps.Copy(existingGroup.routes, routes)
		for route, tpl := range routes {
			existingGroup.compiledRoutes[route] = ptre.MustCompile(tpl, &ptre.Options{
				Encode: func(uri string, token any) string {
					return url.PathEscape(uri)
				},
			})
		}
		return existingGroup
	} else {
		compiledRoutes := make(map[string]func(any) (string, error), len(routes))
		for route, tpl := range routes {
			compiledRoutes[route] = ptre.MustCompile(tpl, &ptre.Options{
				Encode: func(uri string, token any) string {
					return url.PathEscape(uri)
				},
			})
		}

		childGroup := &Group{
			baseURL:        "", // Child groups don't have base URLs
			routes:         routes,
			compiledRoutes: compiledRoutes,
			name:           name,
			path:           path,
			parent:         u,
			children:       make(map[string]*Group),
			urlTemplate:    "",
			templateVars:   make(map[string]string),
		}

		u.children[name] = childGroup
		return childGroup
	}
}

// Template Management Methods

// SetURLTemplate sets the URL template string for this group, enabling template-based URL generation.
// When a template is set, this group becomes a "template owner" and all URL generation for this
// group and its descendants will use template rendering instead of path concatenation.
//
// Template Syntax:
//   - Use {variable_name} to insert template variables
//   - {base_url} is automatically available (the root group's base URL)
//   - {route_path} is automatically available (the compiled route with parameters)
//
// Example templates:
//   - "{base_url}/api/{version}{route_path}"
//   - "{protocol}://{host}/{locale}/{section}{route_path}"
//   - "{base_url}/{env}/{service}{route_path}"
//
// To disable template rendering and revert to path concatenation, pass an empty string.
func (u *Group) SetURLTemplate(template string) {
	u.urlTemplate = template
}

// SetTemplateVar sets a template variable that will be available for substitution in URL templates.
// Template variables follow a hierarchical inheritance pattern where child groups can override
// parent variables.
//
// Variable Precedence (highest to lowest priority):
//  1. Built in variables (base_url, route_path) - cannot be overridden
//  2. Current group's variables
//  3. Parent group's variables (recursively up the hierarchy)
//
// Common use cases:
//   - SetTemplateVar("locale", "en-US") for internationalization
//   - SetTemplateVar("version", "v2") for API versioning
//   - SetTemplateVar("env", "staging") for environment-specific URLs
//   - SetTemplateVar("region", "eu-west") for regional deployments
func (u *Group) SetTemplateVar(key, value string) {
	u.templateVars[key] = value
}

// GetTemplateVar retrieves a template variable value from this group's local variables only.
// This method does NOT search the hierarchy - it only returns variables directly set on this group.
// Use CollectTemplateVars() to get the complete set of variables with inheritance applied.
//
// Returns:
//   - value: the variable value if found
//   - exists: true if the variable exists in this group's local variables
//
// Example:
//
//	value, exists := group.GetTemplateVar("locale")
//	if exists {
//	    fmt.Printf("Locale is set to: %s\n", value)
//	}
func (u *Group) GetTemplateVar(key string) (string, bool) {
	value, exists := u.templateVars[key]
	return value, exists
}

// AddRoutes dynamically adds new routes to this group at runtime.
// Routes are immediately compiled and available for URL building. This method is useful
// for conditional route registration or dynamic route generation based on configuration.
//
// Parameters:
//   - routes: a map of route names to path templates (e.g., "users": "/users/:id")
//
// Path templates follow the same syntax as route registration:
//   - Static segments: "/users/profile"
//   - Parameters: "/users/:id" or "/posts/:postId/comments/:commentId"
//   - Optional parameters: "/search/:query?"
//
// Example:
//
//	group.AddRoutes(map[string]string{
//	    "webhooks": "/webhooks/:event",
//	    "status":   "/status",
//	})
func (u *Group) AddRoutes(routes map[string]string) {
	// Add routes to the routes map
	for route, tpl := range routes {
		u.routes[route] = tpl
		// Compile the route template
		u.compiledRoutes[route] = ptre.MustCompile(tpl, &ptre.Options{
			Encode: func(uri string, token any) string {
				return url.PathEscape(uri)
			},
		})
	}
}

// Template discovery methods

// FindTemplateOwner traverses up the group hierarchy to locate the first ancestor group
// (including the current group) that defines a URL template.
//
// The method performs a depth-first search starting from the current group and moving
// up the parent chain until it finds a group with a non-empty urlTemplate field.
//
// Returns:
//   - *Group: The group that owns the URL template, or nil if no template is found
//     in the entire hierarchy chain.
//
// This method is essential for template-based URL construction as it determines
// which group's template should be used for rendering the final URL.
func (u *Group) FindTemplateOwner() *Group {
	// Check current group first
	if u.urlTemplate != "" {
		return u
	}

	// If no parent, return nil (no template found in hierarchy)
	if u.parent == nil {
		return nil
	}

	// Recursively check parent chain
	return u.parent.FindTemplateOwner()
}

// CollectTemplateVars aggregates template variables from the entire group hierarchy,
// implementing a child-overrides-parent precedence system.
//
// The collection process starts from the root group and moves down to the current group,
// ensuring that variables defined in child groups override those with the same key
// defined in parent groups. This allows for flexible variable inheritance and
// specialization at different hierarchy levels.
//
// Variable Precedence Rules (highest to lowest priority):
//  1. Built in dynamic variables (route_path, base_url)
//  2. Current group's templateVars
//  3. Parent groups' templateVars (closer ancestors override distant ones)
//
// Returns:
//   - map[string]string: A merged map of all template variables with proper precedence
//     applied. Keys are variable names, values are their string values.
//
// Example:
//
//	If parent has {"lang": "en", "theme": "light"} and child has {"lang": "es"},
//	the result will be {"lang": "es", "theme": "light"}.
func (u *Group) CollectTemplateVars() map[string]string {
	vars := make(map[string]string)

	// Collect variables from root to current group (parent first, child overrides)
	u.collectTemplateVarsRecursive(vars)

	return vars
}

// collectTemplateVarsRecursive is an internal helper method that implements the recursive
// variable collection algorithm for the template system.
//
// The method performs a post-order traversal of the group hierarchy, first collecting
// variables from parent groups, then adding/overriding with the current group's variables.
// This ensures the correct precedence order where child variables take priority over
// parent variables with the same key.
//
// Parameters:
//   - vars: A map[string]string that accumulates variables as the recursion progresses.
//     This map is modified in-place during the traversal.
//
// The method uses Go's maps.Copy function to efficiently merge template variables,
// which automatically handles key conflicts by using the source map's values
// (current group) over the destination map's values (accumulated parent variables).
func (u *Group) collectTemplateVarsRecursive(vars map[string]string) {
	// First collect parent variables (if parent exists)
	if u.parent != nil {
		u.parent.collectTemplateVarsRecursive(vars)
	}

	// Then add/override with current group's variables
	maps.Copy(vars, u.templateVars)
}

// renderTemplatedURL constructs URLs using the template-based rendering system,
// providing flexible URL structure independent of group hierarchy.
//
// This method implements the core template rendering logic by:
//  1. Locating the template owner group in the hierarchy
//  2. Compiling the specified route with provided parameters
//  3. Collecting template variables from all groups in the hierarchy
//  4. Adding built-in dynamic variables (route_path, base_url)
//  5. Performing string substitution on the template
//  6. Appending query parameters if provided
//
// Template Variables Added Automatically:
//   - route_path: The compiled route path (e.g., "/about-us" or "/user/123")
//   - base_url: The root group's base URL
//
// Parameters:
//   - routeName: Name of the route to render (must exist in current group's routes)
//   - params: Path parameters for route template compilation (e.g., {id: "123"})
//   - queries: Optional query string parameters to append to the final URL
//
// Returns:
//   - string: The fully rendered URL with template variables substituted
//   - error: Any error from route compilation or template processing
//
// Example:
//
//	With template "{protocol}://{host}/{lang}{route_path}" and variables
//	{"protocol": "https", "host": "example.com", "lang": "en"},
//	a route "/about" becomes "https://example.com/en/about".
func (u *Group) renderTemplatedURL(routeName string, params Params, queries ...Query) (string, error) {
	// Find the template owner (should exist since this method is called when template is found)
	templateOwner := u.FindTemplateOwner()
	if templateOwner == nil {
		return "", fmt.Errorf("no template owner found")
	}

	// Compile route template with parameters to get the route path
	compiled, ok := u.compiledRoutes[routeName]
	if !ok {
		return "", fmt.Errorf("route %s not found", routeName)
	}

	routePath, err := compiled(params)
	if err != nil {
		return "", fmt.Errorf("failed to build route: %s", err)
	}

	// Collect template variables from the hierarchy
	templateVars := u.CollectTemplateVars()

	// Add dynamic variables
	templateVars["route_path"] = routePath
	templateVars["base_url"] = u.getRootGroup().baseURL

	// Substitute template variables in the template string
	finalURL := SubstituteTemplate(templateOwner.urlTemplate, templateVars)

	// Append query parameters using existing logic
	if len(queries) > 0 {
		return JoinURL(finalURL, "", queries...), nil
	}

	return finalURL, nil
}

// SubstituteTemplate performs string substitution on URL templates using the
// specified variable map, implementing the {variable_name} placeholder syntax.
//
// Template Syntax:
//   - Placeholders use curly brace notation: {variable_name}
//   - Variable names are case-sensitive and can contain letters, numbers, and underscores
//   - Nested braces are not supported: {{variable}} is treated as literal text
//   - Missing variables: If a placeholder's variable is not found in the vars map,
//     the placeholder is left unchanged in the output string
//
// Supported Placeholder Examples:
//   - {protocol} → "https" (if vars["protocol"] = "https")
//   - {host} → "example.com" (if vars["host"] = "example.com")
//   - {route_path} → "/about" (built-in dynamic variable)
//   - {missing} → "{missing}" (unchanged if not in vars map)
//
// Parameters:
//   - template: The template string containing {variable} placeholders
//   - vars: Map of variable names to their string values for substitution
//
// Returns:
//   - string: The template with all found variables substituted, unfound placeholders
//     remain unchanged for debugging purposes
//
// Example:
//
//	SubstituteTemplate("{proto}://{host}/{path}", map[string]string{
//	  "proto": "https", "host": "api.example.com", "path": "v1"
//	})
//	Returns: "https://api.example.com/v1"
func SubstituteTemplate(template string, vars map[string]string) string {
	result := template

	// Replace all {variable} placeholders
	for key, value := range vars {
		placeholder := "{" + key + "}"
		result = strings.ReplaceAll(result, placeholder, value)
	}

	return result
}

type Builder struct {
	helper    *Group
	routeName string
	params    Params
	query     Query
}

func (b *Builder) WithParam(key string, value any) *Builder {
	b.params[key] = fmt.Sprint(value)
	return b
}

func (b *Builder) WithQuery(key string, value any) *Builder {
	b.query[key] = fmt.Sprint(value)
	return b
}

func (b *Builder) Build() (string, error) {
	return b.helper.Render(b.routeName, b.params, b.query)
}

func (b *Builder) MustBuild() string {
	s, err := b.Build()
	if err != nil {
		panic(err)
	}
	return s
}

func JoinURL(base, path string, queries ...Query) string {
	u, err := url.Parse(base)
	if err != nil {
		//hack... if base cant be parsed, we trait as a string
		u = &url.URL{Path: base}
	}

	if strings.HasPrefix(path, "/") {
		u.Path = path
	} else {
		if !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
		u.Path += path
	}

	qs := []string{}
	for _, query := range queries {
		for k, v := range query {
			qs = append(qs, k+"="+url.QueryEscape(v))
		}
	}

	rawQuery := strings.Join(qs, "&")
	if u.RawQuery != "" {
		if rawQuery != "" {
			u.RawQuery = u.RawQuery + "&" + rawQuery
		}
	} else {
		u.RawQuery = rawQuery
	}

	return u.String()
}
