package agent

import (
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	cReset  = "\033[0m"
	cDim    = "\033[2m"
	cBold   = "\033[1m"
	cGreen  = "\033[32m"
	cYellow = "\033[33m"
	cRed    = "\033[31m"
	cCyan   = "\033[36m"
	cBlue   = "\033[34m"
)

// Logger is a small colorized CLI logger.
type Logger struct {
	out     io.Writer
	err     io.Writer
	color   bool
	verbose bool
}

// NewLogger writes to stdout/stderr.
func NewLogger(verbose bool) *Logger {
	color := os.Getenv("NO_COLOR") == "" && isTTY(os.Stdout)
	return &Logger{out: os.Stdout, err: os.Stderr, color: color, verbose: verbose}
}

func isTTY(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

func (l *Logger) paint(code, s string) string {
	if l == nil || !l.color {
		return s
	}
	return code + s + cReset
}

func (l *Logger) Infof(format string, args ...any) {
	l.printf(l.out, l.paint(cCyan, "▸ ")+format, args...)
}

func (l *Logger) Actionf(format string, args ...any) {
	l.printf(l.out, l.paint(cBlue, "  → ")+format, args...)
}

func (l *Logger) Okf(format string, args ...any) {
	l.printf(l.out, l.paint(cGreen, "  ✓ ")+format, args...)
}

func (l *Logger) Warnf(format string, args ...any) {
	l.printf(l.out, l.paint(cYellow, "  ! ")+format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	l.printf(l.err, l.paint(cRed, "  ✗ ")+format, args...)
}

func (l *Logger) Debugf(format string, args ...any) {
	if l == nil || !l.verbose {
		return
	}
	l.printf(l.out, l.paint(cDim, "  · ")+format, args...)
}

func (l *Logger) Banner(s string) {
	line := strings.Repeat("─", 56)
	l.printf(l.out, l.paint(cDim, line))
	l.printf(l.out, l.paint(cBold, s))
}

func (l *Logger) Print(s string) {
	if l == nil {
		fmt.Fprint(os.Stdout, s)
		return
	}
	fmt.Fprint(l.out, s)
	if !strings.HasSuffix(s, "\n") {
		fmt.Fprintln(l.out)
	}
}

func (l *Logger) printf(w io.Writer, format string, args ...any) {
	if l == nil {
		fmt.Fprintf(os.Stdout, format+"\n", args...)
		return
	}
	fmt.Fprintf(w, format+"\n", args...)
}

// SilentLogger discards all output (tests).
func SilentLogger() *Logger {
	return &Logger{out: io.Discard, err: io.Discard, color: false, verbose: false}
}
