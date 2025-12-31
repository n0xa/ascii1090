package debug

import (
	"fmt"
	"io"
)

var writer io.Writer = io.Discard

// SetOutput sets the debug output destination
func SetOutput(w io.Writer) {
	writer = w
}

// Log writes a debug message
func Log(format string, args ...interface{}) {
	fmt.Fprintf(writer, format+"\n", args...)
}

// Enabled returns true if debug logging is enabled
func Enabled() bool {
	return writer != io.Discard
}
