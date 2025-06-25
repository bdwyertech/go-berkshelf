package berksfile

import (
	"fmt"
	"os"
)

// ParseFile parses a Berksfile from a file path
func ParseFile(filepath string) (*Berksfile, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Berksfile: %w", err)
	}

	return ParseString(string(data))
}
