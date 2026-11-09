package engine_test

import (
	"strconv"
	"strings"
	"testing"

	"nightexpress/engine"
)

// Paths are written as choice ids (readable, robust to text edits). The UI
// tests replay the same paths as numeric --script input in every language.
var endingPaths = map[string][]string{
	// Best ending: report early and get the chief's key, keep the tea round,
	// hear Bahodir's gossip before the dining car closes, study the decoy and
	// win Lola's trust before 01:30, steal a look at the telegram in the
	// blackout, feed Sherzod, find the professor, reunite father and daughter,
	// and let the chief block the baggage car at Saksovul.
	"E1": {
		"help_lola", "next", "yes", "goodnight", "go", "quiet",
		"report", "key", // chief's 23:00 round
		"tea", "next",
		"decline", // Gena's card game
		"dining", "chat", "back",
		"sorry", // tickets were never checked
		"c4", "briefcase", "talk_first", "talk_open", "back",
		"search", // the 01:30 blackout
		"rear", "listen_bread", "thank", "open", "tell_lola", "leave",
		"c4", "fetch", "leave",
		"wait",
		"stand_ok", "on", // Saksovul
		"wait", "report_cold", "wait", "wait",
		"door", "evidence_win",
	},
	// The professor is never found: he is taken at Saksovul.
	"E5": {
		"check_grey", "next", "yes", "goodnight", "go", "quiet",
		"fine", "tea", "next", "wait", "decline", "wait", "sorry",
		"wait", "stay", "wait", "wait", "wait", "wait", "wait",
		"follow", "end",
	},
	// Duties neglected: the chief relieves you before half past midnight.
	"E7": {
		"check_grey", "next", "ask", "goodnight", "go", "quiet",
		"fine", "staff", "sleep", "sorry", "decline", "staff", "sorry",
	},
	// The professor is saved, but without his daughter he will not testify.
	"E2": {
		"help_lola", "next", "yes", "goodnight", "go", "quiet",
		"report", "key", "tea", "next", "decline",
		"dining", "chat", "back", "sorry",
		"c4", "briefcase", "talk_first", "talk_open", "back",
		"search",
		"rear", "listen_bread", "thank", "open", "tell_lola", "leave",
		"wait", "wait",
		"stand_ok", "on",
		"wait", "report_cold", "wait", "wait",
		"door", "evidence",
	},
	// Find the professor, then sell his hiding place at Saksovul.
	"E3": {
		"help_lola", "next", "yes", "goodnight", "go", "quiet",
		"report", "key", "rear", "sorry", "rear", "decline", "rear", "open", "leave",
		"sorry", "wait", "wait", "stay",
		"wait", "wait", "wait", "wait", "wait",
		"sell",
	},
	// Accuse an innocent at Qumtepa (same night as E8, different last choice).
	"E4": {
		"help_halima", "next", "yes", "goodnight", "go", "quiet",
		"fine", "c6", "talk", "sorry",
		"c6", "decline", "c6", "hen", "keep", "sorry",
		"dining", "chat", "back",
		"wait", "stay", "wait", "wait", "wait", "wait", "wait",
		"follow", "keep_map",
		"wait", "report_cold", "wait", "wait",
		"door", "accuse_lola",
	},
	// Provoke Rudin until he attacks while you have no ally.
	"E6": {
		"check_grey", "next", "yes", "goodnight", "go", "report", "back",
		"fine", "c2", "sorry", "c2", "decline", "c2", "search", "pocket",
		"sorry", "push", "defy",
		"c2", "talk", "fight",
	},
	// Secret ending: the map under the hen goes to the reporter.
	"E8": {
		"help_halima", "next", "yes", "goodnight", "go", "quiet",
		"fine", "c6", "talk", "sorry", // tea was forgotten
		"c6", "decline", "c6", "hen", "keep", "sorry",
		"dining", "chat", "back",
		"wait", "stay", "wait", "wait", "wait", "wait", "wait",
		"follow", "keep_map",
		"wait", "report_cold", "wait", "wait",
		"door", "reporter",
	},
}

// ScriptFor returns the numeric --script string for a path.
func scriptFor(t testing.TB, st *engine.Story, ids []string) string {
	t.Helper()
	nums, err := st.ScriptNumbers(ids)
	if err != nil {
		t.Fatal(err)
	}
	parts := make([]string, len(nums))
	for i, n := range nums {
		parts[i] = strconv.Itoa(n)
	}
	return strings.Join(parts, ",")
}

func TestEndingPaths(t *testing.T) {
	st := loadStructure(t)
	for _, want := range []string{"E1", "E2", "E3", "E4", "E5", "E6", "E7", "E8"} {
		s := st.NewState()
		for i, id := range endingPaths[want] {
			if _, err := st.ChooseByID(&s, id); err != nil {
				t.Fatalf("%s step %d (%s) at %s %s: %v", want, i, id, st.Nodes[s.Node].ID, st.Clock(&s), err)
			}
		}
		if s.Ending != want {
			t.Fatalf("%s path ended at node %s (%s), ending %q", want, st.Nodes[s.Node].ID, st.Clock(&s), s.Ending)
		}
		t.Logf("%s --script %q", want, scriptFor(t, st, endingPaths[want]))
	}
}
