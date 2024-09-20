package anthropic_provider

import (
	"fmt"
	"strings"
	"testing"
)

func TestSanitizeLines(t *testing.T) {

	// I should probably remove this test.
	// I created it just to investigate the strings.Split semantics in go's standard library.

	input := "Hello\nWorld\n"
	lines := strings.Split(input, "\n")

	for i, line := range lines {
		fmt.Printf("Line %d: %s\n", i, line)
	}

	if len(lines) != 3 {
		t.Errorf("Expected 3 lines, got %d", len(lines))
	}

}
