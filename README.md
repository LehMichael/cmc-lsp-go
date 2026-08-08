# cmc-lsp-go

Language tooling for Siemens Create MyConfig (CMC) scripts (`.upscr`) and script libraries (`.uplib`). The repository provides:

- `cmc-lsp`: an LSP server over standard input/output
- `cmc-fmt`: a standalone formatter
- `cmc-check`: a project/script validator
- A lexer and parser for the documented CMC 4.8 language, including library definitions

The language implementation is based on the examples and reference in the [SINUMERIK Create MyConfig operating manual](https://cache.industry.siemens.com/dl/files/192/109769192/att_991335/v1/840Dsl_Create_MyConfig_op_man_en-US.pdf), especially sections 7.8 and 7.9.

## Install

Go 1.26 or newer is required.

```sh
go install github.com/lehmichael/cmc-lsp-go/cmd/cmc-lsp@latest
go install github.com/lehmichael/cmc-lsp-go/cmd/cmc-fmt@latest
go install github.com/lehmichael/cmc-lsp-go/cmd/cmc-check@latest
```

To build the current checkout instead:

```sh
make build
```

The resulting binaries are written to `bin/`.

## Editor setup

Configure an LSP client with:

- Command: `cmc-lsp`
- Transport: standard input/output
- File extensions: `.upscr`, `.uplib`
- Language ID: `cmc`

The server supports full document synchronization, live syntax diagnostics, whole-document formatting, document/workspace symbols, completion, hover documentation, definitions, and references. It uses UTF-16 positions as required by the default LSP encoding.

### Project and single-file modes

When the workspace contains a `.upproj` file, the server reads its XML and resolves the Windows-style paths in `ScriptLibList` and `<script ref="...">` elements. Referenced `.upscr` and `.uplib` files share a project index, so library functions and project variables participate in completion, hover, go-to-definition, references, and workspace-symbol searches.

Files that are not referenced by a `.upproj` remain in single-file mode and do not inherit unrelated project symbols.

## Formatter

Format standard input:

```sh
cmc-fmt < script.upscr
```

Format files in place:

```sh
cmc-fmt -w script.upscr library.uplib
```

Check formatting without changing files:

```sh
cmc-fmt -check script.upscr
```

The default style uses four spaces per nesting level, spaces around operators, one space after commas, and two spaces before trailing comments. Use `-tab-size`, `-tabs`, or `-comment-spaces` to customize the CLI. LSP formatting honors the editor's `tabSize` and `insertSpaces` settings and accepts the optional `cmcCommentSpaces` formatting property.

## Project validation

Validate exactly the scripts and libraries referenced by a project:

```sh
cmc-check Project/my-project.upproj
```

Validate every script below a directory, optionally including formatting:

```sh
cmc-check -format path/to/project
```

`cmc-check` understands both UTF-8 (with or without BOM) and ISO-8859-1, matching the encodings supported by CMC.

## Supported language elements

- Assignments: `=`, `:=`, `?=`, `+=`, `-=`, `*=`, `/=`, `|=`, `&=`, and `~`
- Arithmetic, comparison, logical, bitwise, and string-concatenation operators
- `If`/`ElsIf`/`ElIf`/`Else`/`EndIf` and `While`/`EndWhile`
- NC, drive, and display sections, including dynamic replacement sections
- Qualified `NC[...]`, `PS[...]`, and `BD[...]` data access
- Indexed identifiers and nested `$(Up.variable)` replacements
- Strings, booleans, null, decimal, version, BICO, binary, and hexadecimal literals
- Procedure/function calls and `.uplib` `proc`/`func` definitions
- `#include` directives and semicolon comments

CMC has context-dependent host functions and system variables that vary by CMC release and package type. The server treats function names and data identifiers as open-ended rather than rejecting undocumented vendor or user-defined names.

## Development

```sh
make test
make vet
```

Regenerate enum helpers after changing an enum:

```sh
make generate
```
