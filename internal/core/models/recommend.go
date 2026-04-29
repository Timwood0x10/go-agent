package models

import "time"

// RecommendResult is a generic Agent output structure. In general-purpose scenarios,
// use the Content and Metadata fields as the primary data carriers. Fields such as
// TotalPrice, MatchScore, Occasion, and Season are optional and only meaningful in
// specific domains (e.g. e-commerce recommendations).
type RecommendResult struct {
	SessionID  string           `json:"session_id"`
	UserID     string           `json:"user_id"`
	Items      []*RecommendItem `json:"items"`
	Reason     string           `json:"reason"`
	TotalPrice float64          `json:"total_price"`
	MatchScore float64          `json:"match_score"`
	Occasion   Occasion         `json:"occasion"`
	Season     string           `json:"season"`
	Feedback   *UserFeedback    `json:"feedback"`
	Metadata   map[string]any   `json:"metadata"`
	CreatedAt  time.Time        `json:"created_at"`
}

// RecommendItem is a generic data item produced by an Agent. The Content field
// (a JSON string) and Metadata field (a key-value map) serve as universal data
// carriers for any scenario. Domain-specific fields such as Price, Brand, ImageURL,
// Category, Colors, and MatchReason are optional and should only be populated when
// the downstream consumer expects them.
type RecommendItem struct {
	ItemID           string         `json:"item_id"`
	Category         string         `json:"category"`
	Name             string         `json:"name"`
	Brand            string         `json:"brand"`
	Price            float64        `json:"price"`
	URL              string         `json:"url"`
	ImageURL         string         `json:"image_url"`
	AgentPreferences []StyleTag     `json:"style"`
	Colors           []string       `json:"colors"`
	Description      string         `json:"description"`
	MatchReason      string         `json:"match_reason"`
	Content          string         `json:"content"` // add Content field saved by agI
	Metadata         map[string]any `json:"metadata"`
}

// NewRecommendResult creates a new RecommendResult.
func NewRecommendResult(sessionID, userID string) *RecommendResult {
	return &RecommendResult{
		SessionID: sessionID,
		UserID:    userID,
		Items:     make([]*RecommendItem, 0),
		Metadata:  make(map[string]any),
		CreatedAt: time.Now(),
	}
}

// AddItem appends an item to the result. Only nil items are rejected;
// all other items are accepted regardless of whether domain-specific
// fields (e.g. Price) are populated.
func (r *RecommendResult) AddItem(item *RecommendItem) {
	if item == nil {
		return
	}
	r.Items = append(r.Items, item)
	r.TotalPrice += item.Price
}

// CalculateScore returns a normalised score in [0, 1] based on item count.
// Uses 20 as the reference maximum (matching the default maxItems in the aggregator).
// Returns 0.0 when there are no items.
func (r *RecommendResult) CalculateScore() float64 {
	if len(r.Items) == 0 {
		return 0.0
	}
	const refMax = 20.0
	score := float64(len(r.Items)) / refMax
	if score > 1.0 {
		score = 1.0
	}
	r.MatchScore = score
	return r.MatchScore
}
