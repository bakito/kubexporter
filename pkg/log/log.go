package log

import "fmt"

const (
	// ColorReset reset color
	ColorReset = "\033[0m"
	// ColorGreen green
	ColorGreen = "\033[32m"
	// Check a green check tick
	Check = ColorGreen + "✓" + ColorReset
)

// YALI yet another logger interface ;)
type YALI interface {
	Printf(format string, a ...interface{})
	Checkf(format string, a ...interface{})
}

// New logger
func New(quiet bool) YALI {
	return &log{
		quiet: quiet,
	}
}

type log struct {
	quiet bool
}

// Printf print a message
func (l *log) Printf(format string, a ...interface{}) {
	if !l.quiet {
		fmt.Printf(format, a...)
	}
}

// Checkf print a check message
func (l *log) Checkf(format string, a ...interface{}) {
	l.Printf(fmt.Sprintf("  %s %s", Check, format), a...)
}
