package lsp

import (
	"encoding/json"
	"errors"
	"fmt"
)

type message struct {
	Jsonrpc string `json:"jsonrpc"`
}

type responseMessage struct {
	message
	/**
	 * The request id.
	 */
	ID messageID `json:"id,omitempty"`

	/**
	 * The result of a request. This member is REQUIRED on success.
	 * This member MUST NOT exist if there was an error invoking the method.
	 */
	Result lSPAny `json:"result,omitempty"`

	/**
	 * The error object in case a request fails.
	 */
	Error *responseError `json:"error,omitempty"`
}

type requestMessage struct {
	message
	ID     messageID `json:"id"`
	Method string    `json:"method"`
	Params params    `json:"params,omitempty"`
}

type messageID interface {
	isMessageID()
}

type intMessageID int

func (intMessageID) isMessageID() {}

type stringMessageID string

func (stringMessageID) isMessageID() {}

func (msg *requestMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		message
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	if len(raw.ID) > 0 {
		switch raw.ID[0] {
		case '"':
			var s string
			if err := json.Unmarshal(raw.ID, &s); err != nil {
				return err
			}
			msg.ID = stringMessageID(s)
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '-':
			var i int
			if err := json.Unmarshal(raw.ID, &i); err != nil {
				return err
			}
			msg.ID = intMessageID(i)
		case 'n':
			msg.ID = nil
		default:
			return fmt.Errorf("expected string or number, got: %s", raw.ID)
		}
	} else {
		msg.ID = nil
	}

	msg.Jsonrpc = raw.Jsonrpc
	msg.Method = raw.Method
	msg.Params = nil

	switch msg.Method {
	case "initialize":
		var i initializeParams
		if err := json.Unmarshal(raw.Params, &i); err != nil {
			return err
		}
		msg.Params = i
	case "shutdown":
	case "exit":
	default:
		return errors.New("unknown method")
	}

	return nil
}

type params interface {
	isParams()
}

type workProgressParams struct {
	workDoneToken progressToken
}

func (workProgressParams) isParams() {}

type progressToken interface {
	isProgressToken()
}

type stringProgressToken string

func (stringProgressToken) isProgressToken() {}

type intProgressToken int

func (intProgressToken) isProgressToken() {}

type (
	documentURI string
	uRI         string
)

type initializeParams struct {
	workProgressParams

	/**
	 * The process Id of the parent process that started the server. Is null if
	 * the process has not been started by another process. If the parent
	 * process is not alive then the server should exit (see exit notification)
	 * its process.
	 */
	ProcessID *int `json:"processId,omitempty"`

	/**
	 * Information about the client
	 *
	 * @since 3.15.0
	 */
	ClientInfo struct {
		/**
		 * The name of the client as defined by the client.
		 */
		Name string `json:"name"`

		/**
		 * The client's version as defined by the client.
		 */
		Version *string `json:"version,omitempty"`
	} `json:"clientInfo"`

	/**
	 * The locale the client is currently showing the user interface
	 * in. This must not necessarily be the locale of the operating
	 * system.
	 *
	 * Uses IETF language tags as the value's syntax
	 * (See https://en.wikipedia.org/wiki/IETF_language_tag)
	 *
	 * @since 3.16.0
	 */
	Locale *string `json:"locale,omitempty"`

	/**
	 * The rootPath of the workspace. Is null
	 * if no folder is open.
	 *
	 * @deprecated in favour of `rootUri`.
	 */
	RootPath *string `json:"RootPath,omitempty"`

	/**
	 * The rootUri of the workspace. Is null if no
	 * folder is open. If both `rootPath` and `rootUri` are set
	 * `rootUri` wins.
	 *
	 * @deprecated in favour of `workspaceFolders`
	 */
	RootURI *documentURI `json:"RootURI,omitempty"`

	/**
	 * User provided initialization options.
	 */
	InitializationOptions *string `json:"InitializationOptions,omitempty"`

	/**
	 * The capabilities provided by the client (editor or tool)
	 */
	Capabilities clientCapabilities `json:"capabilities"`

	/**
	 * The initial trace setting. If omitted trace is disabled ('off').
	 */
	Trace *traceValue `json:"trace,omitempty"`

	/**
	 * The workspace folders configured in the client when the server starts.
	 * This property is only available if the client supports workspace folders.
	 * It can be `null` if the client supports workspace folders but none are
	 * configured.
	 *
	 * @since 3.6.0
	 */
	WorkspaceFolders []workspaceFolder `json:"workspaceFolders,omitempty"`
}

type traceValue string

const (
	TraceValueOff      traceValue = "off"
	TraceValueMessages traceValue = "messages"
	TraceValueVerbose  traceValue = "verbose"
)

type workspaceFolder struct {
	/**
	 * The associated URI for this workspace folder.
	 */
	URI uRI `json:"uri"`

	/**
	 * The name of the workspace folder. Used to refer to this
	 * workspace folder in the user interface.
	 */
	Name string `json:"name"`
}

