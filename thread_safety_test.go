package urlkit

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/flosch/pongo2/v6"
)

// Thread Safety Notes for URLKit Template Helpers:
//
// The URLKit RouteManager is designed for the following usage pattern:
// 1. Single threaded initialization: Register all routes during app startup
// 2. Multi threaded read only access: Use template helpers during request handling
//
// The template helpers are thread safe for concurrent read only access to
// pre registered routes. However, the RouteManager itself is NOT thread safe
// for concurrent registration operations.
//
// This is the expected and recommended usage pattern for web applications.

// TestConcurrentTemplateRendering tests that template helpers can be called
// safely from multiple goroutines simultaneously
func TestConcurrentTemplateRendering(t *testing.T) {
	// Setup test route manager
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":         "/",
		"user_profile": "/users/:id/profile",
		"product":      "/products/:id",
		"category":     "/categories/:slug",
	})
	manager.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users":    "/users/:id?",
		"products": "/products",
		"search":   "/search",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	// Test concurrent access to multiple helpers
	const numGoroutines = 100
	const operationsPerGoroutine = 50

	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*operationsPerGoroutine)

	// Test url helper concurrency
	t.Run("URLHelperConcurrency", func(t *testing.T) {
		wg.Add(numGoroutines)
		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer wg.Done()
				urlHelper := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

				for j := 0; j < operationsPerGoroutine; j++ {
					userID := routineID*operationsPerGoroutine + j
					params := map[string]any{"id": userID}

					result, err := urlHelper(
						pongo2.AsValue("frontend"),
						pongo2.AsValue("user_profile"),
						pongo2.AsValue(params),
					)

					if err != nil {
						errorChan <- fmt.Errorf("routine %d operation %d: %v", routineID, j, err)
						return
					}

					expected := fmt.Sprintf("https://example.com/users/%d/profile", userID)
					if result.String() != expected {
						errorChan <- fmt.Errorf("routine %d operation %d: expected %s, got %s",
							routineID, j, expected, result.String())
						return
					}
				}
			}(i)
		}

		wg.Wait()
		close(errorChan)

		// Check for errors
		for err := range errorChan {
			t.Error(err)
		}
	})
}

// TestConcurrentHelperAccess tests that all template helpers work correctly
// when accessed concurrently
func TestConcurrentHelperAccess(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":         "/",
		"user_profile": "/users/:id/profile",
		"products":     "/products",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)

	const numWorkers = 50
	const operationsPerWorker = 20

	var wg sync.WaitGroup
	errors := make([]error, 0)
	errorMutex := sync.Mutex{}

	addError := func(err error) {
		errorMutex.Lock()
		errors = append(errors, err)
		errorMutex.Unlock()
	}

	// Test different helpers concurrently
	tests := []struct {
		name     string
		helper   string
		testFunc func(helper func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error), workerID, opID int) error
	}{
		{
			name:   "url_helper",
			helper: "url",
			testFunc: func(helper func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error), workerID, opID int) error {
				result, err := helper(
					pongo2.AsValue("frontend"),
					pongo2.AsValue("user_profile"),
					pongo2.AsValue(map[string]any{"id": workerID*100 + opID}),
				)
				if err != nil {
					return err.OrigError
				}
				if result.String() == "" {
					return fmt.Errorf("empty result")
				}
				return nil
			},
		},
		{
			name:   "route_path_helper",
			helper: "route_path",
			testFunc: func(helper func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error), workerID, opID int) error {
				result, err := helper(
					pongo2.AsValue("frontend"),
					pongo2.AsValue("user_profile"),
					pongo2.AsValue(map[string]any{"id": workerID*100 + opID}),
				)
				if err != nil {
					return err.OrigError
				}
				if result.String() == "" {
					return fmt.Errorf("empty result")
				}
				return nil
			},
		},
		{
			name:   "has_route_helper",
			helper: "has_route",
			testFunc: func(helper func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error), workerID, opID int) error {
				result, err := helper(
					pongo2.AsValue("frontend"),
					pongo2.AsValue("user_profile"),
				)
				if err != nil {
					return err.OrigError
				}
				if !result.Bool() {
					return fmt.Errorf("expected route to exist")
				}
				return nil
			},
		},
		{
			name:   "route_template_helper",
			helper: "route_template",
			testFunc: func(helper func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error), workerID, opID int) error {
				result, err := helper(
					pongo2.AsValue("frontend"),
					pongo2.AsValue("user_profile"),
				)
				if err != nil {
					return err.OrigError
				}
				if result.String() == "" {
					return fmt.Errorf("empty template")
				}
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			helper := helpers[test.helper].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

			wg.Add(numWorkers)
			for i := 0; i < numWorkers; i++ {
				go func(workerID int) {
					defer wg.Done()
					for j := 0; j < operationsPerWorker; j++ {
						if err := test.testFunc(helper, workerID, j); err != nil {
							addError(fmt.Errorf("worker %d op %d: %v", workerID, j, err))
						}
					}
				}(i)
			}
			wg.Wait()
		})
	}

	// Report any errors
	if len(errors) > 0 {
		for _, err := range errors {
			t.Error(err)
		}
	}
}

