package lsp

import (
	"bytes"
	"encoding/json"
	"errors"
)

type Message struct {
	Jsonrpc string `json:"jsonrpc,omitempty"`
}

type LspMessageKind interface{ isLspMessageKind() }

type lspMessage struct{ message LspMessageKind }

func (message *lspMessage) UnmarshalJSON(data []byte) error {
	var envelope struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return err
	}
	if envelope.Method != "" {
		if len(envelope.ID) == 0 || bytes.Equal(bytes.TrimSpace(envelope.ID), []byte("null")) {
			var notification notificationMessage
			if err := json.Unmarshal(data, &notification); err != nil {
				return err
			}
			message.message = notification
			return nil
		}
		var request requestMessage
		if err := json.Unmarshal(data, &request); err != nil {
			return err
		}
		message.message = request
		return nil
	}
	var response responseMessage
	if err := json.Unmarshal(data, &response); err != nil {
		return err
	}
	message.message = response
	return nil
}

type messageID interface{ isMessageID() }

type intMessageID int

func (intMessageID) isMessageID() {}

type stringMessageID string

func (stringMessageID) isMessageID() {}

func parseMessageID(raw json.RawMessage) (messageID, error) {
	if len(raw) == 0 {
		return nil, errors.New("missing message id")
	}
	if raw[0] == '"' {
		var value string
		if err := json.Unmarshal(raw, &value); err != nil {
			return nil, err
		}
		return stringMessageID(value), nil
	}
	var value int
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	return intMessageID(value), nil
}

type requestMessage struct {
	Message
	ID     messageID       `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

func (requestMessage) isLspMessageKind() {}

func (message *requestMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Message
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	id, err := parseMessageID(raw.ID)
	if err != nil {
		return err
	}
	message.Message = raw.Message
	message.ID = id
	message.Method = raw.Method
	message.Params = raw.Params
	return nil
}

type notificationMessage struct {
	Message
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

func (notificationMessage) isLspMessageKind() {}

type responseMessage struct {
	Message
	ID     messageID      `json:"id"`
	Result any            `json:"result"`
	Error  *responseError `json:"error,omitempty"`
}

func (responseMessage) isLspMessageKind() {}

func (message responseMessage) MarshalJSON() ([]byte, error) {
	if message.Error != nil {
		return json.Marshal(struct {
			Message
			ID    messageID      `json:"id"`
			Error *responseError `json:"error"`
		}{Message: message.Message, ID: message.ID, Error: message.Error})
	}
	return json.Marshal(struct {
		Message
		ID     messageID `json:"id"`
		Result any       `json:"result"`
	}{Message: message.Message, ID: message.ID, Result: message.Result})
}

func (message *responseMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Message
		ID     json.RawMessage `json:"id"`
		Result json.RawMessage `json:"result"`
		Error  *responseError  `json:"error"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	id, err := parseMessageID(raw.ID)
	if err != nil {
		return err
	}
	message.Message = raw.Message
	message.ID = id
	if len(raw.Result) == 0 || bytes.Equal(bytes.TrimSpace(raw.Result), []byte("null")) {
		message.Result = nil
	} else {
		message.Result = raw.Result
	}
	message.Error = raw.Error
	return nil
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

const (
	ParseError           = -32700
	InvalidRequest       = -32600
	MethodNotFound       = -32601
	InvalidParams        = -32602
	InternalError        = -32603
	ServerNotInitialized = -32002
)

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type lspRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type textDocumentIdentifier struct {
	URI string `json:"uri"`
}

type versionedTextDocumentIdentifier struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type textDocumentItem struct {
	URI        string `json:"uri"`
	LanguageID string `json:"languageId"`
	Version    int    `json:"version"`
	Text       string `json:"text"`
}

type initializeParams struct {
	RootURI               *string `json:"rootUri,omitempty"`
	RootPath              *string `json:"rootPath,omitempty"`
	Locale                string  `json:"locale,omitempty"`
	Capabilities          any     `json:"capabilities,omitempty"`
	InitializationOptions struct {
		CMCDatabasePath string `json:"cmcDatabasePath,omitempty"`
	} `json:"initializationOptions,omitempty"`
	WorkspaceFolders []struct {
		URI  string `json:"uri"`
		Name string `json:"name"`
	} `json:"workspaceFolders,omitempty"`
}

type didOpenParams struct {
	TextDocument textDocumentItem `json:"textDocument"`
}

type didChangeParams struct {
	TextDocument   versionedTextDocumentIdentifier `json:"textDocument"`
	ContentChanges []struct {
		Text string `json:"text"`
	} `json:"contentChanges"`
}

type didCloseParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type documentParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
}

type formattingParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Options      struct {
		TabSize          int  `json:"tabSize"`
		InsertSpaces     bool `json:"insertSpaces"`
		CMCCommentSpaces int  `json:"cmcCommentSpaces,omitempty"`
	} `json:"options"`
}

type semanticTokens struct {
	Data []uint32 `json:"data"`
}

type textDocumentPositionParams struct {
	TextDocument textDocumentIdentifier `json:"textDocument"`
	Position     position               `json:"position"`
}

type diagnostic struct {
	Range    lspRange `json:"range"`
	Severity int      `json:"severity,omitempty"`
	Code     string   `json:"code,omitempty"`
	Source   string   `json:"source,omitempty"`
	Message  string   `json:"message"`
}

type textEdit struct {
	Range   lspRange `json:"range"`
	NewText string   `json:"newText"`
}

type location struct {
	URI   string   `json:"uri"`
	Range lspRange `json:"range"`
}

type workspaceSymbolParams struct {
	Query string `json:"query"`
}

type symbolInformation struct {
	Name          string   `json:"name"`
	Kind          int      `json:"kind"`
	Location      location `json:"location"`
	ContainerName string   `json:"containerName,omitempty"`
}

type documentSymbol struct {
	Name           string           `json:"name"`
	Detail         string           `json:"detail,omitempty"`
	Kind           int              `json:"kind"`
	Range          lspRange         `json:"range"`
	SelectionRange lspRange         `json:"selectionRange"`
	Children       []documentSymbol `json:"children,omitempty"`
}

type completionItem struct {
	Label         string         `json:"label"`
	Kind          int            `json:"kind,omitempty"`
	Detail        string         `json:"detail,omitempty"`
	Documentation *markupContent `json:"documentation,omitempty"`
	InsertText    string         `json:"insertText,omitempty"`
}

type markupContent struct {
	Kind  string `json:"kind"`
	Value string `json:"value"`
}

type signatureHelp struct {
	Signatures      []signatureInformation `json:"signatures"`
	ActiveSignature int                    `json:"activeSignature,omitempty"`
	ActiveParameter int                    `json:"activeParameter,omitempty"`
}

type signatureInformation struct {
	Label         string                 `json:"label"`
	Documentation *markupContent         `json:"documentation,omitempty"`
	Parameters    []parameterInformation `json:"parameters,omitempty"`
}

type parameterInformation struct {
	Label         string         `json:"label"`
	Documentation *markupContent `json:"documentation,omitempty"`
}
