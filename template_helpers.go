package urlkit

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/flosch/pongo2/v6"
)

// TemplateHelperConfig defines configuration for template helpers
type TemplateHelperConfig struct {
	// Error reporting configuration
	EnableStructuredErrors bool // When true, returns JSON error objects instead of simple strings
	EnableErrorLogging     bool // When true, logs errors for production debugging
}

// LocaleConfig defines configuration for localization helpers
type LocaleConfig struct {
	// Default locale to use when no locale can be detected
	DefaultLocale string

	// List of supported locales (e.g., ["en", "es", "fr"])
	SupportedLocales []string

	// Map of group names to their supported locales
	// Allows fine-grained control over which locales are available for which groups
	// e.g., "frontend" -> ["en", "es"], "api" -> ["en", "fr", "de"]
	LocaleGroups map[string][]string

	// Custom locale detection function (legacy, for backward compatibility)
	// Takes template context data and returns detected locale
	LocaleDetector func(context any) string

	// Locale detection strategies in priority order
	// Uses the new multi-strategy detection system
	DetectionStrategies []LocaleDetectionStrategy

	// Locale fallback strategy
	// When true, falls back to DefaultLocale if detected locale is not supported
	// When false, returns error for unsupported locales
	EnableLocaleFallback bool

	// Hierarchical locale group structure
	// When true, supports URLKit's hierarchical groups for locale organization
	// e.g., frontend.en.help, frontend.es.help
	EnableHierarchicalLocales bool

	// Locale validation options
	EnableLocaleValidation bool // Validate detected locales against supported list
}

// DefaultTemplateHelperConfig returns default configuration
func DefaultTemplateHelperConfig() *TemplateHelperConfig {
	return &TemplateHelperConfig{
		EnableStructuredErrors: false, // Default to simple error strings for backwards compatibility
		EnableErrorLogging:     false, // Default to no logging
	}
}

// DefaultLocaleConfig returns default locale configuration
func DefaultLocaleConfig() *LocaleConfig {
	return &LocaleConfig{
		DefaultLocale:             "en",
		SupportedLocales:          []string{"en"},
		LocaleGroups:              make(map[string][]string),
		LocaleDetector:            defaultLocaleDetector,                        // Legacy support
		DetectionStrategies:       []LocaleDetectionStrategy{LocaleFromContext}, // Default to context-based
		EnableLocaleFallback:      true,
		EnableHierarchicalLocales: false,
		EnableLocaleValidation:    true,
	}
}

// LocaleDetectionStrategy defines different locale detection strategies
type LocaleDetectionStrategy int

const (
	// LocaleFromContext extracts locale from template context data
	LocaleFromContext LocaleDetectionStrategy = iota
	// LocaleFromURL parses locale from URL path (e.g., /en/path)
	LocaleFromURL
	// LocaleFromHeader extracts from Accept-Language header
	LocaleFromHeader
	// LocaleFromCookie extracts from locale cookie
	LocaleFromCookie
)

// LocaleDetectionContext provides context for locale detection
type LocaleDetectionContext struct {
	// TemplateContext is the template data passed to rendering
	TemplateContext map[string]any
	// URLPath is the current request URL path for URL-based detection
	URLPath string
	// AcceptLanguage is the Accept-Language header value
	AcceptLanguage string
	// CookieLocale is the locale from cookie
	CookieLocale string
	// DefaultLocale is the fallback locale
	DefaultLocale string
}

// defaultLocaleDetector is the default locale detection function
// It looks for locale in template context data under common keys
func defaultLocaleDetector(context any) string {
	if context == nil {
		return ""
	}

	// Try to cast context to map[string]any
	contextMap, ok := context.(map[string]any)
	if !ok {
		return ""
	}

	// Look for locale in common context keys
	localeKeys := []string{"locale", "lang", "language", "current_locale"}
	for _, key := range localeKeys {
		if locale, exists := contextMap[key]; exists {
			if localeStr, ok := locale.(string); ok && localeStr != "" {
				return localeStr
			}
		}
	}

	return ""
}

// contextBasedLocaleDetector extracts locale from template context with enhanced key search
func contextBasedLocaleDetector(detectionContext *LocaleDetectionContext) string {
	if detectionContext == nil || detectionContext.TemplateContext == nil {
		return ""
	}

	// Enhanced locale key search with priority order
	localeKeys := []string{
		"locale",         // Primary
		"current_locale", // Explicit current
		"user_locale",    // User preference
		"lang",           // Short form
		"language",       // Full form
		"i18n_locale",    // Internationalization
		"request_locale", // Request-specific
	}

	for _, key := range localeKeys {
		if locale, exists := detectionContext.TemplateContext[key]; exists {
			if localeStr, ok := locale.(string); ok && localeStr != "" {
				return localeStr
			}
		}
	}

	return ""
}

// urlBasedLocaleDetector extracts locale from URL path
// Supports patterns like: /en/path, /locale/en/path
func urlBasedLocaleDetector(detectionContext *LocaleDetectionContext, supportedLocales []string) string {
	if detectionContext == nil || detectionContext.URLPath == "" {
		return ""
	}

	path := strings.TrimPrefix(detectionContext.URLPath, "/")
	parts := strings.Split(path, "/")

	if len(parts) == 0 {
		return ""
	}

	// Check first path segment as locale
	firstSegment := parts[0]
	for _, locale := range supportedLocales {
		if firstSegment == locale {
			return locale
		}
	}

	// Check for /locale/{locale} pattern
	if len(parts) >= 2 && parts[0] == "locale" {
		localeSegment := parts[1]
		for _, locale := range supportedLocales {
			if localeSegment == locale {
				return locale
			}
		}
	}

	return ""
}

