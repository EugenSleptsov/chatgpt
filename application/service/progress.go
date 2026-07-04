package service

// ProgressReporter delivers chat status messages outside the normal
// request→response cycle: transient "Идет …" notices removed when the work
// finishes, and permanent verbose announcements of tool invocations.
// Implemented by the Telegram bot. Callers hold it as an optional dependency —
// use the package-level StartProgress/Announce helpers, which tolerate nil.
type ProgressReporter interface {
	// StartProgress posts a transient status message and returns a function
	// that deletes it when the work completes.
	StartProgress(chatID int64, text string) (done func())
	// Announce posts a permanent informational message.
	Announce(chatID int64, text string)
}

// StartProgress posts a transient status via p, tolerating a nil reporter.
func StartProgress(p ProgressReporter, chatID int64, text string) func() {
	if p == nil {
		return func() {}
	}
	return p.StartProgress(chatID, text)
}

// Announce posts a permanent message via p, tolerating a nil reporter.
func Announce(p ProgressReporter, chatID int64, text string) {
	if p != nil {
		p.Announce(chatID, text)
	}
}
