package engine

import (
	"encoding/json"
	"os"
	"sync"
	"testing"
)

// observation is what a player can see: where, when, the duty/suspicion pair,
// and exactly which choices are on offer.
type observation struct {
	node, minute, duty, susp int
	avail                    uint64
}

func observeAll(st *Story, opts exploreOpts) map[observation]bool {
	var mu sync.Mutex
	seen := map[observation]bool{}
	opts.observe = func(minute int, s *State, m dsSet) {
		masks := make([]dsSet, len(st.Nodes[s.Node].Choices))
		for i, c := range st.Nodes[s.Node].Choices {
			masks[i] = evalDS(c.Requires, s)
		}
		var local []observation
		for d := 0; d <= StatMax; d++ {
			for sp := 0; sp <= StatMax; sp++ {
				if m[d]&(1<<uint(sp)) == 0 {
					continue
				}
				var avail uint64
				for i := range masks {
					if masks[i][d]&(1<<uint(sp)) != 0 {
						avail |= 1 << uint(i)
					}
				}
				local = append(local, observation{s.Node, minute, d, sp, avail})
			}
		}
		mu.Lock()
		for _, o := range local {
			seen[o] = true
		}
		mu.Unlock()
	}
	st.explore(0, opts)
	return seen
}

// The reductions used by Explore (dropping dead flags and events, capping
// monotone stats) must not change anything a player could observe. Compare
// against a search that merges only identical states, up to 01:20 (the exact search grows too fast to go further in a unit test).
func TestPruningIsSound(t *testing.T) {
	if testing.Short() {
		t.Skip("slow")
	}
	b, err := os.ReadFile("../story/graph.json")
	if err != nil {
		t.Fatal(err)
	}
	var g GraphJSON
	if err := json.Unmarshal(b, &g); err != nil {
		t.Fatal(err)
	}
	st, err := Compile(&g, nil)
	if err != nil {
		t.Fatal(err)
	}
	const horizon = 200 // 01:20
	pruned := observeAll(st, exploreOpts{horizon: horizon})
	exact := observeAll(st, exploreOpts{horizon: horizon, noPrune: true})
	t.Logf("%d observable situations before 01:20", len(exact))
	if len(pruned) != len(exact) {
		t.Errorf("pruned search saw %d situations, exact search %d", len(pruned), len(exact))
	}
	missing := 0
	for o := range exact {
		if !pruned[o] {
			missing++
		}
	}
	if missing > 0 {
		t.Errorf("%d situations reachable in the exact search are missing from the pruned one", missing)
	}
}

func TestDSShiftMatchesClamp(t *testing.T) {
	for d := 0; d <= StatMax; d++ {
		for s := 0; s <= StatMax; s++ {
			for _, k := range []int{-3, -2, -1, 1, 2, 3} {
				m := dsPoint(d, s).addDuty(k).addSusp(k)
				gd, gs := m.first()
				if gd != clamp(d+k, 0, StatMax) || gs != clamp(s+k, 0, StatMax) || m.count() != 1 {
					t.Fatalf("(%d,%d)%+d -> (%d,%d)", d, s, k, gd, gs)
				}
			}
		}
	}
}
