package engine_test

import (
	"encoding/json"
	"io/fs"
	"reflect"
	"sort"
	"strings"
	"testing"
	"unicode"

	"nightexpress/engine"
	"nightexpress/story"
)

func readTexts(t *testing.T) map[string]map[string]string {
	t.Helper()
	out := map[string]map[string]string{}
	for _, lang := range engine.Languages {
		b, err := fs.ReadFile(story.FS, "text/"+lang+".json")
		if err != nil {
			t.Fatal(err)
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("%s: %v", lang, err)
		}
		out[lang] = m
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// The full story (graph + every language) loads and validates.
func TestStoryLoads(t *testing.T) {
	if _, err := engine.Load(story.FS); err != nil {
		t.Fatal(err)
	}
}

func TestLocalisationKeySets(t *testing.T) {
	texts := readTexts(t)
	en := sortedKeys(texts["en"])
	for _, lang := range engine.Languages {
		if got := sortedKeys(texts[lang]); !reflect.DeepEqual(got, en) {
			t.Errorf("%s key set differs from en (%d vs %d keys)", lang, len(got), len(en))
		}
	}
	// Every key the graph refers to exists (and nothing the graph needs is missing).
	st := loadStructure(t)
	for _, k := range st.TextKeys() {
		if _, ok := texts["en"][k]; !ok {
			t.Errorf("graph key %q missing from texts", k)
		}
	}
}

func TestLocalisationPlaceholdersAndEmpties(t *testing.T) {
	texts := readTexts(t)
	for _, k := range sortedKeys(texts["en"]) {
		want := engine.Placeholders(texts["en"][k])
		for _, lang := range engine.Languages {
			v := texts[lang][k]
			if strings.TrimSpace(v) == "" {
				t.Errorf("%s %s: empty", lang, k)
				continue
			}
			if got := engine.Placeholders(v); !reflect.DeepEqual(got, want) {
				t.Errorf("%s %s: placeholders %v, en has %v", lang, k, got, want)
			}
		}
	}
}

func TestUzbekApostrophes(t *testing.T) {
	for k, v := range readTexts(t)["uz"] {
		if strings.ContainsAny(v, "ʻʼ’‘`") {
			t.Errorf("uz %s: use the ASCII apostrophe for o', g' and tutuq belgisi", k)
		}
	}
}

// Prose must actually be translated: Russian prose is Cyrillic, Uzbek prose
// differs from the English.
func TestTranslationsAreTranslated(t *testing.T) {
	texts := readTexts(t)
	for k, en := range texts["en"] {
		if !strings.HasPrefix(k, "n.") && !strings.HasPrefix(k, "r.") {
			continue
		}
		ru, uz := texts["ru"][k], texts["uz"][k]
		cyr := 0
		for _, r := range ru {
			if unicode.Is(unicode.Cyrillic, r) {
				cyr++
			}
		}
		if cyr < len([]rune(ru))/3 {
			t.Errorf("ru %s does not look Russian", k)
		}
		if uz == en {
			t.Errorf("uz %s is identical to English", k)
		}
	}
}
