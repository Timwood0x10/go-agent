// Package experience provides tests for experience distillation service.
package experience

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	storage_models "goagent/internal/storage/postgres/models"
)

// MockLLMClient is a mock implementation of LLM client.
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) Generate(ctx context.Context, prompt string) (string, error) {
	args := m.Called(ctx, prompt)
	return args.String(0), args.Error(1)
}

func (m *MockLLMClient) GenerateStream(ctx context.Context, prompt string) (<-chan string, error) {
	args := m.Called(ctx, prompt)
	return args.Get(0).(<-chan string), args.Error(1)
}

func (m *MockLLMClient) IsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

// MockEmbeddingClient is a mock implementation of embedding client.
type MockEmbeddingClient struct {
	mock.Mock
}

func (m *MockEmbeddingClient) Embed(ctx context.Context, text string) ([]float64, error) {
	args := m.Called(ctx, text)
	return args.Get(0).([]float64), args.Error(1)
}

func (m *MockEmbeddingClient) GetModel() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockEmbeddingClient) GetTimeout() time.Duration {
	args := m.Called()
	return args.Get(0).(time.Duration)
}

// MockExperienceRepository is a mock implementation of experience repository.
type MockExperienceRepository struct {
	mock.Mock
}

func (m *MockExperienceRepository) Create(ctx context.Context, exp *storage_models.Experience) error {
	args := m.Called(ctx, exp)
	return args.Error(0)
}

func (m *MockExperienceRepository) GetByID(ctx context.Context, id string) (*storage_models.Experience, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*storage_models.Experience), args.Error(1)
}

func (m *MockExperienceRepository) Update(ctx context.Context, exp *storage_models.Experience) error {
	args := m.Called(ctx, exp)
	return args.Error(0)
}

func (m *MockExperienceRepository) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockExperienceRepository) SearchByVector(ctx context.Context, embedding []float64, tenantID string, limit int) ([]*storage_models.Experience, error) {
	args := m.Called(ctx, embedding, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage_models.Experience), args.Error(1)
}

func (m *MockExperienceRepository) SearchByKeyword(ctx context.Context, query, tenantID string, limit int) ([]*storage_models.Experience, error) {
	args := m.Called(ctx, query, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage_models.Experience), args.Error(1)
}

func (m *MockExperienceRepository) IncrementUsageCount(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockExperienceRepository) ListByType(ctx context.Context, expType, tenantID string, limit int) ([]*storage_models.Experience, error) {
	args := m.Called(ctx, expType, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage_models.Experience), args.Error(1)
}

func (m *MockExperienceRepository) ListByAgent(ctx context.Context, agentID, tenantID string, limit int) ([]*storage_models.Experience, error) {
	args := m.Called(ctx, agentID, tenantID, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*storage_models.Experience), args.Error(1)
}

// TestShouldDistill tests the ShouldDistill method.
func TestShouldDistill(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name     string
		task     *TaskResult
		expected bool
	}{
		{
			name: "successful task with sufficient length",
			task: &TaskResult{
				Task:     "Optimize database query performance",
				Result:   "Added index on user_id column",
				Success:  true,
				TenantID: "test-tenant",
			},
			expected: true,
		},
		{
			name: "failed task",
			task: &TaskResult{
				Task:     "Optimize database query performance",
				Result:   "Failed to add index",
				Success:  false,
				TenantID: "test-tenant",
			},
			expected: false,
		},
		{
			name: "task too short",
			task: &TaskResult{
				Task:     "Short",
				Result:   "Added index",
				Success:  true,
				TenantID: "test-tenant",
			},
			expected: false,
		},
		{
			name: "result too short",
			task: &TaskResult{
				Task:     "Optimize database query performance",
				Result:   "Done",
				Success:  true,
				TenantID: "test-tenant",
			},
			expected: false,
		},
		{
			name:     "nil task",
			task:     nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewDistillationService(nil, nil, nil)
			result := service.ShouldDistill(ctx, tt.task)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestDistillWithNilTask tests distillation with nil task.
func TestDistillWithNilTask(t *testing.T) {
	ctx := context.Background()

	service := NewDistillationService(nil, nil, nil)
	result, err := service.Distill(ctx, nil)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "task result is nil")
}

// TestDistillWithMissingTenantID tests distillation with missing tenant ID.
func TestDistillWithMissingTenantID(t *testing.T) {
	ctx := context.Background()

	service := NewDistillationService(nil, nil, nil)
	task := &TaskResult{
		Task:     "Optimize database query",
		Result:   "Added index",
		Success:  true,
		TenantID: "", // Missing tenant ID
	}

	result, err := service.Distill(ctx, task)

	assert.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "tenant ID is required")
}
