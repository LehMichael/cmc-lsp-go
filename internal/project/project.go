// Package project loads CMC .upproj XML files and resolves their script graph.
package project

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/textencoding"
)

type Project struct {
	Path       string
	Root       string
	Version    string
	AppVersion string
	Scripts    []string
	Libraries  []string
	Missing    []string
}

func Load(path string) (*Project, error) {
	input, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	text, _ := textencoding.Decode(input)
	decoder := xml.NewDecoder(strings.NewReader(text))
	result := &Project{Path: filepath.Clean(path), Root: filepath.Dir(filepath.Clean(path))}
	seen := map[string]struct{}{}
	depth := 0
	scriptLibraryDepth := -1

	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		switch token := token.(type) {
		case xml.StartElement:
			depth++
			if token.Name.Local == "pack" {
				result.Version = attribute(token.Attr, "version")
				result.AppVersion = attribute(token.Attr, "appVersion")
			}
			if token.Name.Local == "list" && attribute(token.Attr, "name") == "ScriptLibList" {
				scriptLibraryDepth = depth
			}
			ref := attribute(token.Attr, "ref")
			extension := strings.ToLower(filepath.Ext(strings.ReplaceAll(ref, "\\", "/")))
			if ref == "" || !IsSourceExtension(extension) {
				continue
			}
			resolved := ResolveReference(result.Root, ref)
			if _, ok := seen[resolved]; ok {
				continue
			}
			seen[resolved] = struct{}{}
			if extension == ".uplib" || scriptLibraryDepth >= 0 {
				result.Libraries = append(result.Libraries, resolved)
			} else {
				result.Scripts = append(result.Scripts, resolved)
			}
			if _, err := os.Stat(resolved); err != nil {
				result.Missing = append(result.Missing, resolved)
			}
		case xml.EndElement:
			if scriptLibraryDepth == depth && token.Name.Local == "list" {
				scriptLibraryDepth = -1
			}
			depth--
		}
	}
	return result, nil
}

// IsSourceExtension reports whether an extension is interpreted as CMC source.
// .tea files use the same syntax as scripts and are commonly used for data.
func IsSourceExtension(extension string) bool {
	switch strings.ToLower(extension) {
	case ".upscr", ".uplib", ".tea":
		return true
	default:
		return false
	}
}

func ResolveReference(projectRoot, reference string) string {
	reference = filepath.FromSlash(strings.ReplaceAll(reference, "\\", "/"))
	return filepath.Clean(filepath.Join(projectRoot, reference))
}

func (project *Project) Files() []string {
	files := make([]string, 0, len(project.Libraries)+len(project.Scripts))
	files = append(files, project.Libraries...)
	files = append(files, project.Scripts...)
	return files
}

func (project *Project) Contains(path string) bool {
	path = filepath.Clean(path)
	return slices.Contains(project.Libraries, path) || slices.Contains(project.Scripts, path)
}

func Find(root string) ([]*Project, error) {
	var paths []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "tmp") {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".upproj") {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(paths)
	projects := make([]*Project, 0, len(paths))
	for _, path := range paths {
		loaded, err := Load(path)
		if err != nil {
			return nil, err
		}
		projects = append(projects, loaded)
	}
	return projects, nil
}

func attribute(attributes []xml.Attr, name string) string {
	for _, attribute := range attributes {
		if attribute.Name.Local == name {
			return attribute.Value
		}
	}
	return ""
}
