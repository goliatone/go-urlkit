package urlkit

import (
	"fmt"
	"maps"
	"net/url"
	"reflect"
	"slices"
	"strings"
	"unicode"
)

func JoinURL(base, path string, queries ...Query) string {
	u, err := url.Parse(base)
	if err != nil {
		u = &url.URL{Path: base}
	}

	if path != "" {
		if strings.HasPrefix(path, "/") {
			u.Path = path
		} else {
			if !strings.HasSuffix(u.Path, "/") {
				if u.Path == "" {
					u.Path = "/"
				} else {
					u.Path += "/"
				}
			}
			u.Path += path
		}
	}

	if len(queries) > 0 {
		var newPairs []string
		for _, query := range queries {
			if len(query) == 0 {
				continue
			}
			keys := slices.Sorted(maps.Keys(query))
			for _, key := range keys {
				newPairs = append(newPairs, encodeQueryPair(key, query[key]))
			}
		}

		if len(newPairs) > 0 {
			var builder strings.Builder
			if existing := u.RawQuery; existing != "" {
				builder.WriteString(existing)
			}
			for _, pair := range newPairs {
				if builder.Len() > 0 {
					builder.WriteByte('&')
				}
				builder.WriteString(pair)
			}
			u.RawQuery = builder.String()
		}
	}

	return u.String()
}

func groupDisplayName(u *Group) string {
	if u == nil {
		return ""
	}

	if name := u.FullName(); name != "" {
		return name
	}

	u.mu.RLock()
	name := u.name
	parent := u.parent
	u.mu.RUnlock()

	if name != "" {
		return name
	}

	if parent == nil {
		return "(root)"
	}

	return "(unnamed)"
}

func coerceParams(source Params) Params {
	if source == nil {
		return nil
	}

	normalized := make(Params, len(source))
	for key, value := range source {
		normalized[key] = fmt.Sprint(value)
	}
	return normalized
}

func combineQueries(single Query, multi map[string][]string) []Query {
	var queries []Query

	if len(single) > 0 {
		queries = append(queries, cloneQuery(single))
	}

	if len(multi) > 0 {
		keys := slices.Sorted(maps.Keys(multi))
		for _, key := range keys {
			values := multi[key]
			if len(values) == 0 {
				queries = append(queries, Query{key: ""})
				continue
			}
			for _, value := range values {
				queries = append(queries, Query{key: fmt.Sprint(value)})
			}
		}
	}

	return queries
}

