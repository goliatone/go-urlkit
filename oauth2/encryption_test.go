package oauth2

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// TestEncryptDecryptState tests the round-trip encryption and decryption of state data
func TestEncryptDecryptState(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		state       string
		data        interface{}
		expectError bool
	}{
		{
			name:  "simple string data",
			key:   "this-is-a-24-char-key-ok",
			state: "test-state",
			data:  "simple string",
		},
		{
			name:  "struct data",
			key:   "this-is-a-24-char-key-ok",
			state: "test-state-struct",
			data: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "123", Name: "Test User"},
		},
		{
			name:  "complex nested data",
			key:   "this-is-a-32-character-key-ok!!!",
			state: "complex-state",
			data: map[string]interface{}{
				"user_id":   "user123",
				"return_to": "/dashboard",
				"metadata": map[string]string{
					"source": "web",
					"device": "desktop",
				},
				"timestamp": time.Now().Unix(),
			},
		},
		{
			name:  "empty data",
			key:   "this-is-a-24-char-key-ok",
			state: "empty-state",
			data:  "",
		},
		{
			name:  "nil data",
			key:   "this-is-a-24-char-key-ok",
			state: "nil-state",
			data:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt the state
			encrypted, err := EncryptState([]byte(tt.key), tt.state, tt.data)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}
			if err != nil {
				t.Fatalf("encryption failed: %v", err)
			}

			// Verify encrypted state is not empty and different from original
			if encrypted == "" {
				t.Error("encrypted state is empty")
			}
			if encrypted == tt.state {
				t.Error("encrypted state should be different from original")
			}

			// Decrypt the state
			decryptedState, decryptedData, err := DecryptState[interface{}]([]byte(tt.key), encrypted)
			if err != nil {
				t.Fatalf("decryption failed: %v", err)
			}

			// Verify decrypted state matches original
			if decryptedState != tt.state {
				t.Errorf("decrypted state mismatch: got %q, want %q", decryptedState, tt.state)
			}

			// For complex comparisons, we'll do a basic check
			if tt.data != nil && decryptedData == nil {
				t.Error("decrypted data is nil when original was not")
			}
		})
	}
}

