# CMC language tooling guide

This repository implements the parser, formatter, project model, database
catalog, and Language Server Protocol support for Siemens Create MyConfig
(CMC). Keep behavior consistent across `.upscr`, `.uplib`, `.upact`, and `.tea`
unless the language explicitly distinguishes library definitions.

## Language references

The bundled references are:

- `docs/reference/ONE_Create_MyConfig_op_man_0626_en-US.txt` and the adjacent
  German edition for current language behavior and localized descriptions.
- `docs/reference/840Dsl_Create_MyConfig_op_man_en-US.txt` and the adjacent
  German edition for legacy behavior and details omitted by the newer manual.
- The matching PDFs when table layout, figures, or extraction ambiguity
  matters.

Useful 06/2019 manual locations include:

- Sections 6.3.8.5-6.3.8.6 (printed pages 94-98): Diff system variables.
- Pages 419-420: `func`/`proc` documentation comments.
- Section 7.11 (printed pages 447-472): Expert system variables under
  `Up.$Pack`, `Up.$Dialog`, `Up.$Step`, and `Up.$Env`.

The official searchable HTML edition is available from Siemens at
`https://support.industry.siemens.com/cs/attachments/109769192/125621338379.zip?download=true`.
It contains separate `de-DE` and `en-US` topics and is useful when extracting
localized table entries. Do not infer undocumented CMC semantics solely from
identifier names; prefer the manual and clearly label fallbacks.

The 06/2026 manual moves the CMC language to chapter 6. Its sections 6.16 and
6.17 document XML and PLC ConfigData functions that are absent from the older
manual. The English 6.16.6 heading and both editions' `RemoveXmlElement`
signature contain known copy/paste errors; use the German heading and call
examples to resolve them.

Section 8.9 (printed pages 373-391) is the current system-variable reference.
It covers package configuration, dialog fields and enumerations, direct and
dialog-scoped steps, and the runtime environment. Keep its English and German
descriptions synchronized in `internal/lsp/current_system_variables.go`.

`DataBase` is an ignored, locally supplied Siemens catalog of NC, drive, and
system parameter descriptions. It must not be committed or distributed in
release artifacts. Preserve the ability to update it by replacing the local
directory as-is. `cmc_projekt` is a large ignored reference project used for
corpus testing and must not be committed.

## Completion behavior

Completion is context-sensitive; do not solve member completion by returning
or merely reordering the global keyword and symbol list.

- `internal/lsp/index.go` owns completion dispatch and project-defined symbol
  completion. Contextual completion must run before the global fallback.
- `internal/lsp/system_variables.go` handles documented members below known
  system-variable parents such as `Up.$Step[...].` and suppresses unrelated
  global items in those contexts.
- `Up.` and nested package-variable paths must offer only direct `Up` children.
  Root completion combines documented roots such as `$Pack`, `$Dialog`,
  `$Step`, and `$Env` with project-defined package variables. Nested paths must
  collapse deeper definitions to their next segment (for example,
  `Up.Group.Child.Value` contributes `Child` at `Up.Group.`).
- `$...`, `P...`, and `R...` completion comes from the loaded `database.Catalog`,
  not from identifiers encountered in source files. Keep the result bounded:
  Siemens catalogs can contain thousands of machine-data entries and tens of
  thousands of drive-parameter records. Return an LSP `CompletionList` with
  `isIncomplete: true` when a prefix has more matches than the limit so the
  client can request a narrower prefix.
- Catalog completion labels use `database.Parameter.Identifier`; lookup remains
  case-insensitive and normalizes numeric `P`/`R` identifiers. Completion
  details and markdown documentation should use the catalog's type, brief, and
  full hover text. Distinguish machine data, setting data, option data,
  SINAMICS write parameters (`p`), SINAMICS read parameters (`r`), and other
  system variables.
- The description `CMC data or package variable` applies only to identifiers
  beginning with `Up.`. Do not apply it to machine data, drive parameters,
  callables, or generic identifiers.

Regression coverage for these rules belongs in
`internal/lsp/completion_test.go`; catalog prefix matching is covered in
`internal/database/catalog_test.go`.

## Validation in sandboxed agent sessions

Run `go test ./...`, `go vet ./...`, and `make build` as usual. On macOS, a
sandboxed command may be unable to write the default Go cache under
`~/Library/Caches`, the module cache under `~/go/pkg/mod`, or temporary Xcode
cache files under `/var/folders`. Treat cache-location warnings separately from
compiler/test failures and verify command exit status and produced binaries.
If an alternate `GOCACHE` is necessary, choose one stable location for the
whole session (or request the required permission) and reuse it. Do not create
and delete a project-local cache around each command; caches are meant to
persist, and an untracked cache directory pollutes worktree status.

## Related repositories

- `../tree-sitter-cmc`: syntax grammar and highlight queries. Regenerate and
  commit `src/parser.c`, `src/grammar.json`, and `src/node-types.json` after
  grammar changes.
- `../zed-cmc`: Zed language registration and LSP adapter. Pin its grammar
  revision after committing Tree-sitter changes.

When adding a file extension or user-visible syntax, check all three
repositories. Run `go test ./...`, `go vet ./...`, and `make build` here;
`tree-sitter generate`, `tree-sitter test`, and `tree-sitter build --wasm` in
the grammar; and `cargo fmt --check`, `cargo check`, and Clippy in the Zed
extension.
