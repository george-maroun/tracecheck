package all

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
)

func ExampleLogRTraceID() {
	log := logr.Discard()
	log = log.WithValues("key", "value")
}

func ExampleInvalid() {
	// function pointer is not supported
	log := logr.Discard()
	logFn := log.Info
	logFn("message", "key1") // cannot be detected
}

func ExampleLogr() {
	err := fmt.Errorf("error")

	log := logr.Discard()
	log = log.WithValues("traceId", "traceVal", "key")   // want `odd number of arguments passed as key-value pairs for logging`

	log3 := logr.FromContextOrDiscard(context.TODO())
	args := []interface{}{"abc"}
	log3.Error(err, "message", args...) // not supported
}
