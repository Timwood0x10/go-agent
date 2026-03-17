// Package models defines data structures for the storage system.
package models

import "time"

// Secret represents sensitive data with encryption.
// This stores API keys, passwords, and other sensitive information with AES-GCM encryption.
type Secret struct {
	ID         string                 `json:"id"`
	TenantID   string                 `json:"tenant_id"`
	Key        string                 `json:"key"`
	Value      []byte                 `json:"value"`
	KeyVersion int                    `json:"key_version"`
	Algorithm  string                 `json:"algorithm"`
	ExpiresAt  time.Time              `json:"expires_at"`
	Metadata   map[string]interface{} `json:"metadata"`
	CreatedAt  time.Time              `json:"created_at"`
}

// TableName returns the table name for this model.
func (s *Secret) TableName() string {
	return "secrets"
}

// EncryptionAlgorithm constants.
const (
	AlgorithmAESGCM = "aes-gcm"
)

// IsExpired checks if the secret has expired.
func (s *Secret) IsExpired() bool {
	return !s.ExpiresAt.IsZero() && time.Now().After(s.ExpiresAt)
}

// IsValid checks if the secret is valid and not expired.
func (s *Secret) IsValid() bool {
	return !s.IsExpired() && len(s.Value) > 0
}
