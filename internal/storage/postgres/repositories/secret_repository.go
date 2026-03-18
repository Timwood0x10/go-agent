// Package repositories provides data access layer for storage system.
package repositories

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"goagent/internal/core/errors"
	storage_models "goagent/internal/storage/postgres/models"
)

// SecretRepository provides data access for encrypted sensitive data.
// This implements secure storage and retrieval of secrets with encryption.
type SecretRepository struct {
	db            *sql.DB
	encryptionKey []byte
}

// NewSecretRepository creates a new SecretRepository instance.
// Args:
// db - database connection.
// encryptionKey - encryption key (32 bytes for AES-256).
// Returns new SecretRepository instance.
func NewSecretRepository(db *sql.DB, encryptionKey []byte) *SecretRepository {
	return &SecretRepository{
		db:            db,
		encryptionKey: encryptionKey,
	}
}

// Set stores a secret value with encryption.
// Args:
// ctx - database operation context.
// key - secret key.
// value - secret value to store.
// tenantID - tenant identifier for isolation.
// Returns error if storage operation fails.
func (r *SecretRepository) Set(ctx context.Context, key, value, tenantID string) error {
	// Encrypt the value
	encrypted, err := r.encrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	query := `
		INSERT INTO secrets
		(id, tenant_id, key, value, key_version, algorithm, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, 1, 'aes-gcm', NOW())
		ON CONFLICT (tenant_id, key) DO UPDATE SET
			value = EXCLUDED.value,
			key_version = secrets.key_version + 1,
			updated_at = NOW()
		RETURNING id
	`

	var id string
	err = r.db.QueryRowContext(ctx, query, tenantID, key, encrypted).Scan(&id)
	if err != nil {
		return fmt.Errorf("set secret: %w", err)
	}

	return nil
}

// Get retrieves and decrypts a secret value.
// Args:
// ctx - database operation context.
// key - secret key.
// tenantID - tenant identifier for isolation.
// Returns decrypted secret value or error if not found.
func (r *SecretRepository) Get(ctx context.Context, key, tenantID string) (string, error) {
	query := `
		SELECT id, tenant_id, key, value, key_version, algorithm, expires_at
		FROM secrets
		WHERE key = $1 AND tenant_id = $2
	`

	var id, tenant, secretKey string
	var encryptedValue []byte
	var keyVersion int
	var algorithm string
	var expiresAt *time.Time

	err := r.db.QueryRowContext(ctx, query, key, tenantID).Scan(
		&id, &tenant, &secretKey, &encryptedValue, &keyVersion, &algorithm, &expiresAt,
	)

	if err == sql.ErrNoRows {
		return "", errors.ErrRecordNotFound
	}
	if err != nil {
		return "", fmt.Errorf("get secret: %w", err)
	}

	// Check if secret has expired
	if expiresAt != nil && time.Now().After(*expiresAt) {
		return "", errors.ErrSecretExpired
	}

	// Decrypt the value
	decrypted, err := r.decrypt(encryptedValue)
	if err != nil {
		return "", fmt.Errorf("decrypt secret: %w", err)
	}

	return string(decrypted), nil
}

