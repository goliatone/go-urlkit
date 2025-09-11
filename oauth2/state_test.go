package oauth2

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// TestMemoryStateStore tests the basic functionality of MemoryStateStore
func TestMemoryStateStore(t *testing.T) {
	store := NewMemoryStateStore()

	// Test storing and validating states
	testCases := []struct {
		state    string
		expected bool
	}{
		{"state1", true},
		{"state2", true},
		{"state3", true},
	}

	// Store all states
	for _, tc := range testCases {
		result := store.Store(tc.state)
		if result != tc.expected {
			t.Errorf("Store(%q) = %v, want %v", tc.state, result, tc.expected)
		}
	}

	// Validate all states (should succeed once)
	for _, tc := range testCases {
		result := store.Validate(tc.state)
		if result != tc.expected {
			t.Errorf("Validate(%q) first call = %v, want %v", tc.state, result, tc.expected)
		}
	}

	// Validate again (should fail - consume-once pattern)
	for _, tc := range testCases {
		result := store.Validate(tc.state)
		if result != false {
			t.Errorf("Validate(%q) second call = %v, want false", tc.state, result)
		}
	}
}

// TestMemoryStateStoreInvalidStates tests validation of non-existent states
func TestMemoryStateStoreInvalidStates(t *testing.T) {
	store := NewMemoryStateStore()

	invalidStates := []string{
		"nonexistent",
		"",
		"never-stored",
		"fake-state",
	}

	for _, state := range invalidStates {
		result := store.Validate(state)
		if result {
			t.Errorf("Validate(%q) = true for non-existent state, want false", state)
		}
	}
}

// TestMemoryStateStoreDuplicateStorage tests storing the same state multiple times
func TestMemoryStateStoreDuplicateStorage(t *testing.T) {
	store := NewMemoryStateStore()
	state := "duplicate-test"

	// Store the same state multiple times
	for i := 0; i < 5; i++ {
		result := store.Store(state)
		if !result {
			t.Errorf("Store(%q) iteration %d = false, want true", state, i)
		}
	}

	// Should be able to validate once
	if !store.Validate(state) {
		t.Error("Validate should succeed after multiple stores")
	}

	// Second validation should fail
	if store.Validate(state) {
		t.Error("Second validation should fail")
	}
}

// TestMemoryStateStoreEmptyState tests handling of empty states
func TestMemoryStateStoreEmptyState(t *testing.T) {
	store := NewMemoryStateStore()

	// Store empty state
	result := store.Store("")
	if !result {
		t.Error("Store empty state should succeed")
	}

	// Validate empty state
	result = store.Validate("")
	if !result {
		t.Error("Validate empty state should succeed")
	}

	// Second validation should fail
	result = store.Validate("")
	if result {
		t.Error("Second validation of empty state should fail")
	}
}

// TestMemoryStateStoreConcurrency tests thread safety of MemoryStateStore
func TestMemoryStateStoreConcurrency(t *testing.T) {
	store := NewMemoryStateStore()

	const numGoroutines = 100
	const statesPerGoroutine = 100

	var wg sync.WaitGroup

	// Concurrently store states
	wg.Add(numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < statesPerGoroutine; j++ {
				state := fmt.Sprintf("state-%d-%d", routineID, j)
				if !store.Store(state) {
					t.Errorf("Failed to store state %s", state)
				}
			}
		}(i)
	}
	wg.Wait()

	// Concurrently validate states
	wg.Add(numGoroutines)
	validationResults := make([]bool, numGoroutines*statesPerGoroutine)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < statesPerGoroutine; j++ {
				state := fmt.Sprintf("state-%d-%d", routineID, j)
				index := routineID*statesPerGoroutine + j
				validationResults[index] = store.Validate(state)
			}
		}(i)
	}
	wg.Wait()

	// Check that all validations succeeded
	for i, result := range validationResults {
		if !result {
			t.Errorf("Validation failed for index %d", i)
		}
	}

	// Try validating again - all should fail
	wg.Add(numGoroutines)
	secondValidationResults := make([]bool, numGoroutines*statesPerGoroutine)
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < statesPerGoroutine; j++ {
				state := fmt.Sprintf("state-%d-%d", routineID, j)
				index := routineID*statesPerGoroutine + j
				secondValidationResults[index] = store.Validate(state)
			}
		}(i)
	}
	wg.Wait()

	// Check that all second validations failed
	for i, result := range secondValidationResults {
		if result {
			t.Errorf("Second validation should have failed for index %d", i)
		}
	}
}

// TestMemoryStateStoreMixedOperations tests mixed store/validate operations
func TestMemoryStateStoreMixedOperations(t *testing.T) {
	store := NewMemoryStateStore()

	const numWorkers = 50
	var wg sync.WaitGroup

	// Mix of store and validate operations
	wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go func(workerID int) {
			defer wg.Done()

			// Each worker stores some states
			states := make([]string, 10)
			for j := 0; j < 10; j++ {
				state := fmt.Sprintf("worker-%d-state-%d", workerID, j)
				states[j] = state
				if !store.Store(state) {
					t.Errorf("Worker %d failed to store %s", workerID, state)
				}
			}

			// Small delay to let other workers store states
			time.Sleep(time.Millisecond)

			// Then validates its own states
			for _, state := range states {
				if !store.Validate(state) {
					t.Errorf("Worker %d failed to validate %s", workerID, state)
				}
			}
		}(i)
	}
	wg.Wait()
}

