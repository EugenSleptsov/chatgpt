package logger

import (
	"GPTBot/infrastructure/util"
	"log"
)

type SystemLog struct {
}

func NewSystem() *SystemLog {
	return &SystemLog{}
}

func (l *SystemLog) Log(message string) {
	log.Print(message)
}

func (l *SystemLog) Logf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *SystemLog) LogToFile(file string, lines []string) {
	err := util.AddLines(file, lines)
	if err != nil {
		return
	}
}
