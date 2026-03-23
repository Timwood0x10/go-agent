// Package distillation provides memory distillation functionality for agent experience extraction.
package distillation

import (
	"fmt"
	"strings"
)

// Message represents a single message in a conversation.
type Message struct {
	Role    string
	Content string
}

// ExperienceExtractor extracts problem-solution pairs from conversations.
type ExperienceExtractor struct {
	questionDetector *QuestionDetector
	noiseFilter      *NoiseFilter
	enableCrossTurn  bool
}

// NewExperienceExtractor creates a new ExperienceExtractor instance.
func NewExperienceExtractor() *ExperienceExtractor {
	return &ExperienceExtractor{
		questionDetector: NewQuestionDetector(),
		noiseFilter:      NewNoiseFilter(),
		enableCrossTurn:  true,
	}
}

// NewExperienceExtractorWithConfig creates a new ExperienceExtractor instance with custom configuration.
func NewExperienceExtractorWithConfig(enableCrossTurn bool) *ExperienceExtractor {
	return &ExperienceExtractor{
		questionDetector: NewQuestionDetector(),
		noiseFilter:      NewNoiseFilter(),
		enableCrossTurn:  enableCrossTurn,
	}
}

// ExtractExperiences extracts problem-solution pairs from a conversation.
// It supports cross-turn extraction where the solution may appear after 2 messages.
//
// Args:
//
//	messages - the conversation messages.
//
// Returns:
//
//	[]Experience - extracted experiences.
func (e *ExperienceExtractor) ExtractExperiences(messages []Message) []Experience {
	var experiences []Experience

	for i := 0; i < len(messages)-1; i++ {
		current := messages[i]
		next := messages[i+1]

		// Only process user messages that are problems
		if current.Role != "user" {
			continue
		}

		// Check if this is a problem/question
		if !e.questionDetector.Detect(current.Content) {
			continue
		}

		// Filter out noise
		if e.noiseFilter.IsNoise(current.Content) {
			continue
		}

		// Direct extraction: user → assistant
		if next.Role == "assistant" {
			exp := e.extractDirectExperience(current, next)
			if exp != nil {
				experiences = append(experiences, *exp)
			}
		}

		// Cross-turn extraction: user → assistant (after 2 messages)
		if e.enableCrossTurn && i+3 < len(messages) {
			m2 := messages[i+2]
			m3 := messages[i+3]

			if m3.Role == "assistant" {
				exp := e.extractCrossTurnExperience(current, next, m2, m3)
				if exp != nil {
					experiences = append(experiences, *exp)
				}
			}
		}
	}

	return experiences
}

// extractDirectExperience extracts an experience from direct user-assistant pair.
//
// Args:
//
//	user - the user message.
//	assistant - the assistant message.
//
// Returns:
//
//	*Experience - extracted experience, nil if extraction fails.
func (e *ExperienceExtractor) extractDirectExperience(user, assistant Message) *Experience {
	if user.Content == "" || assistant.Content == "" {
		return nil
	}

	// Filter out noise from assistant response
	if e.noiseFilter.IsNoise(assistant.Content) {
		return nil
	}

	// Extract problem and solution
	problem := strings.TrimSpace(user.Content)
	solution := strings.TrimSpace(assistant.Content)

	// Validate extraction
	if problem == "" || solution == "" {
		return nil
	}

	// Extract the core solution (avoid verbose responses)
	solution = e.extractCoreSolution(solution)

	return &Experience{
		Problem:    problem,
		Solution:   solution,
		Confidence: e.calculateConfidence(problem, solution),
	}
}

// extractCrossTurnExperience extracts an experience from multi-turn conversation.
// This handles cases where the solution appears after additional clarification.
//
// Args:
//
//	user - the original user message.
//	a1 - first assistant response (clarification).
//	m2 - second user message (clarification details).
//	a2 - second assistant response (final solution).
//
// Returns:
//
//	*Experience - extracted experience, nil if extraction fails.
func (e *ExperienceExtractor) extractCrossTurnExperience(user, a1, m2, a2 Message) *Experience {
	if user.Content == "" || a2.Content == "" {
		return nil
	}

	// Filter out noise
	if e.noiseFilter.IsNoise(a2.Content) {
		return nil
	}

	// Combine problem context
	problem := user.Content
	if m2.Role == "user" && !e.noiseFilter.IsNoise(m2.Content) {
		problem += " " + m2.Content
	}

	// Extract solution
	solution := strings.TrimSpace(a2.Content)

	// Validate extraction
	if problem == "" || solution == "" {
		return nil
	}

	// Extract the core solution
	solution = e.extractCoreSolution(solution)

	return &Experience{
		Problem:    strings.TrimSpace(problem),
		Solution:   solution,
		Confidence: e.calculateConfidence(problem, solution),
	}
}

// extractCoreSolution extracts the core solution from a potentially verbose response.
// It removes common conversational fillers and focuses on actionable content.
//
// Args:
//
//	solution - the full solution text.
//
// Returns:
//
//	string - the core solution.
func (e *ExperienceExtractor) extractCoreSolution(solution string) string {
	// Remove common prefixes
	prefixes := []string{
		"here's how to fix it:", "to fix this:", "the solution is:",
		"you can fix it by:", "try this:", "here's the solution:",
	}

	lower := strings.ToLower(solution)
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			solution = strings.TrimSpace(solution[len(prefix):])
			break
		}
	}

	// Remove common suffixes
	suffixes := []string{
		"let me know if this helps", "hope this helps", "let me know if you need more help",
	}

	lower = strings.ToLower(solution)
	for _, suffix := range suffixes {
		if strings.HasSuffix(lower, suffix) {
			solution = strings.TrimSpace(solution[:len(solution)-len(suffix)])
			break
		}
	}

	// If solution is too long, truncate to first few sentences
	if len(solution) > 500 {
		// Find first period, newline, or end of first block
		truncateAt := 500
		for i, c := range solution {
			if i > 200 && (c == '.' || c == '\n') {
				truncateAt = i + 1
				break
			}
		}
		solution = solution[:truncateAt] + "..."
	}

	return solution
}

// calculateConfidence calculates the confidence score for an extracted experience.
// It considers factors like solution length, keyword presence, and structure.
//
// Args:
//
//	problem - the problem description.
//	solution - the solution description.
//
// Returns:
//
//	float64 - confidence score between 0 and 1.
func (e *ExperienceExtractor) calculateConfidence(problem, solution string) float64 {
	confidence := 0.5

	// Bonus for longer solutions (more detailed)
	if len(solution) > 100 {
		confidence += 0.1
	}
	if len(solution) > 200 {
		confidence += 0.1
	}

	// Bonus for specific action verbs
	actionVerbs := []string{"restart", "run", "execute", "install", "configure", "update", "delete", "create"}
	lower := strings.ToLower(solution)
	for _, verb := range actionVerbs {
		if strings.Contains(lower, verb) {
			confidence += 0.05
			break
		}
	}

	// Penalty for very short solutions
	if len(solution) < 20 {
		confidence -= 0.2
	}

	// Ensure confidence is within valid range
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// FormatExperience formats an experience as a compact string representation.
//
// Args:
//
//	exp - the experience to format.
//
// Returns:
//
//	string - formatted experience string.
func FormatExperience(exp *Experience) string {
	return fmt.Sprintf("%s → %s", exp.Problem, exp.Solution)
}
