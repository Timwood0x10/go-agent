package models

import (
	"testing"
)

func TestGender(t *testing.T) {
	tests := []struct {
		name     string
		gender   Gender
		expected string
	}{
		{"male", GenderMale, "male"},
		{"female", GenderFemale, "female"},
		{"other", GenderOther, "other"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.gender) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.gender)
			}
		})
	}
}

func TestStyleTag(t *testing.T) {
	tests := []struct {
		name     string
		tag      StyleTag
		expected string
	}{
		{"casual", StyleCasual, "casual"},
		{"formal", StyleFormal, "formal"},
		{"street", StyleStreet, "street"},
		{"sporty", Sporty, "sporty"},
		{"minimalist", StyleMinimalist, "minimalist"},
		{"vintage", StyleVintage, "vintage"},
		{"bohemian", StyleBohemian, "bohemian"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.tag) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.tag)
			}
		})
	}
}

func TestOccasion(t *testing.T) {
	tests := []struct {
		name     string
		occasion Occasion
		expected string
	}{
		{"daily", OccasionDaily, "daily"},
		{"work", OccasionWork, "work"},
		{"party", OccasionParty, "party"},
		{"date", OccasionDate, "date"},
		{"sports", OccasionSports, "sports"},
		{"formal", OccasionFormal, "formal"},
		{"vacation", OccasionVacation, "vacation"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.occasion) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.occasion)
			}
		})
	}
}

func TestAgentType(t *testing.T) {
	tests := []struct {
		name      string
		agentType AgentType
		expected  string
	}{
		{"leader", AgentTypeLeader, "leader"},
		{"top", AgentTypeTop, "agent_top"},
		{"bottom", AgentTypeBottom, "agent_bottom"},
		{"shoes", AgentTypeShoes, "agent_shoes"},
		{"head", AgentTypeHead, "agent_head"},
		{"accessory", AgentTypeAccessory, "agent_accessory"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.agentType) != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, tt.agentType)
			}
		})
	}
}

func TestUserProfile(t *testing.T) {
	t.Run("create user profile", func(t *testing.T) {
		profile := NewUserProfile("user123", "John Doe")

		if profile.UserID != "user123" {
			t.Errorf("expected user123, got %s", profile.UserID)
		}
		if profile.Name != "John Doe" {
			t.Errorf("expected John Doe, got %s", profile.Name)
		}
		if profile.Preferences == nil {
			t.Errorf("expected preferences to be initialized")
		}
	})

	t.Run("validate valid profile", func(t *testing.T) {
		profile := NewUserProfile("user123", "John Doe")
		profile.Age = 25

		err := profile.Validate()
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	})

	t.Run("validate invalid user id", func(t *testing.T) {
		profile := NewUserProfile("", "John Doe")

		err := profile.Validate()
		if err == nil {
			t.Errorf("expected error for empty user id")
		}
	})

	t.Run("validate invalid age", func(t *testing.T) {
		profile := NewUserProfile("user123", "John Doe")
		profile.Age = 200

		err := profile.Validate()
		if err == nil {
			t.Errorf("expected error for invalid age")
		}
	})

	t.Run("has style", func(t *testing.T) {
		profile := &UserProfile{
			Style: []StyleTag{StyleCasual, StyleMinimalist},
		}

		if !profile.HasStyle(StyleCasual) {
			t.Errorf("expected to have casual style")
		}
		if profile.HasStyle(StyleFormal) {
			t.Errorf("expected not to have formal style")
		}
	})
}

func TestUserFeedback(t *testing.T) {
	t.Run("create user feedback", func(t *testing.T) {
		feedback := &UserFeedback{
			Liked:   true,
			Rating:  5,
			Comment: "Great style!",
		}

		if !feedback.Liked {
			t.Errorf("expected liked to be true")
		}
		if feedback.Rating != 5 {
			t.Errorf("expected rating 5, got %d", feedback.Rating)
		}
	})

	t.Run("is valid rating", func(t *testing.T) {
		feedback := &UserFeedback{Rating: 3}
		if !feedback.IsValid() {
			t.Errorf("expected valid rating")
		}
	})

	t.Run("set rating", func(t *testing.T) {
		feedback := &UserFeedback{}
		if !feedback.SetRating(4) {
			t.Errorf("expected to set rating 4")
		}
		if feedback.SetRating(6) {
			t.Errorf("expected invalid rating 6 to fail")
		}
	})
}

