package lsp

import "encoding/json"

type Message struct {
	Jsonrpc string `json:"jsonrpc"`
}

type RequestMessage struct {
	Message
	ID     int    `json:"id"`
	Method string `json:"method"`
	Params Params `json:"params"`
}

func (msg *RequestMessage) UnmarshalJSON(data []byte) error {
	var raw struct {
		Message
		ID     int             `json:"id"`
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	msg.Jsonrpc = raw.Jsonrpc
	msg.ID = raw.ID
	msg.Method = raw.Method

	switch msg.Method {
	case "initialize":
		var i InitializeParams
		if err := json.Unmarshal(raw.Params, &i); err != nil {
			return err
		}
		msg.Params = i
	}

	return nil
}

type Params interface {
	isParams()
}

type WorkProgressParams struct {
	workDoneToken progressToken
}

func (WorkProgressParams) isParams() {}

type progressToken interface {
	isProgressToken()
}

type stringProgressToken string

func (stringProgressToken) isProgressToken() {}

type intProgressToken int

func (intProgressToken) isProgressToken() {}

type (
	DocumentURI string
	URI         string
)

type InitializeParams struct {
	WorkProgressParams

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
	RootURI *DocumentURI `json:"RootURI,omitempty"`

	/**
	 * User provided initialization options.
	 */
	InitializationOptions *string `json:"InitializationOptions,omitempty"`

	/**
	 * The capabilities provided by the client (editor or tool)
	 */
	Capabilities ClientCapabilities `json:"capabilities"`

	/**
	 * The initial trace setting. If omitted trace is disabled ('off').
	 */
	Trace *TraceValue `json:"trace,omitempty"`

	/**
	 * The workspace folders configured in the client when the server starts.
	 * This property is only available if the client supports workspace folders.
	 * It can be `null` if the client supports workspace folders but none are
	 * configured.
	 *
	 * @since 3.6.0
	 */
	WorkspaceFolders []WorkspaceFolder `json:"workspaceFolders,omitempty"`
}

type TraceValue string

const (
	TraceValueOff      TraceValue = "off"
	TraceValueMessages TraceValue = "messages"
	TraceValueVerbose  TraceValue = "verbose"
)

type WorkspaceFolder struct {
	/**
	 * The associated URI for this workspace folder.
	 */
	URI URI `json:"uri"`

	/**
	 * The name of the workspace folder. Used to refer to this
	 * workspace folder in the user interface.
	 */
	Name string `json:"name"`
}

type ClientCapabilities struct {
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
		WorkspaceEdit *WorkspaceEditClientCapabilities `json:"workspaceEdit,omitempty"`

		/**
		 * Capabilities specific to the `workspace/didChangeConfiguration`
		 * notification.
		 */
		DidChangeConfiguration *DidChangeConfigurationClientCapabilities `json:"didChangeConfiguration,omitempty"`

		/**
		 * Capabilities specific to the `workspace/didChangeWatchedFiles`
		 * notification.
		 */
		DidChangeWatchedFiles *DidChangeWatchedFilesClientCapabilities `json:"didChangeWatchedFiles,omitempty"`

		/**
		 * Capabilities specific to the `workspace/symbol` request.
		 */
		Symbol *WorkspaceSymbolClientCapabilities `json:"symbol,omitempty"`

		/**
		 * Capabilities specific to the `workspace/executeCommand` request.
		 */
		ExecuteCommand *ExecuteCommandClientCapabilities `json:"executeCommand,omitempty"`

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
		SemanticTokens *SemanticTokensWorkspaceClientCapabilities `json:"semanticTokens,omitempty"`

		/**
		 * Capabilities specific to the code lens requests scoped to the
		 * workspace.
		 *
		 * @since 3.16.0
		 */
		CodeLens *CodeLensWorkspaceClientCapabilities `json:"codeLens,omitempty"`

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
		InlineValue *InlineValueWorkspaceClientCapabilities `json:"inlineValue,omitempty"`

		/**
		 * Client workspace capabilities specific to inlay hints.
		 *
		 * @since 3.17.0
		 */
		InlayHint *InlayHintWorkspaceClientCapabilities `json:"inlayHint,omitempty"`

		/**
		 * Client workspace capabilities specific to diagnostics.
		 *
		 * @since 3.17.0.
		 */
		Diagnostics *DiagnosticWorkspaceClientCapabilities `json:"diagnostics,omitempty"`
	} `json:"workspace"`

	/**
	 * Text document specific client capabilities.
	 */
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`

	/**
	 * Capabilities specific to the notebook document support.
	 *
	 * @since 3.17.0
	 */
	NotebookDocument *NotebookDocumentClientCapabilities `json:"notebookDocument,omitempty"`

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
		ShowMessage *ShowMessageRequestClientCapabilities `json:"showMessage,omitempty"`

		/**
		 * Client capabilities for the show document request.
		 *
		 * @since 3.16.0
		 */
		ShowDocument *ShowDocumentClientCapabilities `json:"showDocument,omitempty"`
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
		RegularExpressions *RegularExpressionsClientCapabilities `json:"regularExpressions,omitempty"`

		/**
		 * Client capabilities specific to the client's markdown parser.
		 *
		 * @since 3.16.0
		 */
		Markdown *MarkdownClientCapabilities `json:"markdown,omitempty"`

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
		PositionEncodings []PositionEncodingKind `json:"positionEncodings,omitempty"`
	} `json:"general,omitempty"`

	/**
	 * Experimental client capabilities.
	 */
	Experimental *LSPAny `json:"experimental,omitempty"`
}

type LSPAny interface {
	isLspAny()
}

type LSPObject map[string]LSPAny

func (LSPObject) isLspAny() {}

type LSPArray []LSPAny

func (LSPArray) isLspAny() {}

type LSPString string

func (LSPString) isLspAny() {}

type LSPInteger int

func (LSPInteger) isLspAny() {}

type LSPUInteger uint

func (LSPUInteger) isLspAny() {}

type LSPDecimal float64

func (LSPDecimal) isLspAny() {}

type LSPBoolean bool

func (LSPBoolean) isLspAny() {}

type WorkspaceEditClientCapabilities struct {
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
	ResourceOperations []ResourceOperationKind `json:"resourceOperations,omitempty"`

	/**
	 * The failure handling strategy of a client if applying the workspace edit
	 * fails.
	 *
	 * @since 3.13.0
	 */
	FailureHandling *FailureHandlingKind `json:"failureHandling,omitempty"`

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

type DidChangeConfigurationClientCapabilities struct {
	/**
	 * Did change configuration notification supports dynamic registration.
	 *
	 * @since 3.6.0 to support the new pull model.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DidChangeWatchedFilesClientCapabilities struct {
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

type WorkspaceSymbolClientCapabilities struct {
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
		ValueSet []SymbolKind `json:"valueSet,omitempty"`
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
		ValueSet []SymbolTag `json:"valueSet,omitempty"`
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

type ExecuteCommandClientCapabilities struct {
	/**
	 * Execute command supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type SemanticTokensWorkspaceClientCapabilities struct {
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

type CodeLensWorkspaceClientCapabilities struct {
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

// InlineValueWorkspaceClientCapabilities Client workspace capabilities specific to inline values.
// since 3.17.0
type InlineValueWorkspaceClientCapabilities struct {
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

// InlayHintWorkspaceClientCapabilities Client workspace capabilities specific to inlay hints.
// since 3.17.0
type InlayHintWorkspaceClientCapabilities struct {
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

// DiagnosticWorkspaceClientCapabilities Workspace client capabilities specific to diagnostic pull requests.
// since 3.17.0
type DiagnosticWorkspaceClientCapabilities struct {
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

// TextDocumentClientCapabilities Text document specific client capabilities.
type TextDocumentClientCapabilities struct {
	Synchronization *TextDocumentSyncClientCapabilities `json:"synchronization,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/completion` request.
	 */
	Completion *CompletionClientCapabilities `json:"completion,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/hover` request.
	 */
	Hover *HoverClientCapabilities `json:"hover,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/signatureHelp` request.
	 */
	SignatureHelp *SignatureHelpClientCapabilities `json:"signatureHelp,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/declaration` request.
	 *
	 * @since 3.14.0
	 */
	Declaration *DeclarationClientCapabilities `json:"declaration,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/definition` request.
	 */
	Definition *DefinitionClientCapabilities `json:"definition,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/typeDefinition` request.
	 *
	 * @since 3.6.0
	 */
	TypeDefinition *TypeDefinitionClientCapabilities `json:"typeDefinition,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/implementation` request.
	 *
	 * @since 3.6.0
	 */
	Implementation *ImplementationClientCapabilities `json:"implementation,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/references` request.
	 */
	References *ReferenceClientCapabilities `json:"references,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentHighlight` request.
	 */
	DocumentHighlight *DocumentHighlightClientCapabilities `json:"documentHighlight,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentSymbol` request.
	 */
	DocumentSymbol *DocumentSymbolClientCapabilities `json:"documentSymbol,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/codeAction` request.
	 */
	CodeAction *CodeActionClientCapabilities `json:"codeAction,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/codeLens` request.
	 */
	CodeLens *CodeLensClientCapabilities `json:"codeLens,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentLink` request.
	 */
	DocumentLink *DocumentLinkClientCapabilities `json:"documentLink,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/documentColor` and the
	 * `textDocument/colorPresentation` request.
	 *
	 * @since 3.6.0
	 */
	ColorProvider *DocumentColorClientCapabilities `json:"colorProvider,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/formatting` request.
	 */
	Formatting *DocumentFormattingClientCapabilities `json:"formatting,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/rangeFormatting` request.
	 */
	RangeFormatting *DocumentRangeFormattingClientCapabilities `json:"rangeFormatting,omitempty"`

	/** request.
	 * Capabilities specific to the `textDocument/onTypeFormatting` request.
	 */
	OnTypeFormatting *DocumentOnTypeFormattingClientCapabilities `json:"onTypeFormatting,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/rename` request.
	 */
	Rename *RenameClientCapabilities `json:"rename,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/publishDiagnostics`
	 * notification.
	 */
	PublishDiagnostics *PublishDiagnosticsClientCapabilities `json:"publishDiagnostics,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/foldingRange` request.
	 *
	 * @since 3.10.0
	 */
	FoldingRange *FoldingRangeClientCapabilities `json:"foldingRange,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/selectionRange` request.
	 *
	 * @since 3.15.0
	 */
	SelectionRange *SelectionRangeClientCapabilities `json:"selectionRange,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/linkedEditingRange` request.
	 *
	 * @since 3.16.0
	 */
	LinkedEditingRange *LinkedEditingRangeClientCapabilities `json:"linkedEditingRange,omitempty"`

	/**
	 * Capabilities specific to the various call hierarchy requests.
	 *
	 * @since 3.16.0
	 */
	CallHierarchy *CallHierarchyClientCapabilities `json:"callHierarchy,omitempty"`

	/**
	 * Capabilities specific to the various semantic token requests.
	 *
	 * @since 3.16.0
	 */
	SemanticTokens *SemanticTokensClientCapabilities `json:"semanticTokens,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/moniker` request.
	 *
	 * @since 3.16.0
	 */
	Moniker *MonikerClientCapabilities `json:"moniker,omitempty"`

	/**
	 * Capabilities specific to the various type hierarchy requests.
	 *
	 * @since 3.17.0
	 */
	TypeHierarchy *TypeHierarchyClientCapabilities `json:"typeHierarchy,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/inlineValue` request.
	 *
	 * @since 3.17.0
	 */
	InlineValue *InlineValueClientCapabilities `json:"inlineValue,omitempty"`

	/**
	 * Capabilities specific to the `textDocument/inlayHint` request.
	 *
	 * @since 3.17.0
	 */
	InlayHint *InlayHintClientCapabilities `json:"inlayHint,omitempty"`

	/**
	 * Capabilities specific to the diagnostic pull model.
	 *
	 * @since 3.17.0
	 */
	Diagnostic *DiagnosticClientCapabilities `json:"diagnostic,omitempty"`
}

