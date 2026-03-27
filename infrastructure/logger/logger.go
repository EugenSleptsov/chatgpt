package logger

// Log is the basic logging contract (console output).
type Log interface {
	Log(message string)
	Logf(format string, v ...interface{})
}

// FileLog adds file-based logging.
type FileLog interface {
	LogToFile(file string, lines []string)
}
