package urlkit

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/flosch/pongo2"
)

// TemplateHelperConfig defines configuration for template helpers
type TemplateHelperConfig struct {
	// Context key names for extracting contextual data
	CurrentRouteKey  string
	CurrentParamsKey string
	CurrentQueryKey  string
	// Error reporting configuration
	EnableStructuredErrors bool // When true, returns JSON error objects instead of simple strings
	EnableErrorLogging     bool // When true, logs errors for production debugging
}

// DefaultTemplateHelperConfig returns default configuration
func DefaultTemplateHelperConfig() *TemplateHelperConfig {
	return &TemplateHelperConfig{
		CurrentRouteKey:        "current_route_name",
		CurrentParamsKey:       "current_params",
		CurrentQueryKey:        "current_query",
		EnableStructuredErrors: false, // Default to simple error strings for backwards compatibility
		EnableErrorLogging:     false, // Default to no logging
	}
}

// TemplateHelpers returns a map of template helper functions for use with template engines
func TemplateHelpers(manager *RouteManager, config *TemplateHelperConfig) map[string]interface{} {
	if config == nil {
		config = DefaultTemplateHelperConfig()
	}

	helpers := make(map[string]interface{})

	// Core URL helpers
	helpers["url"] = urlHelper(manager, config)
	helpers["route_path"] = routePathHelper(manager, config)
	helpers["has_route"] = hasRouteHelper(manager, config)

	// Phase 2: Debugging helpers
	helpers["route_template"] = routeTemplateHelper(manager, config)
	helpers["route_vars"] = routeVarsHelper(manager, config)
	helpers["route_exists"] = routeExistsHelper(manager, config)
	helpers["url_abs"] = urlAbsHelper(manager, config)

	return helpers
}

// TemplateError represents structured error information for template helpers
type TemplateError struct {
	Helper  string                 `json:"helper"`
	Type    string                 `json:"type"`
	Message string                 `json:"message"`
	Context map[string]interface{} `json:"context,omitempty"`
}

// formatError creates appropriate error response based on configuration
func formatError(helper, errorType, message string, context map[string]interface{}, config *TemplateHelperConfig) *pongo2.Value {
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

// urlHelperArgs represents parsed arguments for URL helpers
type urlHelperArgs struct {
	Group  string
	Route  string
	Params map[string]interface{}
	Query  map[string]string
}

// parseArgs parses variadic pongo2.Value arguments into structured data
func parseArgs(args ...*pongo2.Value) (*urlHelperArgs, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("at least 2 arguments required: group and route")
	}

	result := &urlHelperArgs{
		Params: make(map[string]interface{}),
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
		if params, ok := paramsVal.(map[string]interface{}); ok {
			result.Params = params
		} else if paramsVal != nil {
			return nil, fmt.Errorf("params must be a map")
		}
	}

	// Optional fourth argument: query map
	if len(args) > 3 && args[3] != nil {
		queryVal := fromPongoValue(args[3])
		if queryMap, ok := queryVal.(map[string]interface{}); ok {
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
func fromPongoValue(val *pongo2.Value) interface{} {
	if val == nil {
		return nil
	}

	// Get the interface{} from pongo2.Value
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
	case map[string]interface{}:
		return v
	case []interface{}:
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
				result := make(map[string]interface{})
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
			errorMsg := fmt.Sprintf("#error:url:%s", err.Error())
			return pongo2.AsValue(errorMsg), nil
		}

		// Get the group
		group := manager.Group(parsedArgs.Group)
		if group == nil {
			errorMsg := fmt.Sprintf("#error:url:group '%s' not found", parsedArgs.Group)
			return pongo2.AsValue(errorMsg), nil
		}

		// Build the URL using the fluent API
		builder := group.Builder(parsedArgs.Route)
		if builder == nil {
			errorMsg := fmt.Sprintf("#error:url:route '%s' not found in group '%s'", parsedArgs.Route, parsedArgs.Group)
			return pongo2.AsValue(errorMsg), nil
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
			errorMsg := fmt.Sprintf("#error:url:%s", err.Error())
			return pongo2.AsValue(errorMsg), nil
		}

		return pongo2.AsValue(url), nil
	}
}

// routePathHelper returns a template function that generates URL paths (without base URL)
func routePathHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		parsedArgs, err := parseArgs(args...)
		if err != nil {
			errorMsg := fmt.Sprintf("#error:route_path:%s", err.Error())
			return pongo2.AsValue(errorMsg), nil
		}

		// Get the group
		group := manager.Group(parsedArgs.Group)
		if group == nil {
			errorMsg := fmt.Sprintf("#error:route_path:group '%s' not found", parsedArgs.Group)
			return pongo2.AsValue(errorMsg), nil
		}

		// Build the URL using the fluent API
		builder := group.Builder(parsedArgs.Route)
		if builder == nil {
			errorMsg := fmt.Sprintf("#error:route_path:route '%s' not found in group '%s'", parsedArgs.Route, parsedArgs.Group)
			return pongo2.AsValue(errorMsg), nil
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
			errorMsg := fmt.Sprintf("#error:route_path:%s", err.Error())
			return pongo2.AsValue(errorMsg), nil
		}

		// For now, return the full URL (this can be enhanced later to strip base URL)
		return pongo2.AsValue(url), nil
	}
}

