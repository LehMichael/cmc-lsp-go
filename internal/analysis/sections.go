package analysis

import (
	"path/filepath"
	"strings"

	"github.com/lehmichael/cmc-lsp-go/internal/diag"
	"github.com/lehmichael/cmc-lsp-go/internal/document"
	"github.com/lehmichael/cmc-lsp-go/internal/lexer"
	"github.com/lehmichael/cmc-lsp-go/internal/parser"
	"github.com/lehmichael/cmc-lsp-go/internal/workspace"
)

type sectionState uint8

const (
	sectionUnset sectionState = 1 << iota
	sectionNC
	sectionDrive
	sectionDisplay
	sectionUnknown
)

type sectionChecker struct {
	overlay     *workspace.Overlay
	diagnostics []diag.Diagnostic
	parsed      map[string]parser.Ast
	stack       map[string]struct{}
}

// ValidateSectionAccesses checks that unqualified area data uses the active
// section and that fully qualified identifiers are only used for reads.
// Included scripts inherit the caller's section and return their final state.
// Diagnostics from included files are left for those documents to publish.
func ValidateSectionAccesses(path string, ast parser.Ast, overlay *workspace.Overlay) []diag.Diagnostic {
	path = filepath.Clean(path)
	checker := sectionChecker{
		overlay: overlay,
		parsed:  map[string]parser.Ast{path: ast},
		stack:   map[string]struct{}{path: {}},
	}
	checker.walkStatements(path, ast, sectionUnset, true)
	return checker.diagnostics
}

func (checker *sectionChecker) walkStatements(path string, statements []parser.Statement, state sectionState, report bool) sectionState {
	for _, statement := range statements {
		switch kind := statement.Kind.(type) {
		case parser.SectionSwitch:
			state = stateForSection(kind.Kind)
		case parser.Assignment:
			checker.validateWrite(kind.Target, state, report)
			checker.validateExpression(kind.Value.Kind, state, report)
		case parser.DeleteStatement:
			checker.validateWrite(kind.Identifier, state, report)
		case parser.CallStatement:
			checker.validateCall(parser.CallExpression(kind), state, report)
		case parser.IfBlock:
			checker.validateOptionalExpression(kind.Condition, state, report)
			branchStates := checker.walkStatements(path, kind.ThenBranch, state, report)
			for _, branch := range kind.ElseIfBranch {
				checker.validateOptionalExpression(branch.Condition, state, report)
				branchStates |= checker.walkStatements(path, branch.ThenBranch, state, report)
			}
			if kind.ElseBranch == nil {
				branchStates |= state
			} else {
				branchStates |= checker.walkStatements(path, kind.ElseBranch.ThenBranch, state, report)
			}
			state = branchStates
		case parser.WhileBlock:
			checker.validateOptionalExpression(kind.Condition, state, report)
			state |= checker.walkStatements(path, kind.Body, state, report)
		case parser.FunctionStatement:
			// A library callable inherits its caller's section, which is not
			// statically known at the definition site.
			checker.walkStatements(path, kind.Body, sectionUnknown, report)
		case parser.PreprocessorStatement:
			if include, ok := kind.Kind.(parser.IncludePpStatement); ok {
				state = checker.walkInclude(path, include.Path, state)
			}
		}
	}
	return state
}

func (checker *sectionChecker) walkInclude(path, quotedPath string, state sectionState) sectionState {
	if checker.overlay == nil || quotedPath == "" {
		return state
	}
	includePath := strings.Trim(quotedPath, "\"")
	includePath = includeFilePath(filepath.Dir(path), includePath)
	if _, circular := checker.stack[includePath]; circular {
		return state
	}
	ast, ok := checker.parsed[includePath]
	if !ok {
		text, err := checker.overlay.Read(includePath)
		if err != nil {
			return state
		}
		tokens, diagnostics := lexer.Tokenize(document.CMCText(includePath, text))
		ast, _ = parser.Parse(tokens, diagnostics)
		checker.parsed[includePath] = ast
	}
	checker.stack[includePath] = struct{}{}
	state = checker.walkStatements(includePath, ast, state, false)
	delete(checker.stack, includePath)
	return state
}

func (checker *sectionChecker) validateWrite(identifier parser.IdentifierExpression, state sectionState, report bool) {
	if identifier.Section != nil {
		if report {
			checker.add(diag.FullyQualifiedIdentifierWrite, identifier)
		}
		return
	}
	checker.validateIdentifier(identifier, state, report)
}

func (checker *sectionChecker) validateOptionalExpression(expression *parser.Expression, state sectionState, report bool) {
	if expression != nil {
		checker.validateExpression(expression.Kind, state, report)
	}
}

