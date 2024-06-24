package anthropic_provider

import (
	"strings"
	"testing"
)

func TestSanitizeLines(t *testing.T) {

	input := "Hello\nWorld\n"
	lines := strings.Split(input, "\n")

	if len(lines) != 2 {
		t.Errorf("Expected 2 lines, got %d", len(lines))
	}

}
