package urlkit

import (
	"errors"
	"fmt"
	"maps"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"sync"

	ptre "github.com/soongo/path-to-regexp"
)

type Params = map[string]any
type Query map[string]string

var (
	ErrGroupNotFound = errors.New("group not found")
	ErrRouteNotFound = errors.New("route not found")
)

type Resolver interface {
	Resolve(groupPath, route string, params Params, query Query) (string, error)
}

// NavigationNode represents a prebuilt navigation entry constructed from a group route.
// It captures enough information for templates to render menus without recomputing URLs.
type NavigationNode struct {
	Group     string `json:"group"`      // Dot-qualified group name (e.g., "frontend.en")
	Route     string `json:"route"`      // Route identifier within the group (e.g., "about")
	FullRoute string `json:"full_route"` // Fully qualified route name (e.g., "frontend.en.about")
	Path      string `json:"path"`       // Raw route template (e.g., "/about" or "/users/:id")
	URL       string `json:"url"`        // Resolved URL including host/base path
	Params    Params `json:"params,omitempty"`
}

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

// TemplateSubstitutionError represents a failure to replace all placeholders in a template.
type TemplateSubstitutionError struct {
	Group         string
	Route         string
	TemplateOwner string
	Template      string
	Missing       []string
}

func (e TemplateSubstitutionError) Error() string {
	return fmt.Sprintf(
		"template substitution failed for group %q route %q (template owner %q): missing variables %v",
		e.Group,
		e.Route,
		e.TemplateOwner,
		e.Missing,
	)
}

type RouteManager struct {
	mu     sync.RWMutex
	groups map[string]*Group
}

type Config struct {
	Groups []GroupConfig `json:"groups" yaml:"groups"`
}

// GroupConfig defines the configuration structure for a group when loading from JSON/YAML.
// It supports both traditional path concatenation and template based URL generation.
type GroupConfig struct {
	Name    string            `json:"name" yaml:"name"`
	BaseURL string            `json:"base_url,omitempty" yaml:"base_url,omitempty"`
	Path    string            `json:"path,omitempty" yaml:"path,omitempty"`
	Routes  map[string]string `json:"routes,omitempty" yaml:"routes,omitempty"`
	Paths   map[string]string `json:"paths,omitempty" yaml:"paths,omitempty"` // legacy support
	Groups  []GroupConfig     `json:"groups,omitempty" yaml:"groups,omitempty"`

	// Template Configuration Fields

	// URLTemplate defines the URL structure using placeholder syntax.
	// Example: "{protocol}://{host}/{locale}/{section}{route_path}"
	// When set, this group becomes a template owner and uses template rendering
	// instead of simple path concatenation. Template variables are substituted
	// using {variable_name} syntax.
	URLTemplate string `json:"url_template,omitempty" yaml:"url_template,omitempty"`

	// TemplateVars contains key-value pairs that this group contributes to template rendering.
	// Child groups can override parent variables, following a precedence rule where
	// child variables take priority over parent variables.
	// Special variables:
	//   - base_url: Automatically set to the group's base URL
	//   - route_path: Automatically set to the compiled route path with parameters
	TemplateVars map[string]string `json:"template_vars,omitempty" yaml:"template_vars,omitempty"`
}

func (g GroupConfig) effectiveRoutes() map[string]string {
	if len(g.Routes) > 0 {
		return g.Routes
	}
	if len(g.Paths) > 0 {
		return g.Paths
	}
	return map[string]string{}
}