type clientCapabilities struct {
	/**
	 * Workspace specific client capabilities.
	 */
	Workspace struct {
		/**
		 * The client supports applying batch edits
		 * to the workspace by supporting the request
		 * 'workspace/applyEdit'
		 */
		ApplyEdit *bool `json:"ApplyEdit,omitempty"`

		/**
		 * Capabilities specific to `WorkspaceEdit`s
		 */
		WorkspaceEdit *workspaceEditClientCapabilities `json:"workspaceEdit,omitempty"`

		/**
		 * Capabilities specific to the `workspace/didChangeConfiguration`
		 * notification.
		 */
		DidChangeConfiguration *didChangeConfigurationClientCapabilities `json:"didChangeConfiguration,omitempty"`

		/**
		 * Capabilities specific to the `workspace/didChangeWatchedFiles`
		 * notification.
		 */
		DidChangeWatchedFiles *didChangeWatchedFilesClientCapabilities `json:"didChangeWatchedFiles,omitempty"`

		/**
		 * Capabilities specific to the `workspace/symbol` request.
		 */
		Symbol *workspaceSymbolClientCapabilities `json:"symbol,omitempty"`

		/**
		 * Capabilities specific to the `workspace/executeCommand` request.
		 */
		ExecuteCommand *executeCommandClientCapabilities `json:"executeCommand,omitempty"`

		/**
		 * The client has support for workspace folders.
		 *
		 * @since 3.6.0
		 */
		WorkspaceFolders *bool `json:"workspaceFolders,omitempty"`

		/**
		 * The client supports `workspace/configuration` requests.
		 *
		 * @since 3.6.0
		 */
		Configuration *bool `json:"configuration,omitempty"`

		/**
		 * Capabilities specific to the semantic token requests scoped to the
		 * workspace.
		 *
		 * @since 3.16.0
		 */
		SemanticTokens *semanticTokensWorkspaceClientCapabilities `json:"semanticTokens,omitempty"`

		/**
		 * Capabilities specific to the code lens requests scoped to the
		 * workspace.
		 *
		 * @since 3.16.0
		 */
		CodeLens *codeLensWorkspaceClientCapabilities `json:"codeLens,omitempty"`

		/**
		 * The client has support for file requests/notifications.
		 *
		 * @since 3.16.0
		 */
		FileOperations *struct {
			/**
			 * Whether the client supports dynamic registration for file
			 * requests/notifications.
			 */
			DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

			/**
			 * The client has support for sending didCreateFiles notifications.
			 */
			DidCreate *bool `json:"didCreate,omitempty"`

			/**
			 * The client has support for sending willCreateFiles requests.
			 */
			WillCreate *bool `json:"willCreate,omitempty"`

			/**
			 * The client has support for sending didRenameFiles notifications.
			 */
			DidRename *bool `json:"didRename,omitempty"`

			/**
			 * The client has support for sending willRenameFiles requests.
			 */
			WillRename *bool `json:"willRename,omitempty"`

			/**
			 * The client has support for sending didDeleteFiles notifications.
			 */
			DidDelete *bool `json:"didDelete,omitempty"`

			/**
			 * The client has support for sending willDeleteFiles requests.
			 */
			WillDelete *bool `json:"willDelete,omitempty"`
		} `json:"fileOperations,omitempty"`

		/**
		 * Client workspace capabilities specific to inline values.
		 *
		 * @since 3.17.0
		 */
		InlineValue *inlineValueWorkspaceClientCapabilities `json:"inlineValue,omitempty"`

		/**
		 * Client workspace capabilities specific to inlay hints.
		 *
		 * @since 3.17.0
		 */
		InlayHint *inlayHintWorkspaceClientCapabilities `json:"inlayHint,omitempty"`

		/**
		 * Client workspace capabilities specific to diagnostics.
		 *
		 * @since 3.17.0.
		 */
		Diagnostics *diagnosticWorkspaceClientCapabilities `json:"diagnostics,omitempty"`
	} `json:"workspace"`

	/**
	 * Text document specific client capabilities.
	 */
	TextDocument *textDocumentClientCapabilities `json:"textDocument,omitempty"`

	/**
	 * Capabilities specific to the notebook document support.
	 *
	 * @since 3.17.0
	 */
	NotebookDocument *notebookDocumentClientCapabilities `json:"notebookDocument,omitempty"`

	/**
	 * Window specific client capabilities.
	 */
	Window *struct {
		/**
		 * It indicates whether the client supports server initiated
		 * progress using the `window/workDoneProgress/create` request.
		 *
		 * The capability also controls Whether client supports handling
		 * of progress notifications. If set servers are allowed to report a
		 * `workDoneProgress` property in the request specific server
		 * capabilities.
		 *
		 * @since 3.15.0
		 */
		WorkDoneProgress *bool `json:"workDoneProgress,omitempty"`

		/**
		 * Capabilities specific to the showMessage request
		 *
		 * @since 3.16.0
		 */
		ShowMessage *showMessageRequestClientCapabilities `json:"showMessage,omitempty"`

		/**
		 * Client capabilities for the show document request.
		 *
		 * @since 3.16.0
		 */
		ShowDocument *showDocumentClientCapabilities `json:"showDocument,omitempty"`
	} `json:"window,omitempty"`

	/**
	 * General client capabilities.
	 *
	 * @since 3.16.0
	 */
	General *struct {
		/**
		 * Client capability that signals how the client
		 * handles stale requests (e.g. a request
		 * for which the client will not process the response
		 * anymore since the information is outdated).
		 *
		 * @since 3.17.0
		 */
		StaleRequestSupport *struct {
			/**
			 * The client will actively cancel the request.
			 */
			Cancel *bool `json:"cancel,omitempty"`

			/**
			 * The list of requests for which the client
			 * will retry the request if it receives a
			 * response with error code `ContentModified``
			 */
			RetryOnContentModified []string `json:"retryOnContentModified,omitempty"`
		} `json:"staleRequestSupport,omitempty"`

		/**
		 * Client capabilities specific to regular expressions.
		 *
		 * @since 3.16.0
		 */
		RegularExpressions *regularExpressionsClientCapabilities `json:"regularExpressions,omitempty"`

		/**
		 * Client capabilities specific to the client's markdown parser.
		 *
		 * @since 3.16.0
		 */
		Markdown *markdownClientCapabilities `json:"markdown,omitempty"`

		/**
		 * The position encodings supported by the client. Client and server
		 * have to agree on the same position encoding to ensure that offsets
		 * (e.g. character position in a line) are interpreted the same on both
		 * side.
		 *
		 * To keep the protocol backwards compatible the following applies: if
		 * the value 'utf-16' is missing from the array of position encodings
		 * servers can assume that the client supports UTF-16. UTF-16 is
		 * therefore a mandatory encoding.
		 *
		 * If omitted it defaults to ['utf-16'].
		 *
		 * Implementation considerations: since the conversion from one encoding
		 * into another requires the content of the file / line the conversion
		 * is best done where the file is read which is usually on the server
		 * side.
		 *
		 * @since 3.17.0
		 */
		PositionEncodings []positionEncodingKind `json:"positionEncodings,omitempty"`
	} `json:"general,omitempty"`

	/**
	 * Experimental client capabilities.
	 */
	Experimental lSPAny `json:"experimental,omitempty"`
}

type lSPAny any

type workspaceEditClientCapabilities struct {
	/**
	 * The client supports versioned document changes in `WorkspaceEdit`s
	 */
	DocumentChanges *bool `json:"documentChanges,omitempty"`

	/**
	 * The resource operations the client supports. Clients should at least
	 * support 'create', 'rename' and 'delete' files and folders.
	 *
	 * @since 3.13.0
	 */
	ResourceOperations []resourceOperationKind `json:"resourceOperations,omitempty"`

	/**
	 * The failure handling strategy of a client if applying the workspace edit
	 * fails.
	 *
	 * @since 3.13.0
	 */
	FailureHandling *failureHandlingKind `json:"failureHandling,omitempty"`

	/**
	 * Whether the client normalizes line endings to the client specific
	 * setting.
	 * If set to `true` the client will normalize line ending characters
	 * in a workspace edit to the client specific new line character(s).
	 *
	 * @since 3.16.0
	 */
	NormalizesLineEndings *bool `json:"normalizesLineEndings,omitempty"`

	/**
	 * Whether the client in general supports change annotations on text edits,
	 * create file, rename file and delete file changes.
	 *
	 * @since 3.16.0
	 */
	ChangeAnnotationSupport *struct {
		/**
		 * Whether the client groups edits with equal labels into tree nodes,
		 * for instance all edits labelled with "Changes in Strings" would
		 * be a tree node.
		 */
		GroupsOnLabel *bool `json:"groupsOnLabel,omitempty"`
	} `json:"changeAnnotationSupport,omitempty"`
}

type didChangeConfigurationClientCapabilities struct {
	/**
	 * Did change configuration notification supports dynamic registration.
	 *
	 * @since 3.6.0 to support the new pull model.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type didChangeWatchedFilesClientCapabilities struct {
	/**
	 * Did change watched files notification supports dynamic registration.
	 * Please note that the current protocol doesn't support static
	 * configuration for file changes from the server side.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Whether the client has support for relative patterns
	 * or not.
	 *
	 * @since 3.17.0
	 */
	RelativePatternSupport *bool `json:"relativePatternSupport,omitempty"`
}

