package models

import "time"

// Gender represents user gender.
type Gender string

const (
	GenderMale   Gender = "male"
	GenderFemale Gender = "female"
	GenderOther Gender = "other"
)

// StyleTag represents fashion style tags.
type StyleTag string

const (
	StyleCasual     StyleTag = "casual"
	StyleFormal     StyleTag = "formal"
	StyleStreet    StyleTag = "street"
	Sporty         StyleTag = "sporty"
	StyleMinimalist StyleTag = "minimalist"
	StyleVintage   StyleTag = "vintage"
	StyleBohemian  StyleTag = "bohemian"
)

// Occasion represents usage scenarios.
type Occasion string

const (
	OccasionDaily      Occasion = "daily"
	OccasionWork       Occasion = "work"
	OccasionParty      Occasion = "party"
	OccasionDate       Occasion = "date"
	OccasionSports     Occasion = "sports"
	OccasionFormal     Occasion = "formal"
	OccasionVacation   Occasion = "vacation"
)

// SessionStatus represents session state.
type SessionStatus string

const (
	SessionStatusPending    SessionStatus = "pending"
	SessionStatusProcessing SessionStatus = "processing"
	SessionStatusCompleted SessionStatus = "completed"
	SessionStatusFailed    SessionStatus = "failed"
	SessionStatusExpired   SessionStatus = "expired"
)

// AgentType represents agent types.
type AgentType string

const (
	AgentTypeLeader    AgentType = "leader"
	AgentTypeTop       AgentType = "agent_top"
	AgentTypeBottom    AgentType = "agent_bottom"
	AgentTypeShoes    AgentType = "agent_shoes"
	AgentTypeHead      AgentType = "agent_head"
	AgentTypeAccessory AgentType = "agent_accessory"
)

// AgentStatus represents agent running state.
type AgentStatus string

const (
	AgentStatusStarting AgentStatus = "starting"
	AgentStatusReady    AgentStatus = "ready"
	AgentStatusBusy    AgentStatus = "busy"
	AgentStatusStopping AgentStatus = "stopping"
	AgentStatusOffline AgentStatus = "offline"
)

// PriceRange represents budget range.
type PriceRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

// NewPriceRange creates a new PriceRange.
func NewPriceRange(min, max float64) *PriceRange {
	return &PriceRange{Min: min, Max: max}
}

// IsValid checks if the price range is valid.
func (p *PriceRange) IsValid() bool {
	return p != nil && p.Min >= 0 && p.Max >= p.Min
}

// Contains checks if the price is within range.
func (p *PriceRange) Contains(price float64) bool {
	if !p.IsValid() {
		return true
	}
	return price >= p.Min && price <= p.Max
}

// Time fields for tracking.
var (
	DefaultSessionTTL = 24 * time.Hour
	DefaultTaskTTL    = 1 * time.Hour
)
