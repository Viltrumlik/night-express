// Package ui runs Night Express in a terminal: the title screen, the game
// loop, commands, and the non-interactive --script mode used by tests.
package ui

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"nightexpress/engine"
)

// Options configure a session.
type Options struct {
	Lang       string   // "uz", "ru" or "en"; "" asks on the title screen (script mode defaults to "en")
	Hard       bool     // hard mode: the journal hides clue hints
	New        bool     // skip the title menu and start a new night
	Script     []string // non-nil: play these inputs non-interactively and exit
	Seed       int64    // seed for the ambient lines (cosmetic randomness only)
	In         io.Reader
	Out        io.Writer
	Width      int  // wrap width in columns
	Color      bool // emit ANSI colour
	Typewriter bool // print prose letter by letter (interactive only)
}

// DefaultScriptSeed keeps --script transcripts reproducible when --seed is not given.
const DefaultScriptSeed = 1979

// ErrNoEnding is returned when a script runs out before the night ends.
var ErrNoEnding = errors.New("script ended before an ending was reached")

// ErrQuit is returned when the player quits.
var ErrQuit = errors.New("quit")

type session struct {
	st     *engine.Story
	opt    Options
	sc     *screen
	lang   string
	hard   bool
	seed   int64
	rng    *rand.Rand
	s      engine.State
	lines  <-chan string // interactive input
	script []string
	pos    int
}

// Run plays a session and returns the id of the ending reached ("" if none).
func Run(st *engine.Story, opt Options) (string, error) {
	if opt.Out == nil {
		opt.Out = io.Discard
	}
	if opt.Width <= 0 {
		opt.Width = 80
	}
	g := &session{st: st, opt: opt, lang: opt.Lang, hard: opt.Hard, seed: opt.Seed}
	g.sc = &screen{out: opt.Out, color: opt.Color, width: opt.Width, delay: 6 * time.Millisecond}
	if opt.Script != nil {
		g.script = opt.Script
		if g.lang == "" {
			g.lang = "en"
		}
		if g.seed == 0 {
			g.seed = DefaultScriptSeed
		}
		g.newNight()
		return g.loop()
	}
	if opt.In == nil {
		return "", errors.New("no input")
	}
	ch := make(chan string)
	go func() {
		sc := bufio.NewScanner(opt.In)
		for sc.Scan() {
			ch <- sc.Text()
		}
		close(ch)
	}()
	g.lines = ch
	g.sc.typewriter = opt.Typewriter
	g.sc.skip = ch
	if !g.title() {
		g.sc.println(g.t("ui.bye"))
		return "", ErrQuit
	}
	return g.loop()
}

func (g *session) t(key string) string { return g.st.Text(g.lang, key) }

func (g *session) newNight() {
	g.s = g.st.NewState()
	if g.seed == 0 {
		g.seed = time.Now().UnixNano()
	}
	g.rng = rand.New(rand.NewSource(g.seed))
}

// readLine returns the next input (script token or typed line).
func (g *session) readLine() (string, bool) {
	if g.script != nil {
		if g.pos >= len(g.script) {
			return "", false
		}
		tok := strings.TrimSpace(g.script[g.pos])
		g.pos++
		g.sc.println("> " + tok)
		return tok, true
	}
	if len(g.sc.pending) > 0 {
		l := g.sc.pending[0]
		g.sc.pending = g.sc.pending[1:]
		return strings.TrimSpace(l), true
	}
	l, ok := <-g.lines
	return strings.TrimSpace(l), ok
}

func (g *session) prompt() (string, bool) {
	if g.script == nil {
		g.sc.print(g.sc.style(ansiBold, g.t("ui.prompt")+" > "))
	}
	return g.readLine()
}

const trainArt = `
        ____                     _______________   _______________
   ____/_[]_\___________   ___  |  []  []  []  | |  []  []  []  |
  |  __    ___   ____   |=|   |=|              |=|              |
  |_|__|__|___|_|____|__| |___| |______________| |______________|
 ~~(@)~~(@)~~~~~~~(@)~~(@)~~~~~~~(@)~~~~~~(@)~~~~~~(@)~~~~~~(@)~~~
`

