// Package postgres provides utility functions for vector operations.
package postgres

import (
	"fmt"
	"strings"
)

// FormatVector converts []float64 to pgvector format string.
// This properly formats the embedding array to avoid double brackets.
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
		builder.WriteString(fmt.Sprintf("%f", v))
	}

	builder.WriteString("]")
	return builder.String()
}