// headerBasedLocaleDetector extracts locale from Accept-Language header
func headerBasedLocaleDetector(detectionContext *LocaleDetectionContext, supportedLocales []string) string {
	if detectionContext == nil || detectionContext.AcceptLanguage == "" {
		return ""
	}

	// Parse Accept-Language header (simple implementation)
	// Format: en-US,en;q=0.9,es;q=0.8
	languages := strings.Split(detectionContext.AcceptLanguage, ",")

	for _, lang := range languages {
		// Remove quality factor (;q=0.9)
		langCode := strings.TrimSpace(strings.Split(lang, ";")[0])

		// Check exact match first
		for _, locale := range supportedLocales {
			if langCode == locale {
				return locale
			}
		}

		// Check language prefix (en-US -> en)
		if strings.Contains(langCode, "-") {
			prefix := strings.Split(langCode, "-")[0]
			for _, locale := range supportedLocales {
				if prefix == locale {
					return locale
				}
			}
		}
	}

	return ""
}

// cookieBasedLocaleDetector extracts locale from cookie
func cookieBasedLocaleDetector(detectionContext *LocaleDetectionContext, supportedLocales []string) string {
	if detectionContext == nil || detectionContext.CookieLocale == "" {
		return ""
	}

	// Validate cookie locale against supported locales
	for _, locale := range supportedLocales {
		if detectionContext.CookieLocale == locale {
			return locale
		}
	}

	return ""
}

// multiStrategyLocaleDetector combines multiple detection strategies with priority order
func multiStrategyLocaleDetector(detectionContext *LocaleDetectionContext, supportedLocales []string, strategies []LocaleDetectionStrategy) string {
	if detectionContext == nil {
		return ""
	}

	for _, strategy := range strategies {
		var detected string

		switch strategy {
		case LocaleFromContext:
			detected = contextBasedLocaleDetector(detectionContext)
		case LocaleFromURL:
			detected = urlBasedLocaleDetector(detectionContext, supportedLocales)
		case LocaleFromHeader:
			detected = headerBasedLocaleDetector(detectionContext, supportedLocales)
		case LocaleFromCookie:
			detected = cookieBasedLocaleDetector(detectionContext, supportedLocales)
		}

		if detected != "" {
			return detected
		}
	}

	// Fallback to default
	return detectionContext.DefaultLocale
}

// isLocaleSupported checks if a locale is supported globally or for a specific group
func (c *LocaleConfig) isLocaleSupported(locale string, groupName string) bool {
	if locale == "" {
		return false
	}

	// Check group-specific locales first
	if groupName != "" {
		if groupLocales, exists := c.LocaleGroups[groupName]; exists {
			for _, supportedLocale := range groupLocales {
				if supportedLocale == locale {
					return true
				}
			}
			// If group has specific locales defined, only those are supported for this group
			return false
		}
	}

	// Fall back to global supported locales
	for _, supportedLocale := range c.SupportedLocales {
		if supportedLocale == locale {
			return true
		}
	}

	return false
}

// detectLocale detects locale from context with fallback support
func (c *LocaleConfig) detectLocale(context any, groupName string) string {
	var detectedLocale string

	// Use new multi-strategy detection if strategies are configured
	if len(c.DetectionStrategies) > 0 {
		detectionContext := c.buildDetectionContext(context)
		supportedLocales := c.getSupportedLocalesForGroup(groupName)
		detectedLocale = multiStrategyLocaleDetector(detectionContext, supportedLocales, c.DetectionStrategies)
	} else if c.LocaleDetector != nil {
		// Fall back to legacy custom detector for backward compatibility
		detectedLocale = c.LocaleDetector(context)
	}

	// Validate detected locale if validation is enabled
	if c.EnableLocaleValidation && detectedLocale != "" && !c.isLocaleSupported(detectedLocale, groupName) {
		if c.EnableLocaleFallback && c.isLocaleSupported(c.DefaultLocale, groupName) {
			return c.DefaultLocale
		}
		// If validation enabled but locale not supported and no fallback, return empty
		return ""
	}

	// Check if detected locale is supported
	if detectedLocale != "" && c.isLocaleSupported(detectedLocale, groupName) {
		return detectedLocale
	}

	// Apply fallback strategy
	if c.EnableLocaleFallback && c.isLocaleSupported(c.DefaultLocale, groupName) {
		return c.DefaultLocale
	}

	// If no fallback, return detected locale even if unsupported (will cause error later)
	if detectedLocale != "" {
		return detectedLocale
	}

	// Last resort: return default locale
	return c.DefaultLocale
}

// buildDetectionContext creates a LocaleDetectionContext from template context
func (c *LocaleConfig) buildDetectionContext(context any) *LocaleDetectionContext {
	detectionContext := &LocaleDetectionContext{
		DefaultLocale: c.DefaultLocale,
	}

	// Try to extract context data
	if contextMap, ok := context.(map[string]any); ok {
		detectionContext.TemplateContext = contextMap

		// Extract additional context information if available
		if urlPath, exists := contextMap["url_path"]; exists {
			if urlPathStr, ok := urlPath.(string); ok {
				detectionContext.URLPath = urlPathStr
			}
		}

		if acceptLang, exists := contextMap["accept_language"]; exists {
			if acceptLangStr, ok := acceptLang.(string); ok {
				detectionContext.AcceptLanguage = acceptLangStr
			}
		}

		if cookieLocale, exists := contextMap["cookie_locale"]; exists {
			if cookieLocaleStr, ok := cookieLocale.(string); ok {
				detectionContext.CookieLocale = cookieLocaleStr
			}
		}
	}

	return detectionContext
}

// NewMultiStrategyLocaleConfig creates a LocaleConfig with multiple detection strategies
func NewMultiStrategyLocaleConfig(defaultLocale string, supportedLocales []string, strategies []LocaleDetectionStrategy) *LocaleConfig {
	return &LocaleConfig{
		DefaultLocale:             defaultLocale,
		SupportedLocales:          supportedLocales,
		LocaleGroups:              make(map[string][]string),
		DetectionStrategies:       strategies,
		EnableLocaleFallback:      true,
		EnableHierarchicalLocales: false,
		EnableLocaleValidation:    true,
	}
}