// TestEncryptStateErrors tests error conditions in state encryption
func TestEncryptStateErrors(t *testing.T) {
	tests := []struct {
		name        string
		key         []byte
		state       string
		data        interface{}
		expectError error
	}{
		{
			name:        "invalid key length - too short",
			key:         []byte("short"),
			state:       "test",
			data:        "data",
			expectError: ErrEncryptionFailed,
		},
		{
			name:        "empty key",
			key:         []byte(""),
			state:       "test",
			data:        "data",
			expectError: ErrEncryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := EncryptState(tt.key, tt.state, tt.data)
			if err == nil {
				t.Error("expected error but got none")
			}
			if !errors.Is(err, tt.expectError) {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

// TestDecryptStateErrors tests error conditions in state decryption
func TestDecryptStateErrors(t *testing.T) {
	validKey := []byte("this-is-a-24-char-key-ok")

	// Create a valid encrypted state first
	validEncrypted, err := EncryptState(validKey, "test-state", "test-data")
	if err != nil {
		t.Fatalf("failed to create valid encrypted state: %v", err)
	}

	tests := []struct {
		name          string
		key           []byte
		encryptedData string
		expectError   error
	}{
		{
			name:          "invalid key",
			key:           []byte("wrong-key"),
			encryptedData: validEncrypted,
			expectError:   ErrDecryptionFailed,
		},
		{
			name:          "empty encrypted data",
			key:           validKey,
			encryptedData: "",
			expectError:   ErrDecryptionFailed,
		},
		{
			name:          "corrupted encrypted data",
			key:           validKey,
			encryptedData: "corrupted-data-that-is-not-base64",
			expectError:   ErrDecryptionFailed,
		},
		{
			name:          "invalid base64",
			key:           validKey,
			encryptedData: "invalid-base64-!@#$%^&*()",
			expectError:   ErrDecryptionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := DecryptState[string](tt.key, tt.encryptedData)
			if err == nil {
				t.Error("expected error but got none")
			}
			if !errors.Is(err, tt.expectError) {
				t.Errorf("expected error %v, got %v", tt.expectError, err)
			}
		})
	}
}

// TestEncryptionKeyValidation tests key validation for different lengths
func TestEncryptionKeyValidation(t *testing.T) {
	testData := "test-data"
	testState := "test-state"

	// Test valid key lengths (16, 24, 32 bytes for AES)
	validKeys := []string{
		"16-char-key-test",                 // 16 chars
		"this-is-a-24-char-key-ok",         // 24 chars
		"this-is-a-32-character-key-ok!!!", // 32 chars
	}

	for _, key := range validKeys {
		t.Run("valid_key_"+string(rune(len(key))), func(t *testing.T) {
			encrypted, err := EncryptState([]byte(key), testState, testData)
			if err != nil {
				t.Errorf("valid key length %d should work: %v", len(key), err)
			}

			// Verify decryption works
			_, _, err = DecryptState[string]([]byte(key), encrypted)
			if err != nil {
				t.Errorf("decryption with valid key should work: %v", err)
			}
		})
	}
}

// TestEncryptionDeterminism tests that encryption produces different results each time
func TestEncryptionDeterminism(t *testing.T) {
	key := []byte("this-is-a-24-char-key-ok")
	state := "test-state"
	data := "test-data"

	// Encrypt the same data multiple times
	encrypted1, err1 := EncryptState(key, state, data)
	encrypted2, err2 := EncryptState(key, state, data)

	if err1 != nil || err2 != nil {
		t.Fatalf("encryption should not fail: err1=%v, err2=%v", err1, err2)
	}

	// Results should be different due to random IV
	if encrypted1 == encrypted2 {
		t.Error("encryption should produce different results each time due to random IV")
	}

	// But both should decrypt to the same values
	state1, data1, err1 := DecryptState[string](key, encrypted1)
	state2, data2, err2 := DecryptState[string](key, encrypted2)

	if err1 != nil || err2 != nil {
		t.Fatalf("decryption should not fail: err1=%v, err2=%v", err1, err2)
	}

	if state1 != state2 || data1 != data2 {
		t.Error("decrypted values should be identical")
	}

	if state1 != state || data1 != data {
		t.Error("decrypted values should match original")
	}
}

// TestEncryptionWithDifferentTypes tests encryption/decryption with various Go types
func TestEncryptionWithDifferentTypes(t *testing.T) {
	key := []byte("this-is-a-24-char-key-ok")
	state := "test-state"

	type CustomStruct struct {
		Name      string    `json:"name"`
		Age       int       `json:"age"`
		Active    bool      `json:"active"`
		CreatedAt time.Time `json:"created_at"`
	}

	testTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)
	testStruct := CustomStruct{
		Name:      "Test User",
		Age:       30,
		Active:    true,
		CreatedAt: testTime,
	}

	// Test with struct
	encrypted, err := EncryptState(key, state, testStruct)
	if err != nil {
		t.Fatalf("encryption failed: %v", err)
	}

	decryptedState, decryptedStruct, err := DecryptState[CustomStruct](key, encrypted)
	if err != nil {
		t.Fatalf("decryption failed: %v", err)
	}

	if decryptedState != state {
		t.Errorf("state mismatch: got %q, want %q", decryptedState, state)
	}

	if decryptedStruct.Name != testStruct.Name {
		t.Errorf("name mismatch: got %q, want %q", decryptedStruct.Name, testStruct.Name)
	}
	if decryptedStruct.Age != testStruct.Age {
		t.Errorf("age mismatch: got %d, want %d", decryptedStruct.Age, testStruct.Age)
	}
	if decryptedStruct.Active != testStruct.Active {
		t.Errorf("active mismatch: got %v, want %v", decryptedStruct.Active, testStruct.Active)
	}
}

// TestEncryptionPerformance tests encryption performance
func TestEncryptionPerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping performance test in short mode")
	}

	key := []byte("this-is-a-24-char-key-ok")
	state := "performance-test-state"
	data := strings.Repeat("performance test data ", 100) // ~2KB of data

	const iterations = 1000

	// Measure encryption performance
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := EncryptState(key, state, data)
		if err != nil {
			t.Fatalf("encryption failed on iteration %d: %v", i, err)
		}
	}
	encryptionTime := time.Since(start)

	// Create encrypted data for decryption test
	encrypted, err := EncryptState(key, state, data)
	if err != nil {
		t.Fatalf("failed to create encrypted data: %v", err)
	}

	// Measure decryption performance
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _, err := DecryptState[string](key, encrypted)
		if err != nil {
			t.Fatalf("decryption failed on iteration %d: %v", i, err)
		}
	}
	decryptionTime := time.Since(start)

	t.Logf("Encryption: %d iterations in %v (%.2f ops/sec)",
		iterations, encryptionTime, float64(iterations)/encryptionTime.Seconds())
	t.Logf("Decryption: %d iterations in %v (%.2f ops/sec)",
		iterations, decryptionTime, float64(iterations)/decryptionTime.Seconds())

	// Performance benchmarks (adjust these based on your requirements)
	maxTimePerOp := time.Millisecond * 10
	if encryptionTime/iterations > maxTimePerOp {
		t.Errorf("encryption too slow: %v per operation (max %v)", encryptionTime/iterations, maxTimePerOp)
	}
	if decryptionTime/iterations > maxTimePerOp {
		t.Errorf("decryption too slow: %v per operation (max %v)", decryptionTime/iterations, maxTimePerOp)
	}
}

// BenchmarkEncryptState benchmarks the EncryptState function
func BenchmarkEncryptState(b *testing.B) {
	key := []byte("this-is-a-24-char-key-ok")
	state := "benchmark-state"
	data := "benchmark data for testing encryption performance"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := EncryptState(key, state, data)
		if err != nil {
			b.Fatalf("encryption failed: %v", err)
		}
	}
}

// BenchmarkDecryptState benchmarks the DecryptState function
func BenchmarkDecryptState(b *testing.B) {
	key := []byte("this-is-a-24-char-key-ok")
	state := "benchmark-state"
	data := "benchmark data for testing decryption performance"

	// Prepare encrypted data
	encrypted, err := EncryptState(key, state, data)
	if err != nil {
		b.Fatalf("failed to prepare encrypted data: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := DecryptState[string](key, encrypted)
		if err != nil {
			b.Fatalf("decryption failed: %v", err)
		}
	}
}