func cloneRoutes(routes map[string]string) map[string]string {
	if len(routes) == 0 {
		return map[string]string{}
	}

	clone := make(map[string]string, len(routes))
	for key, value := range routes {
		clone[key] = value
	}
	return clone
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

// NewRouteManagerFromConfig creates a new RouteManager from a Configurator and validates
// the hierarchy during construction.
func NewRouteManagerFromConfig(config Configurator) (*RouteManager, error) {
	manager := &RouteManager{
		groups: map[string]*Group{},
	}

	if config == nil {
		return manager, nil
	}

	for _, groupConfig := range config.GetGroups() {
		if _, err := manager.loadGroupFromConfig(groupConfig, nil); err != nil {
			return nil, err
		}
	}

	return manager, nil
}

func NewRouteManager(config ...*Config) *RouteManager {
	manager := &RouteManager{
		groups: map[string]*Group{},
	}

	// If config is provided, process it
	if len(config) > 0 && config[0] != nil {
		for _, groupConfig := range config[0].Groups {
			if _, err := manager.loadGroupFromConfig(groupConfig, nil); err != nil {
				panic(err)
			}
		}
	}

	return manager
}

func (m *RouteManager) loadGroupFromConfig(cfg GroupConfig, parent *Group) (*Group, error) {
	if cfg.Name == "" {
		return nil, fmt.Errorf("configuration error: group name is required")
	}

	routes := cloneRoutes(cfg.effectiveRoutes())

	if parent == nil {
		if _, exists := m.groups[cfg.Name]; exists {
			return nil, fmt.Errorf("configuration error: duplicate root group %s", cfg.Name)
		}

		m.RegisterGroup(cfg.Name, cfg.BaseURL, routes)
		group := m.Group(cfg.Name)
		group.mu.Lock()
		group.name = cfg.Name
		group.path = cfg.Path
		group.mu.Unlock()

		if cfg.URLTemplate != "" {
			group.SetURLTemplate(cfg.URLTemplate)
		}

		for key, value := range cfg.TemplateVars {
			group.SetTemplateVar(key, value)
		}

		for _, child := range cfg.Groups {
			if child.BaseURL != "" {
				return nil, fmt.Errorf("configuration error: nested group %s cannot specify base_url", child.Name)
			}
			if _, err := m.loadGroupFromConfig(child, group); err != nil {
				return nil, err
			}
		}

		return group, nil
	}

	if cfg.BaseURL != "" {
		return nil, fmt.Errorf("configuration error: nested group %s cannot specify base_url", cfg.Name)
	}

	childGroup := parent.RegisterGroup(cfg.Name, cfg.Path, routes)
	childGroup.mu.Lock()
	childGroup.name = cfg.Name
	if cfg.Path != "" {
		childGroup.path = cfg.Path
	}
	childGroup.mu.Unlock()

	if cfg.URLTemplate != "" {
		childGroup.SetURLTemplate(cfg.URLTemplate)
	}

	for key, value := range cfg.TemplateVars {
		childGroup.SetTemplateVar(key, value)
	}

	for _, child := range cfg.Groups {
		if child.BaseURL != "" {
			return nil, fmt.Errorf("configuration error: nested group %s cannot specify base_url", child.Name)
		}
		if _, err := m.loadGroupFromConfig(child, childGroup); err != nil {
			return nil, err
		}
	}

	return childGroup, nil
}

func (m *RouteManager) RegisterGroup(name, baseURL string, routes map[string]string) *RouteManager {
	m.mu.Lock()
	defer m.mu.Unlock()

	if group, exists := m.groups[name]; exists {
		group.mu.Lock()
		if group.name == "" {
			group.name = name
		}
		group.mu.Unlock()
		group.AddRoutes(routes)
		return m
	}

	group := NewURIHelper(baseURL, routes)
	group.mu.Lock()
	group.name = name
	group.mu.Unlock()
	m.groups[name] = group
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
		group, err := m.GetGroup(name)
		if err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				validation[name] = []string{"Missing group"}
			} else {
				validation[name] = []string{err.Error()}
			}
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

// DebugTree returns a formatted string representing the entire group hierarchy,
// including routes, templates, and effective template variables for each group.
// Output is stable (sorted alphabetically) to simplify inspection and diffing.
func (m *RouteManager) DebugTree() string {
	if m == nil {
		return "RouteManager: <nil>"
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.groups) == 0 {
		return "RouteManager: <empty>"
	}

	var builder strings.Builder
	builder.WriteString("RouteManager Debug Tree:\n")

	rootNames := slices.Sorted(maps.Keys(m.groups))

	for idx, name := range rootNames {
		appendGroupDebug(&builder, m.groups[name], 0)
		if idx < len(rootNames)-1 {
			builder.WriteByte('\n')
		}
	}

	return builder.String()
}

// GetGroup returns the group registered at the given path. The path may reference
// nested groups using dot-notation (e.g., "frontend.en.marketing"). Returns
// ErrGroupNotFound when the requested group does not exist.
func (m *RouteManager) GetGroup(path string) (*Group, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty group path", ErrGroupNotFound)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if group, ok := m.groups[path]; ok {
		return group, nil
	}

	var group *Group
	if strings.Contains(path, ".") {
		group = m.findGroupByPath(path)
	} else {
		group = m.groups[path]
	}

	if group == nil {
		return nil, fmt.Errorf("%w: %s", ErrGroupNotFound, path)
	}

	return group, nil
}

