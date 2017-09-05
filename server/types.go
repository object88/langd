package server

type ClientCapabilities struct {
	// // Below are Sourcegraph extensions. They do not live in lspext since
	// // they are extending the field InitializeParams.Capabilities

	// // XFilesProvider indicates the client provides support for
	// // workspace/xfiles. This is a Sourcegraph extension.
	// XFilesProvider bool `json:"xfilesProvider,omitempty"`

	// // XContentProvider indicates the client provides support for
	// // textDocument/xcontent. This is a Sourcegraph extension.
	// XContentProvider bool `json:"xcontentProvider,omitempty"`

	// // XCacheProvider indicates the client provides support for cache/get
	// // and cache/set.
	// XCacheProvider bool `json:"xcacheProvider,omitempty"`
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type DocumentOnTypeFormattingOptions struct {
	FirstTriggerCharacter string   `json:"firstTriggerCharacter"`
	MoreTriggerCharacter  []string `json:"moreTriggerCharacter,omitempty"`
}

type DocumentURI string

type InitializeParams struct {
	ProcessID int `json:"processId,omitempty"`

	RootURI DocumentURI `json:"rootUri,omitempty"`

	InitializationOptions interface{} `json:"initializationOptions,omitempty"`

	Capabilities ClientCapabilities `json:"capabilities"`
}

type InitializeResult struct {
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

type SaveOptions struct {
	IncludeText bool `json:"includeText"`
}

type TextDocumentSyncKind int

type TextDocumentSyncOptions struct {
	OpenClose         bool                 `json:"openClose,omitempty"`
	Change            TextDocumentSyncKind `json:"change"`
	WillSave          bool                 `json:"willSave,omitempty"`
	WillSaveWaitUntil bool                 `json:"willSaveWaitUntil,omitempty"`
	Save              *SaveOptions         `json:"save,omitempty"`
}

type ServerCapabilities struct {
	TextDocumentSync                 TextDocumentSyncOptions          `json:"textDocumentSync,omitempty"`
	HoverProvider                    bool                             `json:"hoverProvider,omitempty"`
	CompletionProvider               *CompletionOptions               `json:"completionProvider,omitempty"`
	SignatureHelpProvider            *SignatureHelpOptions            `json:"signatureHelpProvider,omitempty"`
	DefinitionProvider               bool                             `json:"definitionProvider,omitempty"`
	ReferencesProvider               bool                             `json:"referencesProvider,omitempty"`
	DocumentHighlightProvider        bool                             `json:"documentHighlightProvider,omitempty"`
	DocumentSymbolProvider           bool                             `json:"documentSymbolProvider,omitempty"`
	WorkspaceSymbolProvider          bool                             `json:"workspaceSymbolProvider,omitempty"`
	CodeActionProvider               bool                             `json:"codeActionProvider,omitempty"`
	CodeLensProvider                 *CodeLensOptions                 `json:"codeLensProvider,omitempty"`
	DocumentFormattingProvider       bool                             `json:"documentFormattingProvider,omitempty"`
	DocumentRangeFormattingProvider  bool                             `json:"documentRangeFormattingProvider,omitempty"`
	DocumentOnTypeFormattingProvider *DocumentOnTypeFormattingOptions `json:"documentOnTypeFormattingProvider,omitempty"`
	RenameProvider                   bool                             `json:"renameProvider,omitempty"`

	// // XWorkspaceReferencesProvider indicates the server provides support for
	// // xworkspace/references. This is a Sourcegraph extension.
	// XWorkspaceReferencesProvider bool `json:"xworkspaceReferencesProvider,omitempty"`

	// // XDefinitionProvider indicates the server provides support for
	// // textDocument/xdefinition. This is a Sourcegraph extension.
	// XDefinitionProvider bool `json:"xdefinitionProvider,omitempty"`

	// // XWorkspaceSymbolByProperties indicates the server provides support for
	// // querying symbols by properties with WorkspaceSymbolParams.symbol. This
	// // is a Sourcegraph extension.
	// XWorkspaceSymbolByProperties bool `json:"xworkspaceSymbolByProperties,omitempty"`
}
