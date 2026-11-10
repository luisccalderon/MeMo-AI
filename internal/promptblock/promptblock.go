// Package promptblock manages the MeMo-AI section of a shared prompt file.
//
// oh-my-pi appends `APPEND_SYSTEM.md` to its system prompt. That file belongs
// to the user, so MeMo-AI only owns the text between two marker comments and
// leaves everything else byte-for-byte intact.
package promptblock

import "strings"

const (
	beginPrefix = "<!-- memo-ai:begin"
	// Begin opens the managed block.
	Begin = beginPrefix + " (managed by memo-ai; edits inside this block are overwritten) -->"
	// End closes the managed block.
	End = "<!-- memo-ai:end -->"
)

// Has reports whether content contains a complete managed block.
func Has(content string) bool {
	_, _, ok := locate(content)
	return ok
}

// Upsert replaces the managed block with body, or appends one when absent.
func Upsert(content, body string) string {
	block := Begin + "\n" + strings.TrimSpace(body) + "\n" + End
	if start, end, ok := locate(content); ok {
		return content[:start] + block + content[end:]
	}
	if strings.TrimSpace(content) == "" {
		return block + "\n"
	}
	return strings.TrimRight(content, "\r\n") + "\n\n" + block + "\n"
}

// Strip removes the managed block. The second result reports whether a block
// was found.
func Strip(content string) (string, bool) {
	start, end, ok := locate(content)
	if !ok {
		return content, false
	}
	before := strings.TrimRight(content[:start], "\r\n")
	after := strings.TrimLeft(content[end:], "\r\n")
	switch {
	case before == "" && after == "":
		return "", true
	case before == "":
		return after, true
	case after == "":
		return before + "\n", true
	default:
		return before + "\n\n" + after, true
	}
}

// locate returns the byte range of the managed block, markers included.
func locate(content string) (int, int, bool) {
	start := strings.Index(content, beginPrefix)
	if start < 0 {
		return 0, 0, false
	}
	rel := strings.Index(content[start:], End)
	if rel < 0 {
		return 0, 0, false
	}
	return start, start + rel + len(End), true
}
