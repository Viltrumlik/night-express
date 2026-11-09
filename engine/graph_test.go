package engine_test

import (
	"encoding/json"
	"io/fs"
	"math/rand"
	"sync"
	"testing"

	"nightexpress/engine"
	"nightexpress/story"
)

// loadStructure compiles the graph without checking texts, so structural
// tests stay independent from the localisation tests.
func loadStructure(t testing.TB) *engine.Story {
	t.Helper()
	b, err := fs.ReadFile(story.FS, "graph.json")
	if err != nil {
		t.Fatal(err)
	}
	var g engine.GraphJSON
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	st, err := engine.Compile(&g, nil)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

var (
	exploreOnce sync.Once
	exploreRep  engine.ExploreReport
)

// explored runs the exhaustive search once per test binary.
func explored(t *testing.T) (*engine.Story, engine.ExploreReport) {
	t.Helper()
	if testing.Short() {
		t.Skip("exhaustive search skipped in -short mode")
	}
	st := loadStructure(t)
	exploreOnce.Do(func() { exploreRep = st.Explore() })
	return st, exploreRep
}

func TestGraphShape(t *testing.T) {
	st := loadStructure(t)
	if n := len(st.Nodes); n < 45 || n > 60 {
		t.Errorf("node count %d outside 45..60", n)
	}
	if len(st.Endings) != 8 {
		t.Fatalf("want exactly 8 endings, got %d", len(st.Endings))
	}
	for i, e := range st.Endings {
		if want := "E" + string(rune('1'+i)); e.ID != want {
			t.Errorf("ending %d is %s, want %s", i, e.ID, want)
		}
	}
	keys := 0
	for _, c := range st.Clues {
		if c.Key {
			keys++
		}
	}
	if len(st.Clues) != 10 || keys != 4 {
		t.Errorf("want 10 clues of which 4 are key clues, got %d/%d", len(st.Clues), keys)
	}
	if len(st.Events) < 5 {
		t.Errorf("want at least 5 timed events, got %d", len(st.Events))
	}
}

// TestExhaustive walks every reachable (node, minutes, stats, flags) state:
// every node and every ending must be reachable, and no reachable non-ending
// state may be left without an available choice.
func TestExhaustive(t *testing.T) {
	st, rep := explored(t)
	if rep.Truncated {
		t.Fatal("search truncated")
	}
	t.Logf("explored %d canonical states covering %d concrete duty/suspicion states (widest minute: %d)",
		rep.States, rep.Concrete, rep.WidestLayer)
	for _, s := range rep.DeadEnds {
		t.Errorf("dead end at node %s, %s, stats %v", st.Nodes[s.Node].ID, st.Clock(&s), s.Stats)
	}
	for i, ok := range rep.NodeReached {
		if !ok {
			t.Errorf("node %s is unreachable", st.Nodes[i].ID)
		}
	}
	for _, e := range st.Endings {
		if rep.Endings[e.ID] == 0 {
			t.Errorf("ending %s is unreachable", e.ID)
		} else {
			t.Logf("ending %s: %d terminal states", e.ID, rep.Endings[e.ID])
		}
	}
}

// Random concrete playthroughs, through the ordinary engine API, must always
// end, never get stuck, and only reach endings the solver also found.
func TestRandomPlaythroughs(t *testing.T) {
	st, rep := explored(t)
	rng := rand.New(rand.NewSource(1979))
	seen := map[string]int{}
	for run := 0; run < 3000; run++ {
		s := st.NewState()
		for steps := 0; s.Ending == ""; steps++ {
			avail := st.Available(&s)
			if len(avail) == 0 {
				t.Fatalf("run %d: stuck at %s %s", run, st.Nodes[s.Node].ID, st.Clock(&s))
			}
			if _, err := st.Choose(&s, avail[rng.Intn(len(avail))]); err != nil {
				t.Fatal(err)
			}
			if steps > 200 {
				t.Fatalf("run %d: no ending after 200 choices", run)
			}
		}
		seen[s.Ending]++
		if rep.Endings[s.Ending] == 0 {
			t.Errorf("random play reached %s, which the solver never found", s.Ending)
		}
	}
	t.Logf("endings over 3000 random nights: %v", seen)
	if seen["E1"] > 3 {
		t.Errorf("E1 reached by %d random nights; the best ending should need a plan", seen["E1"])
	}
}

// The clock must matter: a player who conscientiously tries every option in
// the order offered (the naive "visit everything" strategy) does not reach the
// best ending, while a planned route does.
func TestClockMatters(t *testing.T) {
	st := loadStructure(t)
	for _, name := range []string{"first untried", "last untried"} {
		s := st.NewState()
		tried := map[[2]int]bool{}
		for steps := 0; s.Ending == "" && steps < 500; steps++ {
			avail := st.Available(&s)
			order := avail
			if name == "last untried" {
				order = make([]int, len(avail))
				for i := range avail {
					order[i] = avail[len(avail)-1-i]
				}
			}
			pick := order[0]
			for _, ci := range order {
				if !tried[[2]int{s.Node, ci}] {
					pick = ci
					break
				}
			}
			tried[[2]int{s.Node, pick}] = true
			if _, err := st.Choose(&s, pick); err != nil {
				t.Fatal(err)
			}
		}
		if s.Ending == "" {
			t.Fatalf("%s: never finished", name)
		}
		if s.Ending == "E1" {
			t.Errorf("%s: the naive visit-everything order reached the best ending", name)
		}
		t.Logf("%s: naive order ends in %s at %s with %d clues", name, s.Ending, st.Clock(&s), st.CluesFound(&s))
	}
	// A planned route does reach it.
	s := st.NewState()
	for _, id := range endingPaths["E1"] {
		if _, err := st.ChooseByID(&s, id); err != nil {
			t.Fatal(err)
		}
	}
	if s.Ending != "E1" {
		t.Fatalf("planned route ended in %s", s.Ending)
	}
}
