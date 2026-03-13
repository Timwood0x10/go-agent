package postgres

import (
	"context"

	"styleagent/internal/core/errors"
	"styleagent/internal/core/models"
)

// Repository provides a unified interface for all data access.
type Repository struct {
	Session    *SessionRepository
	Recommend  *RecommendRepository
	Profile    *ProfileRepository
	Vector     *VectorSearcher
	pool       *Pool
}

// NewRepository creates a new Repository with all sub-repositories.
func NewRepository(pool *Pool) *Repository {
	return &Repository{
		Session:    NewSessionRepository(pool),
		Recommend:  NewRecommendRepository(pool),
		Profile:    NewProfileRepository(pool),
		Vector:     NewVectorSearcher(pool),
		pool:       pool,
	}
}

// Pool returns the underlying connection pool.
func (r *Repository) Pool() *Pool {
	return r.pool
}

// Close closes the repository and its pool.
func (r *Repository) Close() error {
	return r.pool.Close()
}

// Transaction executes a function within a transaction.
func (r *Repository) Transaction(ctx context.Context, fn func(repo *Repository) error) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// Create transaction-level repositories
	txRepo := &Repository{
		Session:   NewSessionRepository(r.pool),
		Recommend: NewRecommendRepository(r.pool),
		Profile:  NewProfileRepository(r.pool),
		Vector:   NewVectorSearcher(r.pool),
		pool:     r.pool,
	}

	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return rbErr
		}
		return err
	}

	return tx.Commit()
}

// WithTransaction creates a new repository with a transaction.
func (r *Repository) WithTransaction(ctx context.Context) (*Repository, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}

	txRepo := &Repository{
		Session:   NewSessionRepository(r.pool),
		Recommend: NewRecommendRepository(r.pool),
		Profile:  NewProfileRepository(r.pool),
		Vector:   NewVectorSearcher(r.pool),
		pool:     r.pool,
	}

	// Execute in transaction
	_, err = tx.Exec("SELECT 1")
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	return txRepo, nil
}

// SaveSession saves a session and its results.
func (r *Repository) SaveSession(ctx context.Context, session *models.Session, result *models.RecommendResult) error {
	return r.Transaction(ctx, func(txRepo *Repository) error {
		if err := txRepo.Session.Create(ctx, session); err != nil {
			return err
		}

		if result != nil {
			if err := txRepo.Recommend.Create(ctx, result); err != nil {
				return err
			}
		}

		return nil
	})
}

// GetSessionWithResult retrieves a session with its recommendation result.
func (r *Repository) GetSessionWithResult(ctx context.Context, sessionID string) (*models.Session, *models.RecommendResult, error) {
	session, err := r.Session.GetByID(ctx, sessionID)
	if err != nil {
		return nil, nil, err
	}

	result, err := r.Recommend.GetBySessionID(ctx, sessionID)
	if err != nil && err != errors.ErrRecordNotFound {
		return nil, nil, err
	}

	return session, result, nil
}

// SaveProfile saves a user profile.
func (r *Repository) SaveProfile(ctx context.Context, profile *models.UserProfile) error {
	exists, err := r.Profile.Exists(ctx, profile.UserID)
	if err != nil {
		return err
	}

	if exists {
		return r.Profile.Update(ctx, profile)
	}

	return r.Profile.Create(ctx, profile)
}