type workspaceSymbolClientCapabilities struct {
	/**
	 * Symbol request supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Specific capabilities for the `SymbolKind` in the `workspace/symbol`
	 * request.
	 */
	SymbolKind *struct {
		/**
		 * The symbol kind values the client supports. When this
		 * property exists the client also guarantees that it will
		 * handle values outside its set gracefully and falls back
		 * to a default value when unknown.
		 *
		 * If this property is not present the client only supports
		 * the symbol kinds from `File` to `Array` as defined in
		 * the initial version of the protocol.
		 */
		ValueSet []symbolKind `json:"valueSet,omitempty"`
	} `json:"symbolKind,omitempty"`

	/**
	 * The client supports tags on `SymbolInformation` and `WorkspaceSymbol`.
	 * Clients supporting tags have to handle unknown tags gracefully.
	 *
	 * @since 3.16.0
	 */
	TagSupport *struct {
		/**
		 * The tags supported by the client.
		 */
		ValueSet []symbolTag `json:"valueSet,omitempty"`
	} `json:"tagSupport,omitempty"`

	/**
	 * The client support partial workspace symbols. The client will send the
	 * request `workspaceSymbol/resolve` to the server to resolve additional
	 * properties.
	 *
	 * @since 3.17.0
	 */
	ResolveSupport *struct {
		/**
		 * The properties that a client can resolve lazily. Usually
		 * `location.range`
		 */
		Properties []string `json:"properties,omitempty"`
	} `json:"resolveSupport,omitempty"`
}

type executeCommandClientCapabilities struct {
	/**
	 * Execute command supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type semanticTokensWorkspaceClientCapabilities struct {
	/**
	 * Whether the client implementation supports a refresh request sent from
	 * the server to the client.
	 *
	 * Note that this event is global and will force the client to refresh all
	 * semantic tokens currently shown. It should be used with absolute care
	 * and is useful for situation where a server for example detect a project
	 * wide change that requires such a calculation.
	 */
	RefreshSupport *bool `json:"refreshSupport,omitempty"`
}

type codeLensWorkspaceClientCapabilities struct {
	/**
	 * Whether the client implementation supports a refresh request sent from the
	 * server to the client.
	 *
	 * Note that this event is global and will force the client to refresh all
	 * code lenses currently shown. It should be used with absolute care and is
	 * useful for situation where a server for example detect a project wide
	 * change that requires such a calculation.
	 */
	RefreshSupport *bool `json:"refreshSupport,omitempty"`
}

// inlineValueWorkspaceClientCapabilities Client workspace capabilities specific to inline values.
// since 3.17.0
type inlineValueWorkspaceClientCapabilities struct {
	/**
	 * Whether the client implementation supports a refresh request sent from
	 * the server to the client.
	 *
	 * Note that this event is global and will force the client to refresh all
	 * inline values currently shown. It should be used with absolute care and
	 * is useful for situation where a server for example detect a project wide
	 * change that requires such a calculation.
	 */
	RefreshSupport *bool `json:"refreshSupport,omitempty"`
}

// inlayHintWorkspaceClientCapabilities Client workspace capabilities specific to inlay hints.
// since 3.17.0
type inlayHintWorkspaceClientCapabilities struct {
	/**
	 * Whether the client implementation supports a refresh request sent from
	 * the server to the client.
	 *
	 * Note that this event is global and will force the client to refresh all
	 * inlay hints currently shown. It should be used with absolute care and
	 * is useful for situation where a server for example detects a project wide
	 * change that requires such a calculation.
	 */
	RefreshSupport *bool `json:"refreshSupport,omitempty"`
}

// diagnosticWorkspaceClientCapabilities Workspace client capabilities specific to diagnostic pull requests.
// since 3.17.0
type diagnosticWorkspaceClientCapabilities struct {
	/**
	 * Whether the client implementation supports a refresh request sent from
	 * the server to the client.
	 *
	 * Note that this event is global and will force the client to refresh all
	 * pulled diagnostics currently shown. It should be used with absolute care
	 * and is useful for situation where a server for example detects a project
	 * wide change that requires such a calculation.
	 */
	RefreshSupport *bool `json:"refreshSupport,omitempty"`
}

// textDocumentClientCapabilities Text document specific client capabilities.
type textDocumentClientCapabilities struct {
	Synchronization *textDocumentSyncClientCapabilities `json:"synchronization,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/completion` request.
	 */
	Completion *completionClientCapabilities `json:"completion,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/hover` request.
	 */
	Hover *hoverClientCapabilities `json:"hover,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/signatureHelp` request.
	 */
	SignatureHelp *signatureHelpClientCapabilities `json:"signatureHelp,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/declaration` request.
	 *
	 * @since 3.14.0
	 */
	Declaration *declarationClientCapabilities `json:"declaration,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/definition` request.
	 */
	Definition *definitionClientCapabilities `json:"definition,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/typeDefinition` request.
	 *
	 * @since 3.6.0
	 */
	TypeDefinition *typeDefinitionClientCapabilities `json:"typeDefinition,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/implementation` request.
	 *
	 * @since 3.6.0
	 */
	Implementation *implementationClientCapabilities `json:"implementation,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/references` request.
	 */
	References *referenceClientCapabilities `json:"references,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentHighlight` request.
	 */
	DocumentHighlight *documentHighlightClientCapabilities `json:"documentHighlight,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentSymbol` request.
	 */
	DocumentSymbol *documentSymbolClientCapabilities `json:"documentSymbol,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/codeAction` request.
	 */
	CodeAction *codeActionClientCapabilities `json:"codeAction,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/codeLens` request.
	 */
	CodeLens *codeLensClientCapabilities `json:"codeLens,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentLink` request.
	 */
	DocumentLink *documentLinkClientCapabilities `json:"documentLink,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentColor` and the
	 * `textDocument/colorPresentation` request.
	 *
	 * @since 3.6.0
	 */
	ColorProvider *documentColorClientCapabilities `json:"colorProvider,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/formatting` request.
	 */
	Formatting *documentFormattingClientCapabilities `json:"formatting,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/rangeFormatting` request.
	 */
	RangeFormatting *documentRangeFormattingClientCapabilities `json:"rangeFormatting,omitempty"`

	/** request.
	 * Capabilities specific to the `textDocument/onTypeFormatting` request.
	 */
	OnTypeFormatting *documentOnTypeFormattingClientCapabilities `json:"onTypeFormatting,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/rename` request.
	 */
	Rename *renameClientCapabilities `json:"rename,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/publishDiagnostics`
	 * notification.
	 */
	PublishDiagnostics *publishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/foldingRange` request.
	 *
	 * @since 3.10.0
	 */
	FoldingRange *foldingRangeClientCapabilities `json:"foldingRange,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/selectionRange` request.
	 *
	 * @since 3.15.0
	 */
	SelectionRange *selectionRangeClientCapabilities `json:"selectionRange,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/linkedEditingRange` request.
	 *
	 * @since 3.16.0
	 */
	LinkedEditingRange *linkedEditingRangeClientCapabilities `json:"linkedEditingRange,omitempty"`

	/**
	 * Capabilities specific to the various call hierarchy requests.
	 *
	 * @since 3.16.0
	 */
	CallHierarchy *callHierarchyClientCapabilities `json:"callHierarchy,omitempty"`

	/**
	 * Capabilities specific to the various semantic token requests.
	 *
	 * @since 3.16.0
	 */
	SemanticTokens *semanticTokensClientCapabilities `json:"semanticTokens,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/moniker` request.
	 *
	 * @since 3.16.0
	 */
	Moniker *monikerClientCapabilities `json:"moniker,omitempty"`

	/**
	 * Capabilities specific to the various type hierarchy requests.
	 *
	 * @since 3.17.0
	 */
	TypeHierarchy *typeHierarchyClientCapabilities `json:"typeHierarchy,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/inlineValue` request.
	 *
	 * @since 3.17.0
	 */
	InlineValue *inlineValueClientCapabilities `json:"inlineValue,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/inlayHint` request.
	 *
	 * @since 3.17.0
	 */
	InlayHint *inlayHintClientCapabilities `json:"inlayHint,omitempty"`

	/**
	 * Capabilities specific to the diagnostic pull model.
	 *
	 * @since 3.17.0
	 */
	Diagnostic *diagnosticClientCapabilities `json:"diagnostic,omitempty"`
}

