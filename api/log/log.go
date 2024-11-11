package log

import (
	"GPTBot/util"
	"log"
)

type Log struct {
}

func NewLog() *Log {
	return &Log{}
}

func (l *Log) LogToFile(file string, lines []string) {
	// saving lines to log file
	err := util.AddLines(file, lines)
	if err != nil {
		return
	}
}

func (l *Log) LogError(err error) {
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
}

func (l *Log) LogFatal(err error) {
	if err != nil {
		log.Fatalf("Fatal error: %v\n", err)
	}
}

func (l *Log) LogSystemF(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *Log) LogSystem(v ...any) {
	log.Print(v)
}
