package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/george-maroun/tracecheck"
)

func main() {
	singlechecker.Main(loggercheck.NewAnalyzer())
}
