# Tracecheck

## Description

Tracecheck is a Go linter that ensures proper use and formatting for common logger libraries:
- [kitlog](https://github.com/go-kit/log)
- [klog](https://github.com/kubernetes/klog)
- [logr](https://github.com/go-logr/logr)
- [zap](https://github.com/uber-go/zap)

Here's a summary of functionalities of Tracecheck:
- Check the odd number of key and value pairs for common logger libraries
- Check for the use of a traceId with the logger in functions that take a context as argument
- Add a traceId and spanId when absent using the -fix flag

It's recommended to use Tracecheck with [golangci-lint](https://golangci-lint.run/usage/linters/#loggercheck).

Based on [Loggercheck](https://github.com/timonwong/loggercheck#readme)

## Badges

[![License](https://img.shields.io/github/license/george-maroun/tracecheck.svg)](/LICENSE)
[![Release](https://img.shields.io/github/release/george-maroun/tracecheck.svg)](https://github.com/george-maroun/tracecheck/releases/latest)

## Install

```shell
go install github.com/george-maroun/tracecheck/cmd/tracecheck
```

## Usage

```
Tracecheck: Checks key value pairs for common logger libraries (kitlog,logr,klog,zap).

Usage: tracecheck [-flag] [package]


Flags:
  -V    print version and exit
  -all
        no effect (deprecated)
  -c int
        display offending line with this many lines of context (default -1)
  -cpuprofile string
        write CPU profile to this file
  -debug string
        debug flags, any subset of "fpstv"
  -disable value
        comma-separated list of disabled logger checker (kitlog,klog,logr,zap) (default kitlog)
  -fix
        apply all suggested fixes
  -flags
        print analyzer flags in JSON
  -json
        emit JSON output
  -memprofile string
        write memory profile to this file
  -noprintflike
        require printf-like format specifier not present in args
  -requirestringkey
        require all logging keys to be inlined constant strings
  -rulefile string
        path to a file contains a list of rules
  -source
        no effect (deprecated)
  -tags string
        no effect (deprecated)
  -test
        indicates whether test files should be analyzed, too (default true)
  -trace string
        write trace log to this file
  -v    no effect (deprecated)
```

## Example

Run: tracecheck -fix ./...

If a traceId is missing from the logger in a function that takes a context as arguments:
-> Tracecheck adds an import to go.opentelemetry.io/otel/trace, a span declaration, and traceId and spanId key-value arguments to the logger.

```go
package fix_import

import (
	"github.com/go-logr/zapr"
	"go.uber.org/zap"
	"context"
)

func SomeFunc(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	log := zapr.NewLogger(zap.L()).WithValues("eventType", eventType, "deliverID", deliveryID)
	return nil
}
```

```go
package fix_import

import (
	"context"
	"github.com/go-logr/zapr"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

func SomeFunc(ctx context.Context, eventType, deliveryID string, payload []byte) error {
	span := trace.SpanFromContext(ctx)
	log := zapr.NewLogger(zap.L()).WithValues("traceId", span.SpanContext().TraceID().String(), "spanId", span.SpanContext().SpanID().String(), "eventType", eventType, "deliverID", deliveryID)
	return nil
}

```

```
a.go:10:23: missing traceId in logging keys
```