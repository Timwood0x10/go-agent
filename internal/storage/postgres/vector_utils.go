// Package postgres provides utility functions for vector operations.
package postgres

import (
	"fmt"
	"math"
	"strings"
)

// FormatVector converts []float64 to pgvector format string.
// This properly formats the embedding array to avoid double brackets.
// Uses %.6f format to limit decimal places to 6 for compact representation.
func FormatVector(embedding []float64) string {
	if len(embedding) == 0 {
		return "[]"
	}

	var builder strings.Builder
	builder.WriteString("[")

	for i, v := range embedding {
		if i > 0 {
			builder.WriteString(",")
		}
		fmt.Fprintf(&builder, "%.6f", v)
	}

	builder.WriteString("]")
	return builder.String()
}

// NormalizeVector normalizes a vector to unit length.
// This is required for pgvector's cosine distance operator (<=>).
func NormalizeVector(embedding []float64) []float64 {
	if len(embedding) == 0 {
		return embedding
	}

	// Calculate the norm (length) of the vector
	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	// Avoid division by zero
	if norm == 0 {
		return embedding
	}

	// Normalize each component
	normalized := make([]float64, len(embedding))
	for i, v := range embedding {
		normalized[i] = v / norm
	}

	return normalized
}
