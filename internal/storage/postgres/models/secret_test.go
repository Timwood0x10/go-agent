// Package models provides comprehensive tests for secret model.
package models

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestSecret_AlgorithmConstants tests algorithm constants.
func TestSecret_AlgorithmConstants(t *testing.T) {
	assert.Equal(t, "aes-gcm", AlgorithmAESGCM)
}

// TestSecret_TableName tests table name returns correct value.
func TestSecret_TableName(t *testing.T) {
	secret := &Secret{}
	assert.Equal(t, "secrets", secret.TableName())
}

// TestSecret_ValidFields tests valid field assignment.
func TestSecret_ValidFields(t *testing.T) {
	secret := &Secret{
		ID:         "secret-id",
		TenantID:   "tenant-1",
		Key:        "api_key",
		Value:      []byte("encrypted_value"),
		KeyVersion: 1,
		Algorithm:  AlgorithmAESGCM,
		ExpiresAt:  time.Now().Add(24 * time.Hour),
		Metadata:   map[string]interface{}{"key": "value"},
		CreatedAt:  time.Now(),
	}

	assert.Equal(t, "secret-id", secret.ID)
	assert.Equal(t, "tenant-1", secret.TenantID)
	assert.Equal(t, "api_key", secret.Key)
	assert.NotNil(t, secret.Value)
	assert.Equal(t, 1, secret.KeyVersion)
	assert.Equal(t, AlgorithmAESGCM, secret.Algorithm)
	assert.False(t, secret.IsExpired())
	assert.True(t, secret.IsValid())
}

// TestSecret_EmptyFields tests handling of empty fields.
func TestSecret_EmptyFields(t *testing.T) {
	secret := &Secret{}

	assert.Empty(t, secret.ID)
	assert.Empty(t, secret.TenantID)
	assert.Empty(t, secret.Key)
	assert.Nil(t, secret.Value)
	assert.Equal(t, 0, secret.KeyVersion)
	assert.Empty(t, secret.Algorithm)
	assert.True(t, secret.ExpiresAt.IsZero())
	assert.Nil(t, secret.Metadata)
	assert.True(t, secret.CreatedAt.IsZero())
	assert.False(t, secret.IsValid())
}

// TestSecret_EmptyValue tests handling of empty value.
func TestSecret_EmptyValue(t *testing.T) {
	secret := &Secret{
		ID:         "secret-id",
		Key:        "api_key",
		Value:      []byte{},
		KeyVersion: 1,
		Algorithm:  AlgorithmAESGCM,
		CreatedAt:  time.Now(),
	}

	assert.Empty(t, secret.Value)
	assert.False(t, secret.IsValid())
}

// TestSecret_NilValue tests handling of nil value.
func TestSecret_NilValue(t *testing.T) {
	secret := &Secret{
		ID:         "secret-id",
		Key:        "api_key",
		Value:      nil,
		KeyVersion: 1,
		Algorithm:  AlgorithmAESGCM,
		CreatedAt:  time.Now(),
	}

	assert.Nil(t, secret.Value)
	assert.False(t, secret.IsValid())
}

// TestSecret_IsExpired tests expiration logic.
func TestSecret_IsExpired(t *testing.T) {
	tests := []struct {
		name      string
		expiresAt time.Time
		expected  bool
	}{
		{
			name:      "expired secret",
			expiresAt: time.Now().Add(-1 * time.Hour),
			expected:  true,
		},
		{
			name:      "not expired secret",
			expiresAt: time.Now().Add(1 * time.Hour),
			expected:  false,
		},
		{
			name:      "zero expires time",
			expiresAt: time.Time{},
			expected:  false,
		},
		{
			name:      "exactly expired",
			expiresAt: time.Now(),
			expected:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := &Secret{
				ID:        "secret-id",
				Key:       "api_key",
				Value:     []byte("encrypted_value"),
				ExpiresAt: tt.expiresAt,
				CreatedAt: time.Now(),
			}
			assert.Equal(t, tt.expected, secret.IsExpired())
		})
	}
}

// TestSecret_IsValid tests validity check.
func TestSecret_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		secret   *Secret
		expected bool
	}{
		{
			name: "valid secret",
			secret: &Secret{
				ID:        "secret-id",
				Key:       "api_key",
				Value:     []byte("encrypted_value"),
				ExpiresAt: time.Now().Add(1 * time.Hour),
				CreatedAt: time.Now(),
			},
			expected: true,
		},
		{
			name: "expired secret",
			secret: &Secret{
				ID:        "secret-id",
				Key:       "api_key",
				Value:     []byte("encrypted_value"),
				ExpiresAt: time.Now().Add(-1 * time.Hour),
				CreatedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "empty value",
			secret: &Secret{
				ID:        "secret-id",
				Key:       "api_key",
				Value:     []byte{},
				ExpiresAt: time.Now().Add(1 * time.Hour),
				CreatedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "nil value",
			secret: &Secret{
				ID:        "secret-id",
				Key:       "api_key",
				Value:     nil,
				ExpiresAt: time.Now().Add(1 * time.Hour),
				CreatedAt: time.Now(),
			},
			expected: false,
		},
		{
			name: "no expiration",
			secret: &Secret{
				ID:        "secret-id",
				Key:       "api_key",
				Value:     []byte("encrypted_value"),
				ExpiresAt: time.Time{},
				CreatedAt: time.Now(),
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.secret.IsValid())
		})
	}
}

