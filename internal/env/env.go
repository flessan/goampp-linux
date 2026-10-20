//go:build windows

// Package env holds the process-wide install folder and log sink shared by
// every layer.
package env

// Process-wide environment shared by every layer: where GoAMPP is installed
// and where log lines go. Lower layers use these instead of reaching into the
// application state.

// envBaseDir is the folder goampp.exe runs from.
var BaseDir string

// logSink receives log lines; the application points it at its log buffer.
var LogSink func(string)

func Log(line string) {
	if LogSink != nil {
		LogSink(line)
	}
}