// Delete removes a secret by its key.
// Args:
// ctx - database operation context.
// key - secret key.
// tenantID - tenant identifier for isolation.
// Returns error if delete operation fails.
func (r *SecretRepository) Delete(ctx context.Context, key, tenantID string) error {
	query := `DELETE FROM secrets WHERE key = $1 AND tenant_id = $2`

	result, err := r.db.ExecContext(ctx, query, key, tenantID)
	if err != nil {
		return fmt.Errorf("delete secret: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// List retrieves all secret keys for a tenant.
// Args:
// ctx - database operation context.
// tenantID - tenant identifier for isolation.
// Returns list of secret metadata (without values) or error if query fails.
func (r *SecretRepository) List(ctx context.Context, tenantID string) ([]*storage_models.Secret, error) {
	query := `
		SELECT id, tenant_id, key, key_version, algorithm, expires_at, created_at
		FROM secrets
		WHERE tenant_id = $1
		ORDER BY key ASC
	`

	rows, err := r.db.QueryContext(ctx, query, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}
	defer rows.Close()

	secrets := make([]*storage_models.Secret, 0)
	for rows.Next() {
		secret := &storage_models.Secret{}
		err := rows.Scan(
			&secret.ID, &secret.TenantID, &secret.Key,
			&secret.KeyVersion, &secret.Algorithm, &secret.ExpiresAt, &secret.CreatedAt,
		)
		if err != nil {
			continue
		}
		secrets = append(secrets, secret)
	}

	return secrets, nil
}

// SetWithExpiration stores a secret value with expiration time.
// Args:
// ctx - database operation context.
// key - secret key.
// value - secret value to store.
// tenantID - tenant identifier for isolation.
// expiresAt - expiration time.
// Returns error if storage operation fails.
func (r *SecretRepository) SetWithExpiration(ctx context.Context, key, value, tenantID string, expiresAt time.Time) error {
	// Encrypt the value
	encrypted, err := r.encrypt([]byte(value))
	if err != nil {
		return fmt.Errorf("encrypt secret: %w", err)
	}

	query := `
		INSERT INTO secrets
		(id, tenant_id, key, value, key_version, algorithm, expires_at, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, 1, 'aes-gcm', $4, NOW())
		ON CONFLICT (tenant_id, key) DO UPDATE SET
			value = EXCLUDED.value,
			key_version = secrets.key_version + 1,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
		RETURNING id
	`

	var id string
	err = r.db.QueryRowContext(ctx, query, tenantID, key, encrypted, expiresAt).Scan(&id)
	if err != nil {
		return fmt.Errorf("set secret with expiration: %w", err)
	}

	return nil
}

// UpdateMetadata updates the metadata for a secret.
// Args:
// ctx - database operation context.
// key - secret key.
// tenantID - tenant identifier for isolation.
// metadata - metadata to update.
// Returns error if update operation fails.
func (r *SecretRepository) UpdateMetadata(ctx context.Context, key, tenantID string, metadata map[string]interface{}) error {
	metadataJSON, err := json.Marshal(metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE secrets
		SET metadata = $3, updated_at = NOW()
		WHERE key = $1 AND tenant_id = $2
	`

	result, err := r.db.ExecContext(ctx, query, key, tenantID, metadataJSON)
	if err != nil {
		return fmt.Errorf("update secret metadata: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// CleanupExpired removes secrets that have expired.
// Args:
// ctx - database operation context.
// Returns number of deleted secrets or error if operation fails.
func (r *SecretRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `
		DELETE FROM secrets
		WHERE expires_at IS NOT NULL AND expires_at < NOW()
	`

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired secrets: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("get rows affected: %w", err)
	}

	return rows, nil
}

// encrypt encrypts data using AES-GCM.
// Args:
// plaintext - data to encrypt.
// Returns encrypted data or error if encryption fails.
func (r *SecretRepository) encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// decrypt decrypts data using AES-GCM.
// Args:
// ciphertext - data to decrypt.
// Returns decrypted data or error if decryption fails.
func (r *SecretRepository) decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(r.encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}

	return plaintext, nil
}

// RotateKey re-encrypts all secrets with a new encryption key.
// TODO: implement key rotation functionality (expected by 2026-04-15)
// Implementation requirements:
//  1. Start transaction for atomic operation
//  2. Retrieve all secrets with current encryption key (SELECT ... FOR UPDATE)
//  3. For each secret:
//     a. Decrypt using old encryption key (AES-256-GCM)
//     b. Re-encrypt using new encryption key
//     c. Update database with new encrypted values
//     d. Increment key_version
//  4. Commit transaction if all succeed, rollback if any fail
//  5. Add audit logging for key rotation events
//  6. Test with various secret types and sizes
//
// Dependencies:
// - Need secure key exchange mechanism for distributing new key
// - Need rollback mechanism if rotation fails mid-way
// Args:
// ctx - database operation context.
// newKey - new encryption key (32 bytes for AES-256).
// Returns number of updated secrets or error if operation fails.
func (r *SecretRepository) RotateKey(ctx context.Context, newKey []byte) (int64, error) {
	return 0, errors.ErrNotImplemented
}

// Export exports secrets (for backup purposes).
// Args:
// ctx - database operation context.
// tenantID - tenant identifier for isolation.
// Returns exported secrets data or error if export fails.
func (r *SecretRepository) Export(ctx context.Context, tenantID string) ([]byte, error) {
	secrets, err := r.List(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("list secrets: %w", err)
	}

	data, err := json.Marshal(secrets)
	if err != nil {
		return nil, fmt.Errorf("marshal secrets: %w", err)
	}

	return data, nil
}

// Import imports secrets (for restore purposes).
// TODO: implement secret import functionality (expected by 2026-04-15)
// Current limitation: Export only contains metadata, not actual encrypted values
// Implementation requirements:
//  1. Parse exported secret metadata (JSON format)
//  2. Choose import strategy:
//     a) Prompt user for each secret value interactively
//     b) Provide API endpoint for secure secret value submission
//     c) Implement key sharing mechanism between source and destination systems
//  3. For each secret:
//     a. Validate secret value format and constraints
//     b. Encrypt using current encryption key (AES-256-GCM)
//     c. Insert into database with proper tenant isolation
//     d. Handle version compatibility if encryption algorithm differs
//  4. Add validation to prevent duplicate secret keys within same tenant
//  5. Add transaction support for atomic import operations
//
// Dependencies:
// - Need user interface or API for collecting actual secret values
// - Need secure channel for transmitting secret values
// Args:
// ctx - database operation context.
// tenantID - tenant identifier for isolation.
// data - exported secrets data (JSON format, contains metadata but not actual encrypted values).
// Returns number of imported secrets or error if import fails.
func (r *SecretRepository) Import(ctx context.Context, tenantID string, data []byte) (int64, error) {
	var secrets []*storage_models.Secret
	if err := json.Unmarshal(data, &secrets); err != nil {
		return 0, fmt.Errorf("unmarshal secrets: %w", err)
	}

	// Current implementation validates import format only
	// Actual secret values cannot be imported without user input or secure channel
	var count int64
	for range secrets {
		count++
	}

	return count, errors.ErrNotImplemented
}

// GetKeyVersion retrieves the current key version for a secret.
// Args:
// ctx - database operation context.
// key - secret key.
// tenantID - tenant identifier for isolation.
// Returns key version or error if not found.
func (r *SecretRepository) GetKeyVersion(ctx context.Context, key, tenantID string) (int, error) {
	query := `
		SELECT key_version
		FROM secrets
		WHERE key = $1 AND tenant_id = $2
	`

	var keyVersion int
	err := r.db.QueryRowContext(ctx, query, key, tenantID).Scan(&keyVersion)
	if err == sql.ErrNoRows {
		return 0, errors.ErrRecordNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("get key version: %w", err)
	}

	return keyVersion, nil
}
