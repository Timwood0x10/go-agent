package models

import "time"

// Session represents a user conversation session.
type Session struct {
	SessionID   string           `json:"session_id"`
	UserID      string           `json:"user_id"`
	UserProfile *UserProfile     `json:"user_profile"`
	Input       string           `json:"input"`
	Status      SessionStatus    `json:"status"`
	Tasks       []*Task          `json:"tasks"`
	Results     []*TaskResult    `json:"results"`
	FinalOutput *RecommendResult `json:"final_output"`
	Metadata    map[string]any   `json:"metadata"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`
	ExpiredAt   time.Time        `json:"expired_at"`
}

// NewSession creates a new Session.
func NewSession(sessionID, userID, input string) *Session {
	now := time.Now()
	return &Session{
		SessionID: sessionID,
		UserID:    userID,
		Input:     input,
		Status:    SessionStatusPending,
		Tasks:     make([]*Task, 0),
		Results:   make([]*TaskResult, 0),
		Metadata:  make(map[string]any),
		CreatedAt: now,
		UpdatedAt: now,
		ExpiredAt: now.Add(DefaultSessionTTL),
	}
}

// IsExpired checks if the session has expired.
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiredAt)
}

// IsCompleted checks if the session is completed.
func (s *Session) IsCompleted() bool {
	return s.Status == SessionStatusCompleted || s.Status == SessionStatusFailed
}

// AddTask adds a task to the session.
func (s *Session) AddTask(task *Task) {
	s.Tasks = append(s.Tasks, task)
	s.UpdatedAt = time.Now()
}

// AddResult adds a task result to the session.
func (s *Session) AddResult(result *TaskResult) {
	s.Results = append(s.Results, result)
	s.UpdatedAt = time.Now()
}

// SetStatus updates the session status.
func (s *Session) SetStatus(status SessionStatus) {
	s.Status = status
	s.UpdatedAt = time.Now()
}

// Progress returns the completion progress (0.0 - 1.0).
func (s *Session) Progress() float64 {
	if len(s.Tasks) == 0 {
		return 0.0
	}
	return float64(len(s.Results)) / float64(len(s.Tasks))
}
