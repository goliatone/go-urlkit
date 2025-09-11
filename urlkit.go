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

type GroupConfig struct {
	Name    string            `json:"name"`
	BaseURL string            `json:"base_url,omitempty"`
	Path    string            `json:"path,omitempty"`
	Paths   map[string]string `json:"paths"`
	Groups  []GroupConfig     `json:"groups,omitempty"`
}

func NewRouteManagerFromConfig(config Config) *RouteManager {
	manager := NewRouteManager()
	for _, groupConfig := range config.Groups {
		manager.RegisterGroup(groupConfig.Name, groupConfig.BaseURL, groupConfig.Paths)
		rootGroup := manager.Group(groupConfig.Name)
		manager.parseNestedGroups(groupConfig, rootGroup)
	}
	return manager
}

// parseNestedGroups recursively processes nested groups in the configuration
func (m *RouteManager) parseNestedGroups(config GroupConfig, parent *Group) {
	for _, childConfig := range config.Groups {
		childGroup := parent.RegisterGroup(childConfig.Name, childConfig.Path, childConfig.Paths)

		if childConfig.BaseURL != "" {
			panic(fmt.Errorf("nested group %s cannot specify base_url, only root groups can have base URLs", childConfig.Name))
		}

		m.parseNestedGroups(childConfig, childGroup)
	}
}

func NewRouteManager() *RouteManager {
	return &RouteManager{
		groups: map[string]*Group{},
	}
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

type Group struct {
	// Existing fields
	baseURL        string
	routes         map[string]string
	compiledRoutes map[string]func(any) (string, error)

	// New fields for hierarchical support
	name     string            // The name of this group relative to its parent
	path     string            // The path prefix for this group (e.g., "/en", "/v1")
	parent   *Group            // Pointer to parent group (nil for root groups)
	children map[string]*Group // Map of child groups
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
		// Initialize new fields
		name:     "", // Root groups have empty name
		path:     "", // Root groups have empty path
		parent:   nil,
		children: make(map[string]*Group),
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

	// Compile route template with parameters
	routePath, err := compiled(params)
	if err != nil {
		return "", fmt.Errorf("failed to build route: %s", err)
	}

	// Build hierarchical path by walking up the parent chain
	fullPath := u.getFullPath() + routePath

	// Get base URL from root group
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
// It panics if the child group is not found, consistent with RouteManager.Group behavior.
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

	// Recursively build path from parent chain
	parentPath := u.parent.getFullPath()
	return parentPath + u.path
}

// getRootGroup finds and returns the root group by traversing up the parent chain.
func (u *Group) getRootGroup() *Group {
	if u.parent == nil {
		return u // This is the root group
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
		// Create new child group
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
		}

		u.children[name] = childGroup
		return childGroup
	}
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