// TestTemplateHelpersWithPrepopulatedRoutes tests template helpers thread safety
// with a realistic usage pattern: routes are registered during app initialization,
// then template helpers are used concurrently during request handling
func TestTemplateHelpersWithPrepopulatedRoutes(t *testing.T) {
	manager := NewRouteManager()

	// Setup routes during initialization (single threaded phase)
	for i := 0; i < 20; i++ {
		groupName := fmt.Sprintf("group_%d", i)
		routes := map[string]string{
			"route_1": fmt.Sprintf("/path_%d/1", i),
			"route_2": fmt.Sprintf("/path_%d/2", i),
		}
		manager.RegisterGroup(groupName, "https://api.example.com", routes)
	}

	const numWorkers = 50
	const operationsPerWorker = 100
	var wg sync.WaitGroup

	// Test concurrent template helper usage (realistic multi threaded phase)
	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlHelper := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	errorCount := int64(0)
	var errorMutex sync.Mutex

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				groupID := workerID % 20
				groupName := fmt.Sprintf("group_%d", groupID)

				result, err := urlHelper(
					pongo2.AsValue(groupName),
					pongo2.AsValue("route_1"),
					pongo2.AsValue(map[string]any{"id": j}),
				)

				if err != nil {
					errorMutex.Lock()
					errorCount++
					errorMutex.Unlock()
					t.Errorf("Worker %d iteration %d: %v", workerID, j, err)
					return
				}
				if result.String() == "" {
					errorMutex.Lock()
					errorCount++
					errorMutex.Unlock()
					t.Errorf("Worker %d iteration %d: empty result", workerID, j)
					return
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Template helpers thread safety test failed with %d errors", errorCount)
	}
}

