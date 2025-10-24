package urlkit

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestJSONConfigIntegrationLifecycle(t *testing.T) {
	const configJSON = `{
  "groups": [
    {
      "name": "cms",
      "base_url": "https://cms.example.com",
      "url_template": "{base_url}/{locale}{route_path}",
      "routes": {
        "home": "/"
      },
      "groups": [
        {
          "name": "en",
          "template_vars": {
            "locale": "en"
          },
          "routes": {
            "landing": "/landing"
          },
          "groups": [
            {
              "name": "blog",
              "template_vars": {
                "section": "blog"
              },
              "routes": {
                "article": "/blog/:slug"
              }
            }
          ]
        },
        {
          "name": "es",
          "template_vars": {
            "locale": "es"
          },
          "routes": {
            "landing": "/inicio"
          },
          "groups": [
            {
              "name": "blog",
              "template_vars": {
                "section": "blog"
              },
              "routes": {
                "article": "/blog/:slug"
              }
            }
          ]
        }
      ]
    }
  ]
}`

	var cfg Config
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}

	manager, err := NewRouteManagerFromConfig(cfg)
	if err != nil {
		t.Fatalf("NewRouteManagerFromConfig returned error: %v", err)
	}

	expected := map[string][]string{
		"cms":         {"home"},
		"cms.en":      {"landing"},
		"cms.es":      {"landing"},
		"cms.en.blog": {"article"},
		"cms.es.blog": {"article"},
	}

	if err := manager.Validate(expected); err != nil {
		t.Fatalf("Validate returned error: %v", err)
	}

	url, err := manager.Resolve("cms.es.blog", "article", Params{"slug": "bienvenidos"}, Query{"draft": "false"})
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if want := "https://cms.example.com/es/blog/bienvenidos?draft=false"; url != want {
		t.Fatalf("expected %q, got %q", want, url)
	}

	type articleParams struct {
		Slug string `urlkit:"slug"`
	}

	urlWithStruct, err := manager.ResolveWith("cms.en.blog", "article", articleParams{Slug: "welcome"}, map[string]any{
		"utm_source": "integration-test",
	})
	if err != nil {
		t.Fatalf("ResolveWith returned error: %v", err)
	}
	if want := "https://cms.example.com/en/blog/welcome?utm_source=integration-test"; urlWithStruct != want {
		t.Fatalf("expected %q, got %q", want, urlWithStruct)
	}
}

func TestMultiLocaleNavigationIntegration(t *testing.T) {
	config := Config{
		Groups: []GroupConfig{
			{
				Name:        "frontend",
				BaseURL:     "https://example.com",
				URLTemplate: "{base_url}/{locale}{route_path}",
				Groups: []GroupConfig{
					{
						Name:         "en",
						TemplateVars: map[string]string{"locale": "en"},
						Groups: []GroupConfig{
							{
								Name:         "account",
								TemplateVars: map[string]string{"section": "account"},
								Routes: map[string]string{
									"profile":  "/account/profile",
									"settings": "/account/settings",
								},
							},
						},
					},
					{
						Name:         "es",
						TemplateVars: map[string]string{"locale": "es"},
						Groups: []GroupConfig{
							{
								Name:         "account",
								TemplateVars: map[string]string{"section": "cuenta"},
								Routes: map[string]string{
									"profile":  "/cuenta/perfil",
									"settings": "/cuenta/ajustes",
								},
							},
						},
					},
				},
			},
		},
	}

	manager, err := NewRouteManagerFromConfig(config)
	if err != nil {
		t.Fatalf("NewRouteManagerFromConfig returned error: %v", err)
	}

	accountRoutes := []string{"profile", "settings"}
	paramsFn := func(locale string) func(route string) Params {
		return func(route string) Params {
			return Params{
				"user":   "42",
				"locale": locale,
				"route":  route,
			}
		}
	}

	checkNavigation := func(groupPath, expectedLocale string, expectedURLs map[string]string) {
		group := manager.Group(groupPath)
		nodes, err := group.Navigation(accountRoutes, paramsFn(expectedLocale))
		if err != nil {
			t.Fatalf("Navigation returned error for %s: %v", groupPath, err)
		}

		if len(nodes) != len(accountRoutes) {
			t.Fatalf("expected %d navigation nodes for %s, got %d", len(accountRoutes), groupPath, len(nodes))
		}

		for _, node := range nodes {
			wantURL, ok := expectedURLs[node.Route]
			if !ok {
				t.Fatalf("unexpected route %s in navigation output", node.Route)
			}
			if node.URL != wantURL {
				t.Fatalf("route %s expected URL %q, got %q", node.Route, wantURL, node.URL)
			}
			if node.Group != group.FullName() {
				t.Fatalf("expected group name %q, got %q", group.FullName(), node.Group)
			}
			if !reflect.DeepEqual(node.Params, paramsFn(expectedLocale)(node.Route)) {
				t.Fatalf("route %s params mismatch: %+v", node.Route, node.Params)
			}
		}
	}

	checkNavigation("frontend.en.account", "en", map[string]string{
		"profile":  "https://example.com/en/account/profile",
		"settings": "https://example.com/en/account/settings",
	})

	checkNavigation("frontend.es.account", "es", map[string]string{
		"profile":  "https://example.com/es/cuenta/perfil",
		"settings": "https://example.com/es/cuenta/ajustes",
	})
}
