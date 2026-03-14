package leader

import (
	"context"
	"testing"

	"goagent/internal/core/models"
	"goagent/internal/llm/output"
	"goagent/internal/protocol/ahp"
)

func TestProfileParser_Parse(t *testing.T) {
	parser := NewProfileParser(
		nil,                        // llmAdapter
		output.NewTemplateEngine(), // template
		"{{.input}}",               // promptTpl
		output.NewValidator(),      // validator
		3,                          // maxRetries
	)

	tests := []struct {
		name    string
		input   string
		wantErr bool
		checkFn func(*models.UserProfile) bool
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

func TestLeaderAgent_New(t *testing.T) {
	parser := NewProfileParser(
		nil,
		output.NewTemplateEngine(),
		"{{.input}}",
		output.NewValidator(),
		3,
	)
	planner := NewTaskPlanner(3)
	registry := map[models.AgentType]string{
		models.AgentTypeTop: "agent_top",
	}
	dispatcher := NewTaskDispatcher(registry, 2, 30)
	aggregator := NewResultAggregator(true, 10)

	agent := New("leader1", parser, planner, dispatcher, aggregator, nil, nil, nil)

	if agent.ID() != "leader1" {
		t.Errorf("expected leader1, got %s", agent.ID())
	}
	if agent.Type() != models.AgentTypeLeader {
		t.Errorf("expected AgentTypeLeader")
	}
}

func TestLeaderAgent_DefaultConfig(t *testing.T) {
	cfg := DefaultLeaderAgentConfig()

	if cfg.Type != models.AgentTypeLeader {
		t.Errorf("expected AgentTypeLeader")
	}
	if cfg.MaxParallelTasks != 10 {
		t.Errorf("expected MaxParallelTasks 10")
	}
}

func TestLeaderAgent_StartStop(t *testing.T) {
	parser := NewProfileParser(
		nil,
		output.NewTemplateEngine(),
		"{{.input}}",
		output.NewValidator(),
		3,
	)
	planner := NewTaskPlanner(3)
	registry := map[models.AgentType]string{}
	dispatcher := NewTaskDispatcher(registry, 2, 30)
	aggregator := NewResultAggregator(true, 10)

	agent := New("leader1", parser, planner, dispatcher, aggregator, nil, nil, nil)

	// Start
	err := agent.Start(context.Background())
	if err != nil {
		t.Errorf("Start() error = %v", err)
	}

	if agent.Status() != models.AgentStatusReady {
		t.Errorf("expected status Ready after Start")
	}

	// Start again should fail
	err = agent.Start(context.Background())
	if err == nil {
		t.Error("Start() should return error when already started")
	}

	// Stop
	err = agent.Stop(context.Background())
	if err != nil {
		t.Errorf("Stop() error = %v", err)
	}

	if agent.Status() != models.AgentStatusOffline {
		t.Errorf("expected status Offline after Stop")
	}

	// Stop again should fail
	err = agent.Stop(context.Background())
	if err == nil {
		t.Error("Stop() should return error when not running")
	}
}

func TestLeaderAgent_Process(t *testing.T) {
	parser := NewProfileParser(
		nil,
		output.NewTemplateEngine(),
		"{{.input}}",
		output.NewValidator(),
		3,
	)
	planner := NewTaskPlanner(3)
	registry := map[models.AgentType]string{}
	dispatcher := NewTaskDispatcher(registry, 2, 30)
	aggregator := NewResultAggregator(true, 10)

	agent := New("leader1", parser, planner, dispatcher, aggregator, nil, nil, nil)

	// Process without starting should auto-start
	result, err := agent.Process(context.Background(), "I want casual style")
	if err != nil {
		t.Errorf("Process() error = %v", err)
	}
	_ = result
}

func TestLeaderAgent_ProcessNotReady(t *testing.T) {
	parser := NewProfileParser(
		nil,
		output.NewTemplateEngine(),
		"{{.input}}",
		output.NewValidator(),
		3,
	)
	planner := NewTaskPlanner(3)
	registry := map[models.AgentType]string{}
	dispatcher := NewTaskDispatcher(registry, 2, 30)
	aggregator := NewResultAggregator(true, 10)

	agent := New("leader1", parser, planner, dispatcher, aggregator, nil, nil, nil)

	// Start then set to busy
	agent.Start(context.Background())
	// Note: can't easily set to busy without proper implementation

	// Process after stop should auto-start
	agent.Stop(context.Background())
	result, err := agent.Process(context.Background(), "test")
	if err != nil {
		t.Errorf("Process() error = %v", err)
	}
	_ = result
}

func TestLeaderAgent_SendReceiveMessage(t *testing.T) {
	parser := NewProfileParser(
		nil,
		output.NewTemplateEngine(),
		"{{.input}}",
		output.NewValidator(),
		3,
	)
	planner := NewTaskPlanner(3)
	registry := map[models.AgentType]string{}
	dispatcher := NewTaskDispatcher(registry, 2, 30)
	aggregator := NewResultAggregator(true, 10)
	queue := ahp.NewMessageQueue("leader1", &ahp.QueueOptions{MaxSize: 10})

	// Create using the concrete type
	leader := &leaderAgent{
		id:           "leader1",
		agentType:    models.AgentTypeLeader,
		status:       models.AgentStatusReady,
		config:       DefaultLeaderAgentConfig(),
		parser:       parser,
		planner:      planner,
		dispatcher:   dispatcher,
		aggregator:   aggregator,
		messageQueue: queue,
	}

	// Test SendMessage
	msg := ahp.NewMessage(ahp.AHPMethodTask, "leader1", "sub1", "task1", "session1")
	err := leader.SendMessage(context.Background(), msg)
	if err != nil {
		t.Errorf("SendMessage() error = %v", err)
	}

	// Test ReceiveMessage
	_, err = leader.ReceiveMessage(context.Background())
	if err != nil {
		t.Errorf("ReceiveMessage() error = %v", err)
	}
}

func TestLeaderAgent_Heartbeat(t *testing.T) {
	parser := NewProfileParser(
		nil,
		output.NewTemplateEngine(),
		"{{.input}}",
		output.NewValidator(),
		3,
	)
	planner := NewTaskPlanner(3)
	registry := map[models.AgentType]string{}
	dispatcher := NewTaskDispatcher(registry, 2, 30)
	aggregator := NewResultAggregator(true, 10)
	hbMon := ahp.NewHeartbeatMonitor(ahp.DefaultHeartbeatConfig())

	leader := &leaderAgent{
		id:           "leader1",
		agentType:    models.AgentTypeLeader,
		status:       models.AgentStatusReady,
		config:       DefaultLeaderAgentConfig(),
		parser:       parser,
		planner:      planner,
		dispatcher:   dispatcher,
		aggregator:   aggregator,
		heartbeatMon: hbMon,
	}

	err := leader.Heartbeat(context.Background())
	if err != nil {
		t.Errorf("Heartbeat() error = %v", err)
	}

	if !leader.IsAlive() {
		t.Error("IsAlive() should return true after heartbeat")
	}
}
