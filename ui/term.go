package ui

import (
	"io"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

// ANSI styles. They are emitted only when colour is enabled.
const (
	ansiReset  = "\x1b[0m"
	ansiBold   = "\x1b[1m"
	ansiDim    = "\x1b[2m"
	ansiItalic = "\x1b[3m"
	ansiCyan   = "\x1b[36m"
	ansiYellow = "\x1b[33m"
	ansiMag    = "\x1b[35m"
	ansiGreen  = "\x1b[32m"
)

// IsTerminal reports whether f is a character device (a terminal), without
// depending on golang.org/x/term.
func IsTerminal(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// ColorEnabled follows the NO_COLOR convention (https://no-color.org).
func ColorEnabled(out *os.File) bool {
	if _, set := os.LookupEnv("NO_COLOR"); set {
		return false
	}
	return IsTerminal(out)
}

// TermWidth reads $COLUMNS (default 80, clamped to 40..200).
func TermWidth() int {
	w := 80
	if s := os.Getenv("COLUMNS"); s != "" {
		n := 0
		for _, r := range s {
			if r < '0' || r > '9' {
				n = 0
				break
			}
			n = n*10 + int(r-'0')
		}
		if n > 0 {
			w = n
		}
	}
	return max(40, min(w, 200))
}

// Wrap word-wraps text to width columns, keeping existing line breaks.
// Width is counted in runes, which is right for Latin and Cyrillic text.
func Wrap(text string, width int) string {
	var out strings.Builder
	for li, line := range strings.Split(text, "\n") {
		if li > 0 {
			out.WriteByte('\n')
		}
		indent := ""
		for _, r := range line {
			if r != ' ' {
				break
			}
			indent += " "
		}
		col := 0
		for wi, word := range strings.Fields(line) {
			n := utf8.RuneCountInString(word)
			switch {
			case wi == 0:
				out.WriteString(indent)
				col = len(indent)
			case col+1+n > width:
				out.WriteByte('\n')
				out.WriteString(indent)
				col = len(indent)
			default:
				out.WriteByte(' ')
				col++
			}
			out.WriteString(word)
			col += n
		}
	}
	return out.String()
}

// screen bundles output settings.
type screen struct {
	out        io.Writer
	color      bool
	width      int
	typewriter bool
	delay      time.Duration
	skip       <-chan string // lines typed while the typewriter runs skip it
	pending    []string      // lines consumed by a skip that were not empty
}

func (sc *screen) style(code, s string) string {
	if !sc.color || s == "" {
		return s
	}
	return code + s + ansiReset
}

func (sc *screen) print(s string) { io.WriteString(sc.out, s) }

func (sc *screen) println(s string) { io.WriteString(sc.out, s+"\n") }

// prose prints wrapped text, with the typewriter effect when enabled.
func (sc *screen) prose(text, code string) {
	wrapped := Wrap(text, sc.width-2)
	if !sc.typewriter || sc.skip == nil {
		sc.println(sc.style(code, wrapped))
		return
	}
	if sc.color && code != "" {
		sc.print(code)
	}
	runes := []rune(wrapped)
	for i, r := range runes {
		select {
		case line, ok := <-sc.skip:
			if ok && strings.TrimSpace(line) != "" {
				sc.pending = append(sc.pending, line)
			}
			sc.print(string(runes[i:]))
			if sc.color && code != "" {
				sc.print(ansiReset)
			}
			sc.print("\n")
			return
		default:
		}
		sc.print(string(r))
		if r != ' ' && r != '\n' {
			time.Sleep(sc.delay)
		}
	}
	if sc.color && code != "" {
		sc.print(ansiReset)
	}
	sc.print("\n")
}

// Hang wraps text and indents every line after the first by indent spaces.
func Hang(text string, width, indent int) string {
	lines := strings.Split(Wrap(text, width), "\n")
	if len(lines) <= 1 {
		return Wrap(text, width)
	}
	first := lines[0]
	rest := Wrap(strings.Join(lines[1:], " "), width-indent)
	pad := strings.Repeat(" ", indent)
	return first + "\n" + pad + strings.ReplaceAll(rest, "\n", "\n"+pad)
}