// TestSecret_KeyVersion tests key version handling.
func TestSecret_KeyVersion(t *testing.T) {
	tests := []struct {
		name       string
		keyVersion int
	}{
		{"initial version", 1},
		{"second version", 2},
		{"high version", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := &Secret{
				ID:         "secret-id",
				Key:        "api_key",
				Value:      []byte("encrypted_value"),
				KeyVersion: tt.keyVersion,
				Algorithm:  AlgorithmAESGCM,
				CreatedAt:  time.Now(),
			}
			assert.Equal(t, tt.keyVersion, secret.KeyVersion)
		})
	}
}

// TestSecret_Algorithm tests algorithm field.
func TestSecret_Algorithm(t *testing.T) {
	secret := &Secret{
		ID:        "secret-id",
		Key:       "api_key",
		Value:     []byte("encrypted_value"),
		Algorithm: AlgorithmAESGCM,
		CreatedAt: time.Now(),
	}

	assert.Equal(t, AlgorithmAESGCM, secret.Algorithm)
}

// TestSecret_MetadataComplex tests handling of complex metadata.
func TestSecret_MetadataComplex(t *testing.T) {
	secret := &Secret{
		ID:    "secret-id",
		Key:   "api_key",
		Value: []byte("encrypted_value"),
		Metadata: map[string]interface{}{
			"string": "value",
			"number": 123,
			"bool":   true,
			"array":  []string{"a", "b", "c"},
			"object": map[string]interface{}{"nested": "value"},
		},
		CreatedAt: time.Now(),
	}

	assert.NotNil(t, secret.Metadata)
	assert.Equal(t, "value", secret.Metadata["string"])
	assert.Equal(t, 123, secret.Metadata["number"])
	assert.True(t, secret.Metadata["bool"].(bool))
	assert.Len(t, secret.Metadata["array"].([]string), 3)
	assert.NotNil(t, secret.Metadata["object"])
}

// TestSecret_NilMetadata tests handling of nil metadata.
func TestSecret_NilMetadata(t *testing.T) {
	secret := &Secret{
		ID:        "secret-id",
		Key:       "api_key",
		Value:     []byte("encrypted_value"),
		Metadata:  nil,
		CreatedAt: time.Now(),
	}

	assert.Nil(t, secret.Metadata)
}

// TestSecret_EmptyKey tests handling of empty key.
func TestSecret_EmptyKey(t *testing.T) {
	secret := &Secret{
		ID:        "secret-id",
		Key:       "",
		Value:     []byte("encrypted_value"),
		CreatedAt: time.Now(),
	}

	assert.Empty(t, secret.Key)
}

// TestSecret_LongValue tests handling of long encrypted value.
func TestSecret_LongValue(t *testing.T) {
	longValue := make([]byte, 10000)
	for i := range longValue {
		longValue[i] = 'a'
	}

	secret := &Secret{
		ID:        "secret-id",
		Key:       "api_key",
		Value:     longValue,
		CreatedAt: time.Now(),
	}

	assert.Len(t, secret.Value, 10000)
	assert.True(t, secret.IsValid())
}

// TestSecret_MultipleVersions tests handling of multiple key versions.
func TestSecret_MultipleVersions(t *testing.T) {
	secrets := make([]*Secret, 5)
	for i := 0; i < 5; i++ {
		secrets[i] = &Secret{
			ID:         "secret-id",
			Key:        "api_key",
			Value:      []byte("encrypted_value"),
			KeyVersion: i + 1,
			Algorithm:  AlgorithmAESGCM,
			CreatedAt:  time.Now(),
		}
		assert.Equal(t, i+1, secrets[i].KeyVersion)
	}
}

// TestSecret_ZeroKeyVersion tests handling of zero key version.
func TestSecret_ZeroKeyVersion(t *testing.T) {
	secret := &Secret{
		ID:         "secret-id",
		Key:        "api_key",
		Value:      []byte("encrypted_value"),
		KeyVersion: 0,
		Algorithm:  AlgorithmAESGCM,
		CreatedAt:  time.Now(),
	}

	assert.Equal(t, 0, secret.KeyVersion)
}