func (m *RouteManager) Group(path string) *Group {
	group, err := m.GetGroup(path)
	if err != nil {
		panic(err)
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
	rawParts := strings.Split(path, ".")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil
		}
		parts = append(parts, part)
	}

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
		currentGroup.mu.RLock()
		childGroup, exists := currentGroup.children[parts[i]]
		currentGroup.mu.RUnlock()
		if !exists {
			return nil
		}
		currentGroup = childGroup
	}

	return currentGroup
}

// EnsureGroup ensures that the full group path exists, creating intermediate
// groups as needed. The path must start with an existing root group name.
// Intermediate segments can optionally define a custom path using the syntax
// "name:/custom-path". Missing segments default to "/name". Returns the final
// group or an ErrGroupNotFound if the root group does not exist.
func (m *RouteManager) EnsureGroup(path string) (*Group, error) {
	if path == "" {
		return nil, fmt.Errorf("%w: empty group path", ErrGroupNotFound)
	}

	if group, err := m.GetGroup(path); err == nil {
		return group, nil
	}

	parts := strings.Split(path, ".")
	if len(parts) == 0 {
		return nil, fmt.Errorf("%w: empty group path", ErrGroupNotFound)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	root, exists := m.groups[parts[0]]
	if !exists {
		return nil, fmt.Errorf("%w: %s", ErrGroupNotFound, parts[0])
	}

	current := root
	for idx, rawSegment := range parts[1:] {
		name, segmentPath, err := parseEnsureSegment(rawSegment)
		if err != nil {
			return nil, fmt.Errorf("ensure group: %w", err)
		}

		current.mu.RLock()
		child, ok := current.children[name]
		current.mu.RUnlock()
		if ok {
			current = child
			continue
		}

		current = current.registerChildLocked(name, segmentPath, map[string]string{})

		current.mu.Lock()
		if current.path == "" {
			current.path = segmentPath
		}
		if current.path == "" {
			current.mu.Unlock()
			return nil, fmt.Errorf("ensure group: empty path for segment %q at position %d", name, idx+2)
		}
		current.mu.Unlock()
	}

	return current, nil
}

// AddRoutes attaches additional routes to an existing group identified by the
// provided path. Routes in the map are compiled immediately and overwrite any
// existing routes with the same name. The path supports dot-notation for nested
// groups.
func (m *RouteManager) AddRoutes(path string, routes map[string]string) (*Group, error) {
	group, err := m.GetGroup(path)
	if err != nil {
		return nil, err
	}

	group.AddRoutes(routes)
	return group, nil
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
	mu             sync.RWMutex
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
	u.mu.RLock()
	defer u.mu.RUnlock()

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
	u.mu.RLock()
	compiled, ok := u.compiledRoutes[routeName]
	u.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("%w: route %q in group %s", ErrRouteNotFound, routeName, groupDisplayName(u))
	}

	// Check if template rendering mode is available
	templateOwner := u.FindTemplateOwner()
	if templateOwner != nil {
		// Use template rendering mode
		return u.renderTemplatedURL(routeName, compiled, params, queries...)
	}

	// Fall back to existing path concatenation mode
	routePath, err := compiled(params)
	if err != nil {
		return "", fmt.Errorf("failed to build route: %s", err)
	}

	fullPath := joinURLPath(u.getFullPath(), routePath)

	rootGroup := u.getRootGroup()
	rootGroup.mu.RLock()
	baseURL := rootGroup.baseURL
	rootGroup.mu.RUnlock()

	return JoinURL(baseURL, fullPath, queries...), nil
}

func (u *Group) Route(routeName string) (string, error) {
	u.mu.RLock()
	route, ok := u.routes[routeName]
	u.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("%w: route %q in group %s", ErrRouteNotFound, routeName, groupDisplayName(u))
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
	u.mu.RLock()
	group, exists := u.children[name]
	u.mu.RUnlock()
	if !exists {
		panic(fmt.Errorf("%w: %s.%s", ErrGroupNotFound, groupDisplayName(u), name))
	}
	return group
}

// getFullPath builds the full path by traversing up the parent chain.
// It accumulates path segments from child to root, excluding the route itself.
func (u *Group) getFullPath() string {
	if u == nil {
		return ""
	}

	u.mu.RLock()
	path := u.path
	parent := u.parent
	u.mu.RUnlock()

	if parent == nil {
		return path
	}

	return parent.getFullPath() + path
}

// getRootGroup finds and returns the root group by traversing up the parent chain.
func (u *Group) getRootGroup() *Group {
	if u == nil {
		return nil
	}

	u.mu.RLock()
	parent := u.parent
	u.mu.RUnlock()

	if parent == nil {
		return u
	}
	return parent.getRootGroup()
}

