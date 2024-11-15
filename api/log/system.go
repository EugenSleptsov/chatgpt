package log

import (
	"GPTBot/util"
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

func (l *SystemLog) LogFatal(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func (l *SystemLog) LogError(err error) {
	if err != nil {
		log.Print(err)
	}
}

func (l *SystemLog) LogToFile(file string, lines []string) {
	err := util.AddLines(file, lines)
	if err != nil {
		return
	}
}
