package lsp

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/lehmichael/cmc-lsp-go/internal/database"
	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	"github.com/lehmichael/cmc-lsp-go/internal/document"
	"github.com/lehmichael/cmc-lsp-go/internal/formatter"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
	"github.com/lehmichael/cmc-lsp-go/internal/project"
	"github.com/lehmichael/cmc-lsp-go/internal/source"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

type Lsp struct {
	rd          io.Reader
	wr          io.Writer
	overlay     *workspace.Overlay
	initialized bool
	shutdown    bool
	projects    []*project.Project
	index       []symbolOccurrence
	parameters  *database.Catalog
	locale      string
}

type header struct {
	contentLength int64
	contentType   *string
}

func NewLsp(reader io.Reader, writer io.Writer) *Lsp {
	return &Lsp{rd: reader, wr: writer, overlay: workspace.NewOverlay()}
}

func (server *Lsp) Start() int {
	reader := bufio.NewReader(server.rd)
	for {
		payload, err := readFrame(reader)
		if errors.Is(err, io.EOF) {
			return 1
		}
		if err != nil {
			return 1
		}
		if result := server.handleMessage(string(payload)); result >= 0 {
			return result
		}
	}
}

func readFrame(reader *bufio.Reader) ([]byte, error) {
	length := -1
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) && line == "" {
				return nil, io.EOF
			}
			return nil, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, found := strings.Cut(line, ":")
		if !found {
			return nil, fmt.Errorf("malformed header: %q", line)
		}
		if strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			parsed, err := strconv.Atoi(strings.TrimSpace(value))
			if err != nil || parsed < 0 {
				return nil, fmt.Errorf("invalid Content-Length")
			}
			length = parsed
		}
	}
	if length < 0 {
		return nil, errors.New("missing Content-Length")
	}
	payload := make([]byte, length)
	_, err := io.ReadFull(reader, payload)
	return payload, err
}

func (server *Lsp) handleMessage(payload string) int {
	var message lspMessage
	if err := json.Unmarshal([]byte(payload), &message); err != nil {
		server.respond(nil, nil, &responseError{Code: ParseError, Message: err.Error()})
		return -1
	}

	switch message := message.message.(type) {
	case notificationMessage:
		return server.handleNotification(message)
	case requestMessage:
		server.handleRequest(message)
	}
	return -1
}

func (server *Lsp) handleNotification(message notificationMessage) int {
	if message.Method == "exit" {
		if server.shutdown {
			return 0
		}
		return 1
	}
	if server.shutdown {
		return -1
	}

	switch message.Method {
	case "initialized", "$/cancelRequest", "$/setTrace":
		return -1
	case "textDocument/didOpen":
		var params didOpenParams
		if decodeParams(message.Params, &params) != nil {
			return -1
		}
		if server.overlay.Open(params.TextDocument.URI, params.TextDocument.Text, params.TextDocument.Version) == nil {
			server.reindex()
			server.publishDiagnostics(params.TextDocument.URI, params.TextDocument.Text, params.TextDocument.Version)
		}
	case "textDocument/didChange":
		var params didChangeParams
		if decodeParams(message.Params, &params) != nil || len(params.ContentChanges) == 0 {
			return -1
		}
		// The server advertises full synchronization; the final change is the
		// complete current document.
		text := params.ContentChanges[len(params.ContentChanges)-1].Text
		if server.overlay.Update(params.TextDocument.URI, text, params.TextDocument.Version) == nil {
			server.reindex()
			server.publishDiagnostics(params.TextDocument.URI, text, params.TextDocument.Version)
		}
	case "textDocument/didSave":
		var params documentParams
		if decodeParams(message.Params, &params) == nil {
			if text, err := server.overlay.ReadURI(params.TextDocument.URI); err == nil {
				server.publishDiagnostics(params.TextDocument.URI, text, 0)
			}
		}
	case "textDocument/didClose":
		var params didCloseParams
		if decodeParams(message.Params, &params) == nil {
			_ = server.overlay.Close(params.TextDocument.URI)
			server.reindex()
			server.notify("textDocument/publishDiagnostics", map[string]any{
				"uri": params.TextDocument.URI, "diagnostics": []diagnostic{},
			})
		}
	}
	return -1
}

