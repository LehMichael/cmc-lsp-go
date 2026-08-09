package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/lehmichael/cmc-lsp-go/internal/formatter"
	"github.com/lehmichael/cmc-lsp-go/internal/textencoding"
)

func main() {
	write := flag.Bool("w", false, "write the result to each file")
	check := flag.Bool("check", false, "report files that are not formatted")
	tabSize := flag.Int("tab-size", 4, "indentation width")
	useTabs := flag.Bool("tabs", false, "indent with tabs")
	commentSpaces := flag.Int("comment-spaces", 2, "spaces before a trailing comment")
	alignAssignments := flag.Bool("align-consecutive-assignments", true, "align operators in consecutive assignments")
	alignComments := flag.Bool("align-trailing-comments", true, "align comments on consecutive code lines")
	flag.Parse()

	options := formatter.DefaultOptions()
	options.TabSize = *tabSize
	options.InsertSpaces = !*useTabs
	options.CommentSpaces = *commentSpaces
	options.AlignConsecutiveAssignments = *alignAssignments
	options.AlignTrailingComments = *alignComments

	if flag.NArg() == 0 {
		if *write {
			fmt.Fprintln(os.Stderr, "cmc-fmt: -w requires at least one file")
			os.Exit(2)
		}
		input, err := io.ReadAll(os.Stdin)
		if err != nil {
			fail(err)
		}
		text, encoding := textencoding.Decode(input)
		output := formatter.Format(text, options)
		if *check {
			if output != text {
				os.Exit(1)
			}
			return
		}
		encoded, err := textencoding.Encode(output, encoding)
		if err != nil {
			fail(err)
		}
		_, _ = os.Stdout.Write(encoded)
		return
	}

	exitCode := 0
	for _, path := range flag.Args() {
		input, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "cmc-fmt: %s: %v\n", path, err)
			exitCode = 2
			continue
		}
		text, encoding := textencoding.Decode(input)
		output := formatter.Format(text, options)
		if *check {
			if output != text {
				fmt.Fprintln(os.Stderr, path)
				if exitCode == 0 {
					exitCode = 1
				}
			}
			continue
		}
		if *write {
			encoded, err := textencoding.Encode(output, encoding)
			if err == nil {
				err = os.WriteFile(path, encoded, 0o644)
			}
			if err != nil {
				fmt.Fprintf(os.Stderr, "cmc-fmt: %s: %v\n", path, err)
				exitCode = 2
			}
			continue
		}
		_, _ = io.WriteString(os.Stdout, output)
	}
	os.Exit(exitCode)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "cmc-fmt:", err)
	os.Exit(2)
}
