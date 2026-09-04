package logger

import (
	"fmt"
	"io"
	"os"
	"strings"
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
	defer l.mu.Unlock()

	formattedMsg := fmt.Sprintf(msg, args...)
	timestamp := time.Now().Format("15:04:05.000")

	// Trigger registered hooks (e.g. IPC broadcast)
	for _, hook := range l.hooks {
		hook(lvl, prefix, formattedMsg)
	}

	if l.format == "json" {
		fmt.Fprintf(l.out, `{"time":"%s","level":"%s","message":%q}`+"\n", timestamp, prefix, formattedMsg)
		return
	}

	// ANSI Styles & Colors
	reset := "\033[0m"
	dim := "\033[90m"
	bold := "\033[1m"

	cyan := "\033[36m"
	green := "\033[32m"
	yellow := "\033[33m"
	red := "\033[31m"
	magenta := "\033[35m"
	blue := "\033[34m"

	var tagColor string
	var icon string
	switch lvl {
	case DEBUG:
		tagColor = cyan
		icon = "🔍"
	case INFO:
		tagColor = green
		icon = "🦫"
	case WARN:
		tagColor = yellow
		icon = "⚠️"
	case ERROR:
		tagColor = red
		icon = "🚨"
	}

	// Highlight specialized sections
	displayMsg := formattedMsg
	if strings.Contains(displayMsg, "On-Chain") || strings.Contains(displayMsg, "Broadcasted") {
		displayMsg = magenta + bold + displayMsg + reset
	} else if strings.Contains(displayMsg, "Order executed") || strings.Contains(displayMsg, "Payout claimed") {
		displayMsg = green + bold + displayMsg + reset
	} else if strings.Contains(displayMsg, "Signal on Market") {
		displayMsg = blue + displayMsg + reset
	} else if strings.Contains(displayMsg, "Arbitrage Edge") || strings.Contains(displayMsg, "Parity edge") {
		displayMsg = yellow + bold + displayMsg + reset
	}

	tag := fmt.Sprintf("%s%s[%s %-4s]%s", bold, tagColor, icon, prefix, reset)
	fmt.Fprintf(l.out, "  %s%s%s %s  %s\n", dim, timestamp, reset, tag, displayMsg)
}

func Section(title string) {
	defaultLogger.mu.Lock()
	defer defaultLogger.mu.Unlock()

	reset := "\033[0m"
	bold := "\033[1m"
	cyan := "\033[36m"
	line := strings.Repeat("─", 50)

	fmt.Fprintf(defaultLogger.out, "\n  %s%s┌%s┐%s\n", bold, cyan, line, reset)
	fmt.Fprintf(defaultLogger.out, "  %s%s│  📌 %-46s│%s\n", bold, cyan, title, reset)
	fmt.Fprintf(defaultLogger.out, "  %s%s└%s┘%s\n\n", bold, cyan, line, reset)
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
