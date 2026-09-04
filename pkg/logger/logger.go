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

type HookFunc func(level Level, prefix, message string)

type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	format string
	level  Level
	hooks  []HookFunc
}

var defaultLogger = &Logger{
	out:    os.Stderr,
	format: "text",
	level:  INFO,
	hooks:  make([]HookFunc, 0),
}

func Init(out io.Writer, format string, debug bool) {
	if out == nil {
		out = os.Stderr
	}
	lvl := INFO
	if debug {
		lvl = DEBUG
	}
	defaultLogger.mu.Lock()
	defaultLogger.out = out
	defaultLogger.format = format
	defaultLogger.level = lvl
	defaultLogger.mu.Unlock()
}

func AddHook(h HookFunc) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()
	defaultLogger.hooks = append(defaultLogger.hooks, h)
}

func (l *Logger) log(lvl Level, prefix, msg string, args ...interface{}) {
	if lvl < l.level {
		return
	}
	l.mu.Lock()
	formattedMsg := fmt.Sprintf(msg, args...)
	timestamp := time.Now().Format("15:04:05.000")

	// Trigger registered hooks (e.g. IPC broadcast)
	for _, hook := range l.hooks {
		hook(lvl, prefix, formattedMsg)
	}

	if l.format == "json" {
		fmt.Fprintf(l.out, `{"time":"%s","level":"%s","message":%q}`+"\n", timestamp, prefix, formattedMsg)
	} else {
		colorReset := "\033[0m"
		dimColor := "\033[90m"
		var colorCode string
		var icon string
		switch lvl {
		case DEBUG:
			colorCode = "\033[36m" // Cyan
			icon = "🔍"
		case INFO:
			colorCode = "\033[32m" // Green
			icon = "🦫"
		case WARN:
			colorCode = "\033[33m" // Yellow
			icon = "⚠️"
		case ERROR:
			colorCode = "\033[31m" // Red
			icon = "🚨"
		}
		fmt.Fprintf(l.out, "%s%s%s %s[%s]%s %s\n", dimColor, timestamp, colorReset, colorCode, icon+" "+prefix, colorReset, formattedMsg)
	}
	l.mu.Unlock()
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