// notebookDocumentClientCapabilities Capabilities specific to the notebook document support.
// since 3.17.0
type notebookDocumentClientCapabilities struct {
	/**
	 * Capabilities specific to notebook document synchronization
	 *
	 * @since 3.17.0
	 */
	Synchronization *notebookDocumentSyncClientCapabilities `json:"synchronization,omitempty"`
}

// notebookDocumentSyncClientCapabilities Notebook specific client capabilities.
// since 3.17.0
type notebookDocumentSyncClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is
	 * set to `true` the client supports the new
	 * `(NotebookDocumentSyncRegistrationOptions & NotebookDocumentSyncOptions)`
	 * return value for the corresponding server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports sending execution summary data per cell.
	 */
	ExecutionSummarySupport *bool `json:"executionSummarySupport,omitempty"`
}

// showMessageRequestClientCapabilities Show message request client capabilities
type showMessageRequestClientCapabilities struct {
	/**
	 * Capabilities specific to the `MessageActionItem` type.
	 */
	MessageActionItem *struct {
		/**
		 * Whether the client supports additional attributes which
		 * are preserved and sent back to the server in the
		 * request's response.
		 */
		AdditionalPropertiesSupport *bool `json:"additionalPropertiesSupport,omitempty"`
	} `json:"messageActionItem,omitempty"`
}

// showDocumentClientCapabilities Client capabilities for the show document request.
// since 3.16.0
type showDocumentClientCapabilities struct {
	/**
	 * The client has support for the show document
	 * request.
	 */
	Support *bool `json:"support,omitempty"`
}

// regularExpressionsClientCapabilities Client capabilities specific to regular expressions.
type regularExpressionsClientCapabilities struct {
	/**
	 * The engine's name.
	 */
	Engine string `json:"engine"`

	/**
	 * The engine's version.
	 */
	Version *string `json:"Version,omitempty"`
}

// markdownClientCapabilities Client capabilities specific to the used markdown parser.
// since 3.16.0
type markdownClientCapabilities struct {
	/**
	 * The name of the parser.
	 */
	Parser string `json:"parser"`

	/**
	 * The version of the parser.
	 */
	Version *string `json:"version,omitempty"`

	/**
	 * A list of HTML tags that the client allows / supports in
	 * Markdown.
	 *
	 * @since 3.17.0
	 */
	AllowedTags []string `json:"allowedTags,omitempty"`
}

// positionEncodingKind A type indicating how positions are encoded, specifically what column offsets mean.
// @since 3.17.0
type positionEncodingKind string

// A set of predefined position encoding kinds.
// since 3.17.0
const (
	/**
	 * Character offsets count UTF-8 code units (e.g bytes).
	 */
	PositionEncodingKindUTF8 string = "utf-8"

	/**
	 * Character offsets count UTF-16 code units.
	 *
	 * This is the default and must always be supported
	 * by servers
	 */
	PositionEncodingKindUTF16 string = "utf-16"

	/**
	 * Character offsets count UTF-32 code units.
	 *
	 * Implementation note: these are the same as Unicode code points,
	 * so this `PositionEncodingKind` may also be used for an
	 * encoding-agnostic representation of character offsets.
	 */
	PositionEncodingKindUTF32 string = "utf-32"
)

// resourceOperationKind The kind of resource operations supported by the client.
type resourceOperationKind string

const (
	/**
	 * Supports creating new files and folders.
	 */
	ResourceOperationKindCreate resourceOperationKind = "create"
	/**
	 * Supports renaming existing files and folders.
	 */
	ResourceOperationKindRename = "rename"
	/**
	 * Supports deleting existing files and folders.
	 */
	ResourceOperationKindDelete = "delete"
)

type failureHandlingKind string

const (
	/**
	 * Applying the workspace change is simply aborted if one of the changes
	 * provided fails. All operations executed before the failing operation
	 * stay executed.
	 */
	FailureHandlingKindAbort failureHandlingKind = "abort"
	/**
	 * All operations are executed transactional. That means they either all
	 * succeed or no changes at all are applied to the workspace.
	 */
	FailureHandlingKindTransactional failureHandlingKind = "transactional"
	/**
	 * If the workspace edit contains only textual file changes they are
	 * executed transactional. If resource changes (create, rename or delete
	 * file) are part of the change the failure handling strategy is abort.
	 */
	FailureHandlingKindTextOnlyTransactional failureHandlingKind = "textOnlyTransactional"
	/**
	 * The client tries to undo the operations already executed. But there is no
	 * guarantee that this is succeeding.
	 */
	FailureHandlingKindUndo failureHandlingKind = "undo"
)

type symbolKind int

const (
	SymbolKindFile          symbolKind = 1
	SymbolKindModule        symbolKind = 2
	SymbolKindNamespace     symbolKind = 3
	SymbolKindPackage       symbolKind = 4
	SymbolKindClass         symbolKind = 5
	SymbolKindMethod        symbolKind = 6
	SymbolKindProperty      symbolKind = 7
	SymbolKindField         symbolKind = 8
	SymbolKindConstructor   symbolKind = 9
	SymbolKindEnum          symbolKind = 10
	SymbolKindInterface     symbolKind = 11
	SymbolKindFunction      symbolKind = 12
	SymbolKindVariable      symbolKind = 13
	SymbolKindConstant      symbolKind = 14
	SymbolKindString        symbolKind = 15
	SymbolKindNumber        symbolKind = 16
	SymbolKindBoolean       symbolKind = 17
	SymbolKindArray         symbolKind = 18
	SymbolKindObject        symbolKind = 19
	SymbolKindKey           symbolKind = 20
	SymbolKindNull          symbolKind = 21
	SymbolKindEnumMember    symbolKind = 22
	SymbolKindStruct        symbolKind = 23
	SymbolKindEvent         symbolKind = 24
	SymbolKindOperator      symbolKind = 25
	SymbolKindTypeParameter symbolKind = 26
)

type symbolTag int

const (
	SymbolTagDeprecated symbolTag = 1
)

type textDocumentSyncClientCapabilities struct {
	/**
	 * Whether text document synchronization supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports sending will save notifications.
	 */
	WillSave *bool `json:"willSave,omitempty"`

	/**
	 * The client supports sending a will save request and
	 * waits for a response providing text edits which will
	 * be applied to the document before it is saved.
	 */
	WillSaveWaitUntil *bool `json:"willSaveWaitUntil,omitempty"`

	/**
	 * The client supports did save notifications.
	 */
	DidSave *bool `json:"didSave,omitempty"`
}

