package server

import (
	"fmt"
	"io"
	"os"

	"github.com/rs/zerolog"
)

const (
	EnvLogFilePath = "LOG_FILE_PATH"
	EnvLogLevel    = "LOG_LEVEL"
)

// LoadLogger loads the logger from env.
func LoadLogger() (zerolog.Logger, error) {
	logWritter, err := getLogWriter()
	if err != nil {
		return zerolog.Nop(), err
	}
	logLvl := getLogLevel()
	return zerolog.New(logWritter).Level(logLvl).With().Timestamp().Logger(), nil
}

func getLogWriter() (io.Writer, error) {
	logFilePath := os.Getenv(EnvLogFilePath)
	if len(logFilePath) == 0 {
		fmt.Printf("Env %s, not found logging to Stdout\n", EnvLogFilePath)
		return os.Stdout, nil
	}
	// Open a file for writing logs
	logFile, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return nil, err
	}

	fmt.Printf("\nlog file path %s found in env %s", logFilePath, EnvLogFilePath)
	return logFile, nil
}

func getLogLevel() zerolog.Level {
	logLevelStr := os.Getenv(EnvLogLevel)
	lvl, err := zerolog.ParseLevel(logLevelStr)
	if err != nil || len(lvl.String()) == 0 {
		fmt.Printf("not possible to load log level in env %s, using error level\n", EnvLogLevel)
		return zerolog.ErrorLevel
	}
	fmt.Printf("log level %s parsed from env %s\n", lvl.String(), EnvLogLevel)
	return lvl
}
