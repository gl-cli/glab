package iostreams

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/mattn/go-colorable"
	"github.com/mgutz/ansi"
)

type ColorPalette struct {
	// Magenta outputs ANSI color if stdout is a tty
	Magenta func(string) string
	// Cyan outputs ANSI color if stdout is a tty
	Cyan func(string) string
	// Red outputs ANSI color if stdout is a tty
	Red func(string) string
	// Yellow outputs ANSI color if stdout is a tty
	Yellow func(string) string
	// Blue outputs ANSI color if stdout is a tty
	Blue func(string) string
	// Green outputs ANSI color if stdout is a tty
	Green func(string) string
	// Gray outputs ANSI color if stdout is a tty
	Gray func(string) string
	// Bold outputs ANSI color if stdout is a tty
	Bold func(string) string
}

func (s *IOStreams) Color() *ColorPalette {
	colorEnabled := s.ColorEnabled()
	return &ColorPalette{
		Magenta: makeColorFunc(colorEnabled, "magenta"),
		Cyan:    makeColorFunc(colorEnabled, "cyan"),
		Red:     makeColorFunc(colorEnabled, "red"),
		Yellow:  makeColorFunc(colorEnabled, "yellow"),
		Blue:    makeColorFunc(colorEnabled, "blue"),
		Green:   makeColorFunc(colorEnabled, "green"),
		Gray:    makeColorFunc(colorEnabled, "black+h"),
		Bold:    makeColorFunc(colorEnabled, "default+b"),
	}
}

// NewColorable returns an output stream that handles ANSI color sequences on Windows
func NewColorable(out io.Writer) io.Writer {
	if outFile, isFile := out.(*os.File); isFile {
		return colorable.NewColorable(outFile)
	}
	return out
}

func makeColorFunc(colorEnabled bool, color string) func(string) string {
	isColorfulOutput := colorEnabled && isStdoutTerminal()

	if isColorfulOutput && color == "black+h" && Is256ColorSupported() {
		return func(t string) string {
			return fmt.Sprintf("\x1b[%d;5;%dm%s\x1b[m", 38, 242, t)
		}
	}

	cf := ansi.ColorFunc(color)
	return func(arg string) string {
		if isColorfulOutput {
			return cf(arg)
		}
		return arg
	}
}

// detectIsColorEnabled determines whether color output should be enabled based on environment variables.
// It follows the NO_COLOR specification (https://no-color.org/) with an override mechanism:
//
// - If NO_COLOR environment variable exists (with any value), color is disabled by default
// - If COLOR_ENABLED is set to "1" or "true", it overrides NO_COLOR and forces color to be enabled
// - If NO_COLOR doesn't exist, color is enabled by default
//
// This allows users to disable color globally with NO_COLOR while still providing an escape hatch
// via COLOR_ENABLED for specific use cases.
func detectIsColorEnabled() bool {
	// Check if NO_COLOR environment variable exists (any value disables color)
	_, noColorVarExists := os.LookupEnv("NO_COLOR")

	// If NO_COLOR exists, check if COLOR_ENABLED explicitly overrides it
	if noColorVarExists {
		colorEnabled := os.Getenv("COLOR_ENABLED")
		return colorEnabled == "1" || colorEnabled == "true"
	}

	// If NO_COLOR doesn't exist, color is enabled by default
	return true
}

func Is256ColorSupported() bool {
	term := os.Getenv("TERM")
	colorterm := os.Getenv("COLORTERM")

	return strings.Contains(term, "256") ||
		strings.Contains(term, "24bit") ||
		strings.Contains(term, "truecolor") ||
		strings.Contains(colorterm, "256") ||
		strings.Contains(colorterm, "24bit") ||
		strings.Contains(colorterm, "truecolor")
}
