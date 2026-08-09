# CMC language tooling guide

This repository implements the parser, formatter, project model, database
catalog, and Language Server Protocol support for Siemens Create MyConfig
(CMC). Keep behavior consistent across `.upscr`, `.uplib`, `.upact`, and `.tea`
unless the language explicitly distinguishes library definitions.

## Language references

The canonical bundled reference is:

- `docs/reference/840Dsl_Create_MyConfig_op_man_en-US.txt` for fast `rg`
  searches and text extraction.
- `docs/reference/840Dsl_Create_MyConfig_op_man_en-US.pdf` when table layout,
  figures, or extraction ambiguity matters.

Useful manual locations include:

- Sections 6.3.8.5-6.3.8.6 (printed pages 94-98): Diff system variables.
- Pages 419-420: `func`/`proc` documentation comments.
- Section 7.11 (printed pages 447-472): Expert system variables under
  `Up.$Pack`, `Up.$Dialog`, `Up.$Step`, and `Up.$Env`.

The official searchable HTML edition is available from Siemens at
`https://support.industry.siemens.com/cs/attachments/109769192/125621338379.zip?download=true`.
It contains separate `de-DE` and `en-US` topics and is useful when extracting
localized table entries. Do not infer undocumented CMC semantics solely from
identifier names; prefer the manual and clearly label fallbacks.

The committed `DataBase` directory is a replaceable, Siemens-supplied catalog
of NC, drive, and system parameter descriptions. Preserve the ability to
update it by replacing the directory as-is. `cmc_projekt` is a large ignored
reference project used for corpus testing and must not be committed.

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