// title shows the title screen; it returns false if the player quits.
func (g *session) title() bool {
	sc := g.sc
	sc.println(sc.style(ansiYellow, trainArt))
	sc.println(sc.style(ansiBold, "  TUNGI EKSPRESS  ·  НОЧНОЙ ЭКСПРЕСС  ·  NIGHT EXPRESS"))
	sc.println("")
	if g.lang == "" {
		sc.println("  1. O'zbekcha    2. Русский    3. English")
		for g.lang == "" {
			l, ok := g.prompt()
			if !ok || l == "q" {
				g.lang = "en"
				return false
			}
			switch l {
			case "1", "uz":
				g.lang = "uz"
			case "2", "ru":
				g.lang = "ru"
			case "3", "en":
				g.lang = "en"
			}
		}
	}
	sc.println(sc.style(ansiBold, "  "+g.t("ui.title")))
	sc.println(sc.style(ansiDim, "  "+g.t("ui.subtitle")))
	sc.println("")
	if g.opt.New {
		g.newNight()
		return true
	}
	var save *engine.SaveData
	if engine.HasSave() {
		if d, err := engine.ReadSave(); err == nil {
			save = &d
		}
	}
	for {
		sc.println("  1. " + g.t("ui.menu.new"))
		sc.println("  2. " + g.t("ui.menu.hard"))
		if save != nil {
			clock := engine.FormatClock(save.Minutes, g.st.ClockStart)
			sc.println("  3. " + engine.Format(g.t("ui.menu.continue"), map[string]string{"time": clock}))
		}
		sc.println("  q. " + g.t("ui.menu.quit"))
		l, ok := g.prompt()
		if !ok || l == "q" {
			return false
		}
		switch {
		case l == "1":
			g.newNight()
			return true
		case l == "2":
			g.hard = true
			g.newNight()
			return true
		case l == "3" && save != nil:
			if err := g.restore(*save); err != nil {
				sc.println(engine.Format(g.t("ui.error"), map[string]string{"error": err.Error()}))
				continue
			}
			sc.println(g.t("ui.loaded"))
			return true
		}
	}
}

func (g *session) restore(d engine.SaveData) error {
	s, err := g.st.Restore(d)
	if err != nil {
		return err
	}
	g.s = s // the language stays whatever the player is using now
	g.hard = d.Hard
	g.seed = d.Seed
	if g.seed == 0 {
		g.seed = time.Now().UnixNano()
	}
	g.rng = rand.New(rand.NewSource(g.seed + int64(d.Minutes)))
	return nil
}

func (g *session) vars() map[string]string {
	s := &g.s
	left := max(0, g.st.Deadline-s.Minutes)
	return map[string]string{
		"time":      g.st.Clock(s),
		"left":      strconv.Itoa(left),
		"trust":     strconv.Itoa(s.Stats[engine.StatTrust]),
		"duty":      strconv.Itoa(s.Stats[engine.StatDuty]),
		"suspicion": strconv.Itoa(s.Stats[engine.StatSuspicion]),
		"money":     strconv.Itoa(s.Stats[engine.StatMoney]),
		"clues":     strconv.Itoa(g.st.CluesFound(s)),
		"evidence":  strconv.Itoa(g.st.KeyCluesFound(s)),
	}
}

func (g *session) fmt(key string) string { return engine.Format(g.t(key), g.vars()) }

func (g *session) statusBar() string {
	return "🕙 " + g.fmt("ui.status")
}

// show prints the current node and its numbered choices; it returns the
// available choice indices.
func (g *session) show() []int {
	sc := g.sc
	sc.println("")
	sc.prose(g.fmt(g.st.TextKey(&g.s)), "")
	n := g.st.Node(&g.s)
	if len(n.Ambient) > 0 && g.rng != nil {
		sc.println(sc.style(ansiItalic+ansiDim, Wrap(g.t(n.Ambient[g.rng.Intn(len(n.Ambient))]), sc.width-2)))
	}
	if n.Kind == engine.KindEnding {
		return nil
	}
	sc.println("")
	sc.println(sc.style(ansiDim, g.statusBar()))
	avail := g.st.Available(&g.s)
	for i, ci := range avail {
		c := n.Choices[ci]
		cost := engine.Format(g.t("ui.minutes"), map[string]string{"n": strconv.Itoa(c.Cost)})
		label := fmt.Sprintf("%2d. %s %s", i+1, g.t(c.LabelKey), sc.style(ansiDim, "("+cost+")"))
		sc.println(sc.style(ansiCyan, Hang(label, sc.width-2, 4)))
	}
	return avail
}

// loop runs the night until an ending, a quit, or the end of the script.
func (g *session) loop() (string, error) {
	for {
		if g.s.Ending != "" {
			g.showEnding()
			return g.s.Ending, nil
		}
		avail := g.show()
		if len(avail) == 0 {
			return "", fmt.Errorf("dead end at node %s", g.st.Node(&g.s).ID)
		}
		for acted := false; !acted; {
			l, ok := g.prompt()
			if !ok {
				if g.script != nil {
					g.sc.println(g.t("ui.script.end"))
					return "", ErrNoEnding
				}
				return "", ErrQuit
			}
			done, err := g.command(l, avail, &acted)
			if err != nil {
				return "", err
			}
			if done {
				return "", ErrQuit
			}
		}
	}
}