func cloneQuery(source Query) Query {
	if source == nil {
		return nil
	}

	clone := make(Query, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func cloneMultiQuery(source map[string][]string) map[string][]string {
	if source == nil {
		return nil
	}

	clone := make(map[string][]string, len(source))
	for key, values := range source {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func cloneParamsMap(source Params) Params {
	if source == nil {
		return nil
	}

	clone := make(Params, len(source))
	for key, value := range source {
		clone[key] = value
	}
	return clone
}

func mergeParamsInput(target Params, input any) error {
	if input == nil {
		return nil
	}

	switch v := input.(type) {
	// case Params:
	// 	for key, value := range v {
	// 		target[key] = fmt.Sprint(value)
	// 	}
	// 	return nil
	case map[string]any:
		for key, value := range v {
			target[key] = fmt.Sprint(value)
		}
		return nil
	case map[string]string:
		for key, value := range v {
			target[key] = value
		}
		return nil
	default:
		val := reflect.ValueOf(input)
		if !val.IsValid() {
			return nil
		}

		if val.Kind() == reflect.Pointer {
			if val.IsNil() {
				return nil
			}
			val = val.Elem()
		}

		if val.Kind() != reflect.Struct {
			return fmt.Errorf("unsupported params type %T", input)
		}

		return mergeStructParams(target, val)
	}
}

func mergeStructParams(target Params, value reflect.Value) error {
	valueType := value.Type()
	for i := 0; i < valueType.NumField(); i++ {
		field := valueType.Field(i)
		if !field.IsExported() {
			continue
		}

		key, include := paramsKeyFromField(field)
		if !include {
			continue
		}

		fieldValue := value.Field(i).Interface()
		target[key] = fmt.Sprint(fieldValue)
	}
	return nil
}

func paramsKeyFromField(field reflect.StructField) (string, bool) {
	if tag := field.Tag.Get("urlkit"); tag != "" {
		if tag == "-" {
			return "", false
		}
		return tag, true
	}

	if tag := field.Tag.Get("json"); tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] == "-" {
			return "", false
		}
		if parts[0] != "" {
			return parts[0], true
		}
	}

	return lowerFirst(field.Name), true
}

func lowerFirst(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}

func buildParamsFromInput(input any) (Params, error) {
	params := make(Params)
	if err := mergeParamsInput(params, input); err != nil {
		return nil, err
	}
	return params, nil
}

func buildQueryFromInput(input any) (Query, map[string][]string, error) {
	if input == nil {
		return nil, nil, nil
	}

	switch v := input.(type) {
	case Query:
		return cloneQuery(v), nil, nil
	case map[string]string:
		return cloneQuery(Query(v)), nil, nil
	case map[string][]string:
		return nil, cloneMultiQuery(v), nil
	case url.Values:
		return nil, cloneMultiQuery(map[string][]string(v)), nil
	case map[string]any:
		single := make(Query)
		multi := make(map[string][]string)
		for key, value := range v {
			switch typed := value.(type) {
			case nil:
				single[key] = ""
			case string:
				single[key] = typed
			case []string:
				multi[key] = append([]string(nil), typed...)
			case []any:
				values := make([]string, 0, len(typed))
				for _, entry := range typed {
					values = append(values, fmt.Sprint(entry))
				}
				multi[key] = values
			default:
				single[key] = fmt.Sprint(value)
			}
		}

		if len(single) == 0 {
			single = nil
		}
		if len(multi) == 0 {
			multi = nil
		}

		return single, multi, nil
	default:
		return nil, nil, fmt.Errorf("unsupported query type %T", input)
	}
}

func parseEnsureSegment(segment string) (string, string, error) {
	if segment == "" {
		return "", "", fmt.Errorf("empty segment")
	}

	name := segment
	customPath := ""

	if idx := strings.Index(segment, ":"); idx != -1 {
		name = segment[:idx]
		customPath = segment[idx+1:]
	}

	name = strings.TrimSpace(name)
	customPath = strings.TrimSpace(customPath)

	if name == "" {
		return "", "", fmt.Errorf("segment %q missing group name", segment)
	}

	if customPath == "" {
		trimmed := strings.TrimLeft(name, "/")
		if trimmed == "" {
			trimmed = name
		}
		customPath = "/" + trimmed
	} else if !strings.HasPrefix(customPath, "/") {
		customPath = "/" + customPath
	}

	return name, customPath, nil
}

func encodeQueryPair(key, value string) string {
	return url.QueryEscape(key) + "=" + url.QueryEscape(value)
}

func joinURLPath(prefix, route string) string {
	prefixSegments, _, prefixIsRoot := splitPathSegments(prefix)
	routeSegments, routeHasTrailing, routeIsRoot := splitPathSegments(route)

	overlap := longestOverlap(prefixSegments, routeSegments)

	merged := make([]string, len(prefixSegments))
	copy(merged, prefixSegments)
	merged = append(merged, routeSegments[overlap:]...)

	// Handle special case: prefix is "/" and route starts with "/"
	// This should preserve the empty segment creating "//"
	if prefixIsRoot && len(prefixSegments) == 0 && len(routeSegments) > 0 {
		merged = append([]string{""}, merged...)
	}

	if len(merged) == 0 {
		switch {
		case routeIsRoot && prefix == "":
			return "/"
		case routeIsRoot:
			base := ensureLeadingSlash(prefix)
			if base == "" {
				return "/"
			}
			if !strings.HasSuffix(base, "/") {
				base += "/"
			}
			return base
		case prefix != "":
			return ensureLeadingSlash(prefix)
		case route != "":
			return ensureLeadingSlash(route)
		default:
			return ""
		}
	}

	path := "/" + strings.Join(merged, "/")
	if routeHasTrailing && !strings.HasSuffix(path, "/") {
		path += "/"
	}
	return path
}

func splitPathSegments(path string) (segments []string, hasTrailing bool, isRoot bool) {
	switch path {
	case "":
		return nil, false, false
	case "/":
		return nil, true, true
	}

	hasTrailing = strings.HasSuffix(path, "/")
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil, hasTrailing, false
	}

	return strings.Split(trimmed, "/"), hasTrailing, false
}

func longestOverlap(prefix, route []string) int {
	if len(prefix) == 0 || len(route) == 0 {
		return 0
	}
	max := minInt(len(prefix), len(route))
	for k := max; k > 0; k-- {
		if slices.Equal(prefix[len(prefix)-k:], route[:k]) {
			return k
		}
	}
	return 0
}

func ensureLeadingSlash(path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "/") {
		return path
	}
	return "/" + path
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