type completionClientCapabilities struct {
	/**
	 * Whether completion supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports the following `CompletionItem` specific
	 * capabilities.
	 */
	CompletionItem *struct {
		/**
		 * Client supports snippets as insert text.
		 *
		 * A snippet can define tab stops and placeholders with `$1`, `$2`
		 * and `${3:foo}`. `$0` defines the final tab stop, it defaults to
		 * the end of the snippet. Placeholders with equal identifiers are
		 * linked, that is typing in one will update others too.
		 */
		SnippetSupport *bool `json:"snippetSupport,omitempty"`

		/**
		 * Client supports commit characters on a completion item.
		 */
		CommitCharactersSupport *bool `json:"commitCharactersSupport,omitempty"`

		/**
		 * Client supports the follow content formats for the documentation
		 * property. The order describes the preferred format of the client.
		 */
		DocumentationFormat []markupKind `json:"documentationFormat,omitempty"`

		/**
		 * Client supports the deprecated property on a completion item.
		 */
		DeprecatedSupport *bool `json:"deprecatedSupport,omitempty"`

		/**
		 * Client supports the preselect property on a completion item.
		 */
		PreselectSupport *bool `json:"preselectSupport,omitempty"`

		/**
		 * Client supports the tag property on a completion item. Clients
		 * supporting tags have to handle unknown tags gracefully. Clients
		 * especially need to preserve unknown tags when sending a completion
		 * item back to the server in a resolve call.
		 *
		 * @since 3.15.0
		 */
		TagSupport *struct {
			/**
			 * The tags supported by the client.
			 */
			ValueSet []completionItemTag `json:"valueSet,omitempty"`
		} `json:"tagSupport,omitempty"`

		/**
		 * Client supports insert replace edit to control different behavior if
		 * a completion item is inserted in the text or should replace text.
		 *
		 * @since 3.16.0
		 */
		InsertReplaceSupport *bool `json:"insertReplaceSupport,omitempty"`

		/**
		 * Indicates which properties a client can resolve lazily on a
		 * completion item. Before version 3.16.0 only the predefined properties
		 * `documentation` and `detail` could be resolved lazily.
		 *
		 * @since 3.16.0
		 */
		ResolveSupport *struct {
			/**
			 * The properties that a client can resolve lazily.
			 */
			Properties []string `json:"properties,omitempty"`
		} `json:"resolveSupport,omitempty"`

		/**
		 * The client supports the `insertTextMode` property on
		 * a completion item to override the whitespace handling mode
		 * as defined by the client (see `insertTextMode`).
		 *
		 * @since 3.16.0
		 */
		InsertTextModeSupport *struct {
			ValueSet []insertTextMode `json:"valueSet,omitempty"`
		} `json:"insertTextModeSupport,omitempty"`

		/**
		 * The client has support for completion item label
		 * details (see also `CompletionItemLabelDetails`).
		 *
		 * @since 3.17.0
		 */
		LabelDetailsSupport *bool `json:"labelDetailsSupport,omitempty"`
	} `json:"completionItem,omitempty"`

	CompletionItemKind *struct {
		/**
		 * The completion item kind values the client supports. When this
		 * property exists the client also guarantees that it will
		 * handle values outside its set gracefully and falls back
		 * to a default value when unknown.
		 *
		 * If this property is not present the client only supports
		 * the completion items kinds from `Text` to `Reference` as defined in
		 * the initial version of the protocol.
		 */
		ValueSet []completionItemKind `json:"valueSet,omitempty"`
	} `json:"completionItemKind,omitempty"`

	/**
	 * The client supports to send additional context information for a
	 * `textDocument/completion` request.
	 */
	ContextSupport *bool `json:"contextSupport,omitempty"`

	/**
	 * The client's default when the completion item doesn't provide a
	 * `insertTextMode` property.
	 *
	 * @since 3.17.0
	 */
	InsertTextMode *insertTextMode `json:"insertTextMode,omitempty"`

	/**
	 * The client supports the following `CompletionList` specific
	 * capabilities.
	 *
	 * @since 3.17.0
	 */
	CompletionList *struct {
		/**
		 * The client supports the following itemDefaults on
		 * a completion list.
		 *
		 * The value lists the supported property names of the
		 * `CompletionList.itemDefaults` object. If omitted
		 * no properties are supported.
		 *
		 * @since 3.17.0
		 */
		ItemDefaults []string `json:"itemDefaults,omitempty"`
	} `json:"completionList,omitempty"`
}

type hoverClientCapabilities struct {
	/**
	 * Whether hover supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Client supports the follow content formats if the content
	 * property refers to a `literal of type MarkupContent`.
	 * The order describes the preferred format of the client.
	 */
	ContentFormat []markupKind `json:"contentFormat,omitempty"`
}

type signatureHelpClientCapabilities struct {
	/**
	 * Whether signature help supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports the following `SignatureInformation`
	 * specific properties.
	 */
	SignatureInformation *struct {
		/**
		 * Client supports the follow content formats for the documentation
		 * property. The order describes the preferred format of the client.
		 */
		DocumentationFormat []markupKind `json:"documentationFormat,omitempty"`

		/**
		 * Client capabilities specific to parameter information.
		 */
		ParameterInformation *struct {
			/**
			 * The client supports processing label offsets instead of a
			 * simple label string.
			 *
			 * @since 3.14.0
			 */
			LabelOffsetSupport *bool `json:"labelOffsetSupport,omitempty"`
		} `json:"parameterInformation,omitempty"`

		/**
		 * The client supports the `activeParameter` property on
		 * `SignatureInformation` literal.
		 *
		 * @since 3.16.0
		 */
		activeParameterSupport *bool
	} `json:"signatureInformation,omitempty"`

	/**
	 * The client supports to send additional context information for a
	 * `textDocument/signatureHelp` request. A client that opts into
	 * contextSupport will also support the `retriggerCharacters` on
	 * `SignatureHelpOptions`.
	 *
	 * @since 3.15.0
	 */
	ContextSupport *bool `json:"contextSupport,omitempty"`
}

type declarationClientCapabilities struct {
	/**
	 * Whether declaration supports dynamic registration. If this is set to
	 * `true` the client supports the new `DeclarationRegistrationOptions`
	 * return value for the corresponding server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports additional metadata in the form of declaration links.
	 */
	LinkSupport *bool `json:"linkSupport,omitempty"`
}

type definitionClientCapabilities struct {
	/**
	 * Whether definition supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports additional metadata in the form of definition links.
	 *
	 * @since 3.14.0
	 */
	LinkSupport *bool `json:"linkSupport,omitempty"`
}

type typeDefinitionClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `TypeDefinitionRegistrationOptions`
	 * return value for the corresponding server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports additional metadata in the form of definition links.
	 *
	 * @since 3.14.0
	 */
	LinkSupport *bool `json:"linkSupport,omitempty"`
}

type implementationClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `ImplementationRegistrationOptions`
	 * return value for the corresponding server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports additional metadata in the form of definition links.
	 *
	 * @since 3.14.0
	 */
	LinkSupport *bool `json:"linkSupport,omitempty"`
}

