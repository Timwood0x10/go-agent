package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"goagent/internal/core/errors"
	"goagent/internal/core/models"
)

// SessionRepository handles session persistence.
type SessionRepository struct {
	pool *Pool
}

// NewSessionRepository creates a new SessionRepository.
func NewSessionRepository(pool *Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

// Create creates a new session.
func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	profileJSON, err := json.Marshal(session.UserProfile)
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}

	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		INSERT INTO sessions (session_id, user_id, input, status, user_profile, metadata, created_at, updated_at, expired_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err = r.pool.Exec(ctx, query,
		session.SessionID,
		session.UserID,
		session.Input,
		session.Status,
		profileJSON,
		metadataJSON,
		session.CreatedAt,
		session.UpdatedAt,
		session.ExpiredAt,
	)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}

	return nil
}

// GetByID retrieves a session by ID.
func (r *SessionRepository) GetByID(ctx context.Context, sessionID string) (*models.Session, error) {
	query := `
		SELECT session_id, user_id, input, status, user_profile, metadata, created_at, updated_at, expired_at
		FROM sessions WHERE session_id = $1
	`

	var session models.Session
	var profileJSON, metadataJSON []byte

	err := r.pool.QueryRow(ctx, query, sessionID).Scan(
		&session.SessionID,
		&session.UserID,
		&session.Input,
		&session.Status,
		&profileJSON,
		&metadataJSON,
		&session.CreatedAt,
		&session.UpdatedAt,
		&session.ExpiredAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.ErrRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query session: %w", err)
	}

	if err := json.Unmarshal(profileJSON, &session.UserProfile); err != nil {
		return nil, fmt.Errorf("unmarshal profile: %w", err)
	}
	if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
		return nil, fmt.Errorf("unmarshal metadata: %w", err)
	}

	return &session, nil
}

// Update updates a session.
func (r *SessionRepository) Update(ctx context.Context, session *models.Session) error {
	profileJSON, err := json.Marshal(session.UserProfile)
	if err != nil {
		return fmt.Errorf("marshal profile: %w", err)
	}

	metadataJSON, err := json.Marshal(session.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}

	query := `
		UPDATE sessions
		SET status = $1, user_profile = $2, metadata = $3, updated_at = $4
		WHERE session_id = $5
	`

	result, err := r.pool.Exec(ctx, query,
		session.Status,
		profileJSON,
		metadataJSON,
		time.Now(),
		session.SessionID,
	)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// Delete deletes a session.
func (r *SessionRepository) Delete(ctx context.Context, sessionID string) error {
	query := `DELETE FROM sessions WHERE session_id = $1`

	result, err := r.pool.Exec(ctx, query, sessionID)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return errors.ErrRecordNotFound
	}

	return nil
}

// ListByUserID lists sessions by user ID.
func (r *SessionRepository) ListByUserID(ctx context.Context, userID string, limit, offset int) ([]*models.Session, error) {
	query := `
		SELECT session_id, user_id, input, status, user_profile, metadata, created_at, updated_at, expired_at
		FROM sessions WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.pool.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.Session
	for rows.Next() {
		var session models.Session
		var profileJSON, metadataJSON []byte

		if err := rows.Scan(
			&session.SessionID,
			&session.UserID,
			&session.Input,
			&session.Status,
			&profileJSON,
			&metadataJSON,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.ExpiredAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		if err := json.Unmarshal(profileJSON, &session.UserProfile); err != nil {
			return nil, fmt.Errorf("unmarshal profile: %w", err)
		}
		if err := json.Unmarshal(metadataJSON, &session.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}

		sessions = append(sessions, &session)
	}

	return sessions, nil
}

// CleanupExpired removes expired sessions.
func (r *SessionRepository) CleanupExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM sessions WHERE expired_at < $1`

	result, err := r.pool.Exec(ctx, query, time.Now())
	if err != nil {
		return 0, fmt.Errorf("cleanup expired: %w", err)
	}

	return result.RowsAffected()
}