func (server *Lsp) handleRequest(message requestMessage) {
	if server.shutdown {
		server.respond(message.ID, nil, &responseError{Code: InvalidRequest, Message: "server is shutting down"})
		return
	}
	if !server.initialized && message.Method != "initialize" && message.Method != "shutdown" {
		server.respond(message.ID, nil, &responseError{Code: ServerNotInitialized, Message: "server is not initialized"})
		return
	}

	switch message.Method {
	case "initialize":
		var params initializeParams
		if err := decodeParams(message.Params, &params); err != nil {
			server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
			return
		}
		server.initialized = true
		server.loadProjects(params)
		server.respond(message.ID, map[string]any{
			"capabilities": map[string]any{
				"positionEncoding":           "utf-16",
				"textDocumentSync":           1,
				"documentFormattingProvider": true,
				"documentSymbolProvider":     true,
				"documentLinkProvider": map[string]any{
					"resolveProvider": false,
				},
				"definitionProvider": true,
				"referencesProvider": true,
				"renameProvider": map[string]any{
					"prepareProvider": true,
				},
				"workspaceSymbolProvider": true,
				"hoverProvider":           true,
				"signatureHelpProvider": map[string]any{
					"triggerCharacters":   []string{"(", ","},
					"retriggerCharacters": []string{","},
				},
				"semanticTokensProvider": map[string]any{
					"legend": map[string]any{
						"tokenTypes":     semanticTokenTypes,
						"tokenModifiers": semanticTokenModifiers,
					},
					"full": true,
				},
				"completionProvider": map[string]any{
					"triggerCharacters": []string{".", "$"},
				},
			},
			"serverInfo": map[string]string{"name": "cmc-lsp", "version": "0.1.0"},
		}, nil)
	case "shutdown":
		server.shutdown = true
		server.respond(message.ID, nil, nil)
	case "textDocument/formatting":
		server.handleFormatting(message)
	case "textDocument/documentSymbol":
		server.handleDocumentSymbols(message)
	case "textDocument/documentLink":
		server.handleDocumentLinks(message)
	case "textDocument/completion":
		server.handleCompletion(message)
	case "textDocument/hover":
		server.handleHover(message)
	case "textDocument/definition":
		server.handleDefinition(message)
	case "textDocument/references":
		server.handleReferences(message)
	case "textDocument/signatureHelp":
		server.handleSignatureHelp(message)
	case "textDocument/prepareRename":
		server.handlePrepareRename(message)
	case "textDocument/rename":
		server.handleRename(message)
	case "textDocument/semanticTokens/full":
		server.handleSemanticTokens(message)
	case "workspace/symbol":
		server.handleWorkspaceSymbols(message)
	default:
		server.respond(message.ID, nil, &responseError{Code: MethodNotFound, Message: "method not found: " + message.Method})
	}
}

func (server *Lsp) handleFormatting(message requestMessage) {
	var params formattingParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	text, err := server.overlay.ReadURI(params.TextDocument.URI)
	if err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	options := formatter.DefaultOptions()
	options.TabSize = params.Options.TabSize
	options.InsertSpaces = params.Options.InsertSpaces
	if params.Options.CMCCommentSpaces > 0 {
		options.CommentSpaces = params.Options.CMCCommentSpaces
	}
	if params.Options.CMCAlignConsecutiveAssignments != nil {
		options.AlignConsecutiveAssignments = *params.Options.CMCAlignConsecutiveAssignments
	}
	if params.Options.CMCAlignTrailingComments != nil {
		options.AlignTrailingComments = *params.Options.CMCAlignTrailingComments
	}
	path, _ := workspace.URIToPath(params.TextDocument.URI)
	formatted := document.Format(path, text, options)
	if formatted == text {
		server.respond(message.ID, []textEdit{}, nil)
		return
	}
	server.respond(message.ID, []textEdit{{
		Range: lspRange{Start: position{}, End: documentEnd(text)}, NewText: formatted,
	}}, nil)
}

func (server *Lsp) handleDocumentSymbols(message requestMessage) {
	var params documentParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	text, err := server.overlay.ReadURI(params.TextDocument.URI)
	if err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	cmcText := cmcTextForURI(params.TextDocument.URI, text)
	tokens, diagnostics := lexer.Tokenize(cmcText)
	ast, _ := parser.Parse(tokens, diagnostics)
	server.respond(message.ID, symbolsForStatements(text, ast), nil)
}

