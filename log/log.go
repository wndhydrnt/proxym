package log

import (
	"github.com/op/go-logging"
	"os"
	"strings"
)

const defaultLogFormat = "%{time:02.01.2006 15:04:05} [%{level}] %{longfunc}: %{message}"

var AppLog = logging.MustGetLogger("appLog")
var ErrorLog = logging.MustGetLogger("errorLog")

var stderr = logging.NewLogBackend(os.Stderr, "", 0)
var stdout = logging.NewLogBackend(os.Stdout, "", 0)

func logFormatFromString(ev string) string {
	if ev == "" {
		return defaultLogFormat
	}

	return ev
}

func logLevelFromString(ev string) logging.Level {
	ev = strings.ToLower(ev)

	switch ev {
	case "critical":
		return logging.CRITICAL
	case "debug":
		return logging.DEBUG
	case "error":
		return logging.ERROR
	case "info":
		return logging.INFO
	case "notice":
		return logging.NOTICE
	case "warning":
		return logging.WARNING
	default:
		return logging.INFO
	}
}

func init() {
	appLogLevel := logLevelFromString(os.Getenv("PROXYM_LOG_APPLOG_LEVEL"))

	format := logFormatFromString(os.Getenv("PROXYM_LOG_FORMAT"))

	formatter := logging.MustStringFormatter(format)

	stdoutFormatter := logging.NewBackendFormatter(stdout, formatter)
	stderrFormatter := logging.NewBackendFormatter(stderr, formatter)

	stderrLeveled := logging.AddModuleLevel(stderrFormatter)
	stderrLeveled.SetLevel(logging.ERROR, "")

	stdoutLeveled := logging.AddModuleLevel(stdoutFormatter)
	stdoutLeveled.SetLevel(appLogLevel, "")

	AppLog.SetBackend(stdoutLeveled)
	ErrorLog.SetBackend(stderrLeveled)
}
