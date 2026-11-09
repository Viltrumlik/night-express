package ui_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"nightexpress/engine"
	"nightexpress/story"
	"nightexpress/ui"
)

// The same numeric scripts the README documents (all eight endings; the brief
// requires at least E1, E5, E7 and E8). They go through ui.Run,
// exactly the code path of `nightexpress --script`.
var scripts = map[string]string{
	"E1": "2,1,1,1,1,5,1,1,1,1,2,7,1,1,1,2,3,1,1,2,2,7,2,1,2,2,3,2,2,1,8,1,1,8,1,8,8,1,1",
	"E2": "2,1,1,1,1,5,1,1,1,1,2,7,1,1,1,2,3,1,1,2,2,7,2,1,2,2,3,8,8,1,1,8,1,8,8,1,1",
	"E3": "2,1,1,1,1,5,1,1,9,1,8,2,8,3,3,1,8,8,3,8,8,8,8,8,2",
	"E4": "1,1,1,1,1,5,2,6,1,1,5,2,5,1,1,1,6,1,1,8,3,8,8,8,8,8,1,1,8,1,8,8,1,2",
	"E5": "3,1,1,1,1,5,2,1,1,9,2,9,1,8,3,8,8,8,8,8,1,1",
	"E6": "3,1,1,1,1,4,1,2,3,1,2,2,2,1,2,1,3,2,1,1,1",
	"E7": "3,1,2,1,1,5,2,7,2,1,2,6,1",
	"E8": "1,1,1,1,1,5,2,6,1,1,5,2,5,1,1,1,6,1,1,8,3,8,8,8,8,8,1,1,8,1,8,8,1,1",
}

func load(t *testing.T) *engine.Story {
	t.Helper()
	st, err := engine.Load(story.FS)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func play(t *testing.T, st *engine.Story, lang, script string, hard bool) (string, string, error) {
	t.Helper()
	var out bytes.Buffer
	ending, err := ui.Run(st, ui.Options{
		Lang: lang, Hard: hard, Script: strings.Split(script, ","), Out: &out, Width: 80,
	})
	return ending, out.String(), err
}

func TestScriptedEndingsAllLanguages(t *testing.T) {
	st := load(t)
	for _, lang := range engine.Languages {
		for _, id := range []string{"E1", "E2", "E3", "E4", "E5", "E6", "E7", "E8"} {
			t.Run(lang+"/"+id, func(t *testing.T) {
				home := t.TempDir()
				t.Setenv(engine.HomeEnv, home)
				ending, out, err := play(t, st, lang, scripts[id], false)
				if err != nil {
					t.Fatalf("%v\n%s", err, tail(out))
				}
				if ending != id {
					t.Fatalf("reached %q, want %s\n%s", ending, id, tail(out))
				}
				title := st.Text(lang, "ending."+id+".title")
				if !strings.Contains(out, title) {
					t.Errorf("transcript lacks the %s title %q", lang, title)
				}
				p, err := engine.ReadProgress()
				if err != nil || !p.Has(id) {
					t.Errorf("progress not recorded in %s: %+v %v", home, p, err)
				}
				if engine.HasSave() {
					t.Error("script mode must not leave a save behind")
				}
			})
		}
	}
}

func TestTranscriptIsDeterministic(t *testing.T) {
	st := load(t)
	t.Setenv(engine.HomeEnv, t.TempDir())
	_, a, _ := play(t, st, "en", scripts["E7"], false)
	t.Setenv(engine.HomeEnv, t.TempDir())
	_, b, _ := play(t, st, "en", scripts["E7"], false)
	if a != b {
		t.Error("two runs of the same script printed different transcripts")
	}
}

func TestScriptCommands(t *testing.T) {
	st := load(t)
	home := t.TempDir()
	t.Setenv(engine.HomeEnv, home)
	// Follow the E1 route until the decoy map is found (17 choices), use every
	// command, then carry on: save + load returns to the same moment, so the
	// rest of the route still leads to E1.
	e1 := strings.Split(scripts["E1"], ",")
	cmds := []string{"s", "j", "h", "save", "lang ru", "j", "lang uz", "load"}
	script := strings.Join(append(append(append([]string{}, e1[:17]...), cmds...), e1[17:]...), ",")
	ending, out, err := play(t, st, "en", script, false)
	if err != nil {
		t.Fatalf("%v\n%s", err, tail(out))
	}
	if _, err := os.Stat(filepath.Join(home, "save.json")); err != nil {
		t.Errorf("save command did not write %s/save.json", home)
	}
	for _, want := range []string{
		st.Text("en", "ui.journal.title"),
		st.Text("en", "clue.briefcase_decoy.name"),
		st.Text("en", "clue.briefcase_decoy.hint"),
		st.Text("ru", "ui.journal.title"),
		st.Text("ru", "clue.briefcase_decoy.name"),
		st.Text("en", "ui.saved"),
		st.Text("uz", "ui.loaded"),
		"🕙",
	} {
		if !strings.Contains(flat(out), flat(want)) {
			t.Errorf("transcript lacks %q", want)
		}
	}
	if ending != "E1" {
		t.Errorf("after the commands the script should still reach E1, got %q", ending)
	}
	if !strings.Contains(out, st.Text("uz", "ending.E1.title")) {
		t.Error("the language switch to uz should carry through to the ending")
	}
}

func TestHardModeHidesHints(t *testing.T) {
	st := load(t)
	t.Setenv(engine.HomeEnv, t.TempDir())
	_, out, _ := play(t, st, "en", "1,1,1,1,1,2,j", true)
	if !strings.Contains(out, st.Text("en", "clue.briefcase_decoy.name")) {
		t.Error("hard mode should still list the clue")
	}
	if strings.Contains(flat(out), flat(st.Text("en", "clue.briefcase_decoy.hint"))) {
		t.Error("hard mode must hide clue hints")
	}
}

func TestScriptFailures(t *testing.T) {
	st := load(t)
	t.Setenv(engine.HomeEnv, t.TempDir())
	if _, _, err := play(t, st, "en", "1,1", false); !errors.Is(err, ui.ErrNoEnding) {
		t.Errorf("short script: want ErrNoEnding, got %v", err)
	}
	if _, _, err := play(t, st, "en", "9", false); err == nil {
		t.Error("out-of-range choice in a script should be an error")
	}
}

func TestWrap(t *testing.T) {
	text := "Поезд номер семнадцать «Южная звезда» шипит, как самовар перед кипением.\n\nO'zbekcha matn ham to'g'ri o'raladi."
	for _, w := range []int{20, 33, 60} {
		got := ui.Wrap(text, w)
		for _, line := range strings.Split(got, "\n") {
			if utf8.RuneCountInString(line) > w {
				t.Errorf("width %d: line too long: %q", w, line)
			}
		}
		if strings.Join(strings.Fields(got), " ") != strings.Join(strings.Fields(text), " ") {
			t.Errorf("width %d: words changed", w)
		}
		if !strings.Contains(got, "\n\n") {
			t.Errorf("width %d: paragraph break lost", w)
		}
	}
	hung := ui.Hang(" 1. a fairly long choice label that has to wrap (10 min)", 24, 4)
	for i, line := range strings.Split(hung, "\n") {
		if i > 0 && !strings.HasPrefix(line, "    ") {
			t.Errorf("continuation line not indented: %q", line)
		}
	}
}

// flat collapses whitespace so that word-wrapped output can be searched.
func flat(s string) string { return strings.Join(strings.Fields(s), " ") }

func tail(s string) string {
	if len(s) > 1500 {
		return "..." + s[len(s)-1500:]
	}
	return s
}