func (server *Lsp) handleHover(message requestMessage) {
	var params textDocumentPositionParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	text, err := server.overlay.ReadURI(params.TextDocument.URI)
	if err != nil {
		server.respond(message.ID, nil, nil)
		return
	}
	systemVariable := systemVariableAt(text, params.Position)
	documentation, ok := systemVariableHover(systemVariable, server.locale)
	word := wordAt(text, params.Position)
	if !ok {
		documentation, ok = hoverDocumentation[strings.ToLower(word)]
	}
	if !ok && server.parameters != nil {
		documentation, ok = server.parameters.Hover(word)
	}
	if !ok {
		if occurrence := server.definitionAt(params.TextDocument.URI, symbolAt(text, params.Position)); occurrence != nil {
			documentation = callableHover(*occurrence)
		} else {
			server.respond(message.ID, nil, nil)
			return
		}
	}
	server.respond(message.ID, map[string]any{
		"contents": map[string]string{"kind": "markdown", "value": documentation},
	}, nil)
}

func callableHover(occurrence symbolOccurrence) string {
	if occurrence.CallableKind == "" {
		return "`" + occurrence.Name + "`  \n" + occurrence.Detail
	}
	result := "```cmc\n" + occurrence.CallableKind + " " + occurrence.Name + strings.TrimPrefix(occurrence.Detail, occurrence.CallableKind) + "\n```"
	if occurrence.Documentation != "" {
		result += "\n\n" + occurrence.Documentation
	}
	return result
}

func decodeParams(raw json.RawMessage, target any) error {
	if len(raw) == 0 {
		return nil
	}
	return json.Unmarshal(raw, target)
}

func (server *Lsp) publishDiagnostics(uri, text string, version int) {
	cmcText := cmcTextForURI(uri, text)
	tokens, diagnostics := lexer.Tokenize(cmcText)
	ast, diagnostics := parser.Parse(tokens, diagnostics)
	if path, err := workspace.URIToPath(uri); err == nil {
		for _, statement := range ast {
			switch kind := statement.Kind.(type) {
			case parser.PreprocessorStatement:
				if include, ok := kind.Kind.(parser.IncludePpStatement); ok && include.Path != "" {
					includePath := strings.Trim(include.Path, "\"")
					includePath = filepath.FromSlash(strings.ReplaceAll(includePath, "\\", "/"))
					if _, err := server.overlay.Read(filepath.Clean(filepath.Join(filepath.Dir(path), includePath))); err != nil {
						diagnostics = append(diagnostics, diag.Diagnostic{Kind: diag.MissingInclude, Range: statement.Range, Severity: diag.Error})
					}
				}
			case parser.FunctionStatement:
				if !strings.EqualFold(filepath.Ext(path), ".uplib") {
					diagnostics = append(diagnostics, diag.Diagnostic{Kind: diag.FunctionInScript, Range: statement.Range, Severity: diag.Error})
				}
			}
		}
	}
	result := make([]diagnostic, 0, len(diagnostics))
	for _, item := range diagnostics {
		result = append(result, diagnostic{
			Range:    sourceRangeToLSP(text, item.Range),
			Severity: 1,
			Code:     item.Kind.String(),
			Source:   "cmc",
			Message:  diagnosticMessage(item.Kind),
		})
	}
	params := map[string]any{"uri": uri, "diagnostics": result}
	if version != 0 {
		params["version"] = version
	}
	server.notify("textDocument/publishDiagnostics", params)
}

func cmcTextForURI(uri, text string) string {
	path, err := workspace.URIToPath(uri)
	if err != nil {
		return text
	}
	return document.CMCText(path, text)
}

func (server *Lsp) notify(method string, params any) {
	server.writeJSON(map[string]any{"jsonrpc": "2.0", "method": method, "params": params})
}

func (server *Lsp) respond(id messageID, result any, responseErr *responseError) {
	server.writeJSON(responseMessage{Message: Message{Jsonrpc: "2.0"}, ID: id, Result: result, Error: responseErr})
}

func (server *Lsp) writeJSON(value any) {
	payload, err := json.Marshal(value)
	if err != nil {
		return
	}
	_, _ = fmt.Fprintf(server.wr, "Content-Length: %d\r\n\r\n", len(payload))
	_, _ = server.wr.Write(payload)
}

func sourceRangeToLSP(text string, value source.SourceRange) lspRange {
	lines := strings.Split(text, "\n")
	convert := func(location source.SourceLocation) position {
		if location.Line < 0 {
			location.Line = 0
		}
		if location.Line >= len(lines) {
			return documentEnd(text)
		}
		return position{Line: location.Line, Character: runeColumnToUTF16(lines[location.Line], location.Column)}
	}
	return lspRange{Start: convert(value.Start), End: convert(value.End)}
}

