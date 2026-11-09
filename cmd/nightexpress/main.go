// Command nightexpress plays "Night Express", a text-only mystery set on an
// overnight train, in the terminal.
//
//	nightexpress                      title screen, choose language and difficulty
//	nightexpress --lang uz --new      start a new night in Uzbek straight away
//	nightexpress --script "1,1,2"     play a fixed sequence non-interactively
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"nightexpress/engine"
	"nightexpress/story"
	"nightexpress/ui"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	fs := flag.NewFlagSet("nightexpress", flag.ContinueOnError)
	lang := fs.String("lang", "", "language: uz, ru or en (default: ask on the title screen)")
	fast := fs.Bool("fast", false, "print text instantly (no typewriter effect)")
	script := fs.String("script", "", `play a comma-separated list of inputs non-interactively, e.g. "1,3,2,j,1"`)
	seed := fs.Int64("seed", 0, "seed for the few random (cosmetic) beats; scripts default to 1979")
	hard := fs.Bool("hard", false, "hard mode: the journal hides clue hints")
	newGame := fs.Bool("new", false, "skip the title menu and start a new night")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *lang != "" {
		ok := false
		for _, l := range engine.Languages {
			ok = ok || l == *lang
		}
		if !ok {
			fmt.Fprintf(os.Stderr, "unknown language %q (use uz, ru or en)\n", *lang)
			return 2
		}
	}
	st, err := engine.Load(story.FS)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	opt := ui.Options{
		Lang:  *lang,
		Hard:  *hard,
		New:   *newGame,
		Seed:  *seed,
		In:    os.Stdin,
		Out:   os.Stdout,
		Width: ui.TermWidth(),
		Color: ui.ColorEnabled(os.Stdout),
	}
	if fs.Lookup("script") != nil && isSet(fs, "script") {
		opt.Script = splitScript(*script)
		ending, err := ui.Run(st, opt)
		if err != nil {
			fmt.Fprintln(os.Stderr, "nightexpress:", err)
			return 1
		}
		if ending == "" {
			return 1
		}
		return 0
	}
	opt.Typewriter = !*fast && ui.IsTerminal(os.Stdout)
	if _, err := ui.Run(st, opt); err != nil && !errors.Is(err, ui.ErrQuit) {
		fmt.Fprintln(os.Stderr, "nightexpress:", err)
		return 1
	}
	return 0
}

func isSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) { set = set || f.Name == name })
	return set
}

// splitScript turns "1, 2,j , lang ru" into ["1","2","j","lang ru"].
func splitScript(s string) []string {
	out := []string{}
	for _, tok := range strings.Split(s, ",") {
		if tok = strings.TrimSpace(tok); tok != "" {
			out = append(out, tok)
		}
	}
	return out
}
