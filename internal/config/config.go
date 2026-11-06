// Package config reads, merges and writes OpenCode JSON/JSONC configuration.
package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Document is a mutable OpenCode configuration document.
type Document struct {
	Path string
	Data map[string]any
}

// Load reads an OpenCode config file. A missing file returns an empty document.
func Load(path string) (*Document, error) {
	doc := &Document{Path: path, Data: map[string]any{}}
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return doc, nil
		}
		return nil, err
	}
	if len(bytes.TrimSpace(raw)) == 0 {
		return doc, nil
	}
	clean := StripJSONC(raw)
	if err := json.Unmarshal(clean, &doc.Data); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	if doc.Data == nil {
		doc.Data = map[string]any{}
	}
	return doc, nil
}

// Save writes the document atomically with two-space indentation.
func (d *Document) Save() error {
	out, err := json.MarshalIndent(d.Data, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(d.Path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(d.Path), ".opencode-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(out); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, d.Path)
}

// StripJSONC removes // and /* */ comments plus trailing commas, preserving
// string contents. It is intentionally small: OpenCode only needs JSONC input.
func StripJSONC(raw []byte) []byte {
	out := make([]byte, 0, len(raw))
	inString := false
	escaped := false
	for i := 0; i < len(raw); i++ {
		c := raw[i]
		if inString {
			out = append(out, c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			out = append(out, c)
			continue
		}
		if c == '/' && i+1 < len(raw) {
			if raw[i+1] == '/' {
				for i < len(raw) && raw[i] != '\n' {
					i++
				}
				if i < len(raw) {
					out = append(out, '\n')
				}
				continue
			}
			if raw[i+1] == '*' {
				i += 2
				for i+1 < len(raw) && !(raw[i] == '*' && raw[i+1] == '/') {
					i++
				}
				i++
				continue
			}
		}
		out = append(out, c)
	}

	// Remove trailing commas before } or ].
	clean := make([]byte, 0, len(out))
	inString = false
	escaped = false
	for i := 0; i < len(out); i++ {
		c := out[i]
		if inString {
			clean = append(clean, c)
			if escaped {
				escaped = false
			} else if c == '\\' {
				escaped = true
			} else if c == '"' {
				inString = false
			}
			continue
		}
		if c == '"' {
			inString = true
			clean = append(clean, c)
			continue
		}
		if c == ',' {
			j := i + 1
			for j < len(out) && isSpace(out[j]) {
				j++
			}
			if j < len(out) && (out[j] == '}' || out[j] == ']') {
				continue
			}
		}
		clean = append(clean, c)
	}
	return clean
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// Get returns a nested value.
func Get(root map[string]any, keys ...string) (any, bool) {
	var current any = root
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		current, ok = m[key]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

// GetString returns a nested string value.
func GetString(root map[string]any, keys ...string) string {
	v, ok := Get(root, keys...)
	if !ok {
		return ""
	}
	s, _ := v.(string)
	return s
}

// Set writes a nested value, creating intermediate objects.
func Set(root map[string]any, value any, keys ...string) {
	if len(keys) == 0 {
		return
	}
	current := root
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	current[keys[len(keys)-1]] = value
}

// Delete removes a nested key.
func Delete(root map[string]any, keys ...string) {
	if len(keys) == 0 {
		return
	}
	current := root
	for _, key := range keys[:len(keys)-1] {
		next, ok := current[key].(map[string]any)
		if !ok {
			return
		}
		current = next
	}
	delete(current, keys[len(keys)-1])
}

// EnsureMap returns the nested object, creating it when absent.
func EnsureMap(root map[string]any, keys ...string) map[string]any {
	if len(keys) == 0 {
		return root
	}
	current := root
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			next = map[string]any{}
			current[key] = next
		}
		current = next
	}
	return current
}

// DeepMerge merges src into dst; nested objects merge, everything else replaces.
func DeepMerge(dst, src map[string]any) {
	for key, value := range src {
		if srcMap, ok := value.(map[string]any); ok {
			if dstMap, ok := dst[key].(map[string]any); ok {
				DeepMerge(dstMap, srcMap)
				continue
			}
		}
		dst[key] = value
	}
}

// FormatJSON renders a value for display.
func FormatJSON(value any) string {
	out, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return ""
	}
	return string(out)
}

// HasPlaceholder reports whether a string is empty or a templated default.
func HasPlaceholder(value string) bool {
	value = strings.TrimSpace(value)
	return value == "" || strings.Contains(value, "{env:") || strings.HasPrefix(value, "<")
}