func runeColumnToUTF16(line string, column int) int {
	if column <= 0 {
		return 0
	}
	count := 0
	for index, runeValue := range []rune(strings.TrimSuffix(line, "\r")) {
		if index >= column {
			break
		}
		count += utf16.RuneLen(runeValue)
	}
	return count
}

func documentEnd(text string) position {
	lines := strings.Split(text, "\n")
	line := len(lines) - 1
	return position{Line: line, Character: runeColumnToUTF16(lines[line], utf8.RuneCountInString(lines[line]))}
}

func diagnosticMessage(kind diag.DiagnosticKind) string {
	if message, ok := diagnosticMessages[kind]; ok {
		return message
	}
	return kind.String()
}

var diagnosticMessages = map[diag.DiagnosticKind]string{
	diag.UnexpectedToken:                                "Unexpected token",
	diag.FunctionInvalidIdentifier:                      "Invalid function or procedure name",
	diag.FunctionInvalidParameterDef:                    "Parameters must be declared as $ placeholders",
	diag.FunctionMissingOpeningBrace:                    "Function or procedure body is missing an opening brace",
	diag.FunctionBodyClosingBraceMissing:                "Function or procedure body is missing a closing brace",
	diag.SectionInvalidDrive:                            "Invalid drive section; expected [B<bus>_S<slave>_PS<do>]",
	diag.SectionInvalidChan:                             "Invalid channel section; expected [C1] through [C10]",
	diag.SectionFormatUnrecogniced:                      "Unrecognized section",
	diag.ExpressionGroupedMissingClosingParentheses:     "Grouped expression is missing a closing parenthesis",
	diag.ExpressionReplacementMissingClosingParentheses: "Replacement expression is missing a closing parenthesis",
	diag.WhileEndMissing:                                "While block is missing EndWhile",
	diag.IfThenEndMissing:                               "If block is missing EndIf",
	diag.StringUnterminated:                             "Unterminated string literal",
	diag.NumberFormatUnterminated:                       "Unterminated single-quoted literal",
	diag.MissingInclude:                                 "Included file was not found",
	diag.CircularInclude:                                "Circular include",
	diag.FunctionInScript:                               "Function and procedure definitions are only allowed in .uplib files",
}

func symbolsForStatements(text string, statements []parser.Statement) []documentSymbol {
	result := make([]documentSymbol, 0)
	for _, statement := range statements {
		rangeValue := sourceRangeToLSP(text, statement.Range)
		switch kind := statement.Kind.(type) {
		case parser.FunctionStatement:
			name := "<invalid definition>"
			selection := rangeValue
			callableKind := "func"
			if kind.Kind == parser.Procedure {
				callableKind = "proc"
			}
			detail := callableDetail(callableKind, kind.ArgCount, kind.ArgumentDescriptions)
			if kind.Identifier != nil {
				name = parser.IdentifierString(*kind.Identifier)
				selection = sourceRangeToLSP(text, kind.Identifier.Range)
			}
			result = append(result, documentSymbol{Name: name, Detail: detail, Kind: 12, Range: rangeValue, SelectionRange: selection, Children: symbolsForStatements(text, kind.Body)})
		case parser.Assignment:
			result = append(result, documentSymbol{Name: parser.IdentifierString(kind.Target), Detail: "assignment", Kind: 13, Range: rangeValue, SelectionRange: sourceRangeToLSP(text, kind.Target.Range)})
		case parser.SectionSwitch:
			result = append(result, documentSymbol{Name: parser.SectionString(kind.Kind), Detail: "section", Kind: 3, Range: rangeValue, SelectionRange: rangeValue})
		case parser.IfBlock:
			children := symbolsForStatements(text, kind.ThenBranch)
			for _, branch := range kind.ElseIfBranch {
				children = append(children, symbolsForStatements(text, branch.ThenBranch)...)
			}
			if kind.ElseBranch != nil {
				children = append(children, symbolsForStatements(text, kind.ElseBranch.ThenBranch)...)
			}
			result = append(result, documentSymbol{Name: "If", Kind: 19, Range: rangeValue, SelectionRange: rangeValue, Children: children})
		case parser.WhileBlock:
			result = append(result, documentSymbol{Name: "While", Kind: 19, Range: rangeValue, SelectionRange: rangeValue, Children: symbolsForStatements(text, kind.Body)})
		}
	}
	return result
}

