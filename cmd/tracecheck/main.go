package main

import (
	"golang.org/x/tools/go/analysis/singlechecker"

	"github.com/jlewi/roboweb/tracecheck"
)

func main() {
	singlechecker.Main(loggercheck.NewAnalyzer())
}
