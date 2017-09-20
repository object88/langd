package requests

import "fmt"

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

// DidChangeConfigurationParams contains all settings that have changed
type DidChangeConfigurationParams struct {
	// Settings are the changed settings
	Settings map[string]interface{} `json:"settings"`
}

// DidChangeTextDocumentParams is supplied by the client to describe the
// change or changes made to a text document
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#didchangetextdocument-notification
type DidChangeTextDocumentParams struct {
	/**
	 * The document that did change. The version number points
	 * to the version after all provided content changes have
	 * been applied.
	 */
	TextDocument VersionedTextDocumentIdentifier `json:"textDocument"`

	/**
	 * The actual content changes. The content changes descibe single state changes
	 * to the document. So if there are two content changes c1 and c2 for a document
	 * in state S10 then c1 move the document to S11 and c2 to S12.
	 */
	ContentChanges []TextDocumentContentChangeEvent `json:"contentChanges"`
}

// DidCloseTextDocumentParams is sent from the client to the server when the
// document got closed in the client
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#didclosetextdocument-notification
type DidCloseTextDocumentParams struct {
	// TextDocument specifies the document that was closed.
	TextDocument TextDocumentIdentifier `json:"textDocument"`
}

// DidOpenTextDocumentParams is supplied by the client to describe when a
// document is opened in the editor
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#didopentextdocument-notification
type DidOpenTextDocumentParams struct {
	// TextDocument is the document that was opened.
	TextDocument TextDocumentItem `json:"textDocument"`
}

// DidSaveTextDocumentParams is supplied by the client when a file is written
// to disk
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#didsavetextdocument-notification
type DidSaveTextDocumentParams struct {
	// TextDocument is the document that was saved.
	TextDocument TextDocumentIdentifier `json:"textDocument"`

	// Text is the content when saved, and is optional. Depends on the
	// includeText value when the save notifcation was requested.
	Text *string `json:"text,omitempty"`
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

// TextDocumentContentChangeEvent is an event describing a change to a text
// document. If range and rangeLength are omitted the new text is considered
// to be the full content of the document.
type TextDocumentContentChangeEvent struct {
	// Range states the range of the document that changed.
	Range *Range `json:"range,omitempty"`

	// RangeLength is the length of the range that got replaced.
	RangeLength *int `json:"rangeLength,omitempty"`

	// Text is the new text of the range/document.
	Text string `json:"text"`
}

func (tdcce *TextDocumentContentChangeEvent) String() string {
	text := tdcce.Text
	if len(text) > 32 {
		text = fmt.Sprintf("%s...", text[:32])
	}

	if nil == tdcce.Range && nil == tdcce.RangeLength {
		// Replacing the whole doc...
		return fmt.Sprintf("whole doc: %s", text)
	}

	return fmt.Sprintf("%s: len: %d: '%s'", tdcce.Range.String(), *tdcce.RangeLength, text)
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

// TextDocumentSyncOptions specify what the server can handle with regard to
// changes to a text document
type TextDocumentSyncOptions struct {
	// OpenClose determine whether open & close notifications are sent to the
	// server
	OpenClose bool `json:"openClose,omitempty"`

	// Change determines which type of change notifications are sent to the server
	Change TextDocumentSyncKind `json:"change"`

	// WillSave determine whether will-save notifications are sent to the server
	WillSave bool `json:"willSave,omitempty"`

	// WillSaveWaitUntil determines whether will-save-wait-until requests are
	// sent to the server
	WillSaveWaitUntil bool `json:"willSaveWaitUntil,omitempty"`

	// Save specifies what data is sent along with a save notification
	Save *SaveOptions `json:"save,omitempty"`
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

func (r *Range) String() string {
	return fmt.Sprintf("[%d,%d:%d,%d]", r.Start.Line, r.Start.Character, r.End.Line, r.End.Character)
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

// TextDocumentItem is an item to trasnfer a text document from the client
// to the server
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#textdocumentitem
type TextDocumentItem struct {
	// URI is the text document's URI
	URI DocumentURI `json:"uri"`

	// LanguageID is the text document's language identifier
	LanguageID string `json:"languageId"`

	// Version is the version number of this document (it will increase after
	// each change, including undo/redo)
	Version int `json:"version"`

	// Text is the content of the opened text document
	Text string `json:"text"`
}

// TextDocumentPositionParams is a parameter literal used in requests to pass
// a text document and a position inside that document.
// https://github.com/Microsoft/language-server-protocol/blob/master/protocol.md#textdocumentpositionparams
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

// VersionedTextDocumentIdentifier is a TextDocumentIdenfigier with a
// version number
// NOTE: figure out how to do JSON marshalling so that the structure is flattened.
type VersionedTextDocumentIdentifier struct {
	TextDocumentIdentifier

	// Version is the version number of this document.
	Version int `json:"version"`
}