func wordAt(text string, target position) string {
	lines := strings.Split(text, "\n")
	if target.Line < 0 || target.Line >= len(lines) {
		return ""
	}
	runes := []rune(lines[target.Line])
	column := utf16ToRuneColumn(runes, target.Character)
	if column == len(runes) && column > 0 {
		column--
	}
	allowed := func(value rune) bool {
		return unicode.IsLetter(value) || unicode.IsNumber(value) || value == '_' || value == '$'
	}
	start, end := column, column
	for start > 0 && allowed(runes[start-1]) {
		start--
	}
	for end < len(runes) && allowed(runes[end]) {
		end++
	}
	return string(runes[start:end])
}

func utf16ToRuneColumn(runes []rune, target int) int {
	units := 0
	for index, value := range runes {
		next := units + utf16.RuneLen(value)
		if next > target {
			return index
		}
		units = next
	}
	return len(runes)
}

func completions() []completionItem {
	return completionItems
}

var completionItems = []completionItem{
	{Label: "If", Kind: 14, Detail: "If ... EndIf", InsertText: "If "},
	{Label: "ElsIf", Kind: 14, Detail: "Alternative condition", InsertText: "ElsIf "},
	{Label: "Else", Kind: 14}, {Label: "EndIf", Kind: 14},
	{Label: "While", Kind: 14, Detail: "While ... EndWhile", InsertText: "While "},
	{Label: "EndWhile", Kind: 14}, {Label: "true", Kind: 21},
	{Label: "false", Kind: 21}, {Label: "null", Kind: 21},
	{Label: "CHANDATA", Kind: 3, InsertText: "CHANDATA()"},
	{Label: "#include", Kind: 14, InsertText: "#include \"\""},
	{Label: "StringLen", Kind: 3, InsertText: "StringLen()"},
	{Label: "StringMatch", Kind: 3, InsertText: "StringMatch()"},
	{Label: "StringPos", Kind: 3, InsertText: "StringPos()"},
	{Label: "StringReplace", Kind: 3, InsertText: "StringReplace()"},
	{Label: "StringSubStr", Kind: 3, InsertText: "StringSubStr()"},
	{Label: "FileCopy", Kind: 3, InsertText: "FileCopy()"},
	{Label: "FileDelete", Kind: 3, InsertText: "FileDelete()"},
	{Label: "FileExist", Kind: 3, InsertText: "FileExist()"},
	{Label: "FileRead", Kind: 3, InsertText: "FileRead()"},
	{Label: "FileWrite", Kind: 3, InsertText: "FileWrite()"},
	{Label: "Log", Kind: 3, InsertText: "Log()"}, {Label: "Msg", Kind: 3, InsertText: "Msg()"},
	{Label: "Warning", Kind: 3, InsertText: "Warning()"}, {Label: "Error", Kind: 3, InsertText: "Error()"},
	{Label: "Version", Kind: 3, InsertText: "Version()"}, {Label: "DOVar", Kind: 3, InsertText: "DOVar()"},
	{Label: "Return", Kind: 3, InsertText: "Return()"},
}

var hoverDocumentation = map[string]string{
	"if":            "`If condition ... ElsIf condition ... Else ... EndIf` conditionally executes a block.",
	"while":         "`While condition ... EndWhile` repeats a block while its condition is true.",
	"chandata":      "`CHANDATA(n)` selects NC channel `n` for subsequent unqualified data access.",
	"include":       "`#include \"path\"` includes an external script at deployment time.",
	"stringlen":     "`StringLen(string)` returns the number of printable and non-printable characters.",
	"stringmatch":   "`StringMatch(string, search)` returns the part matched by a regular expression.",
	"stringpos":     "`StringPos(string, search, position)` returns the first match position, or `-1`.",
	"stringreplace": "`StringReplace(string, search, replacement)` replaces all regular-expression matches.",
	"stringsubstr":  "`StringSubStr(string, position, length)` returns a substring.",
	"fileexist":     "`FileExist(area, path)` tests whether a file exists. Common areas are `RTS`, `ARC`, `NCU`, and `PCU`.",
	"version":       "`Version(area, product)` returns a product version or `null`.",
	"return":        "`Return(value)` supplies the return value of a function defined in a `.uplib` library.",
}