// TestConcurrentRouteValidation tests concurrent access to route validation
func TestConcurrentRouteValidation(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("frontend", "https://example.com", map[string]string{
		"home":    "/",
		"about":   "/about",
		"contact": "/contact",
	})
	manager.RegisterGroup("api", "https://api.example.com", map[string]string{
		"users":    "/users",
		"products": "/products",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	hasRouteHelper := helpers["has_route"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	const numWorkers = 30
	const checksPerWorker = 100

	var wg sync.WaitGroup
	errorCount := int64(0)
	var errorMutex sync.Mutex

	// Test routes that exist and don't exist
	testCases := []struct {
		group  string
		route  string
		exists bool
	}{
		{"frontend", "home", true},
		{"frontend", "about", true},
		{"frontend", "nonexistent", false},
		{"api", "users", true},
		{"api", "nonexistent", false},
		{"nonexistent_group", "home", false},
	}

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < checksPerWorker; j++ {
				testCase := testCases[j%len(testCases)]

				result, err := hasRouteHelper(
					pongo2.AsValue(testCase.group),
					pongo2.AsValue(testCase.route),
				)

				if err != nil {
					errorMutex.Lock()
					errorCount++
					errorMutex.Unlock()
					t.Errorf("Worker %d check %d: unexpected error: %v", workerID, j, err)
					continue
				}

				if result.Bool() != testCase.exists {
					errorMutex.Lock()
					errorCount++
					errorMutex.Unlock()
					t.Errorf("Worker %d check %d: expected exists=%v for %s.%s, got %v",
						workerID, j, testCase.exists, testCase.group, testCase.route, result.Bool())
				}
			}
		}(i)
	}

	wg.Wait()

	if errorCount > 0 {
		t.Errorf("Thread safety test failed with %d errors", errorCount)
	}
}

// TestHighLoadTemplateHelpers simulates high load conditions
func TestHighLoadTemplateHelpers(t *testing.T) {
	manager := NewRouteManager()

	// Create a substantial number of routes
	for i := 0; i < 20; i++ {
		groupName := fmt.Sprintf("group_%d", i)
		routes := make(map[string]string)
		for j := 0; j < 50; j++ {
			routes[fmt.Sprintf("route_%d", j)] = fmt.Sprintf("/path_%d/:id", j)
		}
		manager.RegisterGroup(groupName, fmt.Sprintf("https://group%d.example.com", i), routes)
	}

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlHelper := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	const numWorkers = 100
	const operationsPerWorker = 500

	start := time.Now()
	var wg sync.WaitGroup
	errorCount := int64(0)
	successCount := int64(0)
	var mutex sync.Mutex

	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < operationsPerWorker; j++ {
				groupID := workerID % 20
				routeID := j % 50

				result, err := urlHelper(
					pongo2.AsValue(fmt.Sprintf("group_%d", groupID)),
					pongo2.AsValue(fmt.Sprintf("route_%d", routeID)),
					pongo2.AsValue(map[string]any{"id": j}),
				)

				mutex.Lock()
				if err != nil {
					errorCount++
				} else if result.String() != "" {
					successCount++
				} else {
					errorCount++
				}
				mutex.Unlock()
			}
		}(i)
	}

	wg.Wait()
	duration := time.Since(start)

	totalOperations := numWorkers * operationsPerWorker
	t.Logf("High load test completed:")
	t.Logf("  Duration: %v", duration)
	t.Logf("  Total operations: %d", totalOperations)
	t.Logf("  Success count: %d", successCount)
	t.Logf("  Error count: %d", errorCount)
	t.Logf("  Operations per second: %.2f", float64(totalOperations)/duration.Seconds())

	if errorCount > 0 {
		t.Errorf("High load test failed with %d errors out of %d operations", errorCount, totalOperations)
	}

	if successCount != int64(totalOperations) {
		t.Errorf("Expected %d successful operations, got %d", totalOperations, successCount)
	}
}

// TestTemplateHelperMemoryUsage tests for memory leaks under concurrent access
func TestTemplateHelperMemoryUsage(t *testing.T) {
	manager := NewRouteManager()
	manager.RegisterGroup("test", "https://example.com", map[string]string{
		"route": "/test/:id",
	})

	config := DefaultTemplateHelperConfig()
	helpers := TemplateHelpers(manager, config)
	urlHelper := helpers["url"].(func(...*pongo2.Value) (*pongo2.Value, *pongo2.Error))

	// Force initial allocation
	_, _ = urlHelper(
		pongo2.AsValue("test"),
		pongo2.AsValue("route"),
		pongo2.AsValue(map[string]any{"id": 1}),
	)

	const numIterations = 1000
	const numWorkers = 10

	var wg sync.WaitGroup
	wg.Add(numWorkers)

	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			for j := 0; j < numIterations; j++ {
				// Create new parameter maps to test for leaks
				params := map[string]any{
					"id":    workerID*numIterations + j,
					"extra": fmt.Sprintf("data_%d_%d", workerID, j),
				}
				query := map[string]any{
					"page":   j % 10,
					"filter": fmt.Sprintf("filter_%d", j),
				}

				result, err := urlHelper(
					pongo2.AsValue("test"),
					pongo2.AsValue("route"),
					pongo2.AsValue(params),
					pongo2.AsValue(query),
				)

				if err != nil {
					t.Errorf("Worker %d iteration %d: %v", workerID, j, err)
					return
				}

				if result.String() == "" {
					t.Errorf("Worker %d iteration %d: empty result", workerID, j)
					return
				}

				// Null out references to help detect leaks
				params = nil
				query = nil
				result = nil
			}
		}(i)
	}

	wg.Wait()

	// This test mainly ensures we don't crash or deadlock under memory pressure
	// Memory leak detection would require additional tooling
	t.Log("Memory usage test completed successfully")
}
