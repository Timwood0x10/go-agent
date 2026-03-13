package leader

import (
	"context"
	"testing"

	"goagent/internal/core/models"
)

func TestProfileParser_Parse(t *testing.T) {
	parser := NewProfileParser()

	tests := []struct {
		name     string
		input    string
		wantErr  bool
		checkFn  func(*models.UserProfile) bool
	}{
		{
			name:    "parse simple input",
			input:   "I want casual style",
			wantErr: false,
			checkFn: func(p *models.UserProfile) bool {
				return len(p.Style) > 0
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parser.Parse(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !tt.checkFn(got) {
				t.Errorf("Parse() check failed")
			}
		})
	}
}

func TestTaskPlanner_Plan(t *testing.T) {
	planner := NewTaskPlanner(3)

	profile := &models.UserProfile{
		Style:     []models.StyleTag{models.StyleCasual},
		Occasions: []models.Occasion{models.OccasionDaily},
		Budget:    models.NewPriceRange(100, 500),
	}

	tasks, err := planner.Plan(context.Background(), profile)
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if len(tasks) == 0 {
		t.Error("Plan() returned empty tasks")
	}

	if len(tasks) > 3 {
		t.Errorf("Plan() returned too many tasks, got %d, want <= 3", len(tasks))
	}
}

func TestTaskPlanner_PlanNilProfile(t *testing.T) {
	planner := NewTaskPlanner(3)

	_, err := planner.Plan(context.Background(), nil)
	if err == nil {
		t.Error("Plan() should return error for nil profile")
	}
}

func TestResultAggregator_Aggregate(t *testing.T) {
	aggregator := NewResultAggregator(true, 10)

	results := []*models.TaskResult{
		{
			TaskID:    "task_1",
			AgentType: models.AgentTypeTop,
			Success:   true,
			Items: []*models.RecommendItem{
				{
					ItemID:   "item_1",
					Category: "top",
					Name:     "T-Shirt",
					Price:    199.00,
				},
			},
		},
		{
			TaskID:    "task_2",
			AgentType: models.AgentTypeBottom,
			Success:   true,
			Items: []*models.RecommendItem{
				{
					ItemID:   "item_2",
					Category: "bottom",
					Name:     "Jeans",
					Price:    299.00,
				},
			},
		},
	}

	result, err := aggregator.Aggregate(context.Background(), results)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if len(result.Items) != 2 {
		t.Errorf("Aggregate() got %d items, want 2", len(result.Items))
	}
}

func TestResultAggregator_AggregateEmpty(t *testing.T) {
	aggregator := NewResultAggregator(false, 10)

	result, err := aggregator.Aggregate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if len(result.Items) != 0 {
		t.Errorf("Aggregate() got %d items, want 0", len(result.Items))
	}
}

func TestResultAggregator_Deduplication(t *testing.T) {
	aggregator := NewResultAggregator(true, 10)

	results := []*models.TaskResult{
		{
			TaskID:    "task_1",
			AgentType: models.AgentTypeTop,
			Success:   true,
			Items: []*models.RecommendItem{
				{
					ItemID:   "item_1",
					Category: "top",
					Name:     "T-Shirt",
					Price:    199.00,
				},
			},
		},
		{
			TaskID:    "task_2",
			AgentType: models.AgentTypeBottom,
			Success:   true,
			Items: []*models.RecommendItem{
				{
					ItemID:   "item_1", // Duplicate
					Category: "top",
					Name:     "T-Shirt",
					Price:    199.00,
				},
			},
		},
	}

	result, err := aggregator.Aggregate(context.Background(), results)
	if err != nil {
		t.Fatalf("Aggregate() error = %v", err)
	}

	if len(result.Items) != 1 {
		t.Errorf("Aggregate() got %d items after dedup, want 1", len(result.Items))
	}
}

func TestTaskDispatcher_Dispatch(t *testing.T) {
	registry := map[models.AgentType]string{
		models.AgentTypeTop:    "agent_top",
		models.AgentTypeBottom: "agent_bottom",
	}
	dispatcher := NewTaskDispatcher(registry, 2, 30)

	profile := &models.UserProfile{
		Style:     []models.StyleTag{models.StyleCasual},
		Occasions: []models.Occasion{models.OccasionDaily},
	}

	tasks := []*models.Task{
		models.NewTask("task_1", models.AgentTypeTop, profile),
		models.NewTask("task_2", models.AgentTypeBottom, profile),
	}

	results, err := dispatcher.Dispatch(context.Background(), tasks)
	if err != nil {
		t.Fatalf("Dispatch() error = %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Dispatch() got %d results, want 2", len(results))
	}
}

func TestTaskDispatcher_DispatchEmpty(t *testing.T) {
	registry := map[models.AgentType]string{}
	dispatcher := NewTaskDispatcher(registry, 2, 30)

	_, err := dispatcher.Dispatch(context.Background(), nil)
	if err == nil {
		t.Error("Dispatch() should return error for empty tasks")
	}
}
