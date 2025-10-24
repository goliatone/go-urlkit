package urlkit

import (
	"fmt"
	"net/url"
	"reflect"
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
				u.Path += "/"
			}
			u.Path += path
		}
	}

	values := u.Query()
	for _, query := range queries {
		for key, value := range query {
			values.Add(key, value)
		}
	}

	u.RawQuery = values.Encode()

	return u.String()
}

func groupDisplayName(u *Group) string {
	if u == nil {
		return ""
	}

	if name := u.FullName(); name != "" {
		return name
	}

	if u.name != "" {
		return u.name
	}

	if u.parent == nil {
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

	for key, values := range multi {
		if len(values) == 0 {
			queries = append(queries, Query{key: ""})
			continue
		}
		for _, value := range values {
			queries = append(queries, Query{key: fmt.Sprint(value)})
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
	case Params:
		for key, value := range v {
			target[key] = fmt.Sprint(value)
		}
		return nil
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
