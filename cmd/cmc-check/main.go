package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/document"
	"github.com/lehmichael/cmc-lsp-go/internal/formatter"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
	"github.com/lehmichael/cmc-lsp-go/internal/project"
	"github.com/lehmichael/cmc-lsp-go/internal/textencoding"
)

func main() {
	checkFormat := flag.Bool("format", false, "also require canonical formatting")
	flag.Parse()
	if flag.NArg() == 0 {
		fmt.Fprintln(os.Stderr, "usage: cmc-check [-format] <file, directory, or .upproj>...")
		os.Exit(2)
	}

	files, err := collectFiles(flag.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "cmc-check:", err)
		os.Exit(2)
	}
	failures := 0
	for _, path := range files {
		input, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "%s: %v\n", path, err)
			failures++
			continue
		}
		text, _ := textencoding.Decode(input)
		tokens, diagnostics := lexer.Tokenize(document.CMCText(path, text))
		_, diagnostics = parser.Parse(tokens, diagnostics)
		for _, item := range diagnostics {
			fmt.Printf("%s:%d:%d: %s\n", path, item.Range.Start.Line+1, item.Range.Start.Column+1, item.Kind.String())
		}
		failures += len(diagnostics)
		if *checkFormat && document.Format(path, text, formatter.DefaultOptions()) != text {
			fmt.Printf("%s: not formatted\n", path)
			failures++
		}
	}
	if failures > 0 {
		fmt.Fprintf(os.Stderr, "cmc-check: %d issue(s) in %d file(s)\n", failures, len(files))
		os.Exit(1)
	}
	fmt.Printf("cmc-check: %d file(s) OK\n", len(files))
}

func collectFiles(arguments []string) ([]string, error) {
	seen := map[string]struct{}{}
	var files []string
	add := func(path string) {
		path = filepath.Clean(path)
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		files = append(files, path)
	}
	for _, argument := range arguments {
		info, err := os.Stat(argument)
		if err != nil {
			return nil, err
		}
		if !info.IsDir() && strings.EqualFold(filepath.Ext(argument), ".upproj") {
			loaded, err := project.Load(argument)
			if err != nil {
				return nil, err
			}
			for _, path := range loaded.Files() {
				add(path)
			}
			continue
		}
		if !info.IsDir() {
			add(argument)
			continue
		}
		err = filepath.WalkDir(argument, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			if !entry.IsDir() && project.IsSourceExtension(extension) {
				add(path)
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	slices.Sort(files)
	return files, nil
}