type referenceClientCapabilities struct {
	/**
	 * Whether references supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type documentHighlightClientCapabilities struct {
	/**
	 * Whether document highlight supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type documentSymbolClientCapabilities struct {
	/**
	 * Whether document symbol supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Specific capabilities for the `SymbolKind` in the
	 * `textDocument/documentSymbol` request.
	 */
	SymbolKind *struct {
		/**
		 * The symbol kind values the client supports. When this
		 * property exists the client also guarantees that it will
		 * handle values outside its set gracefully and falls back
		 * to a default value when unknown.
		 *
		 * If this property is not present the client only supports
		 * the symbol kinds from `File` to `Array` as defined in
		 * the initial version of the protocol.
		 */
		ValueSet []symbolKind `json:"valueSet,omitempty"`
	} `json:"symbolKind,omitempty"`

	/**
	 * The client supports hierarchical document symbols.
	 */
	HierarchicalDocumentSymbolSupport *bool `json:"hierarchicalDocumentSymbolSupport,omitempty"`

	/**
	 * The client supports tags on `SymbolInformation`. Tags are supported on
	 * `DocumentSymbol` if `hierarchicalDocumentSymbolSupport` is set to true.
	 * Clients supporting tags have to handle unknown tags gracefully.
	 *
	 * @since 3.16.0
	 */
	TagSupport *struct {
		/**
		 * The tags supported by the client.
		 */
		ValueSet []symbolTag `json:"valueSet,omitempty"`
	} `json:"tagSupport,omitempty"`

	/**
	 * The client supports an additional label presented in the UI when
	 * registering a document symbol provider.
	 *
	 * @since 3.16.0
	 */
	LabelSupport *bool `json:"labelSupport,omitempty"`
}

type codeActionClientCapabilities struct {
	/**
	 * Whether code action supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The client supports code action literals as a valid
	 * response of the `textDocument/codeAction` request.
	 *
	 * @since 3.8.0
	 */
	CodeActionLiteralSupport *struct {
		/**
		 * The code action kind is supported with the following value
		 * set.
		 */
		CodeActionKind struct {
			/**
			 * The code action kind values the client supports. When this
			 * property exists the client also guarantees that it will
			 * handle values outside its set gracefully and falls back
			 * to a default value when unknown.
			 */
			ValueSet []codeActionKind `json:"valueSet,omitempty"`
		} `json:"codeActionKind"`
	} `json:"codeActionLiteralSupport,omitempty"`

	/**
	 * Whether code action supports the `isPreferred` property.
	 *
	 * @since 3.15.0
	 */
	IsPreferredSupport *bool `json:"isPreferredSupport,omitempty"`

	/**
	 * Whether code action supports the `disabled` property.
	 *
	 * @since 3.16.0
	 */
	DisabledSupport *bool `json:"disabledSupport,omitempty"`

	/**
	 * Whether code action supports the `data` property which is
	 * preserved between a `textDocument/codeAction` and a
	 * `codeAction/resolve` request.
	 *
	 * @since 3.16.0
	 */
	DataSupport *bool `json:"dataSupport,omitempty"`

	/**
	 * Whether the client supports resolving additional code action
	 * properties via a separate `codeAction/resolve` request.
	 *
	 * @since 3.16.0
	 */
	ResolveSupport *struct {
		/**
		 * The properties that a client can resolve lazily.
		 */
		Properties []string `json:"properties,omitempty"`
	} `json:"resolveSupport,omitempty"`

	/**
	 * Whether the client honors the change annotations in
	 * text edits and resource operations returned via the
	 * `CodeAction#edit` property by for example presenting
	 * the workspace edit in the user interface and asking
	 * for confirmation.
	 *
	 * @since 3.16.0
	 */
	HonorsChangeAnnotations *bool `json:"honorsChangeAnnotations,omitempty"`
}

type codeLensClientCapabilities struct {
	/**
	 * Whether code lens supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type documentLinkClientCapabilities struct {
	/**
	 * Whether document link supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Whether the client supports the `tooltip` property on `DocumentLink`.
	 *
	 * @since 3.15.0
	 */
	TooltipSupport *bool `json:"tooltipSupport,omitempty"`
}

type documentColorClientCapabilities struct {
	/**
	 * Whether document color supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type documentFormattingClientCapabilities struct {
	/**
	 * Whether formatting supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type documentRangeFormattingClientCapabilities struct {
	/**
	 * Whether formatting supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type documentOnTypeFormattingClientCapabilities struct {
	/**
	 * Whether on type formatting supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type renameClientCapabilities struct {
	/**
	 * Whether rename supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Client supports testing for validity of rename operations
	 * before execution.
	 *
	 * @since version 3.12.0
	 */
	PrepareSupport *bool `json:"prepareSupport,omitempty"`

	/**
	 * Client supports the default behavior result
	 * (`{ defaultBehavior: boolean }`).
	 *
	 * The value indicates the default behavior used by the
	 * client.
	 *
	 * @since version 3.16.0
	 */
	PrepareSupportDefaultBehavior *prepareSupportDefaultBehavior `json:"prepareSupportDefaultBehavior,omitempty"`

	/**
	 * Whether the client honors the change annotations in
	 * text edits and resource operations returned via the
	 * rename request's workspace edit by for example presenting
	 * the workspace edit in the user interface and asking
	 * for confirmation.
	 *
	 * @since 3.16.0
	 */
	HonorsChangeAnnotations *bool `json:"honorsChangeAnnotations,omitempty"`
}

type publishDiagnosticsClientCapabilities struct {
	/**
	 * Whether the clients accepts diagnostics with related information.
	 */
	RelatedInformation *bool `json:"relatedInformation,omitempty"`

	/**
	 * Client supports the tag property to provide meta data about a diagnostic.
	 * Clients supporting tags have to handle unknown tags gracefully.
	 *
	 * @since 3.15.0
	 */
	TagSupport *struct {
		/**
		 * The tags supported by the client.
		 */
		ValueSet []diagnosticTag `json:"valueSet,omitempty"`
	} `json:"tagSupport,omitempty"`

	/**
	 * Whether the client interprets the version property of the
	 * `textDocument/publishDiagnostics` notification's parameter.
	 *
	 * @since 3.15.0
	 */
	VersionSupport *bool `json:"versionSupport,omitempty"`

	/**
	 * Client supports a codeDescription property
	 *
	 * @since 3.16.0
	 */
	CodeDescriptionSupport *bool `json:"codeDescriptionSupport,omitempty"`

	/**
	 * Whether code action supports the `data` property which is
	 * preserved between a `textDocument/publishDiagnostics` and
	 * `textDocument/codeAction` request.
	 *
	 * @since 3.16.0
	 */
	DataSupport *bool `json:"dataSupport,omitempty"`
}

type foldingRangeClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration for folding range
	 * providers. If this is set to `true` the client supports the new
	 * `FoldingRangeRegistrationOptions` return value for the corresponding
	 * server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * The maximum number of folding ranges that the client prefers to receive
	 * per document. The value serves as a hint, servers are free to follow the
	 * limit.
	 */
	RangeLimit *bool `json:"rangeLimit,omitempty"`

	/**
	 * If set, the client signals that it only supports folding complete lines.
	 * If set, client will ignore specified `startCharacter` and `endCharacter`
	 * properties in a FoldingRange.
	 */
	LineFoldingOnly *bool `json:"lineFoldingOnly,omitempty"`

	/**
	 * Specific options for the folding range kind.
	 *
	 * @since 3.17.0
	 */
	FoldingRangeKind *struct {
		/**
		 * The folding range kind values the client supports. When this
		 * property exists the client also guarantees that it will
		 * handle values outside its set gracefully and falls back
		 * to a default value when unknown.
		 */
		ValueSet []foldingRangeKind `json:"valueSet,omitempty"`
	} `json:"foldingRangeKind,omitempty"`

	/**
	 * Specific options for the folding range.
	 * @since 3.17.0
	 */
	FoldingRange *struct {
		/**
		* If set, the client signals that it supports setting collapsedText on
		* folding ranges to display custom labels instead of the default text.
		*
		* @since 3.17.0
		 */
		CollapsedText *bool `json:"collapsedText,omitempty"`
	} `json:"foldingRange,omitempty"`
}

type selectionRangeClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration for selection range
	 * providers. If this is set to `true` the client supports the new
	 * `SelectionRangeRegistrationOptions` return value for the corresponding
	 * server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type linkedEditingRangeClientCapabilities struct {
	/**
	 * Whether the implementation supports dynamic registration.
	 * If this is set to `true` the client supports the new
	 * `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	 * return value for the corresponding server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type callHierarchyClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `(TextDocumentRegistrationOptions &
	 * StaticRegistrationOptions)` return value for the corresponding server
	 * capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type semanticTokensClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `(TextDocumentRegistrationOptions &
	 * StaticRegistrationOptions)` return value for the corresponding server
	 * capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Which requests the client supports and might send to the server
	 * depending on the server's capability. Please note that clients might not
	 * show semantic tokens or degrade some of the user experience if a range
	 * or full request is advertised by the client but not provided by the
	 * server. If for example the client capability `requests.full` and
	 * `request.range` are both set to true but the server only provides a
	 * range provider the client might not render a minimap correctly or might
	 * even decide to not show any semantic tokens at all.
	 */
	Requests *struct {
		// TODO:
		/**
		 * The client will send the `textDocument/semanticTokens/range` request
		 * if the server provides a corresponding handler.
		 */
		Range *bool `json:"range,omitempty"`

		/**
		 * The client will send the `textDocument/semanticTokens/full` request
		 * if the server provides a corresponding handler.
		 */
		Full *bool `json:"full,omitempty"`
	} `json:"requests,omitempty"`

	/**
	 * The token types that the client supports.
	 */
	TokenTypes []string `json:"tokenTypes,omitempty"`

	/**
	 * The token modifiers that the client supports.
	 */
	TokenModifiers []string `json:"tokenModifiers,omitempty"`

	/**
	 * The formats the clients supports.
	 */
	Formats []tokenFormat `json:"formats,omitempty"`

	/**
	 * Whether the client supports tokens that can overlap each other.
	 */
	OverlappingTokenSupport *bool `json:"overlappingTokenSupport,omitempty"`

	/**
	 * Whether the client supports tokens that can span multiple lines.
	 */
	MultilineTokenSupport *bool `json:"multilineTokenSupport,omitempty"`

	/**
	 * Whether the client allows the server to actively cancel a
	 * semantic token request, e.g. supports returning
	 * ErrorCodes.ServerCancelled. If a server does the client
	 * needs to retrigger the request.
	 *
	 * @since 3.17.0
	 */
	ServerCancelSupport *bool `json:"serverCancelSupport,omitempty"`

	/**
	 * Whether the client uses semantic tokens to augment existing
	 * syntax tokens. If set to `true` client side created syntax
	 * tokens and semantic tokens are both used for colorization. If
	 * set to `false` the client only uses the returned semantic tokens
	 * for colorization.
	 *
	 * If the value is `undefined` then the client behavior is not
	 * specified.
	 *
	 * @since 3.17.0
	 */
	AugmentsSyntaxTokens *bool `json:"augmentsSyntaxTokens,omitempty"`
}

type monikerClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `(TextDocumentRegistrationOptions &
	 * StaticRegistrationOptions)` return value for the corresponding server
	 * capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type typeHierarchyClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `(TextDocumentRegistrationOptions &
	 * StaticRegistrationOptions)` return value for the corresponding server
	 * capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

// inlineValueClientCapabilities Client capabilities specific to inline values.
// since 3.17.0
type inlineValueClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration for inline
	 * value providers.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

// inlayHintClientCapabilities Inlay hint client capabilities.
// since 3.17.0
type inlayHintClientCapabilities struct {
	/**
	 * Whether inlay hints support dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Indicates which properties a client can resolve lazily on an inlay
	 * hint.
	 */
	ResolveSupport *struct {
		/**
		 * The properties that a client can resolve lazily.
		 */
		Properties []string `json:"properties,omitempty"`
	} `json:"resolveSupport,omitempty"`
}

