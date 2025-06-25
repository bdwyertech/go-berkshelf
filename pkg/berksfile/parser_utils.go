package berksfile

import (
	"fmt"

	"github.com/bdwyer/go-berkshelf/pkg/template"
)

// ParseFile parses a Berksfile from a file path
func ParseFile(filepath string) (*Berksfile, error) {
	data, err := template.Render(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read Berksfile: %w", err)
	}

	return ParseString(data)
}
