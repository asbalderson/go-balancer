package logging

import (
	"fmt"
	"log"
)

var Level int = INFO

const (
	DEBUG = 0
	INFO  = 1
	WARN  = 2
	ERROR = 3
)

func Debug(input string, args ...interface{}) {
	if Level > DEBUG {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[DEBUG] - %s", msg)
}

func Info(input string, args ...interface{}) {
	if Level > INFO {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[INFO] - %s", msg)
}

func Warning(input string, args ...interface{}) {
	if Level > WARN {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[WARN] - %s", msg)
}

func Error(input string, args ...interface{}) {
	if Level > ERROR {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[ERROR] - %s", msg)
}
