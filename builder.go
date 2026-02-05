package urlkit

import "fmt"

type Builder struct {
	helper     *Group
	routeName  string
	params     Params
	query      Query
	multiQuery map[string][]string
	err        error
}

func (b *Builder) WithParam(key string, value any) *Builder {
	if b.err != nil {
		return b
	}

	b.params[key] = fmt.Sprint(value)
	return b
}

func (b *Builder) WithParamsMap(values map[string]any) *Builder {
	if b.err != nil {
		return b
	}

	if err := mergeParamsInput(b.params, values); err != nil {
		b.err = err
	}
	return b
}

func (b *Builder) WithStruct(value any) *Builder {
	if b.err != nil {
		return b
	}

	if err := mergeParamsInput(b.params, value); err != nil {
		b.err = err
	}
	return b
}

func (b *Builder) WithQuery(key string, value any) *Builder {
	if b.err != nil {
		return b
	}

	if b.query == nil {
		b.query = make(Query)
	}

	switch v := value.(type) {
	case nil:
		b.query[key] = ""
	case []string:
		b.setMultiQueryValues(key, v)
	case []any:
		values := make([]string, 0, len(v))
		for _, entry := range v {
			values = append(values, fmt.Sprint(entry))
		}
		b.setMultiQueryValues(key, values)
	default:
		b.query[key] = fmt.Sprint(v)
	}

	return b
}

func (b *Builder) WithQueryValues(values map[string][]string) *Builder {
	if b.err != nil {
		return b
	}

	for key, items := range values {
		b.setMultiQueryValues(key, items)
	}

	return b
}

func (b *Builder) setMultiQueryValues(key string, values []string) {
	if b.multiQuery == nil {
		b.multiQuery = make(map[string][]string)
	}

	normalized := make([]string, 0, len(values))
	for _, value := range values {
		normalized = append(normalized, fmt.Sprint(value))
	}
	if len(normalized) == 0 {
		normalized = []string{""}
	}

	b.multiQuery[key] = normalized
	if b.query != nil {
		delete(b.query, key)
	}
}

func (b *Builder) Build() (string, error) {
	if b.err != nil {
		return "", b.err
	}

	params := coerceParams(b.params)

	queries := combineQueries(b.query, b.multiQuery)

	return b.helper.Render(b.routeName, params, queries...)
}

func (b *Builder) MustBuild() string {
	if b.err != nil {
		panic(b.err)
	}

	s, err := b.Build()
	if err != nil {
		panic(err)
	}
	return s
}

func (m *RouteManager) Resolve(groupPath, route string, params Params, query Query) (string, error) {
	group, err := m.GetGroup(groupPath)
	if err != nil {
		return "", err
	}

	normalizedParams := coerceParams(params)
	normalizedQuery := cloneQuery(query)

	queries := combineQueries(normalizedQuery, nil)
	return group.Render(route, normalizedParams, queries...)
}

// RoutePath returns the full group path plus the raw route template.
// It excludes the base URL and does not append query parameters.
func (m *RouteManager) RoutePath(groupPath, route string) (string, error) {
	group, err := m.GetGroup(groupPath)
	if err != nil {
		return "", err
	}

	routeTemplate, err := group.Route(route)
	if err != nil {
		return "", err
	}

	return joinURLPath(group.getFullPath(), routeTemplate), nil
}

func (m *RouteManager) ResolveWith(groupPath, route string, params any, query any) (string, error) {
	normalizedParams, err := buildParamsFromInput(params)
	if err != nil {
		return "", err
	}

	singleQuery, multiQuery, err := buildQueryFromInput(query)
	if err != nil {
		return "", err
	}

	group, err := m.GetGroup(groupPath)
	if err != nil {
		return "", err
	}

	queries := combineQueries(singleQuery, multiQuery)
	return group.Render(route, coerceParams(normalizedParams), queries...)
}
