package harness

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// NormalizeOptions controls which normalizations to apply.
type NormalizeOptions struct {
	StripANSI           bool
	StripTimestamps     bool
	NormalizePaths      bool
	NormalizeWhitespace bool
	SortJSONKeys        bool
	StripVersionHeaders bool
}

// DefaultNormalizeOptions returns options with all normalizations enabled.
func DefaultNormalizeOptions() NormalizeOptions {
	return NormalizeOptions{
		StripANSI:           true,
		StripTimestamps:     true,
		NormalizePaths:      true,
		NormalizeWhitespace: true,
		SortJSONKeys:        true,
		StripVersionHeaders: true,
	}
}

// ansiRegex matches ANSI escape sequences (CSI sequences, OSC sequences, and simple escapes).
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\][^\x1b]*(?:\x1b\\|\x07)|\x1b[^[\]()][0-9;]*[a-zA-Z]?`)

// absPathRegex matches common absolute filesystem paths on Unix systems.
// It matches paths starting with /Users/, /home/, /tmp/, /var/, /opt/, /etc/, /usr/, /private/
// and captures the full path (sequence of non-whitespace characters).
var absPathRegex = regexp.MustCompile(`(?:/Users|/home|/tmp|/var|/opt|/etc|/usr|/private)/\S+`)

// versionRegex matches version-specific build metadata strings like "go-berkshelf/X.Y.Z" or
// "go-berkshelf X.Y.Z" and similar version patterns.
var versionRegex = regexp.MustCompile(`go-berkshelf[/ ]v?\d+\.\d+(?:\.\d+)?(?:[-.]\S+)?`)

// timestampRegex matches common timestamp patterns (ISO 8601 and similar).
var timestampRegex = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}(?:\.\d+)?(?:Z|[+-]\d{2}:?\d{2})?`)

// NormalizeCLIOutput normalizes CLI stdout/stderr text by applying the
// requested transformations in a deterministic order.
func NormalizeCLIOutput(output string, opts NormalizeOptions) string {
	s := output

	if opts.StripANSI {
		s = ansiRegex.ReplaceAllString(s, "")
	}

	if opts.StripTimestamps {
		s = timestampRegex.ReplaceAllString(s, "<TIMESTAMP>")
	}

	if opts.StripVersionHeaders {
		s = versionRegex.ReplaceAllString(s, "<VERSION>")
	}

	if opts.NormalizePaths {
		s = absPathRegex.ReplaceAllString(s, "<PATH>")
	}

	if opts.NormalizeWhitespace {
		s = normalizeWhitespace(s)
	}

	return s
}

// normalizeWhitespace trims trailing whitespace from each line and
// collapses trailing newlines to a single newline (if any content exists).
func normalizeWhitespace(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t\r")
	}

	// Rejoin and strip trailing blank lines, then add exactly one trailing newline
	// if the original had any content.
	result := strings.Join(lines, "\n")
	result = strings.TrimRight(result, "\n")
	if result == "" {
		return ""
	}
	return result + "\n"
}

// NormalizeLockfileJSON normalizes a JSON lockfile string by:
// - Removing "generated_at" fields
// - Replacing absolute filesystem paths with <PATH> placeholder
// - Sorting JSON keys lexicographically
// Returns the normalized JSON string or an error if the input is not valid JSON.
func NormalizeLockfileJSON(content string) (string, error) {
	var raw any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return "", fmt.Errorf("invalid JSON in lockfile: %w", err)
	}

	cleaned := removeGeneratedAt(raw)
	cleaned = replacePathsInJSON(cleaned)

	// Marshal with sorted keys (encoding/json sorts map keys by default)
	// and use indent for readability.
	out, err := marshalSorted(cleaned)
	if err != nil {
		return "", fmt.Errorf("marshaling normalized lockfile: %w", err)
	}

	return string(out) + "\n", nil
}

// removeGeneratedAt recursively removes any "generated_at" keys from JSON objects.
func removeGeneratedAt(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			if k == "generated_at" {
				continue
			}
			result[k] = removeGeneratedAt(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = removeGeneratedAt(item)
		}
		return result
	default:
		return v
	}
}

// replacePathsInJSON recursively replaces absolute filesystem paths in string
// values within JSON structures.
func replacePathsInJSON(v any) any {
	switch val := v.(type) {
	case string:
		return absPathRegex.ReplaceAllString(val, "<PATH>")
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = replacePathsInJSON(v)
		}
		return result
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = replacePathsInJSON(item)
		}
		return result
	default:
		return v
	}
}

// marshalSorted produces JSON with keys sorted lexicographically at every level.
// Go's encoding/json already sorts map[string]any keys, so we just
// need to marshal with indentation.
func marshalSorted(v any) ([]byte, error) {
	// encoding/json sorts map keys by default for map[string]any.
	// We recursively ensure all maps are sorted by converting to ordered output.
	return json.MarshalIndent(sortKeys(v), "", "  ")
}

// sortKeys recursively ensures map keys are in a deterministic order.
// While encoding/json already sorts map[string]any keys during marshal,
// this function makes the sorting explicit for clarity.
func sortKeys(v any) any {
	switch val := v.(type) {
	case map[string]any:
		sorted := make(map[string]any, len(val))
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sorted[k] = sortKeys(val[k])
		}
		return sorted
	case []any:
		result := make([]any, len(val))
		for i, item := range val {
			result[i] = sortKeys(item)
		}
		return result
	default:
		return v
	}
}