// NotebookDocumentClientCapabilities Capabilities specific to the notebook document support.
// since 3.17.0
type NotebookDocumentClientCapabilities struct {
	/**
	 * Capabilities specific to notebook document synchronization
	 *
	 * @since 3.17.0
	 */
	Synchronization *NotebookDocumentSyncClientCapabilities `json:"synchronization,omitempty"`
}

// NotebookDocumentSyncClientCapabilities Notebook specific client capabilities.
// since 3.17.0
type NotebookDocumentSyncClientCapabilities struct {
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

// ShowMessageRequestClientCapabilities Show message request client capabilities
type ShowMessageRequestClientCapabilities struct {
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

// ShowDocumentClientCapabilities Client capabilities for the show document request.
// since 3.16.0
type ShowDocumentClientCapabilities struct {
	/**
	 * The client has support for the show document
	 * request.
	 */
	Support *bool `json:"support,omitempty"`
}

// RegularExpressionsClientCapabilities Client capabilities specific to regular expressions.
type RegularExpressionsClientCapabilities struct {
	/**
	 * The engine's name.
	 */
	Engine string `json:"engine"`

	/**
	 * The engine's version.
	 */
	Version *string `json:"Version,omitempty"`
}

// MarkdownClientCapabilities Client capabilities specific to the used markdown parser.
// since 3.16.0
type MarkdownClientCapabilities struct {
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

// PositionEncodingKind A type indicating how positions are encoded, specifically what column offsets mean.
// @since 3.17.0
type PositionEncodingKind string

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

// ResourceOperationKind The kind of resource operations supported by the client.
type ResourceOperationKind string

const (
	/**
	 * Supports creating new files and folders.
	 */
	ResourceOperationKindCreate ResourceOperationKind = "create"
	/**
	 * Supports renaming existing files and folders.
	 */
	ResourceOperationKindRename = "rename"
	/**
	 * Supports deleting existing files and folders.
	 */
	ResourceOperationKindDelete = "delete"
)

type FailureHandlingKind string

const (
	/**
	 * Applying the workspace change is simply aborted if one of the changes
	 * provided fails. All operations executed before the failing operation
	 * stay executed.
	 */
	FailureHandlingKindAbort FailureHandlingKind = "abort"
	/**
	 * All operations are executed transactional. That means they either all
	 * succeed or no changes at all are applied to the workspace.
	 */
	FailureHandlingKindTransactional FailureHandlingKind = "transactional"
	/**
	 * If the workspace edit contains only textual file changes they are
	 * executed transactional. If resource changes (create, rename or delete
	 * file) are part of the change the failure handling strategy is abort.
	 */
	FailureHandlingKindTextOnlyTransactional FailureHandlingKind = "textOnlyTransactional"
	/**
	 * The client tries to undo the operations already executed. But there is no
	 * guarantee that this is succeeding.
	 */
	FailureHandlingKindUndo FailureHandlingKind = "undo"
)

type SymbolKind int

const (
	SymbolKindFile          SymbolKind = 1
	SymbolKindModule        SymbolKind = 2
	SymbolKindNamespace     SymbolKind = 3
	SymbolKindPackage       SymbolKind = 4
	SymbolKindClass         SymbolKind = 5
	SymbolKindMethod        SymbolKind = 6
	SymbolKindProperty      SymbolKind = 7
	SymbolKindField         SymbolKind = 8
	SymbolKindConstructor   SymbolKind = 9
	SymbolKindEnum          SymbolKind = 10
	SymbolKindInterface     SymbolKind = 11
	SymbolKindFunction      SymbolKind = 12
	SymbolKindVariable      SymbolKind = 13
	SymbolKindConstant      SymbolKind = 14
	SymbolKindString        SymbolKind = 15
	SymbolKindNumber        SymbolKind = 16
	SymbolKindBoolean       SymbolKind = 17
	SymbolKindArray         SymbolKind = 18
	SymbolKindObject        SymbolKind = 19
	SymbolKindKey           SymbolKind = 20
	SymbolKindNull          SymbolKind = 21
	SymbolKindEnumMember    SymbolKind = 22
	SymbolKindStruct        SymbolKind = 23
	SymbolKindEvent         SymbolKind = 24
	SymbolKindOperator      SymbolKind = 25
	SymbolKindTypeParameter SymbolKind = 26
)

type SymbolTag int

const (
	SymbolTagDeprecated SymbolTag = 1
)

type TextDocumentSyncClientCapabilities struct {
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

type CompletionClientCapabilities struct {
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
		DocumentationFormat []MarkupKind `json:"documentationFormat,omitempty"`

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
			ValueSet []CompletionItemTag `json:"valueSet,omitempty"`
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
			ValueSet []InsertTextMode `json:"valueSet,omitempty"`
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
		ValueSet []CompletionItemKind `json:"valueSet,omitempty"`
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
	InsertTextMode *InsertTextMode `json:"insertTextMode,omitempty"`

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

type HoverClientCapabilities struct {
	/**
	 * Whether hover supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

	/**
	 * Client supports the follow content formats if the content
	 * property refers to a `literal of type MarkupContent`.
	 * The order describes the preferred format of the client.
	 */
	ContentFormat []MarkupKind `json:"contentFormat,omitempty"`
}

type SignatureHelpClientCapabilities struct {
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
		DocumentationFormat []MarkupKind `json:"documentationFormat,omitempty"`

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

type DeclarationClientCapabilities struct {
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

type DefinitionClientCapabilities struct {
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

type TypeDefinitionClientCapabilities struct {
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

type ImplementationClientCapabilities struct {
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

type ReferenceClientCapabilities struct {
	/**
	 * Whether references supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DocumentHighlightClientCapabilities struct {
	/**
	 * Whether document highlight supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DocumentSymbolClientCapabilities struct {
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
		ValueSet []SymbolKind `json:"valueSet,omitempty"`
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
		ValueSet []SymbolTag `json:"valueSet,omitempty"`
	} `json:"tagSupport,omitempty"`

	/**
	 * The client supports an additional label presented in the UI when
	 * registering a document symbol provider.
	 *
	 * @since 3.16.0
	 */
	LabelSupport *bool `json:"labelSupport,omitempty"`
}

type CodeActionClientCapabilities struct {
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
			ValueSet []CodeActionKind `json:"valueSet,omitempty"`
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

type CodeLensClientCapabilities struct {
	/**
	 * Whether code lens supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DocumentLinkClientCapabilities struct {
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

type DocumentColorClientCapabilities struct {
	/**
	 * Whether document color supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DocumentFormattingClientCapabilities struct {
	/**
	 * Whether formatting supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DocumentRangeFormattingClientCapabilities struct {
	/**
	 * Whether formatting supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type DocumentOnTypeFormattingClientCapabilities struct {
	/**
	 * Whether on type formatting supports dynamic registration.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type RenameClientCapabilities struct {
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
	PrepareSupportDefaultBehavior *PrepareSupportDefaultBehavior `json:"prepareSupportDefaultBehavior,omitempty"`

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

type PublishDiagnosticsClientCapabilities struct {
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
		ValueSet []DiagnosticTag `json:"valueSet,omitempty"`
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

type FoldingRangeClientCapabilities struct {
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
		ValueSet []FoldingRangeKind `json:"valueSet,omitempty"`
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

type SelectionRangeClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration for selection range
	 * providers. If this is set to `true` the client supports the new
	 * `SelectionRangeRegistrationOptions` return value for the corresponding
	 * server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type LinkedEditingRangeClientCapabilities struct {
	/**
	 * Whether the implementation supports dynamic registration.
	 * If this is set to `true` the client supports the new
	 * `(TextDocumentRegistrationOptions & StaticRegistrationOptions)`
	 * return value for the corresponding server capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type CallHierarchyClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `(TextDocumentRegistrationOptions &
	 * StaticRegistrationOptions)` return value for the corresponding server
	 * capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type SemanticTokensClientCapabilities struct {
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
	Formats []TokenFormat `json:"formats,omitempty"`

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

type MonikerClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `(TextDocumentRegistrationOptions &
	 * StaticRegistrationOptions)` return value for the corresponding server
	 * capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type TypeHierarchyClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration. If this is set to
	 * `true` the client supports the new `(TextDocumentRegistrationOptions &
	 * StaticRegistrationOptions)` return value for the corresponding server
	 * capability as well.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

// InlineValueClientCapabilities Client capabilities specific to inline values.
// since 3.17.0
type InlineValueClientCapabilities struct {
	/**
	 * Whether implementation supports dynamic registration for inline
	 * value providers.
	 */
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

// InlayHintClientCapabilities Inlay hint client capabilities.
// since 3.17.0
type InlayHintClientCapabilities struct {
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

// DiagnosticClientCapabilities Client capabilities specific to diagnostic pull requests.
// since 3.17.0
type DiagnosticClientCapabilities struct {
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

type MarkupKind string

const (
	/**
	 * Plain text is supported as a content format
	 */
	MarkupKindPlaintext MarkupKind = "plaintext"
	/**
	 * Markdown is supported as a content format
	 */
	MarkupKindMarkdown MarkupKind = "markdown"
)

type CompletionItemTag = int

const (
	CompletionItemTagDeprecated CompletionItemTag = 1
)

type InsertTextMode = int

const (
	InsertTextModeAsIs              InsertTextMode = 1
	InsertTextModeAdjustIndentation InsertTextMode = 2
)

type CompletionItemKind = int

const (
	CompletionItemKindText          CompletionItemKind = 1
	CompletionItemKindMethod        CompletionItemKind = 2
	CompletionItemKindFunction      CompletionItemKind = 3
	CompletionItemKindConstructor   CompletionItemKind = 4
	CompletionItemKindField         CompletionItemKind = 5
	CompletionItemKindVariable      CompletionItemKind = 6
	CompletionItemKindClass         CompletionItemKind = 7
	CompletionItemKindInterface     CompletionItemKind = 8
	CompletionItemKindModule        CompletionItemKind = 9
	CompletionItemKindProperty      CompletionItemKind = 10
	CompletionItemKindUnit          CompletionItemKind = 11
	CompletionItemKindValue         CompletionItemKind = 12
	CompletionItemKindEnum          CompletionItemKind = 13
	CompletionItemKindKeyword       CompletionItemKind = 14
	CompletionItemKindSnippet       CompletionItemKind = 15
	CompletionItemKindColor         CompletionItemKind = 16
	CompletionItemKindFile          CompletionItemKind = 17
	CompletionItemKindReference     CompletionItemKind = 18
	CompletionItemKindFolder        CompletionItemKind = 19
	CompletionItemKindEnumMember    CompletionItemKind = 20
	CompletionItemKindConstant      CompletionItemKind = 21
	CompletionItemKindStruct        CompletionItemKind = 22
	CompletionItemKindEvent         CompletionItemKind = 23
	CompletionItemKindOperator      CompletionItemKind = 24
	CompletionItemKindTypeParameter CompletionItemKind = 25
)

// CodeActionKind The kind of a code action.
// Kinds are a hierarchical list of identifiers separated by `.`,
// e.g. `"refactor.extract.function"`.
// The set of kinds is open and client needs to announce the kinds it supports
// to the server during initialization.
type CodeActionKind = string

/**
 * A set of predefined code action kinds.
 */
const (

	/**
	 * Empty kind.
	 */
	CodeActionKindEmpty CodeActionKind = ""

	/**
	 * Base kind for quickfix actions: "quickfix".
	 */
	CodeActionKindQuickFix CodeActionKind = "quickfix"

	/**
	 * Base kind for refactoring actions: "refactor".
	 */
	CodeActionKindRefactor CodeActionKind = "refactor"

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
	CodeActionKindRefactorExtract CodeActionKind = "refactor.extract"

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
	CodeActionKindRefactorInline CodeActionKind = "refactor.inline"

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
	CodeActionKindRefactorRewrite CodeActionKind = "refactor.rewrite"

	/**
	 * Base kind for source actions: `source`.
	 *
	 * Source code actions apply to the entire file.
	 */
	CodeActionKindSource CodeActionKind = "source"

	/**
	 * Base kind for an organize imports source action:
	 * `source.organizeImports`.
	 */
	CodeActionKindSourceOrganizeImports CodeActionKind = "source.organizeImports"

	/**
	 * Base kind for a "fix all" source action: `source.fixAll`.
	 *
	 * "Fix all" actions automatically fix errors that have a clear fix that
	 * do not require user input. They should not suppress errors or perform
	 * unsafe fixes such as generating new types or classes.
	 *
	 * @since 3.17.0
	 */
	CodeActionKindSourceFixAll CodeActionKind = "source.fixAll"
)

type PrepareSupportDefaultBehavior = int

const (
	PrepareSupportDefaultBehaviorIdentifier PrepareSupportDefaultBehavior = 1
)

// DiagnosticTag The diagnostic tags.
// since 3.15.0
type DiagnosticTag = int

const (
	/**
	 * Unused or unnecessary code.
	 *
	 * Clients are allowed to render diagnostics with this tag faded out
	 * instead of having an error squiggle.
	 */
	DiagnosticTagUnnecessary DiagnosticTag = 1
	/**
	 * Deprecated or obsolete code.
	 *
	 * Clients are allowed to rendered diagnostics with this tag strike through.
	 */
	DiagnosticTagDeprecated DiagnosticTag = 2
)

// FoldingRangeKind The type is a string since the value set is extensible
type FoldingRangeKind = string

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

type TokenFormat = string

const (
	TokenFormatRelative TokenFormat = "relative"
)
