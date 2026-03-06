package logging

import (
	"fmt"
	"log"
)

var Level int = INFO

const (
	DEBUG  = 0
	INFO   = 1
	WARN   = 2
	ERROR  = 3
	SILENT = 4
)

func Debug(input string, args ...any) {
	if Level > DEBUG {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[DEBUG] - %s", msg)
}

func Info(input string, args ...any) {
	if Level > INFO {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[INFO] - %s", msg)
}

func Warning(input string, args ...any) {
	if Level > WARN {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[WARN] - %s", msg)
}

func Error(input string, args ...any) {
	if Level > ERROR {
		return
	}
	msg := fmt.Sprintf(input, args...)
	log.Printf("[ERROR] - %s", msg)
}
