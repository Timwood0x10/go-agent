package context

import (
	"context"
	"errors"
	"sync"
	"time"

	"goagent/internal/core/models"
)

// Memory errors.
var (
	ErrUserNotFound    = errors.New("user not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrTaskNotFound    = errors.New("task not found")
)

// SessionMemory stores conversation context for a session.
type SessionMemory struct {
	sessions map[string]*SessionData
	mu       sync.RWMutex
	maxSize  int
	ttl      time.Duration
}

// SessionData holds session information.
type SessionData struct {
	SessionID   string
	UserID     string
	Messages   []Message
	Context    map[string]interface{}
	AccessedAt time.Time
	CreatedAt  time.Time
}

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
	Time    time.Time `json:"time"`
}

// NewSessionMemory creates a new SessionMemory.
func NewSessionMemory(maxSize int, ttl time.Duration) *SessionMemory {
	return &SessionMemory{
		sessions: make(map[string]*SessionData),
		maxSize:  maxSize,
		ttl:      ttl,
	}
}

// Get retrieves session data.
func (m *SessionMemory) Get(ctx context.Context, sessionID string) (*SessionData, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, false
	}

	if time.Since(session.AccessedAt) > m.ttl {
		return nil, false
	}

	session.AccessedAt = time.Now()
	return session, true
}

// Set stores session data.
func (m *SessionMemory) Set(ctx context.Context, sessionID, userID string, messages []Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.sessions) >= m.maxSize {
		m.evictOldest()
	}

	session := &SessionData{
		SessionID:   sessionID,
		UserID:     userID,
		Messages:   messages,
		Context:    make(map[string]interface{}),
		AccessedAt: time.Now(),
		CreatedAt:  time.Now(),
	}

	m.sessions[sessionID] = session
	return nil
}

// AddMessage adds a message to the session.
func (m *SessionMemory) AddMessage(ctx context.Context, sessionID string, msg Message) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return ErrSessionNotFound
	}

	session.Messages = append(session.Messages, msg)
	session.AccessedAt = time.Now()

	return nil
}

// GetMessages returns session messages.
func (m *SessionMemory) GetMessages(ctx context.Context, sessionID string) ([]Message, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return nil, ErrSessionNotFound
	}

	return session.Messages, nil
}

// Delete removes a session.
func (m *SessionMemory) Delete(ctx context.Context, sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, sessionID)
	return nil
}

// Clear removes all sessions.
func (m *SessionMemory) Clear(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.sessions = make(map[string]*SessionData)
	return nil
}

// Size returns the number of sessions.
func (m *SessionMemory) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// evictOldest removes the oldest session.
func (m *SessionMemory) evictOldest() {
	var oldest *SessionData
	var oldestID string

	for id, session := range m.sessions {
		if oldest == nil || session.AccessedAt.Before(oldest.AccessedAt) {
			oldest = session
			oldestID = id
		}
	}

	if oldestID != "" {
		delete(m.sessions, oldestID)
	}
}

// SessionMemory errors.
var (
	ErrSessionNotFound = models.ErrSessionNotFound
)
