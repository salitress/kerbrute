package util

import (
	"os"

	"github.com/op/go-logging"
)

type Logger struct {
	Log *logging.Logger
}

func NewLogger(verbose bool, logFileName string, jsonMode bool) Logger {
	log := logging.MustGetLogger("kerbrute")
	format := logging.MustStringFormatter(
		"\r" + `%{color}%{time:2006/01/02 15:04:05} >  %{message}%{color:reset}`,
	)
	formatNoColor := logging.MustStringFormatter(
		"\r" + `%{time:2006/01/02 15:04:05} >  %{message}`,
	)
	// In --json mode, stdout is reserved for the NDJSON result stream, so
	// human-readable chatter (progress, info, errors) goes to stderr instead.
	logDest := os.Stdout
	if jsonMode {
		logDest = os.Stderr
	}
	backend := logging.NewLogBackend(logDest, "", 0)
	backendFormatter := logging.NewBackendFormatter(backend, format)

	if logFileName != "" {
		outputFile, err := os.Create(logFileName)
		if err != nil {
			panic(err)
		}
		fileBackend := logging.NewLogBackend(outputFile, "", 0)
		fileFormatter := logging.NewBackendFormatter(fileBackend, formatNoColor)
		logging.SetBackend(backendFormatter, fileFormatter)
	} else {
		logging.SetBackend(backendFormatter)
	}

	if verbose {
		logging.SetLevel(logging.DEBUG, "")
	} else {
		logging.SetLevel(logging.INFO, "")
	}
	return Logger{log}
}
