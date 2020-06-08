package recover

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/go-logr/logr"
)

// Panic recovers a panic
func Panic(log logr.Logger) {
	if e := recover(); e != nil {
		if log != nil {
			log.Info("recover from panic", e)
			log.Info(string(debug.Stack()))

		} else {
			fmt.Fprintln(os.Stderr, e)
			debug.PrintStack()
		}
	}
}
