package lsp

import (
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/lehmichael/cmc-lsp-go/internal/database"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
	"github.com/lehmichael/cmc-lsp-go/internal/project"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

type symbolOccurrence struct {
	Name          string
	URI           string
	Range         lspRange
	Kind          int
	Detail        string
	Documentation string
	CallableKind  string
	Description   string
	Arguments     []string
	ArgumentCount int
	Definition    bool
	Project       string
	Pattern       identifierPattern
}

func (server *Lsp) loadProjects(params initializeParams) {
	var roots []string
	for _, folder := range params.WorkspaceFolders {
		if path, err := workspace.URIToPath(folder.URI); err == nil {
			roots = append(roots, path)
		}
	}
	if len(roots) == 0 && params.RootURI != nil {
		if path, err := workspace.URIToPath(*params.RootURI); err == nil {
			roots = append(roots, path)
		}
	}
	if len(roots) == 0 && params.RootPath != nil {
		roots = append(roots, *params.RootPath)
	}
	if databasePath := database.Locate(roots, params.InitializationOptions.CMCDatabasePath); databasePath != "" {
		if catalog, err := database.Load(databasePath, params.Locale); err == nil {
			server.parameters = catalog
		}
	}

	seen := map[string]struct{}{}
	for _, root := range roots {
		projects, err := project.Find(root)
		if err != nil {
			continue
		}
		for _, loaded := range projects {
			if _, ok := seen[loaded.Path]; ok {
				continue
			}
			seen[loaded.Path] = struct{}{}
			server.projects = append(server.projects, loaded)
		}
	}
	server.reindex()
}

func (server *Lsp) reindex() {
	var result []symbolOccurrence
	for _, loaded := range server.projects {
		for _, path := range loaded.Files() {
			text, err := server.overlay.Read(path)
			if err != nil {
				continue
			}
			result = append(result, occurrences(text, workspace.PathToURI(path), loaded.Path)...)
		}
	}
	server.index = result
}

func occurrences(text, uri, projectPath string) []symbolOccurrence {
	tokens, diagnostics := lexer.Tokenize(text)
	statements, _ := parser.Parse(tokens, diagnostics)
	var result []symbolOccurrence

	var addIdentifier func(parser.IdentifierExpression, bool, int, string) int
	addIdentifier = func(identifier parser.IdentifierExpression, definition bool, kind int, detail string) int {
		name := parser.IdentifierString(identifier)
		if name == "" {
			return -1
		}
		index := len(result)
		result = append(result, symbolOccurrence{
			Name: name, URI: uri, Range: sourceRangeToLSP(text, identifier.Range),
			Kind: kind, Detail: detail, Definition: definition, Project: projectPath,
			Pattern: identifierPatternFromAST(identifier),
		})
		for _, segment := range identifier.Segments {
			for _, part := range segment.Parts {
				if replacement, ok := part.(parser.ReplacementIdentifier); ok {
					addIdentifier(parser.IdentifierExpression(replacement), false, 13, "CMC replacement variable")
				}
			}
		}
		return index
	}

	var addExpression func(parser.ExpressionKind)
	addExpression = func(expression parser.ExpressionKind) {
		switch expression := expression.(type) {
		case parser.IdentifierExpression:
			addIdentifier(expression, false, 13, "CMC data or package variable")
		case parser.CallExpression:
			addIdentifier(expression.Identifier, false, 12, "Function or procedure call")
			for _, parameter := range expression.Parameters {
				addExpression(parameter.Kind)
			}
		case parser.InterpolatedStringLiteral:
			for _, replacement := range expression.Replacements {
				addIdentifier(replacement, false, 13, "CMC string replacement variable")
			}
		case parser.GroupedExpression:
			addExpression(expression.Expression.Kind)
		case parser.PrefixedExpression:
			addExpression(expression.Expression.Kind)
		case parser.BinaryExpression:
			addExpression(expression.Left.Kind)
			addExpression(expression.Right.Kind)
		}
	}

	var addStatements func([]parser.Statement)
	addStatements = func(statements []parser.Statement) {
		for _, statement := range statements {
			switch statement := statement.Kind.(type) {
			case parser.Assignment:
				addIdentifier(statement.Target, true, 13, "CMC data or package variable")
				addExpression(statement.Value.Kind)
			case parser.DeleteStatement:
				addIdentifier(statement.Identifier, false, 13, "CMC data or package variable")
			case parser.CallStatement:
				addExpression(parser.CallExpression(statement))
			case parser.FunctionStatement:
				if statement.Identifier != nil {
					callableKind := "func"
					if statement.Kind == parser.Procedure {
						callableKind = "proc"
					}
					index := addIdentifier(*statement.Identifier, true, 12, callableDetail(callableKind, statement.ArgCount, statement.ArgumentDescriptions))
					if index >= 0 {
						result[index].Documentation = callableDocumentation(statement.Description, statement.ArgumentDescriptions)
						result[index].CallableKind = callableKind
						result[index].Description = statement.Description
						result[index].Arguments = append([]string(nil), statement.ArgumentDescriptions...)
						result[index].ArgumentCount = statement.ArgCount
					}
				}
				addStatements(statement.Body)
			case parser.IfBlock:
				if statement.Condition != nil {
					addExpression(statement.Condition.Kind)
				}
				addStatements(statement.ThenBranch)
				for _, branch := range statement.ElseIfBranch {
					if branch.Condition != nil {
						addExpression(branch.Condition.Kind)
					}
					addStatements(branch.ThenBranch)
				}
				if statement.ElseBranch != nil {
					addStatements(statement.ElseBranch.ThenBranch)
				}
			case parser.WhileBlock:
				if statement.Condition != nil {
					addExpression(statement.Condition.Kind)
				}
				addStatements(statement.Body)
			}
		}
	}
	addStatements(statements)
	return result
}

func callableDetail(kind string, argCount int, arguments []string) string {
	labels := make([]string, argCount)
	for index := range labels {
		labels[index] = "$"
		if index < len(arguments) && arguments[index] != "" {
			labels[index] = arguments[index]
		}
	}
	return kind + "(" + strings.Join(labels, ", ") + ")"
}

func callableDocumentation(description string, arguments []string) string {
	var sections []string
	if description != "" {
		sections = append(sections, description)
	}
	for index, argument := range arguments {
		if argument != "" {
			sections = append(sections, "Arg"+integerString(index+1)+": `"+argument+"`")
		}
	}
	return strings.Join(sections, "  \n")
}

func integerString(value int) string {
	if value == 0 {
		return "0"
	}
	var digits [20]byte
	index := len(digits)
	for value > 0 {
		index--
		digits[index] = byte('0' + value%10)
		value /= 10
	}
	return string(digits[index:])
}

func (server *Lsp) projectForURI(uri string) *project.Project {
	path, err := workspace.URIToPath(uri)
	if err != nil {
		return nil
	}
	for _, loaded := range server.projects {
		if loaded.Contains(path) {
			return loaded
		}
	}
	return nil
}

func (server *Lsp) scopedOccurrences(uri string) []symbolOccurrence {
	if loaded := server.projectForURI(uri); loaded != nil {
		var result []symbolOccurrence
		for _, occurrence := range server.index {
			if occurrence.Project == loaded.Path {
				result = append(result, occurrence)
			}
		}
		return result
	}
	text, err := server.overlay.ReadURI(uri)
	if err != nil {
		return nil
	}
	return occurrences(text, uri, "")
}

func (server *Lsp) definitionAt(uri, name string) *symbolOccurrence {
	if name == "" {
		return nil
	}
	for _, occurrence := range server.scopedOccurrences(uri) {
		if occurrence.Definition && strings.EqualFold(occurrence.Name, name) {
			copy := occurrence
			return &copy
		}
	}
	return nil
}

func (server *Lsp) callableDefinitionAt(uri, name string) *symbolOccurrence {
	var result *symbolOccurrence
	for _, occurrence := range server.scopedOccurrences(uri) {
		if !occurrence.Definition || occurrence.CallableKind == "" || !strings.EqualFold(occurrence.Name, name) {
			continue
		}
		if result != nil {
			return nil
		}
		copy := occurrence
		result = &copy
	}
	return result
}

func (server *Lsp) handleCompletion(message requestMessage) {
	var params textDocumentPositionParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	items := append([]completionItem(nil), completionItems...)
	seen := map[string]struct{}{}
	for _, item := range items {
		seen[strings.ToLower(item.Label)] = struct{}{}
	}
	for _, occurrence := range server.scopedOccurrences(params.TextDocument.URI) {
		if !occurrence.Definition {
			continue
		}
		key := strings.ToLower(occurrence.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		insertText := occurrence.Name
		completionKind := 6
		if occurrence.Kind == 12 {
			completionKind = 3
			insertText += "()"
		}
		var documentation *markupContent
		if occurrence.Documentation != "" {
			documentation = &markupContent{Kind: "markdown", Value: occurrence.Documentation}
		}
		items = append(items, completionItem{Label: occurrence.Name, Kind: completionKind, Detail: occurrence.Detail, Documentation: documentation, InsertText: insertText})
	}
	slices.SortFunc(items, func(left, right completionItem) int {
		return strings.Compare(strings.ToLower(left.Label), strings.ToLower(right.Label))
	})
	server.respond(message.ID, items, nil)
}

func (server *Lsp) handleDefinition(message requestMessage) {
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
	if include := server.includeLocationAt(text, params.TextDocument.URI, params.Position); include != nil {
		server.respond(message.ID, *include, nil)
		return
	}
	definition := server.definitionAt(params.TextDocument.URI, symbolAt(text, params.Position))
	if definition == nil {
		server.respond(message.ID, nil, nil)
		return
	}
	server.respond(message.ID, location{URI: definition.URI, Range: definition.Range}, nil)
}

func (server *Lsp) handleReferences(message requestMessage) {
	var params textDocumentPositionParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	text, err := server.overlay.ReadURI(params.TextDocument.URI)
	if err != nil {
		server.respond(message.ID, []location{}, nil)
		return
	}
	occurrences := server.scopedOccurrences(params.TextDocument.URI)
	selected := occurrenceAt(occurrences, params.TextDocument.URI, params.Position)
	fallbackName := symbolAt(text, params.Position)
	var result []location
	for _, occurrence := range occurrences {
		matches := false
		if selected != nil {
			matches = occurrence.Kind == selected.Kind && identifierPatternsOverlap(selected.Pattern, occurrence.Pattern)
		} else {
			matches = strings.EqualFold(occurrence.Name, fallbackName)
		}
		if matches {
			result = append(result, location{URI: occurrence.URI, Range: occurrence.Range})
		}
	}
	server.respond(message.ID, result, nil)
}

func occurrenceAt(occurrences []symbolOccurrence, uri string, target position) *symbolOccurrence {
	var selected *symbolOccurrence
	for _, occurrence := range occurrences {
		if occurrence.URI != uri || !positionInRange(target, occurrence.Range) {
			continue
		}
		if selected == nil || rangeContains(selected.Range, occurrence.Range) {
			copy := occurrence
			selected = &copy
		}
	}
	return selected
}

func rangeContains(outer, inner lspRange) bool {
	startsBeforeOrEqual := !positionLess(inner.Start, outer.Start)
	endsAfterOrEqual := !positionLess(outer.End, inner.End)
	return startsBeforeOrEqual && endsAfterOrEqual && outer != inner
}

func (server *Lsp) handleWorkspaceSymbols(message requestMessage) {
	var params workspaceSymbolParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	query := strings.ToLower(params.Query)
	seen := map[string]struct{}{}
	var result []symbolInformation
	for _, occurrence := range server.index {
		if !occurrence.Definition || !strings.Contains(strings.ToLower(occurrence.Name), query) {
			continue
		}
		key := occurrence.Project + "\x00" + strings.ToLower(occurrence.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, symbolInformation{
			Name: occurrence.Name, Kind: occurrence.Kind,
			Location:      location{URI: occurrence.URI, Range: occurrence.Range},
			ContainerName: filepath.Base(occurrence.Project),
		})
	}
	server.respond(message.ID, result, nil)
}

func symbolAt(text string, target position) string {
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
		return unicode.IsLetter(value) || unicode.IsNumber(value) || value == '_' || value == '$' || value == '.'
	}
	start, end := column, column
	for start > 0 && allowed(runes[start-1]) {
		start--
	}
	for end < len(runes) && allowed(runes[end]) {
		end++
	}
	return strings.Trim(string(runes[start:end]), ".")
}