// NewURLBasedLocaleConfig creates a LocaleConfig optimized for URL-based locale detection
func NewURLBasedLocaleConfig(defaultLocale string, supportedLocales []string) *LocaleConfig {
	return &LocaleConfig{
		DefaultLocale:             defaultLocale,
		SupportedLocales:          supportedLocales,
		LocaleGroups:              make(map[string][]string),
		DetectionStrategies:       []LocaleDetectionStrategy{LocaleFromURL, LocaleFromContext, LocaleFromCookie},
		EnableLocaleFallback:      true,
		EnableHierarchicalLocales: true, // URL-based usually uses hierarchical structure
		EnableLocaleValidation:    true,
	}
}

// NewHeaderBasedLocaleConfig creates a LocaleConfig optimized for Accept-Language header detection
func NewHeaderBasedLocaleConfig(defaultLocale string, supportedLocales []string) *LocaleConfig {
	return &LocaleConfig{
		DefaultLocale:             defaultLocale,
		SupportedLocales:          supportedLocales,
		LocaleGroups:              make(map[string][]string),
		DetectionStrategies:       []LocaleDetectionStrategy{LocaleFromHeader, LocaleFromCookie, LocaleFromContext},
		EnableLocaleFallback:      true,
		EnableHierarchicalLocales: false,
		EnableLocaleValidation:    true,
	}
}

// NewFullStackLocaleConfig creates a comprehensive LocaleConfig with all detection strategies
func NewFullStackLocaleConfig(defaultLocale string, supportedLocales []string) *LocaleConfig {
	return &LocaleConfig{
		DefaultLocale:    defaultLocale,
		SupportedLocales: supportedLocales,
		LocaleGroups:     make(map[string][]string),
		DetectionStrategies: []LocaleDetectionStrategy{
			LocaleFromContext, // Highest priority: explicit template context
			LocaleFromURL,     // Second: URL path-based
			LocaleFromCookie,  // Third: persistent user preference
			LocaleFromHeader,  // Fourth: browser preference
		},
		EnableLocaleFallback:      true,
		EnableHierarchicalLocales: true,
		EnableLocaleValidation:    true,
	}
}

