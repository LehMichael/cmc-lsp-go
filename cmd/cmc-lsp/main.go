package main

import (
	"os"

	"github.com/lehmichael/cmc-lsp-go/internal/lsp"
)

func main() {
	os.Exit(lsp.NewLsp(os.Stdin, os.Stdout))
}