func (checker *sectionChecker) validateExpression(expression parser.ExpressionKind, state sectionState, report bool) {
	switch kind := expression.(type) {
	case parser.IdentifierExpression:
		checker.validateIdentifier(kind, state, report)
	case parser.GroupedExpression:
		checker.validateExpression(kind.Expression.Kind, state, report)
	case parser.PrefixedExpression:
		checker.validateExpression(kind.Expression.Kind, state, report)
	case parser.BinaryExpression:
		checker.validateExpression(kind.Left.Kind, state, report)
		checker.validateExpression(kind.Right.Kind, state, report)
	case parser.CallExpression:
		checker.validateCall(kind, state, report)
	case parser.InterpolatedStringLiteral:
		for _, replacement := range kind.Replacements {
			checker.validateIdentifier(replacement, state, report)
		}
	}
}

func (checker *sectionChecker) validateCall(call parser.CallExpression, state sectionState, report bool) {
	for _, parameter := range call.Parameters {
		checker.validateExpression(parameter.Kind, state, report)
	}
}

func (checker *sectionChecker) validateIdentifier(identifier parser.IdentifierExpression, state sectionState, report bool) {
	expected := identifierArea(identifier)
	if expected == 0 || !report {
		return
	}
	actual := state
	if identifier.Section != nil {
		actual = stateForSection(*identifier.Section)
	}
	if actual&expected != 0 || actual&sectionUnknown != 0 {
		return
	}
	switch expected {
	case sectionNC:
		checker.add(diag.NCDataSectionRequired, identifier)
	case sectionDrive:
		checker.add(diag.DriveDataSectionRequired, identifier)
	case sectionDisplay:
		checker.add(diag.DisplayDataSectionRequired, identifier)
	}
}

func (checker *sectionChecker) add(kind diag.DiagnosticKind, identifier parser.IdentifierExpression) {
	checker.diagnostics = append(checker.diagnostics, diag.Diagnostic{
		Kind: kind, Range: identifier.Range, Severity: diag.Error,
	})
}

func stateForSection(section parser.SectionSwitchKind) sectionState {
	namespace := parser.Unqualified
	switch section := section.(type) {
	case *parser.ChannelSection:
		namespace = section.Namespace
		if namespace == parser.Unqualified {
			return sectionNC
		}
	case *parser.DriveSection:
		namespace = section.Namespace
		if namespace == parser.Unqualified {
			return sectionDrive
		}
	case parser.DisplaySection:
		namespace = section.Namespace
		if namespace == parser.Unqualified {
			return sectionDisplay
		}
	case parser.DynamicSection:
		namespace = section.Namespace
	case parser.InvalidSection:
		return sectionUnknown
	default:
		return sectionUnknown
	}
	switch namespace {
	case parser.Nc, parser.Chandata:
		return sectionNC
	case parser.Ps:
		return sectionDrive
	case parser.Bd:
		return sectionDisplay
	default:
		return sectionUnknown
	}
}

func identifierArea(identifier parser.IdentifierExpression) sectionState {
	if len(identifier.Segments) == 0 || len(identifier.Segments[0].Parts) == 0 {
		return 0
	}
	head, ok := identifier.Segments[0].Parts[0].(parser.LiteralIdentifier)
	if !ok {
		return 0
	}
	name := strings.ToUpper(string(head))
	switch {
	case strings.HasPrefix(name, "$MM_"):
		return sectionDisplay
	case strings.HasPrefix(name, "$MN_"), strings.HasPrefix(name, "$MC_"),
		strings.HasPrefix(name, "$MA_"), strings.HasPrefix(name, "$MNS_"),
		strings.HasPrefix(name, "$MCS_"), strings.HasPrefix(name, "$MAS_"),
		strings.HasPrefix(name, "$SN_"), strings.HasPrefix(name, "$SC_"),
		strings.HasPrefix(name, "$SA_"), strings.HasPrefix(name, "$SNS_"),
		strings.HasPrefix(name, "$SCS_"), strings.HasPrefix(name, "$ON_"):
		return sectionNC
	case numericDriveIdentifier(name):
		return sectionDrive
	case (name == "P" || name == "R") && len(identifier.Segments[0].Parts) > 1:
		if _, dynamic := identifier.Segments[0].Parts[1].(parser.ReplacementIdentifier); dynamic {
			return sectionDrive
		}
	}
	return 0
}

func numericDriveIdentifier(identifier string) bool {
	if len(identifier) < 2 || (identifier[0] != 'P' && identifier[0] != 'R') {
		return false
	}
	for _, character := range identifier[1:] {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}
