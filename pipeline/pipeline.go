// Package pipeline defines the transport-agnostic types that flow
// through the entire handler pipeline: decoder → executor → command.
// It deliberately has ZERO internal dependencies so every layer can import it
// without pulling in the Telegram SDK or any infrastructure package.
package pipeline

// RequestContext is a pre-parsed, transport-agnostic view of an incoming
// message. It is created once at the edge (app/worker) and then threaded
// through decoder, executor and command layers unchanged.
type RequestContext struct {
	ChatID     int64
	MessageID  int
	SenderID   int64
	SenderName string
	Text       string // message text or caption, whichever is non-empty
	ChatTitle  string

	IsEdited  bool
	IsGroup   bool
	IsCommand bool
	IsPhoto   bool
	IsVoice   bool
	IsSticker bool

	// Command-specific (populated only when IsCommand == true).
	CommandName string
	CommandArgs string

	// Media-specific.
	PhotoFileID  string
	VoiceFileID  string
	StickerEmoji string
	Caption      string // original caption (before merging into Text)

	// Reply / forward metadata.
	ReplyToUsername string // username of the author of the replied-to message
	IsForwarded     bool
}

// FileInfo is a transport-agnostic descriptor for a remote file.
type FileInfo struct {
	FilePath string
}

// FileResolver resolves file IDs to downloadable URLs.
// Concrete implementations live in the transport layer (e.g. api/telegram).
type FileResolver interface {
	GetFile(fileID string) (FileInfo, error)
	FileURL(filePath string) string
}
