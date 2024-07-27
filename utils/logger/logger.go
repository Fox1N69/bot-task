package logger

import (
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/sirupsen/logrus"
)

var e *logrus.Entry

type writeHook struct {
	Writer    []io.Writer
	LogLevels []logrus.Level
}

type Logger struct {
	*logrus.Entry
}

// Fire implements the logrus.Hook interface method to write log entries to
// multiple writers configured in writeHook.
//
// It converts the log entry into a string representation and writes it to each
// writer in hook.Writer.
//
// Parameters:
// - entry: Logrus log entry containing the log message and metadata.
//
// Returns an error if there was an issue converting the log entry to string
// or if there was an error while writing to any of the writers.
func (hook *writeHook) Fire(entry *logrus.Entry) error {
	line, err := entry.String()
	if err != nil {
		return err
	}

	for _, w := range hook.Writer {
		w.Write([]byte(line))
	}

	return err
}

// Levels returns the log levels for which this hook should be triggered.
//
// It returns the log levels configured in hook.LogLevels.
func (hook *writeHook) Levels() []logrus.Level {
	return hook.LogLevels
}

// init initializes the logging configuration using logrus library.
//
// It sets up a new logrus logger with caller reporting enabled, colorful output,
// and a custom CallerPrettyfier function. It creates a "logs" directory if it doesn't
// exist, opens or creates the "logs/all.log" file for logging, and configures log levels.
//
// Panics if there is an error while creating the "logs" directory or opening the log file.
func init() {
	l := logrus.New()
	l.SetReportCaller(true)

	l.Formatter = &logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
		CallerPrettyfier: func(f *runtime.Frame) (function, file string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	}

	err := os.MkdirAll("logs", 0755)
	if err != nil {
		panic(fmt.Errorf("failed to create logs directory: %w", err))
	}

	// Получаем имя текущей директории
	dir, err := os.Getwd()
	if err != nil {
		panic(fmt.Errorf("failed to get current working directory: %w", err))
	}

	// Формируем имя файла на основе имени директории
	logFileName := filepath.Join(dir, "logs", fmt.Sprintf("%s-error.log", filepath.Base(dir)))

	allFile, err := os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0640)
	if err != nil {
		panic(fmt.Errorf("failed to open log file: %w", err))
	}

	l.SetOutput(io.Discard)

	l.SetLevel(logrus.DebugLevel)
	l.AddHook(&writeHook{
		Writer:    []io.Writer{os.Stdout},
		LogLevels: logrus.AllLevels,
	})

	l.AddHook(&writeHook{
		Writer:    []io.Writer{allFile},
		LogLevels: []logrus.Level{logrus.ErrorLevel},
	})

	e = logrus.NewEntry(l)
}

// GetLogger returns a Logger instance initialized with the global logrus entry (e).
//
// It provides access to the global logger instance configured in the application.
func GetLogger() Logger {
	return Logger{e}
}

// GetLoggerWithFild returns a new Logger instance with an additional field added.
//
// Parameters:
// - k: Key of the field.
// - v: Value of the field.
//
// Returns a new Logger instance with the specified field added.
func (l *Logger) GetLoggerWithFild(k string, v interface{}) Logger {
	return Logger{l.WithField(k, v)}
}