// ValidateLocaleConfig validates the locale configuration
func (c *LocaleConfig) ValidateLocaleConfig() error {
	if c.DefaultLocale == "" {
		return fmt.Errorf("default locale cannot be empty")
	}

	if len(c.SupportedLocales) == 0 {
		return fmt.Errorf("supported locales cannot be empty")
	}

	// Check if default locale is in supported locales
	defaultSupported := false
	for _, locale := range c.SupportedLocales {
		if locale == c.DefaultLocale {
			defaultSupported = true
			break
		}
	}
	if !defaultSupported {
		return fmt.Errorf("default locale '%s' must be in supported locales list", c.DefaultLocale)
	}

	// Validate group-specific locales
	for groupName, groupLocales := range c.LocaleGroups {
		if len(groupLocales) == 0 {
			return fmt.Errorf("group '%s' has empty locale list", groupName)
		}
		for _, groupLocale := range groupLocales {
			// Check if group locale is in global supported locales
			found := false
			for _, supportedLocale := range c.SupportedLocales {
				if groupLocale == supportedLocale {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("group '%s' locale '%s' is not in global supported locales", groupName, groupLocale)
			}
		}
	}

	return nil
}

// getSupportedLocalesForGroup returns the list of supported locales for a specific group
func (c *LocaleConfig) getSupportedLocalesForGroup(groupName string) []string {
	if groupLocales, exists := c.LocaleGroups[groupName]; exists {
		return groupLocales
	}
	return c.SupportedLocales
}

// LocaleInfo represents locale information for template helpers
type LocaleInfo struct {
	Locale string `json:"locale"`
	URL    string `json:"url"`
}

// TemplateHelpers returns a map of template helper functions for use with template engines
func TemplateHelpers(manager *RouteManager, config *TemplateHelperConfig) map[string]any {
	if config == nil {
		config = DefaultTemplateHelperConfig()
	}

	helpers := make(map[string]any)

	// Core URL helpers (wrapped with panic recovery)
	helpers["url"] = safeTemplateHelper("url", config, urlHelper(manager, config))
	helpers["route_path"] = safeTemplateHelper("route_path", config, routePathHelper(manager, config))
	helpers["has_route"] = safeTemplateHelper("has_route", config, hasRouteHelper(manager, config))
	helpers["route_template"] = safeTemplateHelper("route_template", config, routeTemplateHelper(manager, config))
	helpers["route_vars"] = safeTemplateHelper("route_vars", config, routeVarsHelper(manager, config))
	helpers["route_exists"] = safeTemplateHelper("route_exists", config, routeExistsHelper(manager, config))
	helpers["url_abs"] = safeTemplateHelper("url_abs", config, urlAbsHelper(manager, config))

	// Contextual Helper Functions (work with middleware-injected context)
	helpers["current_route_if"] = safeTemplateHelper("current_route_if", config, currentRouteIfHelper(config))

	return helpers
}

// TemplateHelpersWithLocale returns a map of template helper functions with localization support
// The returned map can be passed to template.WithTemplateFunc() during engine initialization.
//
// Usage:
//
//	manager := NewRouteManager()
//	config := DefaultTemplateHelperConfig()
//	localeConfig := DefaultLocaleConfig()
//	localeConfig.SupportedLocales = []string{"en", "es", "fr"}
//	localeConfig.LocaleGroups["frontend"] = []string{"en", "es"}
//	renderer, err := template.NewRenderer(
//	    template.WithTemplateFunc(urlkit.TemplateHelpersWithLocale(manager, config, localeConfig)),
//	)
//
// Template usage:
//
//	{{ url_i18n('frontend', 'user_profile', {'id': user.id}) }}
//	{{ url_locale('frontend', 'about', 'es') }}
func TemplateHelpersWithLocale(manager *RouteManager, config *TemplateHelperConfig, localeConfig *LocaleConfig) map[string]any {
	if config == nil {
		config = DefaultTemplateHelperConfig()
	}
	if localeConfig == nil {
		localeConfig = DefaultLocaleConfig()
	}

	// Start with standard helpers
	helpers := TemplateHelpers(manager, config)

	// Add localization helpers (wrapped with panic recovery)
	helpers["url_i18n"] = safeTemplateHelper("url_i18n", config, urlI18nHelper(manager, config, localeConfig))
	helpers["url_locale"] = safeTemplateHelper("url_locale", config, urlLocaleHelper(manager, config, localeConfig))
	helpers["url_all_locales"] = safeTemplateHelper("url_all_locales", config, urlAllLocalesHelper(manager, config, localeConfig))
	helpers["has_locale"] = safeTemplateHelper("has_locale", config, hasLocaleHelper(manager, config, localeConfig))
	helpers["current_locale"] = safeTemplateHelper("current_locale", config, currentLocaleHelper(config, localeConfig))

	return helpers
}

// TemplateError represents structured error information for template helpers
type TemplateError struct {
	Helper  string         `json:"helper"`
	Type    string         `json:"type"`
	Message string         `json:"message"`
	Context map[string]any `json:"context,omitempty"`
}

// formatError creates appropriate error response based on configuration
func formatError(helper, errorType, message string, context map[string]any, config *TemplateHelperConfig) *pongo2.Value {
	if config.EnableStructuredErrors {
		errorObj := TemplateError{
			Helper:  helper,
			Type:    errorType,
			Message: message,
			Context: context,
		}
		return pongo2.AsValue(errorObj)
	}

	// Simple error format for backwards compatibility
	var parts []string
	parts = append(parts, "#error", helper, errorType, message)
	errorMsg := strings.Join(parts, ":")

	// Log error if logging is enabled
	if config.EnableErrorLogging {
		fmt.Printf("[URLKit Template Helper Error] %s: %s (Context: %+v)\n", helper, message, context)
	}

	return pongo2.AsValue(errorMsg)
}

// safeGroupAccess safely accesses a group without panicking
func safeGroupAccess(manager *RouteManager, groupName string) *Group {
	defer func() {
		if r := recover(); r != nil {
			// Group access panicked, group doesn't exist
		}
	}()

	return manager.Group(groupName)
}

// safeTemplateHelper wraps template helper functions with comprehensive panic recovery
func safeTemplateHelper(helperName string, config *TemplateHelperConfig, helperFunc func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error)) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (result *pongo2.Value, err *pongo2.Error) {
		defer func() {
			if r := recover(); r != nil {
				// Panic occurred during template helper execution
				context := map[string]any{
					"panic_value": fmt.Sprintf("%v", r),
					"args_count":  len(args),
				}

				// Log the panic if logging is enabled
				if config.EnableErrorLogging {
					fmt.Printf("[URLKit Template Helper Panic] %s: %v\n", helperName, r)
				}

				// Return a graceful error instead of crashing the template engine
				errorMsg := fmt.Sprintf("template helper '%s' encountered an unexpected error", helperName)
				result = formatError(helperName, "panic_recovered", errorMsg, context, config)
				err = nil
			}
		}()

		return helperFunc(args...)
	}
}

// urlHelperArgs represents parsed arguments for URL helpers
type urlHelperArgs struct {
	Group  string
	Route  string
	Params map[string]any
	Query  map[string]string
}

// parseArgs parses variadic pongo2.Value arguments into structured data
func parseArgs(args ...*pongo2.Value) (*urlHelperArgs, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("at least 2 arguments required: group and route")
	}

	result := &urlHelperArgs{
		Params: make(map[string]any),
		Query:  make(map[string]string),
	}

	// First argument: group name
	groupVal := fromPongoValue(args[0])
	if group, ok := groupVal.(string); ok {
		result.Group = group
	} else {
		return nil, fmt.Errorf("group must be a string")
	}

	// Second argument: route name
	routeVal := fromPongoValue(args[1])
	if route, ok := routeVal.(string); ok {
		result.Route = route
	} else {
		return nil, fmt.Errorf("route must be a string")
	}

	// Optional third argument: params map
	if len(args) > 2 && args[2] != nil {
		paramsVal := fromPongoValue(args[2])
		if params, ok := paramsVal.(map[string]any); ok {
			result.Params = params
		} else if paramsVal != nil {
			return nil, fmt.Errorf("params must be a map")
		}
	}

	// Optional fourth argument: query map
	if len(args) > 3 && args[3] != nil {
		queryVal := fromPongoValue(args[3])
		if queryMap, ok := queryVal.(map[string]any); ok {
			// Convert to map[string]string
			for k, v := range queryMap {
				if str, ok := v.(string); ok {
					result.Query[k] = str
				} else if v != nil {
					result.Query[k] = fmt.Sprintf("%v", v)
				}
			}
		} else if queryVal != nil {
			return nil, fmt.Errorf("query must be a map")
		}
	}

	return result, nil
}