func TestTask(t *testing.T) {
	t.Run("create task", func(t *testing.T) {
		task := NewTask("task123", AgentTypeLeader, nil)

		if task.TaskID != "task123" {
			t.Errorf("expected task123, got %s", task.TaskID)
		}
		if task.AgentType != AgentTypeLeader {
			t.Errorf("expected leader agent, got %s", task.AgentType)
		}
	})
}

func TestTaskContext(t *testing.T) {
	t.Run("create task context", func(t *testing.T) {
		ctx := &TaskContext{
			Dependencies: []string{"dep1", "dep2"},
			DepResults:   make(map[string]*TaskResult),
			Coordination: make(map[string]any),
		}

		if len(ctx.Dependencies) != 2 {
			t.Errorf("expected 2 dependencies, got %d", len(ctx.Dependencies))
		}
	})
}

func TestSession(t *testing.T) {
	t.Run("create session", func(t *testing.T) {
		session := NewSession("session123", "user123", "I want a casual outfit")

		if session.SessionID != "session123" {
			t.Errorf("expected session123, got %s", session.SessionID)
		}
		if session.UserID != "user123" {
			t.Errorf("expected user123, got %s", session.UserID)
		}
		if session.Input != "I want a casual outfit" {
			t.Errorf("expected input, got %s", session.Input)
		}
	})

	t.Run("is expired", func(t *testing.T) {
		session := NewSession("session123", "user123", "test")
		if session.IsExpired() {
			t.Errorf("expected non-expired session")
		}
	})
}

func TestRecommendResult(t *testing.T) {
	t.Run("create recommend result", func(t *testing.T) {
		result := NewRecommendResult("session123", "user123")

		if result.SessionID != "session123" {
			t.Errorf("expected session123, got %s", result.SessionID)
		}
		if result.UserID != "user123" {
			t.Errorf("expected user123, got %s", result.UserID)
		}
		if len(result.Items) != 0 {
			t.Errorf("expected empty items, got %d", len(result.Items))
		}
	})

	t.Run("add item to result", func(t *testing.T) {
		result := NewRecommendResult("session123", "user123")
		item := &RecommendItem{
			ItemID:      "item1",
			Category:    "top",
			Name:        "T-Shirt",
			Price:       29.99,
			Description: "Cotton t-shirt",
		}

		result.AddItem(item)

		if len(result.Items) != 1 {
			t.Errorf("expected 1 item, got %d", len(result.Items))
		}
		if result.Items[0].ItemID != "item1" {
			t.Errorf("expected item1, got %s", result.Items[0].ItemID)
		}
	})

	t.Run("calculate score", func(t *testing.T) {
		result := NewRecommendResult("session123", "user123")
		result.AddItem(&RecommendItem{ItemID: "item1", Price: 100})

		score := result.CalculateScore()
		if score < 0 || score > 1 {
			t.Errorf("expected score between 0 and 1, got %f", score)
		}
	})
}

func TestPriceRange(t *testing.T) {
	t.Run("create price range", func(t *testing.T) {
		pr := &PriceRange{Min: 100, Max: 500}

		if pr.Min != 100 {
			t.Errorf("expected 100, got %f", pr.Min)
		}
		if pr.Max != 500 {
			t.Errorf("expected 500, got %f", pr.Max)
		}
	})

	t.Run("is valid", func(t *testing.T) {
		pr := &PriceRange{Min: 100, Max: 500}
		if !pr.IsValid() {
			t.Errorf("expected valid price range")
		}
	})

	t.Run("is valid with invalid range", func(t *testing.T) {
		pr := &PriceRange{Min: 500, Max: 100}
		if pr.IsValid() {
			t.Errorf("expected invalid price range")
		}
	})
}
