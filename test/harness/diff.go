package harness

import (
	"fmt"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

// DiffResult holds comparison results for a single fixture run.
type DiffResult struct {
	LockfileDiff  string
	StdoutDiff    string
	StderrDiff    string
	ExitCodeMatch bool
	RubyExitCode  int
	GoExitCode    int
}

// CompareLockfiles compares two parsed Ruby lockfiles and returns a unified diff.
// It formats both lockfiles to their canonical string representation (which sorts
// GRAPH entries deterministically) and then diffs the text. An empty string means
// the lockfiles are equivalent.
func CompareLockfiles(ruby, goLock *RubyLockfile) string {
	rubyText := FormatRubyLockfile(ruby)
	goText := FormatRubyLockfile(goLock)

	if rubyText == goText {
		return ""
	}

	return unifiedDiff("lockfile", rubyText, goText)
}

// CompareText compares two normalized text outputs and returns a unified diff.
// The label is used in the --- and +++ headers (e.g., "stdout" or "stderr").
// An empty string means the texts are identical.
func CompareText(label, ruby, goOutput string) string {
	if ruby == goOutput {
		return ""
	}

	return unifiedDiff(label, ruby, goOutput)
}

// unifiedDiff produces a unified diff between two strings with the given label
// in the header lines.
func unifiedDiff(label, a, b string) string {
	dmp := diffmatchpatch.New()
	aLines, bLines, lineArray := dmp.DiffLinesToChars(a, b)
	diffs := dmp.DiffMain(aLines, bLines, false)
	diffs = dmp.DiffCharsToLines(diffs, lineArray)
	diffs = dmp.DiffCleanupSemantic(diffs)

	return formatUnifiedDiff(label, diffs)
}

// formatUnifiedDiff converts a list of line-level diffs into unified diff format
// with --- / +++ headers and @@ hunk headers.
func formatUnifiedDiff(label string, diffs []diffmatchpatch.Diff) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "--- ruby/%s\n", label)
	fmt.Fprintf(&sb, "+++ go/%s\n", label)

	// Build hunks from the diffs
	type change struct {
		op   diffmatchpatch.Operation
		text string
	}

	var changes []change
	for _, d := range diffs {
		lines := splitLines(d.Text)
		for _, line := range lines {
			changes = append(changes, change{op: d.Type, text: line})
		}
	}

	// Generate hunks with context lines (up to 3 lines of context)
	const contextLines = 3

	// Find ranges of changed lines and group them into hunks
	type hunkRange struct {
		start, end int // indices into changes slice
	}

	var hunkRanges []hunkRange
	i := 0
	for i < len(changes) {
		if changes[i].op != diffmatchpatch.DiffEqual {
			start := i
			// Extend backwards for context
			for j := 0; j < contextLines && start > 0 && changes[start-1].op == diffmatchpatch.DiffEqual; j++ {
				start--
			}

			// Find end of this change group
			end := i
			for end < len(changes) && changes[end].op != diffmatchpatch.DiffEqual {
				end++
			}

			// Extend forward for context
			for j := 0; j < contextLines && end < len(changes) && changes[end].op == diffmatchpatch.DiffEqual; j++ {
				end++
			}

			// Merge with previous hunk if overlapping
			if len(hunkRanges) > 0 && start <= hunkRanges[len(hunkRanges)-1].end {
				hunkRanges[len(hunkRanges)-1].end = end
			} else {
				hunkRanges = append(hunkRanges, hunkRange{start: start, end: end})
			}

			i = end
		} else {
			i++
		}
	}

	// Output each hunk
	for _, hr := range hunkRanges {
		aStart := 0
		bStart := 0
		// Count lines before this hunk to determine line numbers
		for j := 0; j < hr.start; j++ {
			switch changes[j].op {
			case diffmatchpatch.DiffEqual:
				aStart++
				bStart++
			case diffmatchpatch.DiffDelete:
				aStart++
			case diffmatchpatch.DiffInsert:
				bStart++
			}
		}

		aCount := 0
		bCount := 0
		for j := hr.start; j < hr.end; j++ {
			switch changes[j].op {
			case diffmatchpatch.DiffEqual:
				aCount++
				bCount++
			case diffmatchpatch.DiffDelete:
				aCount++
			case diffmatchpatch.DiffInsert:
				bCount++
			}
		}

		fmt.Fprintf(&sb, "@@ -%d,%d +%d,%d @@\n",
			aStart+1, aCount, bStart+1, bCount)

		for j := hr.start; j < hr.end; j++ {
			line := changes[j].text
			switch changes[j].op {
			case diffmatchpatch.DiffEqual:
				sb.WriteString(" " + line + "\n")
			case diffmatchpatch.DiffDelete:
				sb.WriteString("-" + line + "\n")
			case diffmatchpatch.DiffInsert:
				sb.WriteString("+" + line + "\n")
			}
		}
	}

	return sb.String()
}

// splitLines splits a string into lines, removing the trailing empty element
// that results from a trailing newline.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	lines := strings.Split(s, "\n")
	// Remove trailing empty string from trailing newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
