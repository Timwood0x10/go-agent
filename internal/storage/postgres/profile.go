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

// ProfileRepository handles user profile persistence.
type ProfileRepository struct {
	pool *Pool
}

// NewProfileRepository creates a new ProfileRepository.
func NewProfileRepository(pool *Pool) *ProfileRepository {
	return &ProfileRepository{pool: pool}
}

// Create creates a new user profile.
func (r *ProfileRepository) Create(ctx context.Context, profile *models.UserProfile) error {
	styleJSON, err := json.Marshal(profile.Style)
	if err != nil {
		return fmt.Errorf("marshal style: %w", err)
	}

	budgetJSON, err := json.Marshal(profile.Budget)
	if err != nil {
		return fmt.Errorf("marshal budget: %w", err)
	}

	colorsJSON, err := json.Marshal(profile.Colors)
	if err != nil {
		return fmt.Errorf("marshal colors: %w", err)
	}

	occasionsJSON, err := json.Marshal(profile.Occasions)
	if err != nil {
		return fmt.Errorf("marshal occasions: %w", err)
	}

	preferencesJSON, err := json.Marshal(profile.Preferences)
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}

	query := `
		INSERT INTO user_profiles (user_id, name, gender, age, occupation, style, budget, colors, occasions, body_type, preferences, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
	`

	_, err = r.pool.Exec(ctx, query,
		profile.UserID,
		profile.Name,
		profile.Gender,
		profile.Age,
		profile.Occupation,
		styleJSON,
		budgetJSON,
		colorsJSON,
		occasionsJSON,
		profile.BodyType,
		preferencesJSON,
		profile.CreatedAt,
		profile.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert profile: %w", err)
	}

	return nil
}

// GetByID retrieves a user profile by ID.
func (r *ProfileRepository) GetByID(ctx context.Context, userID string) (*models.UserProfile, error) {
	query := `
		SELECT user_id, name, gender, age, occupation, style, budget, colors, occasions, body_type, preferences, created_at, updated_at
		FROM user_profiles WHERE user_id = $1
	`

	var profile models.UserProfile
	var styleJSON, budgetJSON, colorsJSON, occasionsJSON, preferencesJSON []byte

	err := r.pool.QueryRow(ctx, query, userID).Scan(
		&profile.UserID,
		&profile.Name,
		&profile.Gender,
		&profile.Age,
		&profile.Occupation,
		&styleJSON,
		&budgetJSON,
		&colorsJSON,
		&occasionsJSON,
		&profile.BodyType,
		&preferencesJSON,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, errors.ErrRecordNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query profile: %w", err)
	}

	if err := json.Unmarshal(styleJSON, &profile.Style); err != nil {
		return nil, fmt.Errorf("unmarshal style: %w", err)
	}
	if err := json.Unmarshal(budgetJSON, &profile.Budget); err != nil {
		return nil, fmt.Errorf("unmarshal budget: %w", err)
	}
	if err := json.Unmarshal(colorsJSON, &profile.Colors); err != nil {
		return nil, fmt.Errorf("unmarshal colors: %w", err)
	}
	if err := json.Unmarshal(occasionsJSON, &profile.Occasions); err != nil {
		return nil, fmt.Errorf("unmarshal occasions: %w", err)
	}
	if err := json.Unmarshal(preferencesJSON, &profile.Preferences); err != nil {
		return nil, fmt.Errorf("unmarshal preferences: %w", err)
	}

	return &profile, nil
}

// Update updates a user profile.
func (r *ProfileRepository) Update(ctx context.Context, profile *models.UserProfile) error {
	profile.UpdatedAt = time.Now()

	styleJSON, err := json.Marshal(profile.Style)
	if err != nil {
		return fmt.Errorf("marshal style: %w", err)
	}

	budgetJSON, err := json.Marshal(profile.Budget)
	if err != nil {
		return fmt.Errorf("marshal budget: %w", err)
	}

	colorsJSON, err := json.Marshal(profile.Colors)
	if err != nil {
		return fmt.Errorf("marshal colors: %w", err)
	}

	occasionsJSON, err := json.Marshal(profile.Occasions)
	if err != nil {
		return fmt.Errorf("marshal occasions: %w", err)
	}

	preferencesJSON, err := json.Marshal(profile.Preferences)
	if err != nil {
		return fmt.Errorf("marshal preferences: %w", err)
	}

	query := `
		UPDATE user_profiles
		SET name = $1, gender = $2, age = $3, occupation = $4, style = $5, budget = $6,
		    colors = $7, occasions = $8, body_type = $9, preferences = $10, updated_at = $11
		WHERE user_id = $12
	`

	result, err := r.pool.Exec(ctx, query,
		profile.Name,
		profile.Gender,
		profile.Age,
		profile.Occupation,
		styleJSON,
		budgetJSON,
		colorsJSON,
		occasionsJSON,
		profile.BodyType,
		preferencesJSON,
		profile.UpdatedAt,
		profile.UserID,
	)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
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

// Delete deletes a user profile.
func (r *ProfileRepository) Delete(ctx context.Context, userID string) error {
	query := `DELETE FROM user_profiles WHERE user_id = $1`

	result, err := r.pool.Exec(ctx, query, userID)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
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

// Exists checks if a profile exists.
func (r *ProfileRepository) Exists(ctx context.Context, userID string) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM user_profiles WHERE user_id = $1)`

	var exists bool
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check exists: %w", err)
	}

	return exists, nil
}
