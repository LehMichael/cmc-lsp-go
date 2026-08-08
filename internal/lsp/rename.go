package lsp

import (
	"slices"
	"strings"
	"unicode"
)

func (server *Lsp) handlePrepareRename(message requestMessage) {
	var params textDocumentPositionParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	occurrence, definition := server.renameTarget(params.TextDocument.URI, params.Position)
	if occurrence == nil || definition == nil {
		server.respond(message.ID, nil, nil)
		return
	}
	server.respond(message.ID, map[string]any{"range": occurrence.Range, "placeholder": occurrence.Name}, nil)
}

func (server *Lsp) handleRename(message requestMessage) {
	var params renameParams
	if err := decodeParams(message.Params, &params); err != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: err.Error()})
		return
	}
	if !isSimpleIdentifier(params.NewName) {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: "the new callable name is not a valid static CMC identifier"})
		return
	}
	_, definition := server.renameTarget(params.TextDocument.URI, params.Position)
	if definition == nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: "only unambiguous user-defined functions and procedures can be renamed safely"})
		return
	}
	if !strings.EqualFold(params.NewName, definition.Name) && server.callableDefinitionAt(params.TextDocument.URI, params.NewName) != nil {
		server.respond(message.ID, nil, &responseError{Code: InvalidParams, Message: "a function or procedure with the new name already exists in this project"})
		return
	}

	changes := make(map[string][]textEdit)
	for _, occurrence := range server.scopedOccurrences(params.TextDocument.URI) {
		if occurrence.Kind != 12 || !strings.EqualFold(occurrence.Name, definition.Name) {
			continue
		}
		changes[occurrence.URI] = append(changes[occurrence.URI], textEdit{Range: occurrence.Range, NewText: params.NewName})
	}
	for uri := range changes {
		slices.SortFunc(changes[uri], func(left, right textEdit) int {
			if positionLess(left.Range.Start, right.Range.Start) {
				return -1
			}
			if positionLess(right.Range.Start, left.Range.Start) {
				return 1
			}
			return 0
		})
	}
	server.respond(message.ID, workspaceEdit{Changes: changes}, nil)
}

func (server *Lsp) renameTarget(uri string, target position) (*symbolOccurrence, *symbolOccurrence) {
	text, err := server.overlay.ReadURI(uri)
	if err != nil {
		return nil, nil
	}
	name := symbolAt(text, target)
	if !isSimpleIdentifier(name) {
		return nil, nil
	}
	var selected *symbolOccurrence
	var definition *symbolOccurrence
	definitions := 0
	for _, occurrence := range server.scopedOccurrences(uri) {
		if occurrence.Kind != 12 || !strings.EqualFold(occurrence.Name, name) {
			continue
		}
		if occurrence.URI == uri && positionInRange(target, occurrence.Range) {
			copy := occurrence
			selected = &copy
		}
		if occurrence.Definition && occurrence.CallableKind != "" {
			copy := occurrence
			definition = &copy
			definitions++
		}
	}
	if selected == nil || definitions != 1 {
		return nil, nil
	}
	return selected, definition
}

func isSimpleIdentifier(name string) bool {
	runes := []rune(name)
	if len(runes) == 0 || !(unicode.IsLetter(runes[0]) || runes[0] == '_') {
		return false
	}
	for _, value := range runes[1:] {
		if !unicode.IsLetter(value) && !unicode.IsNumber(value) && value != '_' {
			return false
		}
	}
	return true
}
