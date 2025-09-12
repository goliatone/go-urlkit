package securelink

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// BenchmarkManagerGenerate benchmarks the Manager.Generate method with various scenarios
func BenchmarkManagerGenerate(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32), // 32 bytes for HS256
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		payload := Payload{
			"user_id": "123",
			"action":  "test",
		}
		_, err := manager.Generate("test", payload)
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}

// BenchmarkManagerGenerateNoPayload benchmarks Generate method with no payload
func BenchmarkManagerGenerateNoPayload(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32),
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := manager.Generate("test")
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}

// BenchmarkManagerGenerateMultiplePayloads benchmarks Generate with multiple payloads
func BenchmarkManagerGenerateMultiplePayloads(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32),
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		payload1 := Payload{"user_id": "123"}
		payload2 := Payload{"session": "abc"}
		payload3 := Payload{"role": "admin"}

		_, err := manager.Generate("test", payload1, payload2, payload3)
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}

// BenchmarkManagerGenerateQueryBased benchmarks query-based URL generation
func BenchmarkManagerGenerateQueryBased(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32),
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    true, // Query-based URLs
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	payload := Payload{
		"user_id": "123",
		"action":  "test",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := manager.Generate("test", payload)
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}

// BenchmarkManagerValidate benchmarks the Manager.Validate method
func BenchmarkManagerValidate(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32),
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	// Generate a token to validate
	payload := Payload{
		"user_id": "123",
		"action":  "test",
	}
	link, err := manager.Generate("test", payload)
	if err != nil {
		b.Fatalf("Failed to generate link: %v", err)
	}

	// Extract token from link
	token := strings.TrimPrefix(link, "https://benchmark.example.com/test/")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := manager.Validate(token)
		if err != nil {
			b.Fatalf("Validate failed: %v", err)
		}
	}
}

// BenchmarkInternalGenerate benchmarks the low-level Generate function
func BenchmarkInternalGenerate(b *testing.B) {
	signingKey := strings.Repeat("a", 32)
	expiration := 1 * time.Hour
	signingMethod := jwt.SigningMethodHS256
	data := map[string]any{
		"user_id": "123",
		"action":  "test",
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Generate(data, signingKey, expiration, signingMethod)
		if err != nil {
			b.Fatalf("Generate failed: %v", err)
		}
	}
}

// BenchmarkInternalValidate benchmarks the low-level Validate function
func BenchmarkInternalValidate(b *testing.B) {
	signingKey := strings.Repeat("a", 32)
	expiration := 1 * time.Hour
	signingMethod := jwt.SigningMethodHS256
	data := map[string]any{
		"user_id": "123",
		"action":  "test",
	}

	// Generate a token to validate
	token, err := Generate(data, signingKey, expiration, signingMethod)
	if err != nil {
		b.Fatalf("Failed to generate token: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := Validate(token, signingKey, signingMethod)
		if err != nil {
			b.Fatalf("Validate failed: %v", err)
		}
	}
}

// BenchmarkDifferentSigningMethods compares performance across different algorithms
func BenchmarkDifferentSigningMethods(b *testing.B) {
	testCases := []struct {
		name          string
		signingMethod jwt.SigningMethod
		keyLength     int
	}{
		{"HS256", jwt.SigningMethodHS256, 32},
		{"HS384", jwt.SigningMethodHS384, 48},
		{"HS512", jwt.SigningMethodHS512, 64},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			cfg := Config{
				SigningKey:    strings.Repeat("a", tc.keyLength),
				Expiration:    1 * time.Hour,
				BaseURL:       "https://benchmark.example.com",
				QueryKey:      "token",
				Routes:        map[string]string{"test": "/test"},
				AsQuery:       false,
				SigningMethod: tc.signingMethod,
			}

			manager, err := NewManagerFromConfig(cfg)
			if err != nil {
				b.Fatalf("Failed to create manager: %v", err)
			}

			payload := Payload{
				"user_id": "123",
				"action":  "test",
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := manager.Generate("test", payload)
				if err != nil {
					b.Fatalf("Generate failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkConcurrentGenerate benchmarks Generate under concurrent load
func BenchmarkConcurrentGenerate(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32),
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			payload := Payload{
				"user_id": "123",
				"action":  "test",
			}
			_, err := manager.Generate("test", payload)
			if err != nil {
				b.Fatalf("Generate failed: %v", err)
			}
		}
	})
}

// BenchmarkConcurrentValidate benchmarks Validate under concurrent load
func BenchmarkConcurrentValidate(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32),
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	// Generate a token to validate
	payload := Payload{
		"user_id": "123",
		"action":  "test",
	}
	link, err := manager.Generate("test", payload)
	if err != nil {
		b.Fatalf("Failed to generate link: %v", err)
	}
	token := strings.TrimPrefix(link, "https://benchmark.example.com/test/")

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := manager.Validate(token)
			if err != nil {
				b.Fatalf("Validate failed: %v", err)
			}
		}
	})
}

// BenchmarkPayloadSizes compares performance with different payload sizes
func BenchmarkPayloadSizes(b *testing.B) {
	cfg := Config{
		SigningKey: strings.Repeat("a", 32),
		Expiration: 1 * time.Hour,
		BaseURL:    "https://benchmark.example.com",
		QueryKey:   "token",
		Routes:     map[string]string{"test": "/test"},
		AsQuery:    false,
	}

	manager, err := NewManagerFromConfig(cfg)
	if err != nil {
		b.Fatalf("Failed to create manager: %v", err)
	}

	testCases := []struct {
		name    string
		payload Payload
	}{
		{
			name: "Small",
			payload: Payload{
				"id": "123",
			},
		},
		{
			name: "Medium",
			payload: Payload{
				"user_id":   "123",
				"action":    "test",
				"timestamp": time.Now().Unix(),
				"session":   "abc123def456",
				"role":      "admin",
			},
		},
		{
			name: "Large",
			payload: Payload{
				"user_id":     "123",
				"action":      "test",
				"timestamp":   time.Now().Unix(),
				"session":     "abc123def456ghi789jkl012",
				"role":        "admin",
				"permissions": []string{"read", "write", "delete", "admin"},
				"metadata": map[string]any{
					"ip":         "192.168.1.100",
					"user_agent": "Mozilla/5.0 (Test Browser)",
					"referrer":   "https://example.com/login",
				},
				"description": strings.Repeat("x", 100), // 100 characters
			},
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, err := manager.Generate("test", tc.payload)
				if err != nil {
					b.Fatalf("Generate failed: %v", err)
				}
			}
		})
	}
}
