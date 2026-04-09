package models

import "time"

// RecommendResult represents the final recommendation output.
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

// RecommendItem represents a single recommended item.
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

// AddItem adds an item to the recommendation.
func (r *RecommendResult) AddItem(item *RecommendItem) {
	if item == nil || item.Price < 0 {
		// Skip invalid items with negative prices
		return
	}
	r.Items = append(r.Items, item)
	r.TotalPrice += item.Price
}

// CalculateScore calculates the overall match score.
func (r *RecommendResult) CalculateScore() float64 {
	if len(r.Items) == 0 {
		return 0.0
	}
	// Simplified scoring - can be enhanced with more complex logic
	baseScore := 0.8
	pricePenalty := r.TotalPrice / 1000.0 * 0.1
	r.MatchScore = baseScore - pricePenalty
	if r.MatchScore < 0 {
		r.MatchScore = 0
	}
	return r.MatchScore
}