// Navigation builds a slice of NavigationNode entries for the provided routes.
// The params callback can supply per-route parameter maps which are applied before building URLs.
func (u *Group) Navigation(routes []string, params func(route string) Params) ([]NavigationNode, error) {
	if len(routes) == 0 {
		return []NavigationNode{}, nil
	}

	nodes := make([]NavigationNode, 0, len(routes))
	groupName := u.FQN()

	for _, routeName := range routes {
		if routeName == "" {
			continue
		}

		builder := u.Builder(routeName)

		var providedParams Params
		if params != nil {
			providedParams = params(routeName)
		}

		if len(providedParams) > 0 {
			for key, value := range providedParams {
				builder.WithParam(key, value)
			}
		}

		urlValue, err := builder.Build()
		if err != nil {
			return nil, err
		}

		routePattern, err := u.Route(routeName)
		if err != nil {
			return nil, err
		}

		fullRoute := routeName
		if groupName != "" {
			fullRoute = groupName + "." + routeName
		}

		nodes = append(nodes, NavigationNode{
			Group:     groupName,
			Route:     routeName,
			FullRoute: fullRoute,
			Path:      routePattern,
			URL:       urlValue,
			Params:    cloneParamsMap(providedParams),
		})
	}

	return nodes, nil
}

// FQN returns the group's fully qualified name within the hierarchy (dot notation).
// Root groups return their own name, while nested groups include their ancestors
// (e.g., "frontend.en.marketing"). An empty string indicates the group is detached
// from the manager hierarchy.
func (u *Group) FQN() string {
	if u == nil {
		return ""
	}

	u.mu.RLock()
	name := u.name
	parent := u.parent
	u.mu.RUnlock()

	if parent == nil {
		return name
	}

	parentName := parent.FQN()
	if parentName == "" {
		return name
	}

	if name == "" {
		return parentName
	}

	return parentName + "." + name
}

// RegisterGroup creates and registers a new child group under the current group.
// If a child group with the same name already exists, it merges the routes.
func (u *Group) RegisterGroup(name, path string, routes map[string]string) *Group {
	return u.registerChildLocked(name, path, routes)
}

func (u *Group) registerChildLocked(name, path string, routes map[string]string) *Group {
	u.mu.Lock()
	defer u.mu.Unlock()

	if existingGroup, exists := u.children[name]; exists {
		if len(routes) > 0 {
			existingGroup.AddRoutes(routes)
		}
		if path != "" {
			existingGroup.mu.Lock()
			if existingGroup.path == "" {
				existingGroup.path = path
			}
			existingGroup.mu.Unlock()
		}
		return existingGroup
	}

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
	u.mu.Lock()
	defer u.mu.Unlock()
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
	u.mu.Lock()
	defer u.mu.Unlock()
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
	u.mu.RLock()
	defer u.mu.RUnlock()
	value, exists := u.templateVars[key]
	return value, exists
}