// command handles one line of input. It sets *acted when the story moved on.
func (g *session) command(l string, avail []int, acted *bool) (quit bool, err error) {
	sc := g.sc
	fields := strings.Fields(strings.ToLower(l))
	if len(fields) == 0 {
		return false, nil
	}
	switch fields[0] {
	case "q", "quit", "exit":
		sc.println(g.t("ui.quit"))
		return true, nil
	case "h", "help", "?":
		sc.println(Wrap(g.t("ui.help"), sc.width-2))
		return false, nil
	case "s", "status":
		sc.println(sc.style(ansiGreen, g.fmt("ui.status.full")))
		return false, nil
	case "j", "journal":
		g.journal()
		return false, nil
	case "save":
		if err := engine.WriteSave(g.st.Snapshot(&g.s, g.lang, g.hard, g.seed)); err != nil {
			sc.println(engine.Format(g.t("ui.error"), map[string]string{"error": err.Error()}))
		} else {
			sc.println(g.t("ui.saved"))
		}
		return false, nil
	case "load":
		d, err := engine.ReadSave()
		if err != nil {
			sc.println(g.t("ui.nosave"))
			return false, nil
		}
		if err := g.restore(d); err != nil {
			sc.println(engine.Format(g.t("ui.error"), map[string]string{"error": err.Error()}))
			return false, nil
		}
		sc.println(g.t("ui.loaded"))
		*acted = true // redraw the loaded scene
		return false, nil
	case "lang", "til", "язык":
		want := ""
		if len(fields) > 1 {
			want = fields[1]
		} else {
			sc.println(g.t("ui.lang.ask"))
			l2, ok := g.prompt()
			if !ok {
				return false, nil
			}
			want = strings.ToLower(strings.TrimSpace(l2))
		}
		for _, lg := range engine.Languages {
			if lg == want {
				g.lang = lg
				sc.println(g.t("ui.lang.changed"))
				*acted = true // redraw in the new language
				return false, nil
			}
		}
		sc.println(g.t("ui.unknown"))
		return false, nil
	}
	n, convErr := strconv.Atoi(fields[0])
	if convErr != nil || n < 1 || n > len(avail) {
		if g.script != nil {
			return false, fmt.Errorf("script token %q is not a valid choice here (node %s, %d choices)",
				l, g.st.Node(&g.s).ID, len(avail))
		}
		sc.println(g.t("ui.unknown"))
		return false, nil
	}
	step, err := g.st.Choose(&g.s, avail[n-1])
	if err != nil {
		return false, err
	}
	*acted = true
	if step.ResultKey != "" {
		sc.println("")
		sc.prose(g.fmt(step.ResultKey), ansiYellow)
	}
	if step.Arrival {
		sc.println("")
		sc.println(sc.style(ansiMag, g.t("ui.deadline")))
	} else if step.EventID != "" {
		sc.println("")
		sc.println(sc.style(ansiMag, g.fmt("ui.event")))
	}
	return false, nil
}

func (g *session) journal() {
	sc := g.sc
	s := &g.s
	sc.println(sc.style(ansiBold, g.t("ui.journal.title")))
	found := g.st.CluesFound(s)
	sc.println(engine.Format(g.t("ui.journal.clues"), map[string]string{
		"found": strconv.Itoa(found), "total": strconv.Itoa(len(g.st.Clues)),
	}))
	if found == 0 {
		sc.println("  " + g.t("ui.journal.empty"))
	}
	for _, c := range g.st.Clues {
		if !s.Flags.Has(c.Idx) {
			continue
		}
		line := "  • " + g.t("clue."+c.Flag+".name")
		if !g.hard {
			if c.Key {
				line += " [" + g.t("ui.journal.key") + "]"
			}
			line += " — " + g.t("clue."+c.Flag+".hint")
		}
		sc.println(Wrap(line, sc.width-2))
	}
	sc.println(g.t("ui.journal.notes"))
	shown := false
	for _, n := range g.st.Notes {
		if s.Flags.Has(n.Idx) {
			shown = true
			sc.println(Wrap("  – "+g.t("note."+n.Flag), sc.width-2))
		}
	}
	if !shown {
		sc.println("  " + g.t("ui.journal.empty"))
	}
}

func (g *session) showEnding() {
	sc := g.sc
	id := g.s.Ending
	num := g.st.EndingIndex(id)
	title := g.t("ending." + id + ".title")
	sc.println("")
	sc.prose(g.fmt(g.st.TextKey(&g.s)), "")
	card := engine.Format(g.t("ui.ending.card"), map[string]string{"num": strconv.Itoa(num), "title": title})
	bar := strings.Repeat("═", min(sc.width-2, max(20, len([]rune(card))+4)))
	sc.println("")
	sc.println(sc.style(ansiYellow, bar))
	sc.println(sc.style(ansiBold+ansiYellow, "  "+card))
	sc.println(sc.style(ansiYellow, bar))

	prog, _ := engine.ReadProgress()
	isNew := prog.Add(id)
	if err := engine.WriteProgress(prog); err != nil {
		sc.println(engine.Format(g.t("ui.error"), map[string]string{"error": err.Error()}))
	}
	if g.script == nil {
		_ = engine.RemoveSave() // the night is over; Continue no longer makes sense
	}
	mins := g.s.Minutes
	v := g.vars()
	v["hours"] = strconv.Itoa(mins / 60)
	v["mins"] = strconv.Itoa(mins % 60)
	v["count"] = strconv.Itoa(len(prog.Endings))
	v["list"] = strings.Join(prog.Endings, ", ")
	sc.println(engine.Format(g.t("ui.summary"), v))
	if isNew {
		sc.println(sc.style(ansiGreen, g.t("ui.summary.new")))
	}
}
