# cmc-lsp-go

Language tooling for Siemens Create MyConfig (CMC) scripts (`.upscr`), script libraries (`.uplib`), action scripts (`.upact`), and data exports (`.tea`). The repository provides:

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
- File extensions: `.upscr`, `.uplib`, `.upact`, `.tea`
- Language ID: `cmc`

The server supports full document synchronization, live syntax diagnostics, semantic syntax highlighting, whole-document formatting, document/workspace symbols, completion, hover documentation, definitions, clickable and navigable `#include` paths, dynamic-aware references, signature help, and safe callable rename. It uses UTF-16 positions as required by the default LSP encoding.

### Zed

The [`zed-cmc`](https://github.com/LehMichael/zed-cmc) extension provides a
native CMC file type and uses
[`tree-sitter-cmc`](https://github.com/LehMichael/tree-sitter-cmc) for syntax
highlighting. Install it as a dev extension while it is not yet in Zed's
extension registry, then use the following settings with a local checkout:

```json
{
  "languages": {
    "Create MyConfig": {
      "formatter": "language_server",
      "format_on_save": "on",
      "semantic_tokens": "combined",
      "tab_size": 4,
      "hard_tabs": false
    }
  },
  "lsp": {
    "cmc-lsp": {
      "binary": {
        "path": "/absolute/path/to/cmc-lsp-go/bin/cmc-lsp"
      }
    }
  }
}
```

The Tree-sitter layer supplies stable lexical highlighting—including correct
semicolon comments—while combined semantic tokens refine functions,
namespaces, parameters, and properties using LSP context.

CMC includes commonly use quoted Windows-style paths, which Zed's native
`editor::OpenSelectedFilename` action does not resolve. The language server
exposes them as document links and definitions. To make Vim-mode `gf` use that
resolver for CMC files, add this entry to `~/.config/zed/keymap.json`:

```json
[
  {
    "context": "Editor && vim_mode == normal && (extension == upscr || extension == uplib || extension == upact || extension == tea)",
    "bindings": {
      "g f": "editor::GoToDefinition"
    }
  }
]
```

Without the override, `gd` still opens an include and the include path is
clickable.

### Project and single-file modes

When the workspace contains a `.upproj` file, the server reads its XML and resolves the Windows-style paths in `ScriptLibList` and `<script ref="...">` elements. Referenced `.upscr`, `.uplib`, `.upact`, and `.tea` files share a project index, so library functions and project variables participate in completion, hover, signature help, go-to-definition, references, rename, and workspace-symbol searches.

Files that are not referenced by a `.upproj` remain in single-file mode and do not inherit unrelated project symbols. `.tea` data files and `.upact` action files are parsed and formatted as regular CMC scripts.

In single-file mode, the server discovers `.uplib` libraries beside the active script and in the directories listed by `%UP_LIB_PATH%`, following the CMC Diff search order. Directory entries are processed in filename order and duplicate directories are ignored. To override the inherited environment, pass the platform path list as `cmcLibraryPath` in the LSP `initializationOptions` object.

Hover documentation covers the language-provided functions and procedures from the current and legacy manuals, including XML and PLC ConfigData operations, Siemens standard-library callables, signatures, behavior, and compatibility aliases. German LSP client locales receive German descriptions. Completion recommends the current callable names and omits deprecated aliases.

The current manual's section 8.9 system variables are also documented on
hover. This includes `Up.$Pack`, all current dialog fields and enumeration
values, indexed language-archive entries, `Up.$Step[...]` and dialog-scoped
steps, `Up.$AccessLevelPWDConfig`, `Up.$BasicSecSettings`, and `Up.$Env`.
Indexed IDs are matched independently of their concrete value, and German LSP
client locales receive the corresponding German descriptions.
Completion after a documented system-variable object is member-aware: it only
offers valid immediate fields or enumeration values and includes their type,
access mode, localized description, and manual reference.

### Siemens parameter database

If a Siemens `DataBase` directory is present in the workspace or one of its parent directories, the server loads its `.mdat`, `.svar`, and `.para` XML files at startup. Hover then shows the supplied descriptions for NC machine/setting data, system variables, and SINAMICS `p`/`r` parameters. The files are parsed directly, so the database can be updated by replacing the directory.

The server uses the LSP client's `locale`: German locales use the descriptions in the `DataBase` root, while other locales prefer a matching language directory and then `DataBase/en`. To keep the data elsewhere, pass `cmcDatabasePath` in the LSP `initializationOptions` object.

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

The default style uses four spaces per nesting level, spaces around operators, one space after commas, two spaces before trailing comments, and aligns consecutive assignments and trailing comments. Use `-tab-size`, `-tabs`, `-comment-spaces`, `-align-consecutive-assignments`, or `-align-trailing-comments` to customize the CLI. Boolean flags can be disabled with `-flag=false`. LSP formatting honors the editor's `tabSize` and `insertSpaces` settings and accepts the optional `cmcCommentSpaces`, `cmcAlignConsecutiveAssignments`, and `cmcAlignTrailingComments` formatting properties.

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
- Optional leading `N<number>` machine-data annotations, treated as non-semantic comments
- Qualified `NC[...]`, `PS[...]`, and `BD[...]` data access
- Indexed identifiers and nested `$(Up.variable)` replacements
- Strings with embedded `$(...)` replacements, plus booleans, null, decimal,
  version, BICO, binary, and hexadecimal literals
- Procedure/function calls and `.uplib` `proc`/`func` definitions
- Adjacent `;Description:` and `;Arg1:` through `;Arg9:` library documentation comments, surfaced in completion and hover
- `#include` directives and semicolon comments

CMC has context-dependent host functions and system variables that vary by CMC release and package type. The server treats function names and data identifiers as open-ended rather than rejecting undocumented vendor or user-defined names.

Rename is intentionally limited to unambiguous, statically named user-defined functions and procedures. Vendor parameters, dynamic identifiers, and data variables are refused until their exact symbol boundaries and runtime scoping can be proven safely.

## Development

```sh
make test
make vet
```

Regenerate enum helpers after changing an enum:

```sh
make generate
```