// TestMemoryStateStoreInterface verifies MemoryStateStore implements StateStore
func TestMemoryStateStoreInterface(t *testing.T) {
	var store StateStore = NewMemoryStateStore()

	// Test interface methods
	testState := "interface-test"

	if !store.Store(testState) {
		t.Error("Store method should succeed")
	}

	if !store.Validate(testState) {
		t.Error("Validate method should succeed")
	}

	// Debug should not panic
	store.Debug()
}

// TestMemoryStateStoreDebug tests the Debug method
func TestMemoryStateStoreDebug(t *testing.T) {
	store := NewMemoryStateStore()

	// Debug on empty store should not panic
	store.Debug()

	// Add some states
	states := []string{"debug-state-1", "debug-state-2", "debug-state-3"}
	for _, state := range states {
		store.Store(state)
	}

	// Debug with states should not panic
	store.Debug()

	// Validate one state
	store.Validate("debug-state-1")

	// Debug after partial validation should not panic
	store.Debug()
}

// TestMemoryStateStoreRaceCondition tests for race conditions using the race detector
func TestMemoryStateStoreRaceCondition(t *testing.T) {
	store := NewMemoryStateStore()

	// This test is designed to be run with -race flag
	// go test -race ./oauth2

	const iterations = 1000
	var wg sync.WaitGroup

	// Concurrent readers and writers
	wg.Add(3)

	// Writer goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			state := fmt.Sprintf("race-test-%d", i)
			store.Store(state)
		}
	}()

	// Validator goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			state := fmt.Sprintf("race-test-%d", i)
			// Don't care about result, just testing for races
			store.Validate(state)
		}
	}()

	// Debug caller goroutine
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			store.Debug()
			time.Sleep(time.Microsecond * 100)
		}
	}()

	wg.Wait()
}

// TestMemoryStateStoreMemoryUsage tests memory behavior with many states
func TestMemoryStateStoreMemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory usage test in short mode")
	}

	store := NewMemoryStateStore()

	// Store a large number of states
	const numStates = 10000
	states := make([]string, numStates)

	for i := 0; i < numStates; i++ {
		state := fmt.Sprintf("memory-test-state-%d", i)
		states[i] = state
		if !store.Store(state) {
			t.Errorf("Failed to store state %d", i)
		}
	}

	// Validate all states (this should remove them from memory)
	for i, state := range states {
		if !store.Validate(state) {
			t.Errorf("Failed to validate state %d: %s", i, state)
		}
	}

	// Try to validate again (should all fail)
	for i, state := range states {
		if store.Validate(state) {
			t.Errorf("State %d should not validate twice: %s", i, state)
		}
	}
}

// TestStateStoreImplementation verifies the StateStore interface contract
func TestStateStoreImplementation(t *testing.T) {
	// Test that MemoryStateStore implements StateStore
	var _ StateStore = &MemoryStateStore{}

	store := NewMemoryStateStore()

	// Test the expected behavior according to interface documentation
	state := "contract-test"

	// 1. Store should return true for successful storage
	if !store.Store(state) {
		t.Error("Store should return true for successful storage")
	}

	// 2. Validate should return true for valid state and remove it
	if !store.Validate(state) {
		t.Error("Validate should return true for valid state")
	}

	// 3. Second validate should return false (consume-once pattern)
	if store.Validate(state) {
		t.Error("Second validate should return false (consume-once pattern)")
	}

	// 4. Validate non-existent state should return false
	if store.Validate("nonexistent") {
		t.Error("Validate should return false for non-existent state")
	}
}

// BenchmarkMemoryStateStoreStore benchmarks the Store operation
func BenchmarkMemoryStateStoreStore(b *testing.B) {
	store := NewMemoryStateStore()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		state := fmt.Sprintf("bench-state-%d", i)
		store.Store(state)
	}
}

// BenchmarkMemoryStateStoreValidate benchmarks the Validate operation
func BenchmarkMemoryStateStoreValidate(b *testing.B) {
	store := NewMemoryStateStore()

	// Pre-populate with states
	states := make([]string, b.N)
	for i := 0; i < b.N; i++ {
		state := fmt.Sprintf("bench-state-%d", i)
		states[i] = state
		store.Store(state)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.Validate(states[i])
	}
}

// BenchmarkMemoryStateStoreConcurrent benchmarks concurrent operations
func BenchmarkMemoryStateStoreConcurrent(b *testing.B) {
	store := NewMemoryStateStore()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			state := fmt.Sprintf("concurrent-bench-%d", i)
			store.Store(state)
			store.Validate(state)
			i++
		}
	})
}
