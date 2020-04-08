package common

import (
	"fmt"
	"log"
	"os"
	"path"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Logging utilities
// Use as: LogInfo.Printf(...)

type LoggingLevel int

var loggingLevel LoggingLevel = LogLevelUnknown
var logWriter *lumberjack.Logger

const (
	LogLevelUnknown LoggingLevel = iota
	LogLevelTrace
	LogLevelDebug
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

var (
	// Defaults
	LogTrace   *log.Logger = log.New(os.Stderr, "Trace ", log.LstdFlags)
	LogDebug   *log.Logger = log.New(os.Stderr, "Debug ", log.LstdFlags)
	LogInfo    *log.Logger = log.New(os.Stderr, "Info ", log.LstdFlags)
	LogWarning *log.Logger = log.New(os.Stderr, "Warning ", log.LstdFlags)
	LogError   *log.Logger = log.New(os.Stderr, "Error ", log.LstdFlags)
)

// logAndPrint logs and also writes to the standard output
type logAndPrint struct {
	logger *lumberjack.Logger
	level  LoggingLevel
}

func (l *logAndPrint) Write(data []byte) (int, error) {
	if l.level >= loggingLevel {
		msg := string(data)
		fmt.Print(msg)
		return l.logger.Write(data)
	}
	return 0, nil
}

func newLogAndPrint(level LoggingLevel) *logAndPrint {
	logAndPrint := logAndPrint{
		logger: logWriter,
		level:  level,
	}
	return &logAndPrint
}

// InitLogging initializes a rotating logger. This should be done once by the executable
// filePath is the full path of the log file
func InitLogging(filePath string) {
	logWriter = &lumberjack.Logger{
		Filename: filePath,
		// MaxSize is the maximum size in megabytes of the log file before it gets
		// rotated. It defaults to 100 megabytes.
		MaxSize: 50, // megabytes
		//MaxBackups: 4,
		MaxAge:   31 * 5, //days
		Compress: false,
	}

	// Make sure the directory exists
	err := os.MkdirAll(path.Dir(filePath), os.ModeDir|os.ModePerm)
	Abort(err)

	log.SetOutput(newLogAndPrint(LogLevelTrace))
	LogTrace = log.New(newLogAndPrint(LogLevelTrace), "Trace: ", log.LstdFlags)
	LogDebug = log.New(newLogAndPrint(LogLevelDebug), "Debug: ", log.LstdFlags)
	LogInfo = log.New(newLogAndPrint(LogLevelInfo), "Info: ", log.LstdFlags)
	LogWarning = log.New(newLogAndPrint(LogLevelWarning), "Warning: ", log.LstdFlags)
	LogError = log.New(newLogAndPrint(LogLevelError), "Error: ", log.LstdFlags)
	SetLoggingLevel(LogLevelTrace)
}

func GetLoggingLevel() LoggingLevel {
	return loggingLevel
}

func SetLoggingLevel(level LoggingLevel) {
	if level != loggingLevel {
		loggingLevel = level
		fmt.Printf("Logging level is %q\n", loggingLevel)
	}
}

func (l LoggingLevel) String() string {
	switch l {
	case LogLevelDebug:		return "debug"
	case LogLevelTrace:		return "trace"
	case LogLevelInfo:		return "info"
	case LogLevelWarning:	return"warning"
	case LogLevelError:		return "error"
	default:				return "unknown"
	}
}

func SetLoggingLevelFromName(level string) {
	switch strings.ToLower(level) {
	case "debug":
		SetLoggingLevel(LogLevelDebug)
	case "trace":
		SetLoggingLevel(LogLevelTrace)
	case "info":
		SetLoggingLevel(LogLevelInfo)
	case "warning":
		SetLoggingLevel(LogLevelWarning)
	case "error":
		SetLoggingLevel(LogLevelError)
	default:
		SetLoggingLevel(LogLevelInfo)
	}
}
