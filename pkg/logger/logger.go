package logger

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	format string
	level  Level
}

var defaultLogger = &Logger{
	out:    os.Stderr,
	format: "text",
	level:  INFO,
}

func Init(out io.Writer, format string, debug bool) {
	if out == nil {
		out = os.Stderr
	}
	lvl := INFO
	if debug {
		lvl = DEBUG
	}
	defaultLogger = &Logger{
		out:    out,
		format: format,
		level:  lvl,
	}
}

func (l *Logger) log(lvl Level, prefix, msg string, args ...interface{}) {
	if lvl < l.level {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	formattedMsg := fmt.Sprintf(msg, args...)
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	if l.format == "json" {
		fmt.Fprintf(l.out, `{"time":"%s","level":"%s","message":%q}`+"\n", timestamp, prefix, formattedMsg)
	} else {
		colorReset := "\033[0m"
		colorCode := "\033[37m"
		switch lvl {
		case DEBUG:
			colorCode = "\033[36m" // Cyan
		case INFO:
			colorCode = "\033[32m" // Green
		case WARN:
			colorCode = "\033[33m" // Yellow
		case ERROR:
			colorCode = "\033[31m" // Red
		}
		fmt.Fprintf(l.out, "[%s] %s[%s]%s %s\n", timestamp, colorCode, prefix, colorReset, formattedMsg)
	}
}

func Debugf(msg string, args ...interface{}) {
	defaultLogger.log(DEBUG, "DEBUG", msg, args...)
}

func Infof(msg string, args ...interface{}) {
	defaultLogger.log(INFO, "INFO", msg, args...)
}

func Warnf(msg string, args ...interface{}) {
	defaultLogger.log(WARN, "WARN", msg, args...)
}

func Errorf(msg string, args ...interface{}) {
	defaultLogger.log(ERROR, "ERROR", msg, args...)
}