// hasRouteHelper returns a template function that checks if a route exists
func hasRouteHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
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

		// Check if group exists
		group := manager.Group(groupName)
		if group == nil {
			return pongo2.AsValue(false), nil
		}

		// Check if route exists in group
		builder := group.Builder(routeName)
		exists := builder != nil

		return pongo2.AsValue(exists), nil
	}
}

// routeTemplateHelper returns a template function that returns the raw route template string
func routeTemplateHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 2 {
			errorMsg := "#error:route_template:at least 2 arguments required: group and route"
			return pongo2.AsValue(errorMsg), nil
		}

		groupVal := fromPongoValue(args[0])
		routeVal := fromPongoValue(args[1])

		groupName, ok1 := groupVal.(string)
		routeName, ok2 := routeVal.(string)

		if !ok1 || !ok2 {
			errorMsg := "#error:route_template:group and route must be strings"
			return pongo2.AsValue(errorMsg), nil
		}

		// Get the group
		group := manager.Group(groupName)
		if group == nil {
			errorMsg := fmt.Sprintf("#error:route_template:group '%s' not found", groupName)
			return pongo2.AsValue(errorMsg), nil
		}

		// Get the route template
		template, err := group.Route(routeName)
		if err != nil {
			errorMsg := fmt.Sprintf("#error:route_template:route '%s' not found in group '%s'", routeName, groupName)
			return pongo2.AsValue(errorMsg), nil
		}

		return pongo2.AsValue(template), nil
	}
}

// routeVarsHelper returns a template function that returns template variables for debugging
func routeVarsHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 1 {
			errorMsg := "#error:route_vars:at least 1 argument required: group"
			return pongo2.AsValue(errorMsg), nil
		}

		groupVal := fromPongoValue(args[0])
		groupName, ok := groupVal.(string)

		if !ok {
			errorMsg := "#error:route_vars:group must be a string"
			return pongo2.AsValue(errorMsg), nil
		}

		// Get the group
		group := manager.Group(groupName)
		if group == nil {
			errorMsg := fmt.Sprintf("#error:route_vars:group '%s' not found", groupName)
			return pongo2.AsValue(errorMsg), nil
		}

		// Get the template variables
		vars := group.CollectTemplateVars()

		// Convert to map[string]interface{} for template use
		result := make(map[string]interface{})
		for k, v := range vars {
			result[k] = v
		}

		return pongo2.AsValue(result), nil
	}
}

// routeExistsHelper returns a template function that checks if an entire route group exists
func routeExistsHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		if len(args) < 1 {
			return pongo2.AsValue(false), nil
		}

		groupVal := fromPongoValue(args[0])
		groupName, ok := groupVal.(string)

		if !ok {
			return pongo2.AsValue(false), nil
		}

		// Check if group exists without panic by using the groups map directly
		// We need to implement this safely, so let's check the group exists by trying to access it
		defer func() {
			if r := recover(); r != nil {
				// Group doesn't exist, handled in the defer
			}
		}()

		group := manager.Group(groupName)
		exists := group != nil

		return pongo2.AsValue(exists), nil
	}
}

// urlAbsHelper returns a template function that forces absolute URL generation with base URL
func urlAbsHelper(manager *RouteManager, config *TemplateHelperConfig) func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
	return func(args ...*pongo2.Value) (*pongo2.Value, *pongo2.Error) {
		parsedArgs, err := parseArgs(args...)
		if err != nil {
			errorMsg := fmt.Sprintf("#error:url_abs:%s", err.Error())
			return pongo2.AsValue(errorMsg), nil
		}

		// Get the group
		group := manager.Group(parsedArgs.Group)
		if group == nil {
			errorMsg := fmt.Sprintf("#error:url_abs:group '%s' not found", parsedArgs.Group)
			return pongo2.AsValue(errorMsg), nil
		}

		// Build the URL using the fluent API
		builder := group.Builder(parsedArgs.Route)
		if builder == nil {
			errorMsg := fmt.Sprintf("#error:url_abs:route '%s' not found in group '%s'", parsedArgs.Route, parsedArgs.Group)
			return pongo2.AsValue(errorMsg), nil
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
			errorMsg := fmt.Sprintf("#error:url_abs:%s", err.Error())
			return pongo2.AsValue(errorMsg), nil
		}

		return pongo2.AsValue(url), nil
	}
}