// fromPongoValue recursively converts pongo2.Value to Go native types
func fromPongoValue(val *pongo2.Value) any {
	if val == nil {
		return nil
	}

	// Get the any from pongo2.Value
	iface := val.Interface()
	if iface == nil {
		return nil
	}

	// Handle different types
	switch v := iface.(type) {
	case string:
		return v
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return v
	case bool:
		return v
	case map[string]any:
		return v
	case []any:
		return v
	default:
		// Try to convert using reflection for other types
		rv := reflect.ValueOf(iface)
		switch rv.Kind() {
		case reflect.String:
			return rv.String()
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return int(rv.Int())
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			return int(rv.Uint())
		case reflect.Float32, reflect.Float64:
			return rv.Float()
		case reflect.Bool:
			return rv.Bool()
		case reflect.Map:
			if rv.Type().Key().Kind() == reflect.String {
				result := make(map[string]any)
				for _, key := range rv.MapKeys() {
					keyStr := key.String()
					value := rv.MapIndex(key).Interface()
					result[keyStr] = value
				}
				return result
			}
		}

		// Fallback to string representation
		return fmt.Sprintf("%v", iface)
	}
}

// urlHelper returns a template function that generates complete URLs
func urlHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		parsedArgs, err := parseArgs(args...)
		if err != nil {
			return formatError("url", "parse_error", err.Error(), map[string]any{"args_count": len(args)}, config), nil
		}

		// Get the group safely
		group := safeGroupAccess(manager, parsedArgs.Group)
		if group == nil {
			context := map[string]any{
				"group_name": parsedArgs.Group,
			}
			return formatError("url", "group_not_found", fmt.Sprintf("group '%s' not found", parsedArgs.Group), context, config), nil
		}

		// Build the URL using the fluent API
		builder := group.Builder(parsedArgs.Route)
		if builder == nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": parsedArgs.Group,
			}
			return formatError("url", "route_not_found", fmt.Sprintf("route '%s' not found in group '%s'", parsedArgs.Route, parsedArgs.Group), context, config), nil
		}

		// Add parameters if provided
		if len(parsedArgs.Params) > 0 {
			for key, value := range parsedArgs.Params {
				builder = builder.WithParam(key, value)
			}
		}

		// Add query parameters if provided
		if len(parsedArgs.Query) > 0 {
			for key, value := range parsedArgs.Query {
				builder = builder.WithQuery(key, value)
			}
		}

		// Build the final URL
		url, err := builder.Build()
		if err != nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": parsedArgs.Group,
				"params":     parsedArgs.Params,
				"query":      parsedArgs.Query,
			}
			return formatError("url", "build_error", err.Error(), context, config), nil
		}

		return pongo2.AsValue(url), nil
	}
}

// routePathHelper returns a template function that generates URL paths (without base URL)
func routePathHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		parsedArgs, err := parseArgs(args...)
		if err != nil {
			return formatError("route_path", "parse_error", err.Error(), map[string]any{"args_count": len(args)}, config), nil
		}

		// Get the group safely
		group := safeGroupAccess(manager, parsedArgs.Group)
		if group == nil {
			context := map[string]any{
				"group_name": parsedArgs.Group,
			}
			return formatError("route_path", "group_not_found", fmt.Sprintf("group '%s' not found", parsedArgs.Group), context, config), nil
		}

		// Build the URL using the fluent API
		builder := group.Builder(parsedArgs.Route)
		if builder == nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": parsedArgs.Group,
			}
			return formatError("route_path", "route_not_found", fmt.Sprintf("route '%s' not found in group '%s'", parsedArgs.Route, parsedArgs.Group), context, config), nil
		}

		// Add parameters if provided
		if len(parsedArgs.Params) > 0 {
			for key, value := range parsedArgs.Params {
				builder = builder.WithParam(key, value)
			}
		}

		// Add query parameters if provided
		if len(parsedArgs.Query) > 0 {
			for key, value := range parsedArgs.Query {
				builder = builder.WithQuery(key, value)
			}
		}

		// Build the URL path only (this would need a method in URLKit to get path without base URL)
		url, err := builder.Build()
		if err != nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": parsedArgs.Group,
				"params":     parsedArgs.Params,
				"query":      parsedArgs.Query,
			}
			return formatError("route_path", "build_error", err.Error(), context, config), nil
		}

		// For now, return the full URL (this can be enhanced later to strip base URL)
		return pongo2.AsValue(url), nil
	}
}

// hasRouteHelper returns a template function that checks if a route exists
func hasRouteHelper(manager *RouteManager, _ *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 2 {
			return pongo2.AsValue(false), nil
		}

		groupVal := fromPongoValue(args[0])
		routeVal := fromPongoValue(args[1])

		groupName, ok1 := groupVal.(string)
		routeName, ok2 := routeVal.(string)

		if !ok1 || !ok2 {
			return pongo2.AsValue(false), nil
		}

		// Check if group exists safely
		group := safeGroupAccess(manager, groupName)
		if group == nil {
			return pongo2.AsValue(false), nil
		}

		// Check if route exists in group
		// Use Route method instead of Builder to avoid potential panic
		_, err := group.Route(routeName)
		exists := err == nil

		return pongo2.AsValue(exists), nil
	}
}

// routeTemplateHelper returns a template function that returns the raw route template string
func routeTemplateHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 2 {
			return formatError("route_template", "insufficient_args", "at least 2 arguments required: group and route", map[string]any{"args_count": len(args)}, config), nil
		}

		groupVal := fromPongoValue(args[0])
		routeVal := fromPongoValue(args[1])

		groupName, ok1 := groupVal.(string)
		routeName, ok2 := routeVal.(string)

		if !ok1 || !ok2 {
			context := map[string]any{
				"group_type": fmt.Sprintf("%T", groupVal),
				"route_type": fmt.Sprintf("%T", routeVal),
			}
			return formatError("route_template", "invalid_args", "group and route must be strings", context, config), nil
		}

		// Get the group safely
		group := safeGroupAccess(manager, groupName)
		if group == nil {
			context := map[string]any{
				"group_name": groupName,
			}
			return formatError("route_template", "group_not_found", fmt.Sprintf("group '%s' not found", groupName), context, config), nil
		}

		// Get the route template
		template, err := group.Route(routeName)
		if err != nil {
			context := map[string]any{
				"route_name": routeName,
				"group_name": groupName,
			}
			return formatError("route_template", "route_not_found", fmt.Sprintf("route '%s' not found in group '%s'", routeName, groupName), context, config), nil
		}

		return pongo2.AsValue(template), nil
	}
}