// diagnosticClientCapabilities Client capabilities specific to diagnostic pull requests.
// since 3.17.0
type diagnosticClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new
	 * `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	 * return value for the corresponding server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Whether the clients supports related documents for document diagnostic
	 * pulls.
	 */
	RelatedDocumentSupport *bool `json:"relatedDocumentSupport,omitempty"`

	/**
	 * Whether the clients accepts diagnostics with related information.
	 */
	RelatedInformation *bool `json:"relatedInformation,omitempty"`

	/**
	 * Client supports the tag property to provide meta data about a diagnostic.
	 * Clients supporting tags have to handle unknown tags gracefully.
	 */
	// tagSupport *ClientDiagnosticsTagOptions

	/**
	 * Client supports a codeDescription property
	 */
	CodeDescriptionSupport *bool `json:"codeDescriptionSupport,omitempty"`

	/**
	 * Whether code action supports the `data` property which is
	 * preserved between a `textDocument/publishDiagnostics` and
	 * `textDocument/codeAction` request.
	 */
	DataSupport *bool `json:"dataSupport,omitempty"`
}

type markupKind string

const (
	/**
	 * Plain text is supported as a content format
	 */
	MarkupKindPlaintext markupKind = "plaintext"
	/**
	 * Markdown is supported as a content format
	 */
	MarkupKindMarkdown markupKind = "markdown"
)

type completionItemTag = int

const (
	CompletionItemTagDeprecated completionItemTag = 1
)

type insertTextMode = int

const (
	InsertTextModeAsIs              insertTextMode = 1
	InsertTextModeAdjustIndentation insertTextMode = 2
)

type completionItemKind = int

const (
	CompletionItemKindText          completionItemKind = 1
	CompletionItemKindMethod        completionItemKind = 2
	CompletionItemKindFunction      completionItemKind = 3
	CompletionItemKindConstructor   completionItemKind = 4
	CompletionItemKindField         completionItemKind = 5
	CompletionItemKindVariable      completionItemKind = 6
	CompletionItemKindClass         completionItemKind = 7
	CompletionItemKindInterface     completionItemKind = 8
	CompletionItemKindModule        completionItemKind = 9
	CompletionItemKindProperty      completionItemKind = 10
	CompletionItemKindUnit          completionItemKind = 11
	CompletionItemKindValue         completionItemKind = 12
	CompletionItemKindEnum          completionItemKind = 13
	CompletionItemKindKeyword       completionItemKind = 14
	CompletionItemKindSnippet       completionItemKind = 15
	CompletionItemKindColor         completionItemKind = 16
	CompletionItemKindFile          completionItemKind = 17
	CompletionItemKindReference     completionItemKind = 18
	CompletionItemKindFolder        completionItemKind = 19
	CompletionItemKindEnumMember    completionItemKind = 20
	CompletionItemKindConstant      completionItemKind = 21
	CompletionItemKindStruct        completionItemKind = 22
	CompletionItemKindEvent         completionItemKind = 23
	CompletionItemKindOperator      completionItemKind = 24
	CompletionItemKindTypeParameter completionItemKind = 25
)

// codeActionKind The kind of a code action.
// Kinds are a hierarchical list of identifiers separated by `.`,
// e.g. `"refactor.extract.function"`.
// The set of kinds is open and client needs to announce the kinds it supports
// to the server during initialization.
type codeActionKind = string

/**
 * A set of predefined code action kinds.
 */
const (

	/**
	 * Empty kind.
	 */
	CodeActionKindEmpty codeActionKind = ""

	/**
	 * Base kind for quickfix actions: "quickfix".
	 */
	CodeActionKindQuickFix codeActionKind = "quickfix"

	/**
	 * Base kind for refactoring actions: "refactor".
	 */
	CodeActionKindRefactor codeActionKind = "refactor"

	/**
	 * Base kind for refactoring extraction actions: "refactor.extract".
	 *
	 * Example extract actions:
	 *
	 * - Extract method
	 * - Extract function
	 * - Extract variable
	 * - Extract interface from class
	 * - ...
	 */
	CodeActionKindRefactorExtract codeActionKind = "refactor.extract"

	/**
	 * Base kind for refactoring inline actions: "refactor.inline".
	 *
	 * Example inline actions:
	 *
	 * - Inline function
	 * - Inline variable
	 * - Inline constant
	 * - ...
	 */
	CodeActionKindRefactorInline codeActionKind = "refactor.inline"

	/**
	 * Base kind for refactoring rewrite actions: "refactor.rewrite".
	 *
	 * Example rewrite actions:
	 *
	 * - Convert JavaScript function to class
	 * - Add or remove parameter
	 * - Encapsulate field
	 * - Make method static
	 * - Move method to base class
	 * - ...
	 */
	CodeActionKindRefactorRewrite codeActionKind = "refactor.rewrite"

	/**
	 * Base kind for source actions: `source`.
	 *
	 * Source code actions apply to the entire file.
	 */
	CodeActionKindSource codeActionKind = "source"

	/**
	 * Base kind for an organize imports source action:
	 * `source.organizeImports`.
	 */
	CodeActionKindSourceOrganizeImports codeActionKind = "source.organizeImports"

	/**
	 * Base kind for a "fix all" source action: `source.fixAll`.
	 *
	 * "Fix all" actions automatically fix errors that have a clear fix that
	 * do not require user input. They should not suppress errors or perform
	 * unsafe fixes such as generating new types or classes.
	 *
	 * @since 3.17.0
	 */
	CodeActionKindSourceFixAll codeActionKind = "source.fixAll"
)

type prepareSupportDefaultBehavior = int

const (
	PrepareSupportDefaultBehaviorIdentifier prepareSupportDefaultBehavior = 1
)

// diagnosticTag The diagnostic tags.
// since 3.15.0
type diagnosticTag = int

const (
	/**
	 * Unused or unnecessary code.
	 *
	 * Clients are allowed to render diagnostics with this tag faded out
	 * instead of having an error squiggle.
	 */
	DiagnosticTagUnnecessary diagnosticTag = 1
	/**
	 * Deprecated or obsolete code.
	 *
	 * Clients are allowed to rendered diagnostics with this tag strike through.
	 */
	DiagnosticTagDeprecated diagnosticTag = 2
)

// foldingRangeKind The type is a string since the value set is extensible
type foldingRangeKind = string

/**
 * A set of predefined range kinds.
 */
const (
	/**
	 * Folding range for a comment
	 */
	FoldingRangeKindComment string = "comment"
	/**
	 * Folding range for imports or includes
	 */
	FoldingRangeKindImports string = "imports"
	/**
	 * Folding range for a region (e.g. `#region`)
	 */
	FoldingRangeKindRegion string = "region"
)

type tokenFormat = string

const (
	TokenFormatRelative tokenFormat = "relative"
)

type responseError struct {
	/**
	 * A number indicating the error type that occurred.
	 */
	Code int `json:"code"`

	/**
	 * A string providing a short description of the error.
	 */
	Message string `json:"message"`

	/**
	 * A primitive or structured value that contains additional
	 * information about the error. Can be omitted.
	 */
	Data lSPAny `json:"data,omitempty"`
}

const (
	// Defined by JSON-RPC
	ParseError     int = -32700
	InvalidRequest int = -32600
	MethodNotFound int = -32601
	InvalidParams  int = -32602
	InternalError  int = -32603

	/**
	 * This is the start range of JSON-RPC reserved error codes.
	 * It doesn't denote a real error code. No LSP error codes should
	 * be defined between the start and end range. For backwards
	 * compatibility the `ServerNotInitialized` and the `UnknownErrorCode`
	 * are left in the range.
	 *
	 * @since 3.16.0
	 */
	jsonrpcReservedErrorRangeStart int = -32099
	/** @deprecated use jsonrpcReservedErrorRangeStart */
	serverErrorStart int = jsonrpcReservedErrorRangeStart

	/**
	 * Error code indicating that a server received a notification or
	 * request before the server received the `initialize` request.
	 */
	ServerNotInitialized int = -32002
	UnknownErrorCode     int = -32001

	/**
	 * This is the end range of JSON-RPC reserved error codes.
	 * It doesn't denote a real error code.
	 *
	 * @since 3.16.0
	 */
	jsonrpcReservedErrorRangeEnd = -32000
	/** @deprecated use jsonrpcReservedErrorRangeEnd */
	serverErrorEnd int = jsonrpcReservedErrorRangeEnd

	/**
	 * This is the start range of LSP reserved error codes.
	 * It doesn't denote a real error code.
	 *
	 * @since 3.16.0
	 */
	lspReservedErrorRangeStart int = -32899

	/**
	 * A request failed but it was syntactically correct, e.g the
	 * method name was known and the parameters were valid. The error
	 * message should contain human readable information about why
	 * the request failed.
	 *
	 * @since 3.17.0
	 */
	RequestFailed int = -32803

	/**
	 * The server cancelled the request. This error code should
	 * only be used for requests that explicitly support being
	 * server cancellable.
	 *
	 * @since 3.17.0
	 */
	ServerCancelled int = -32802

	/**
	 * The server detected that the content of a document got
	 * modified outside normal conditions. A server should
	 * NOT send this error code if it detects a content change
	 * in its unprocessed messages. The result even computed
	 * on an older state might still be useful for the client.
	 *
	 * If a client decides that a result is not of any use anymore
	 * the client should cancel the request.
	 */
	ContentModified int = -32801

	/**
	 * The client has canceled a request and a server has detected
	 * the cancel.
	 */
	RequestCancelled int = -32800

	/**
	 * This is the end range of LSP reserved error codes.
	 * It doesn't denote a real error code.
	 *
	 * @since 3.16.0
	 */
	lspReservedErrorRangeEnd int = -32800
)
