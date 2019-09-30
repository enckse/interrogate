package internal

import (
	"fmt"
)

// Error writes an error message
func Error(message string, err error) {
	fmt.Println(fmt.Sprintf("%s (%v)", message, err))
}

// Fatal writes a message and panics
func Fatal(message string, err error) {
	Error(message, err)
	panic("fatal error ^")
}

// Info writes informational messages
func Info(message string) {
	fmt.Println(message)
}
