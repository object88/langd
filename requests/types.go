package requests

// ClientCapabilities contains specific groups of capabilities of the client
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#initialize-request
type ClientCapabilities struct {
	// Workspace specific client capabilities.
	Workspace *WorkspaceClientCapabilities `json:"workspace,omitempty"`

	// TextDocument specific client capabilities.
	TextDocument *TextDocumentClientCapabilities `json:"textDocument,omitempty"`

	// Experimental client capabilities.
	Experimental interface{} `json:"experimental,omitempty"`
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

// DynamicRegistration contains information about dynamic changes
// to capabilities
type DynamicRegistration struct {
	// DynamicRegistration determines whether the 'Did change'
	// notification supports dynamic registration.
	DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`
}

type InitializeParams struct {
	ProcessID int `json:"processId,omitempty"`

	RootURI DocumentURI `json:"rootUri,omitempty"`

	InitializationOptions interface{} `json:"initializationOptions,omitempty"`

	Capabilities ClientCapabilities `json:"capabilities"`
}

// Example initialization options:
// {
// 	"processId":46081,
// 	"rootPath":"/Users/bropa18/work/src/github.com/object88/immutable",
// 	"rootUri":"file:///Users/bropa18/work/src/github.com/object88/immutable",
// 	"capabilities": {
// 		"workspace":{
// 			"applyEdit":true,
// 			"workspaceEdit":{"documentChanges":true},
// 			"didChangeConfiguration":{"dynamicRegistration":false},
// 			"didChangeWatchedFiles":{"dynamicRegistration":true},
// 			"symbol":{"dynamicRegistration":true},
// 			"executeCommand":{"dynamicRegistration":true}
// 		},
// 		"textDocument":{
// 			"synchronization":{"dynamicRegistration":true,"willSave":true,"willSaveWaitUntil":true,"didSave":true},
// 			"completion":{"dynamicRegistration":true,"completionItem":{"snippetSupport":true}},
// 			"hover":{"dynamicRegistration":true},
// 			"signatureHelp":{"dynamicRegistration":true},
// 			"references":{"dynamicRegistration":true},
// 			"documentHighlight":{"dynamicRegistration":true},
// 			"documentSymbol":{"dynamicRegistration":true},
// 			"formatting":{"dynamicRegistration":true},
// 			"rangeFormatting":{"dynamicRegistration":true},
// 			"onTypeFormatting":{"dynamicRegistration":true},
// 			"definition":{"dynamicRegistration":true},
// 			"codeAction":{"dynamicRegistration":true},
// 			"codeLens":{"dynamicRegistration":true},
// 			"documentLink":{"dynamicRegistration":true},
// 			"rename":{"dynamicRegistration":true}
// 		}
// 	},
// 	"trace":"off"
// }

// InitializeResult is the response to the initialize request, and includes
// information regarding the server's capabilities
type InitializeResult struct {
	// Capabilities describe what the server is capable of handling
	Capabilities ServerCapabilities `json:"capabilities,omitempty"`
}

// SaveOptions includes options that the server can indicate to the client.
type SaveOptions struct {
	// IncludeText specifies whether the client should include file content
	// on save
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

// SignatureHelpOptions specifies how the server can assist with signatures
type SignatureHelpOptions struct {
	// TriggerCharacters are characters that trigger signature help automatically
	TriggerCharacters []string `json:"triggerCharacters,omitempty"`
}

type TextDocumentClientCapabilities struct {
	// Synchronization contains sync-related capabilities
	Synchronization *struct {
		// DynamicRegistration states whether text document synchronization
		// supports dynamic registration.
		DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

		// WillSave state whether the client supports sending will save
		// notifications.
		WillSave *bool `json:"willSave,omitempty"`

		// WillSaveWaitUntil states whether the client supports sending a will
		// save request and waits for a response providing text edits which will
		// be applied to the document before it is saved.
		WillSaveWaitUntil *bool `json:"willSaveWaitUntil,omitempty"`

		// DidSave states whether the client supports did save notifications.
		DidSave *bool `json:"didSave,omitEmpty"`
	} `json:"synchronization,omitempty"`

	// Completion contains capabilities specific to `textDocument/completion`
	Completion *struct {
		// DynamicRegistration states whether completion supports dynamic
		// registration.
		DynamicRegistration *bool `json:"dynamicRegistration,omitempty"`

		// CompletionItem determines whether the client supports the following
		// `CompletionItem` specific capabilities.
		CompletionItem *struct {
			// SnippetSupport states whether the client supports snippets as insert
			// text.
			//
			// A snippet can define tab stops and placeholders with `$1`, `$2`
			// and `${3:foo}`. `$0` defines the final tab stop, it defaults to
			// the end of the snippet. Placeholders with equal identifiers are linked,
			// that is typing in one will update others too.
			SnippetSupport *bool `json:"snippetSupport,omitempty"`
		} `json:"completionItem,omitempty"`
	} `json:"completion,omitempty"`

	// Hover determines capabilities specific to `textDocument/hover`
	Hover *DynamicRegistration `json:"hover,omitempty"`

	// SignatureHelp determines capabilities specific to `textDocument/signatureHelp`
	SignatureHelp *DynamicRegistration `json:"signatureHelp,omitempty"`

	// References determines capabilities specific to `textDocument/references`
	References *DynamicRegistration `json:"references,omitempty"`

	// DocumentHighlight determines capabilities specific to `textDocument/documentHighlight`
	DocumentHighlight *DynamicRegistration `json:"documentHighlight,omitempty"`

	// DocumentSymbol determines capabilities specific to `textDocument/documentSymbol`
	DocumentSymbol *DynamicRegistration `json:"documentSymbol,omitempty"`

	// Formatting determines capabilities specific to `textDocument/formatting`
	Formatting *DynamicRegistration `json:"formatting,omitempty"`

	// RangeFormatting determines capabilities specific to `textDocument/rangeFormatting`
	RangeFormatting *DynamicRegistration `json:"rangeFormatting,omitempty"`

	// OnTypeFormatting determines capabilities specific to `textDocument/onTypeFormatting`
	OnTypeFormatting *DynamicRegistration `json:"onTypeFormatting,omitempty"`

	// Definition determines capabilities specific to `textDocument/definition`
	Definition *DynamicRegistration `json:"definition,omitempty"`

	// CodeAction determines capabilities specific to `textDocument/codeAction`
	CodeAction *DynamicRegistration `json:"codeAction,omitempty"`

	// CodeLens determines capabilities specific to `textDocument/codeLens`
	CodeLens *DynamicRegistration `json:"codeLens,omitempty"`

	// DocumentLink determines capabilities specific to `textDocument/documentLink`
	DocumentLink *DynamicRegistration `json:"documentLink,omitempty"`

	// Rename determines capabilities specific to `textDocument/rename`
	Rename *DynamicRegistration `json:"rename,omitempty"`
}

// TextDocumentSyncKind defines how the host (editor) should sync document
// changes to the language server.
type TextDocumentSyncKind int

const (
	// None specifies that documents should not be synced at all.
	None TextDocumentSyncKind = iota

	// Full specifies that documents are synced by always sending the full
	// content of the document
	Full

	// Incremental specifies that documents are synced by sending the full
	// content on open; after that only incremental updates to the document are
	// send
	Incremental
)

type TextDocumentSyncOptions struct {
	OpenClose         bool                 `json:"openClose,omitempty"`
	Change            TextDocumentSyncKind `json:"change"`
	WillSave          bool                 `json:"willSave,omitempty"`
	WillSaveWaitUntil bool                 `json:"willSaveWaitUntil,omitempty"`
	Save              *SaveOptions         `json:"save,omitempty"`
}

// WorkspaceClientCapabilities contains specific client capabilities.
type WorkspaceClientCapabilities struct {
	// ApplyEdit determines whether the client supports applying batch edits to the workspace by supporting
	// the request 'workspace/applyEdit'
	ApplyEdit *bool `json:"applyEdit"`

	// WorkspaceEdit determines whether the client will handle
	WorkspaceEdit struct {
		// DocumentChanges declares whether the client supports versioned document changes
		DocumentChanges *bool `json:"documentChanges,omitempty"`
	} `json:"workspaceEdit,omitempty"`

	// DidChangeConfiguration specifies whether the `workspace/didChangeConfiguration` notification
	// can dynamically change
	DidChangeConfiguration *DynamicRegistration `json:"didChangeConfiguration,omitempty"`

	// DidChangeWatchedFiles specifies whether the `workspace/didChangeWatchedFiles` notification
	// can dynamically change
	DidChangeWatchedFiles *DynamicRegistration `json:"didChangeWatchedFiles"`

	// Symbol specifies whether the `workspace/symbol` notification can dynamically change
	Symbol *DynamicRegistration `json:"symbol,omitempty"`

	// ExecuteCommand specifies whether the `workspace/executeCommand` request
	// can dynamically change
	ExecuteCommand *DynamicRegistration `json:"executeCommand,omitempty"`
}

// Common?

// DocumentFilter denotes a document through properties like language, schema or pattern.
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#new-documentfilter
type DocumentFilter struct {
	// LanguageID is a language name, like `typescript`.
	Language *string `json:"language,omitempty"`

	// Scheme is a Uri [scheme](#Uri.scheme), like `file` or `untitled`.
	Scheme *string `json:"scheme,omitempty"`

	// Pattern is a glob pattern, like `*.{ts,js}`.
	Pattern *string `json:"pattern,omitempty"`
}

// DocumentSelector is the combination of one or more document filters.
type DocumentSelector []DocumentFilter

// DocumentURI is a document identifier
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#uri
type DocumentURI string

// Location is a spanning location inside a document
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#location
type Location struct {
	URI   DocumentURI `json:"uri"`
	Range Range       `json:"range"`
}

// LogMessageParams is used by the LogMessageNotification to send messages
// from the server to the client.
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#logmessage-notification
type LogMessageParams struct {
	// Type is the message type. See {@link MessageType}
	Type MessageType `json:"type"`

	// Message is the actual message
	Message string `json:"message"`
}

// MessageType is the type of message
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#showmessage-notification
type MessageType int

const (
	_ MessageType = iota

	// Error is an error message.
	Error

	// Warning is a warning message.
	Warning

	// Info is an information message.
	Info

	// Log is a log message.
	Log
)

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

// Registration is used to register for a capability.
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#new-register-capability
type Registration struct {
	// ID used to register the request. The id can be used to deregister the request again.
	ID string `json:"id"`

	// Method or capability to register for
	Method string `json:"method"`

	// RegisterOptions are options necessary for the registration.
	RegisterOptions *[]interface{} `json:"registerOptions"`
}

// RegistrationParams is an collection of Registration
type RegistrationParams struct {
	Registrations []Registration `json:"registrations"`
}

// ShowMessageParams allows the IDE to display a message to the user
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#showmessage-notification
type ShowMessageParams struct {
	// Type is the message type
	Type MessageType `json:"type"`

	// Message is the actual message
	Message string `json:"message"`
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

// TextDocumentRegistrationOptions are additional the text document options
type TextDocumentRegistrationOptions struct {
	// DocumentSelector is used to identify the scope of the registration. If set to null
	// the document selector provided on the client side will be used.
	DocumentSelector *DocumentSelector `json:"documentSelector,omitempty"`
}
