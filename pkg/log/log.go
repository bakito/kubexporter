package log

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// ColorReset reset color.
	ColorReset = "\033[0m"
	// ColorGreen green.
	ColorGreen = "\033[32m"
	// Check a green check tick.
	Check = ColorGreen + "✓" + ColorReset
)

var nonASCII = regexp.MustCompile("[[:^ascii:]]")

// YALI yet another logger interface ;).
type YALI interface {
	Printf(format string, a ...any)
	Checkf(format string, a ...any)
}

// New logger.
func New(quiet, simple bool) YALI {
	return &log{
		quiet:  quiet,
		simple: simple,
	}
}

type log struct {
	quiet  bool
	simple bool
}

// Printf print a message.
func (l *log) Printf(format string, a ...any) {
	if !l.quiet {
		if l.simple {
			format = strings.ReplaceAll(format, "✓", "-")
			format = strings.ReplaceAll(format, ColorReset, "")
			format = strings.ReplaceAll(format, ColorGreen, "")
			format = nonASCII.ReplaceAllLiteralString(format, "")
			format = strings.ReplaceAll(format, "- \t", "\t")
		}
		_, _ = fmt.Printf(format, a...)
	}
}

// Checkf print a check message.
func (l *log) Checkf(format string, a ...any) {
	l.Printf(fmt.Sprintf("  %s %s", Check, format), a...)
}