// AddRoutes dynamically adds new routes to this group at runtime.
// Routes are immediately compiled and available for URL building. Existing routes with the
// same name are replaced and recompiled. This method is useful for conditional route
// registration or dynamic route generation based on configuration.
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
	u.mu.Lock()
	defer u.mu.Unlock()

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
	for current := u; current != nil; {
		current.mu.RLock()
		if current.urlTemplate != "" {
			current.mu.RUnlock()
			return current
		}
		parent := current.parent
		current.mu.RUnlock()
		current = parent
	}
	return nil
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
	var chain []*Group
	for current := u; current != nil; {
		current.mu.RLock()
		parent := current.parent
		current.mu.RUnlock()

		chain = append(chain, current)
		current = parent
	}

	vars := make(map[string]string)
	for i := len(chain) - 1; i >= 0; i-- {
		chain[i].mu.RLock()
		maps.Copy(vars, chain[i].templateVars)
		chain[i].mu.RUnlock()
	}

	return vars
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
func (u *Group) renderTemplatedURL(routeName string, compiled func(any) (string, error), params Params, queries ...Query) (string, error) {
	// Find the template owner (should exist since this method is called when template is found)
	templateOwner := u.FindTemplateOwner()
	if templateOwner == nil {
		return "", fmt.Errorf("no template owner found")
	}

	routePath, err := compiled(params)
	if err != nil {
		return "", fmt.Errorf("failed to build route: %s", err)
	}

	// Collect template variables from the hierarchy
	templateVars := u.CollectTemplateVars()

	// Determine optional route path suffix behavior.
	routePathSuffix, hasSuffix := templateVars["route_path_suffix"]
	if !hasSuffix {
		routePathSuffix = "/"
	}

	routePath = applyRoutePathSuffix(routePath, routePathSuffix)

	// Add dynamic variables
	templateVars["route_path"] = routePath
	root := u.getRootGroup()
	if root == nil {
		return "", fmt.Errorf("missing root group for template rendering")
	}
	root.mu.RLock()
	templateVars["base_url"] = root.baseURL
	root.mu.RUnlock()

	templateOwner.mu.RLock()
	templateString := templateOwner.urlTemplate
	templateOwner.mu.RUnlock()

	if missing := detectMissingTemplateVars(templateString, templateVars); len(missing) > 0 {
		return "", TemplateSubstitutionError{
			Group:         groupDisplayName(u),
			Route:         routeName,
			TemplateOwner: groupDisplayName(templateOwner),
			Template:      templateString,
			Missing:       append([]string(nil), missing...),
		}
	}

	// Substitute template variables in the template string
	finalURL := SubstituteTemplate(templateString, templateVars)

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

func applyRoutePathSuffix(routePath, suffix string) string {
	if routePath == "" || suffix == "" {
		return routePath
	}

	// Avoid duplicating the suffix if it's already present.
	if strings.HasSuffix(routePath, suffix) {
		return routePath
	}

	// Special case: if the route resolves to root and suffix indicates "/",
	// keep single slash to avoid generating "//".
	if suffix == "/" && routePath == "/" {
		return routePath
	}

	return routePath + suffix
}

var placeholderPattern = regexp.MustCompile(`\{([a-zA-Z0-9_]+)\}`)

func detectMissingTemplateVars(template string, vars map[string]string) []string {
	matches := placeholderPattern.FindAllStringSubmatch(template, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		seen[match[1]] = struct{}{}
	}

	var missing []string
	for key := range seen {
		if _, ok := vars[key]; !ok {
			missing = append(missing, key)
		}
	}

	slices.Sort(missing)
	return missing
}

func appendGroupDebug(builder *strings.Builder, group *Group, depth int) {
	if group == nil {
		return
	}

	group.mu.RLock()
	isRoot := group.parent == nil
	baseURL := group.baseURL
	path := group.path
	template := group.urlTemplate
	routesCopy := make(map[string]string, len(group.routes))
	for key, value := range group.routes {
		routesCopy[key] = value
	}
	childMap := make(map[string]*Group, len(group.children))
	childNames := make([]string, 0, len(group.children))
	for name, child := range group.children {
		childMap[name] = child
		childNames = append(childNames, name)
	}
	group.mu.RUnlock()

	indent := strings.Repeat("  ", depth)
	displayName := groupDisplayName(group)
	if displayName == "" {
		displayName = "(unnamed)"
	}

	meta := make([]string, 0, 2)
	if isRoot {
		meta = append(meta, fmt.Sprintf("base=%q", baseURL))
	}
	if path != "" {
		meta = append(meta, fmt.Sprintf("path=%q", path))
	}

	builder.WriteString(indent)
	builder.WriteString("- ")
	builder.WriteString(displayName)
	if len(meta) > 0 {
		builder.WriteString(" (")
		builder.WriteString(strings.Join(meta, ", "))
		builder.WriteString(")")
	}
	builder.WriteByte('\n')

	if template != "" {
		fmt.Fprintf(builder, "%s  template: %q\n", indent, template)
	}

	if vars := group.CollectTemplateVars(); len(vars) > 0 {
		keys := slices.Sorted(maps.Keys(vars))
		fmt.Fprintf(builder, "%s  vars:\n", indent)
		for _, key := range keys {
			fmt.Fprintf(builder, "%s    %s = %q\n", indent, key, vars[key])
		}
	}

	if len(routesCopy) > 0 {
		routeNames := slices.Sorted(maps.Keys(routesCopy))
		fmt.Fprintf(builder, "%s  routes:\n", indent)
		for _, route := range routeNames {
			fmt.Fprintf(builder, "%s    - %s: %s\n", indent, route, routesCopy[route])
		}
	}

	if len(childNames) == 0 {
		return
	}

	slices.Sort(childNames)
	for idx, childName := range childNames {
		appendGroupDebug(builder, childMap[childName], depth+1)
		if idx < len(childNames)-1 {
			builder.WriteByte('\n')
		}
	}
}
