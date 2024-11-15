package log

type Log interface {
	Log(message string)
	Logf(format string, v ...interface{})
}

type ErrorLog interface {
	LogError(err error)
	LogFatal(err error)
}

type FileLog interface {
	LogToFile(file string, lines []string)
}
