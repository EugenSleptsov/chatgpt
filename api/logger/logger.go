package logger

// Log is the basic logging contract (console output).
type Log interface {
	Log(message string)
	Logf(format string, v ...interface{})
}

// ErrorLog adds error-level helpers.
type ErrorLog interface {
	LogError(err error)
	LogFatal(err error)
}

// FileLog adds file-based logging.
type FileLog interface {
	LogToFile(file string, lines []string)
}

// Logger is the full logging contract combining all sub-interfaces.
// Use narrow sub-interfaces (Log, FileLog, etc.) in consumers that
// don't need the full set.
type Logger interface {
	Log
	ErrorLog
	FileLog
}