// routeVarsHelper returns a template function that returns template variables for debugging
func routeVarsHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 1 {
			return formatError("route_vars", "insufficient_args", "at least 1 argument required: group", map[string]any{"args_count": len(args)}, config), nil
		}

		groupVal := fromPongoValue(args[0])
		groupName, ok := groupVal.(string)

		if !ok {
			context := map[string]any{
				"group_type": fmt.Sprintf("%T", groupVal),
			}
			return formatError("route_vars", "invalid_args", "group must be a string", context, config), nil
		}

		// Get the group safely
		group := safeGroupAccess(manager, groupName)
		if group == nil {
			context := map[string]any{
				"group_name": groupName,
			}
			return formatError("route_vars", "group_not_found", fmt.Sprintf("group '%s' not found", groupName), context, config), nil
		}

		// Get the template variables
		vars := group.CollectTemplateVars()

		// Convert to map[string]any for template use
		result := make(map[string]any)
		for k, v := range vars {
			result[k] = v
		}

		return pongo2.AsValue(result), nil
	}
}

// routeExistsHelper returns a template function that checks if an entire route group exists
func routeExistsHelper(manager *RouteManager, _ *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 1 {
			return pongo2.AsValue(false), nil
		}

		groupVal := fromPongoValue(args[0])
		groupName, ok := groupVal.(string)

		if !ok {
			return pongo2.AsValue(false), nil
		}

		// Check if group exists safely
		group := safeGroupAccess(manager, groupName)
		exists := group != nil

		return pongo2.AsValue(exists), nil
	}
}

// urlAbsHelper returns a template function that forces absolute URL generation with base URL
func urlAbsHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		parsedArgs, err := parseArgs(args...)
		if err != nil {
			return formatError("url_abs", "parse_error", err.Error(), map[string]any{"args_count": len(args)}, config), nil
		}

		// Get the group safely
		group := safeGroupAccess(manager, parsedArgs.Group)
		if group == nil {
			context := map[string]any{
				"group_name": parsedArgs.Group,
			}
			return formatError("url_abs", "group_not_found", fmt.Sprintf("group '%s' not found", parsedArgs.Group), context, config), nil
		}

		// Build the URL using the fluent API
		builder := group.Builder(parsedArgs.Route)
		if builder == nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": parsedArgs.Group,
			}
			return formatError("url_abs", "route_not_found", fmt.Sprintf("route '%s' not found in group '%s'", parsedArgs.Route, parsedArgs.Group), context, config), nil
		}

		// Add parameters if provided
		if len(parsedArgs.Params) > 0 {
			for key, value := range parsedArgs.Params {
				builder = builder.WithParam(key, value)
			}
		}

		// Add query parameters if provided
		if len(parsedArgs.Query) > 0 {
			for key, value := range parsedArgs.Query {
				builder = builder.WithQuery(key, value)
			}
		}

		// Build the final URL - this already includes the base URL by default
		// The url_abs helper is essentially the same as url helper for now
		// but it's here for explicit absolute URL generation semantics
		url, err := builder.Build()
		if err != nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": parsedArgs.Group,
				"params":     parsedArgs.Params,
				"query":      parsedArgs.Query,
			}
			return formatError("url_abs", "build_error", err.Error(), context, config), nil
		}

		return pongo2.AsValue(url), nil
	}
}

// currentRouteIfHelper returns a template function that conditionally returns values based on route matching
// Signature: current_route_if(targetRoute, currentRoute, valueIfTrue, [valueIfFalse])
// This helper works with middleware-injected context data passed as template variables
func currentRouteIfHelper(config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 3 {
			return formatError("current_route_if", "insufficient_args", "requires targetRoute, currentRoute, valueIfTrue", map[string]any{"args_count": len(args)}, config), nil
		}

		targetRouteVal := fromPongoValue(args[0])
		currentRouteVal := fromPongoValue(args[1])
		valueIfTrueVal := fromPongoValue(args[2])

		targetRoute, ok1 := targetRouteVal.(string)
		currentRoute, ok2 := currentRouteVal.(string)

		if !ok1 || !ok2 {
			context := map[string]any{
				"target_route_type":  fmt.Sprintf("%T", targetRouteVal),
				"current_route_type": fmt.Sprintf("%T", currentRouteVal),
			}
			return formatError("current_route_if", "invalid_args", "targetRoute and currentRoute must be strings", context, config), nil
		}

		var valueIfFalse any = ""
		if len(args) > 3 {
			valueIfFalse = fromPongoValue(args[3])
		}

		if targetRoute == currentRoute {
			return pongo2.AsValue(valueIfTrueVal), nil
		}

		return pongo2.AsValue(valueIfFalse), nil
	}
}

