package server

type ClientCapabilities struct {
}

type CodeLensOptions struct {
	ResolveProvider bool `json:"resolveProvider,omitempty"`
}

type CompletionOptions struct {
	ResolveProvider   bool     `json:"resolveProvider,omitempty"`
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type DocumentOnTypeFormattingOptions struct {
	FirstTriggerCharacter string   `json:"firstTriggerCharacter"`
	MoreTriggerCharacter  []string `json:"moreTriggerCharacter,omitempty"`
}

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
}

type SignatureHelpOptions struct {
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type TextDocumentSyncKind int

type TextDocumentSyncOptions struct {
	OpenClose         bool                 `json:"openClose,omitempty"`
	Change            TextDocumentSyncKind `json:"change"`
	WillSave          bool                 `json:"willSave,omitempty"`
	WillSaveWaitUntil bool                 `json:"willSaveWaitUntil,omitempty"`
	Save              *SaveOptions         `json:"save,omitempty"`
}

// Common?

// DocumentURI is a document identifier
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#uri
type DocumentURI string

// Location is a spanning location inside a document
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#location
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// Position points to a location in a text document
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#position
type Position struct {
	// Line position in a document (zero-based)
	Line int `json:"line"`

	// Character offset on a line in a document (zero-based)
	Character int `json:"character"`
}

// Range is a contigous block within a document
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#range
type Range struct {
	// Start is the range's start position
	Start Position `json:"start"`

	// End is the range's end position
	End Position `json:"end"`
}

// TextDocumentIdentifier is an identifier for a text document
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#textdocumentidentifier
type TextDocumentIdentifier struct {
	// URI of the text document
	URI DocumentURI `json:"uri"`
}

type TextDocumentPositionParams struct {
	// TextDocument idenfities the document
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Position inside the text document
	Position Position `json:"position"`
}
