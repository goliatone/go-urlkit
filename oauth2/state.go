package oauth2

import (
	"fmt"
	"sync"
)

// StateStore defines the interface for managing OAuth state tokens during the authorization flow.
// State tokens are used to prevent CSRF attacks by ensuring that the authorization response
// corresponds to a request that was initiated by the same client.
//
// Implementations must be thread-safe as they may be accessed concurrently by multiple
// goroutines handling different OAuth flows simultaneously.
//
// The typical flow is:
//  1. Store() - save a state token before redirecting to OAuth provider
//  2. Validate() - verify the state token when handling the callback
//     (this should also remove the token to prevent replay attacks)
type StateStore interface {
	// Store saves a state token for later validation.
	// Returns true if the state was successfully stored, false otherwise.
	//
	// The implementation should ensure the state is stored securely and
	// can be retrieved later for validation. States should have reasonable
	// expiration times to prevent accumulation of stale tokens.
	Store(state string) bool

	// Validate checks if a state token is valid and removes it from storage.
	// Returns true if the state was found and valid, false otherwise.
	//
	// This method implements a "consume-once" pattern - once a state is validated,
	// it should be removed from storage to prevent replay attacks. If the same
	// state is validated again, it should return false.
	Validate(state string) bool

	// Debug outputs debugging information about stored states.
	// This method is intended for development and troubleshooting purposes.
	// Production implementations may choose to make this a no-op for security.
	Debug()
}

// Compile-time check to ensure MemoryStateStore implements StateStore interface
var _ StateStore = &MemoryStateStore{}

// MemoryStateStore is an in-memory implementation of StateStore interface.
// It stores state tokens in a map and provides thread-safe operations using a mutex.
//
// This implementation is suitable for development, testing, and single-instance
// applications. For distributed applications or high-availability scenarios,
// consider implementing a persistent StateStore using Redis, database, or
// other shared storage systems.
//
// Security considerations:
// - States are stored in memory and will be lost on application restart
// - No automatic expiration - states persist until validated or app restart
// - All operations are protected by mutex for thread-safety
// - States are permanently removed after validation (consume-once pattern)
type MemoryStateStore struct {
	// states maps state tokens to empty structs for memory efficiency
	// Using struct{} as value type minimizes memory overhead
	states map[string]struct{}

	// mx protects concurrent access to the states map
	// All public methods must acquire this mutex before accessing states
	mx sync.Mutex
}

// NewMemoryStateStore creates a new in-memory state store.
// The returned store is ready to use and thread-safe.
//
// Example usage:
//
//	store := NewMemoryStateStore()
//	state := "random-state-token"
//
//	// Store state before OAuth redirect
//	if !store.Store(state) {
//		// handle storage error
//	}
//
//	// Later, validate state from OAuth callback
//	if !store.Validate(state) {
//		// handle invalid or missing state
//	}
func NewMemoryStateStore() *MemoryStateStore {
	return &MemoryStateStore{
		states: make(map[string]struct{}),
	}
}

// Store saves a state token for later validation.
// This method is thread-safe and can be called concurrently.
//
// Parameters:
//   - state: The state token to store. Should be a unique, unpredictable value.
//
// Returns:
//   - bool: Always returns true for this implementation. Other implementations
//     might return false if storage fails (e.g., database errors, capacity limits).
//
// The state token will remain stored until it is validated (which removes it)
// or the application restarts (memory-based storage).
func (s *MemoryStateStore) Store(state string) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	s.states[state] = struct{}{}
	return true
}

// Validate checks if a state token exists and removes it from storage.
// This method implements the "consume-once" security pattern to prevent replay attacks.
// This method is thread-safe and can be called concurrently.
//
// Parameters:
//   - state: The state token to validate and remove.
//
// Returns:
//   - bool: true if the state was found and successfully removed,
//     false if the state was not found or already consumed.
//
// Security note: Once a state is validated, it cannot be validated again.
// This prevents an attacker from reusing a valid state token they may have
// intercepted during the OAuth flow.
func (s *MemoryStateStore) Validate(state string) bool {
	s.mx.Lock()
	defer s.mx.Unlock()

	_, exists := s.states[state]
	if exists {
		// Remove the state after validation (consume-once pattern)
		delete(s.states, state)
		return true
	}
	return false
}

// Debug outputs information about currently stored states to stdout.
// This method is intended for development and debugging purposes only.
// This method is thread-safe and can be called concurrently.
//
// The output includes:
// - A header indicating this is memory store debug output
// - Each stored state token (one per line)
// - A footer to clearly mark the end of debug output
//
// Security warning: This method exposes state tokens in plain text.
// Production implementations should either disable this method or
// ensure it's only available in development/debug builds.
func (s *MemoryStateStore) Debug() {
	s.mx.Lock()
	defer s.mx.Unlock()

	fmt.Println("=== Memory Store Debug ====")
	for state := range s.states {
		fmt.Println("state " + state)
	}
	fmt.Println("=== End Memory Store Debug ====")
}