// urlI18nHelper returns a template function that generates URLs with automatic locale detection from context
// Template usage: {{ url_i18n('frontend', 'user_profile', {'id': user.id}) }}
func urlI18nHelper(manager *RouteManager, config *TemplateHelperConfig, localeConfig *LocaleConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		parsedArgs, err := parseArgs(args...)
		if err != nil {
			return formatError("url_i18n", "parse_error", err.Error(), map[string]any{"args_count": len(args)}, config), nil
		}

		// Get template context for locale detection (if available)
		// For now, we'll detect locale from a hypothetical context parameter
		// In practice, this would be injected by middleware into template context
		var detectedLocale string
		if len(args) > 4 {
			// Fifth argument could be context data for locale detection
			contextVal := fromPongoValue(args[4])
			detectedLocale = localeConfig.detectLocale(contextVal, parsedArgs.Group)
		} else {
			// No context provided, use default locale
			detectedLocale = localeConfig.DefaultLocale
		}

		// Check if detected locale is supported for this group
		if !localeConfig.isLocaleSupported(detectedLocale, parsedArgs.Group) {
			if !localeConfig.EnableLocaleFallback {
				context := map[string]any{
					"group_name":      parsedArgs.Group,
					"detected_locale": detectedLocale,
					"supported":       localeConfig.getSupportedLocalesForGroup(parsedArgs.Group),
				}
				return formatError("url_i18n", "unsupported_locale", fmt.Sprintf("locale '%s' is not supported for group '%s'", detectedLocale, parsedArgs.Group), context, config), nil
			}
			// Fall back to default locale
			detectedLocale = localeConfig.DefaultLocale
		}

		// Construct localized group name
		localizedGroupName := parsedArgs.Group
		if localeConfig.EnableHierarchicalLocales && detectedLocale != "" {
			localizedGroupName = parsedArgs.Group + "." + detectedLocale
		}

		// Get the group safely
		group := safeGroupAccess(manager, localizedGroupName)
		if group == nil {
			// If hierarchical locale group doesn't exist, try the original group
			if localeConfig.EnableHierarchicalLocales {
				group = safeGroupAccess(manager, parsedArgs.Group)
			}
			if group == nil {
				context := map[string]any{
					"group_name":           parsedArgs.Group,
					"localized_group_name": localizedGroupName,
					"locale":               detectedLocale,
				}
				return formatError("url_i18n", "group_not_found", fmt.Sprintf("neither localized group '%s' nor base group '%s' found", localizedGroupName, parsedArgs.Group), context, config), nil
			}
		}

		// Build URL using standard logic
		builder := group.Builder(parsedArgs.Route)
		if builder == nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": localizedGroupName,
				"locale":     detectedLocale,
			}
			return formatError("url_i18n", "route_not_found", fmt.Sprintf("route '%s' not found in group '%s'", parsedArgs.Route, localizedGroupName), context, config), nil
		}

		// Add parameters
		if len(parsedArgs.Params) > 0 {
			for key, value := range parsedArgs.Params {
				builder = builder.WithParam(key, value)
			}
		}

		// Add query parameters
		if len(parsedArgs.Query) > 0 {
			for key, value := range parsedArgs.Query {
				builder = builder.WithQuery(key, value)
			}
		}

		// Build the final URL
		url, err := builder.Build()
		if err != nil {
			context := map[string]any{
				"route_name": parsedArgs.Route,
				"group_name": localizedGroupName,
				"locale":     detectedLocale,
				"params":     parsedArgs.Params,
				"query":      parsedArgs.Query,
			}
			return formatError("url_i18n", "build_error", err.Error(), context, config), nil
		}

		return pongo2.AsValue(url), nil
	}
}

// urlLocaleHelper returns a template function that generates URLs for a specific locale
// Template usage: {{ url_locale('frontend', 'about', 'es', {'id': 1}) }}
func urlLocaleHelper(manager *RouteManager, config *TemplateHelperConfig, localeConfig *LocaleConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 3 {
			return formatError("url_locale", "insufficient_args", "requires group, route, and locale", map[string]any{"args_count": len(args)}, config), nil
		}

		// Parse basic arguments
		groupVal := fromPongoValue(args[0])
		routeVal := fromPongoValue(args[1])
		localeVal := fromPongoValue(args[2])

		groupName, ok1 := groupVal.(string)
		routeName, ok2 := routeVal.(string)
		locale, ok3 := localeVal.(string)

		if !ok1 || !ok2 || !ok3 {
			context := map[string]any{
				"group_type":  fmt.Sprintf("%T", groupVal),
				"route_type":  fmt.Sprintf("%T", routeVal),
				"locale_type": fmt.Sprintf("%T", localeVal),
			}
			return formatError("url_locale", "invalid_args", "group, route, and locale must be strings", context, config), nil
		}

		// Parse optional params and query arguments (args[3] and args[4])
		params := make(map[string]any)
		query := make(map[string]string)

		if len(args) > 3 && args[3] != nil {
			paramsVal := fromPongoValue(args[3])
			if paramsMap, ok := paramsVal.(map[string]any); ok {
				params = paramsMap
			} else if paramsVal != nil {
				context := map[string]any{
					"params_type": fmt.Sprintf("%T", paramsVal),
				}
				return formatError("url_locale", "invalid_params", "params must be a map", context, config), nil
			}
		}

		if len(args) > 4 && args[4] != nil {
			queryVal := fromPongoValue(args[4])
			if queryMap, ok := queryVal.(map[string]any); ok {
				for k, v := range queryMap {
					if str, ok := v.(string); ok {
						query[k] = str
					} else if v != nil {
						query[k] = fmt.Sprintf("%v", v)
					}
				}
			} else if queryVal != nil {
				context := map[string]any{
					"query_type": fmt.Sprintf("%T", queryVal),
				}
				return formatError("url_locale", "invalid_query", "query must be a map", context, config), nil
			}
		}

		// Check if locale is supported for this group
		if !localeConfig.isLocaleSupported(locale, groupName) {
			if !localeConfig.EnableLocaleFallback {
				context := map[string]any{
					"group_name":       groupName,
					"requested_locale": locale,
					"supported":        localeConfig.getSupportedLocalesForGroup(groupName),
				}
				return formatError("url_locale", "unsupported_locale", fmt.Sprintf("locale '%s' is not supported for group '%s'", locale, groupName), context, config), nil
			}
			// Fall back to default locale
			locale = localeConfig.DefaultLocale
		}

		// Construct localized group name
		localizedGroupName := groupName
		if localeConfig.EnableHierarchicalLocales && locale != "" {
			localizedGroupName = groupName + "." + locale
		}

		// Get the group safely
		group := safeGroupAccess(manager, localizedGroupName)
		if group == nil {
			// If hierarchical locale group doesn't exist, try the original group
			if localeConfig.EnableHierarchicalLocales {
				group = safeGroupAccess(manager, groupName)
			}
			if group == nil {
				context := map[string]any{
					"group_name":           groupName,
					"localized_group_name": localizedGroupName,
					"locale":               locale,
				}
				return formatError("url_locale", "group_not_found", fmt.Sprintf("neither localized group '%s' nor base group '%s' found", localizedGroupName, groupName), context, config), nil
			}
		}

		// Build URL using standard logic
		builder := group.Builder(routeName)
		if builder == nil {
			context := map[string]any{
				"route_name": routeName,
				"group_name": localizedGroupName,
				"locale":     locale,
			}
			return formatError("url_locale", "route_not_found", fmt.Sprintf("route '%s' not found in group '%s'", routeName, localizedGroupName), context, config), nil
		}

		// Add parameters
		if len(params) > 0 {
			for key, value := range params {
				builder = builder.WithParam(key, value)
			}
		}

		// Add query parameters
		if len(query) > 0 {
			for key, value := range query {
				builder = builder.WithQuery(key, value)
			}
		}

		// Build the final URL
		url, err := builder.Build()
		if err != nil {
			context := map[string]any{
				"route_name": routeName,
				"group_name": localizedGroupName,
				"locale":     locale,
				"params":     params,
				"query":      query,
			}
			return formatError("url_locale", "build_error", err.Error(), context, config), nil
		}

		return pongo2.AsValue(url), nil
	}
}

