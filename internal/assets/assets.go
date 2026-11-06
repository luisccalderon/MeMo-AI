// Package assets embeds the agents, skills, commands and plugin shipped by memo-ai.
package assets

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

//go:embed files
var embedded embed.FS

// File is one embedded asset.
type File struct {
	// Path is slash-separated and relative to files/, for example
	// "agents/memo-orchestrator.md".
	Path string
	Kind string
}

// Read returns an embedded file.
func Read(path string) ([]byte, error) {
	return embedded.ReadFile("files/" + strings.TrimPrefix(path, "/"))
}

// Walk lists every embedded file.
func Walk() ([]File, error) {
	var files []File
	err := fs.WalkDir(embedded, "files", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(path, "files/")
		files = append(files, File{Path: rel, Kind: kindOf(rel)})
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files, err
}

func kindOf(path string) string {
	switch {
	case strings.HasPrefix(path, "agents/"):
		return "agent"
	case strings.HasPrefix(path, "skills/"):
		return "skill"
	case strings.HasPrefix(path, "commands/"):
		return "command"
	case strings.HasPrefix(path, "plugins/"):
		return "plugin"
	default:
		return "file"
	}
}