// urlAllLocalesHelper returns a template function that generates URLs for all available locales
// Template usage: {{ url_all_locales('frontend', 'about', {'id': 1}) }}
// Returns array of LocaleInfo objects with locale and url fields
func urlAllLocalesHelper(manager *RouteManager, config *TemplateHelperConfig, localeConfig *LocaleConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 2 {
			return formatError("url_all_locales", "insufficient_args", "requires group and route", map[string]any{"args_count": len(args)}, config), nil
		}

		// Parse basic arguments
		groupVal := fromPongoValue(args[0])
		routeVal := fromPongoValue(args[1])

		groupName, ok1 := groupVal.(string)
		routeName, ok2 := routeVal.(string)

		if !ok1 || !ok2 {
			context := map[string]any{
				"group_type": fmt.Sprintf("%T", groupVal),
				"route_type": fmt.Sprintf("%T", routeVal),
			}
			return formatError("url_all_locales", "invalid_args", "group and route must be strings", context, config), nil
		}

		// Parse optional params and query arguments
		params := make(map[string]any)
		query := make(map[string]string)

		if len(args) > 2 && args[2] != nil {
			paramsVal := fromPongoValue(args[2])
			if paramsMap, ok := paramsVal.(map[string]any); ok {
				params = paramsMap
			}
		}

		if len(args) > 3 && args[3] != nil {
			queryVal := fromPongoValue(args[3])
			if queryMap, ok := queryVal.(map[string]any); ok {
				for k, v := range queryMap {
					if str, ok := v.(string); ok {
						query[k] = str
					} else if v != nil {
						query[k] = fmt.Sprintf("%v", v)
					}
				}
			}
		}

		// Get supported locales for the group
		supportedLocales := localeConfig.getSupportedLocalesForGroup(groupName)
		var localeInfos []LocaleInfo

		// Generate URL for each supported locale
		for _, locale := range supportedLocales {
			localizedGroupName := groupName
			if localeConfig.EnableHierarchicalLocales && locale != "" {
				localizedGroupName = groupName + "." + locale
			}

			// Get the group safely
			group := safeGroupAccess(manager, localizedGroupName)
			if group == nil && localeConfig.EnableHierarchicalLocales {
				// If hierarchical locale group doesn't exist, try the original group
				group = safeGroupAccess(manager, groupName)
			}

			if group == nil {
				continue // Skip this locale if group doesn't exist
			}

			// Build URL
			builder := group.Builder(routeName)
			if builder == nil {
				continue // Skip this locale if route doesn't exist
			}

			// Add parameters
			for key, value := range params {
				builder = builder.WithParam(key, value)
			}

			// Add query parameters
			for key, value := range query {
				builder = builder.WithQuery(key, value)
			}

			// Build URL
			url, err := builder.Build()
			if err != nil {
				continue // Skip this locale if URL building fails
			}

			localeInfos = append(localeInfos, LocaleInfo{
				Locale: locale,
				URL:    url,
			})
		}

		return pongo2.AsValue(localeInfos), nil
	}
}

// hasLocaleHelper returns a template function that checks if a locale is available for a group
// Template usage: {{ has_locale('frontend', 'es') }}
func hasLocaleHelper(_ *RouteManager, _ *TemplateHelperConfig, localeConfig *LocaleConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 2 {
			return pongo2.AsValue(false), nil
		}

		groupVal := fromPongoValue(args[0])
		localeVal := fromPongoValue(args[1])

		groupName, ok1 := groupVal.(string)
		locale, ok2 := localeVal.(string)

		if !ok1 || !ok2 {
			return pongo2.AsValue(false), nil
		}

		// Check if locale is supported for the group
		supported := localeConfig.isLocaleSupported(locale, groupName)
		return pongo2.AsValue(supported), nil
	}
}

// currentLocaleHelper returns a template function that gets the current locale from template context
// Template usage: {{ current_locale() }}
func currentLocaleHelper(_ *TemplateHelperConfig, localeConfig *LocaleConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		// Get template context for locale detection
		var detectedLocale string
		if len(args) > 0 {
			// First argument could be context data for locale detection
			contextVal := fromPongoValue(args[0])
			detectedLocale = localeConfig.detectLocale(contextVal, "")
		}

		if detectedLocale == "" {
			detectedLocale = localeConfig.DefaultLocale
		}

		return pongo2.AsValue(detectedLocale), nil
	}
}
