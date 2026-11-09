package engine_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"nightexpress/engine"
	"nightexpress/story"
)

// tiny compiles a graph written in JSON (no texts: texts are checked elsewhere).
func tiny(t *testing.T, src string) *engine.Story {
	t.Helper()
	var g engine.GraphJSON
	if err := json.Unmarshal([]byte(src), &g); err != nil {
		t.Fatal(err)
	}
	st, err := engine.Compile(&g, nil)
	if err != nil {
		t.Fatal(err)
	}
	return st
}

func availableIDs(st *engine.Story, s *engine.State) []string {
	var ids []string
	for _, ci := range st.Available(s) {
		ids = append(ids, st.Nodes[s.Node].Choices[ci].ID)
	}
	sort.Strings(ids)
	return ids
}

const condGraph = `{
 "start": "a", "clock_start": "22:00", "deadline": "06:00", "deadline_node": "end",
 "initial": {"trust": 3, "duty": 5, "suspicion": 1},
 "endings": [{"id": "E1", "node": "end"}],
 "nodes": [
  {"id": "a", "kind": "hub", "text_key": "a", "choices": [
   {"id": "always", "label_key": "x", "target": "a", "cost_min": 10,
    "effects": [{"set": "f"}, {"set": "g"}, {"set": "h"}]},
   {"id": "flag", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"flag": "f"}},
   {"id": "notflag", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"not_flag": "f"}},
   {"id": "trust5", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"stat": "trust", "op": ">=", "value": 5}},
   {"id": "susp_lt2", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"stat": "suspicion", "op": "<", "value": 2}},
   {"id": "before23", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"before": "23:00"}},
   {"id": "after23", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"after": "23:00"}},
   {"id": "after0130", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"after": "01:30"}},
   {"id": "visits2", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"visits": "a", "op": ">=", "value": 2}},
   {"id": "count2", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"count": ["f", "g", "h"], "op": ">=", "value": 2}},
   {"id": "all", "label_key": "x", "target": "a", "cost_min": 10,
    "requires": {"all": [{"flag": "f"}, {"stat": "trust", "op": ">=", "value": 5}]}},
   {"id": "any", "label_key": "x", "target": "a", "cost_min": 10,
    "requires": {"any": [{"flag": "f"}, {"stat": "trust", "op": ">=", "value": 5}]}},
   {"id": "not", "label_key": "x", "target": "a", "cost_min": 10, "requires": {"not": {"flag": "f"}}},
   {"id": "finish", "label_key": "x", "target": "end", "cost_min": 10}
  ]},
  {"id": "end", "kind": "ending", "text_key": "e", "late_ok": true, "on_enter": [{"ending": "E1"}]}
 ]
}`

func TestConditions(t *testing.T) {
	st := tiny(t, condGraph)
	s := st.NewState()
	got := availableIDs(st, &s)
	want := []string{"always", "before23", "finish", "not", "notflag", "susp_lt2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("fresh state: got %v, want %v", got, want)
	}

	// Two flags of three, trust 5, 23:30, second visit.
	s.Flags.Set(st.FlagIdx["f"])
	s.Flags.Set(st.FlagIdx["g"])
	s.Stats[engine.StatTrust] = 5
	s.Stats[engine.StatSuspicion] = 2
	s.Minutes = 90
	s.Visits[st.NodeIdx["a"]] = 2
	got = availableIDs(st, &s)
	want = []string{"after23", "all", "always", "any", "count2", "finish", "flag", "trust5", "visits2"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("later state: got %v, want %v", got, want)
	}

	// Time windows cross midnight correctly: 01:30 is 210 minutes after 22:00.
	s.Minutes = 209
	if contains(availableIDs(st, &s), "after0130") {
		t.Error("after 01:30 should not hold at 01:29")
	}
	s.Minutes = 210
	if !contains(availableIDs(st, &s), "after0130") {
		t.Error("after 01:30 should hold at 01:30")
	}
}

func contains(list []string, x string) bool {
	for _, v := range list {
		if v == x {
			return true
		}
	}
	return false
}

func TestEffectsAndClamping(t *testing.T) {
	st := tiny(t, `{
	 "start": "a", "clock_start": "22:00", "deadline": "06:00", "deadline_node": "end",
	 "initial": {"trust": 3, "duty": 5, "suspicion": 1, "money": 5},
	 "endings": [{"id": "E1", "node": "end"}],
	 "nodes": [
	  {"id": "a", "kind": "scene", "text_key": "a", "choices": [
	   {"id": "big", "label_key": "x", "target": "b", "cost_min": 20,
	    "effects": [{"stat": "trust", "add": 20}, {"stat": "duty", "add": -20}, {"stat": "suspicion", "add": 4},
	                {"money": -50}, {"time": 15}, {"set": "f"}]}
	  ]},
	  {"id": "b", "kind": "scene", "text_key": "b", "on_enter": [{"clear": "f"}, {"set": "g"}, {"money": 30}], "choices": [
	   {"id": "fin", "label_key": "x", "target": "end", "cost_min": 10}
	  ]},
	  {"id": "end", "kind": "ending", "text_key": "e", "late_ok": true, "on_enter": [{"ending": "E1"}]}
	 ]
	}`)
	s := st.NewState()
	step, err := st.ChooseByID(&s, "big")
	if err != nil {
		t.Fatal(err)
	}
	if step.EventID != "" || step.Arrival {
		t.Fatalf("unexpected redirect %+v", step)
	}
	if s.Stats[engine.StatTrust] != 10 || s.Stats[engine.StatDuty] != 0 || s.Stats[engine.StatSuspicion] != 5 {
		t.Fatalf("stats not clamped to 0..10: %v", s.Stats)
	}
	if s.Stats[engine.StatMoney] != 30 {
		t.Fatalf("money should floor at 0 then gain 30 on enter, got %d", s.Stats[engine.StatMoney])
	}
	if s.Minutes != 35 {
		t.Fatalf("cost 20 + time effect 15 = 35 minutes, got %d", s.Minutes)
	}
	if st.HasFlag(&s, "f") || !st.HasFlag(&s, "g") {
		t.Fatal("on_enter should clear f and set g")
	}
	if _, err := st.ChooseByID(&s, "fin"); err != nil {
		t.Fatal(err)
	}
	if s.Ending != "E1" {
		t.Fatalf("ending not set: %q", s.Ending)
	}
	if _, err := st.ChooseByID(&s, "fin"); err == nil {
		t.Fatal("choosing after the ending should fail")
	}
}

const eventGraph = `{
 "start": "hub", "clock_start": "22:00", "deadline": "23:30", "deadline_node": "arrive",
 "initial": {"trust": 3, "duty": 5, "suspicion": 1},
 "endings": [{"id": "E1", "node": "end"}],
 "events": [
  {"id": "early", "at": "22:20", "until": "22:40", "target": "ev_early"},
  {"id": "alarm", "at": "22:00", "when": {"flag": "armed"}, "target": "ev_alarm"},
  {"id": "round", "at": "22:30", "target": "ev_round"}
 ],
 "nodes": [
  {"id": "hub", "kind": "hub", "text_key": "h", "choices": [
   {"id": "wait", "label_key": "x", "target": "hub", "cost_min": 30},
   {"id": "wait10", "label_key": "x", "target": "hub", "cost_min": 10},
   {"id": "arm", "label_key": "x", "target": "side", "cost_min": 10, "effects": [{"set": "armed"}]}
  ]},
  {"id": "side", "kind": "scene", "text_key": "s", "choices": [
   {"id": "back", "label_key": "x", "target": "hub", "cost_min": 10}
  ]},
  {"id": "ev_early", "kind": "scene", "text_key": "x", "choices": [{"id": "ok", "label_key": "x", "target": "hub", "cost_min": 10}]},
  {"id": "ev_alarm", "kind": "scene", "text_key": "x", "choices": [{"id": "ok", "label_key": "x", "target": "hub", "cost_min": 10}]},
  {"id": "ev_round", "kind": "scene", "text_key": "x", "choices": [{"id": "ok", "label_key": "x", "target": "hub", "cost_min": 10}]},
  {"id": "arrive", "kind": "scene", "text_key": "x", "late_ok": true, "choices": [{"id": "ok", "label_key": "x", "target": "end", "cost_min": 10}]},
  {"id": "end", "kind": "ending", "text_key": "e", "on_enter": [{"ending": "E1"}]}
 ]
}`

func TestTimedEventsFireOnce(t *testing.T) {
	st := tiny(t, eventGraph)
	s := st.NewState()
	node := func() string { return st.Nodes[s.Node].ID }

	// 22:30: both "early" (22:20-22:40) and "round" (22:30) are due; list order wins.
	step, _ := st.ChooseByID(&s, "wait")
	if node() != "ev_early" || step.EventID != "early" {
		t.Fatalf("22:30: want ev_early, got %s (%+v)", node(), step)
	}
	st.ChooseByID(&s, "ok") // 22:40, back to the hub: "round" is next in line
	if node() != "ev_round" {
		t.Fatalf("22:40: want ev_round, got %s", node())
	}
	st.ChooseByID(&s, "ok") // 22:50
	if node() != "hub" {
		t.Fatalf("22:50: events must not fire twice, got %s", node())
	}
	// Scenes are never interrupted; the flag-gated event waits for the next hub visit.
	st.ChooseByID(&s, "arm") // 23:00, scene "side"
	if node() != "side" {
		t.Fatalf("scenes must not be interrupted, got %s", node())
	}
	st.ChooseByID(&s, "back") // 23:10, hub: the alarm fires now
	if node() != "ev_alarm" {
		t.Fatalf("23:10: want ev_alarm, got %s", node())
	}
	st.ChooseByID(&s, "ok") // 23:20
	if node() != "hub" {
		t.Fatalf("23:20: want hub, got %s", node())
	}
	// Deadline 23:30: any non-late node redirects to the arrival, and the clock snaps to it.
	step, _ = st.ChooseByID(&s, "wait") // would be 23:50
	if node() != "arrive" || !step.Arrival || st.Clock(&s) != "23:30" {
		t.Fatalf("deadline: node %s, step %+v, clock %s", node(), step, st.Clock(&s))
	}
	st.ChooseByID(&s, "ok")
	if s.Ending != "E1" {
		t.Fatalf("late_ok chain should reach the ending, got %q", s.Ending)
	}
}

func TestEventUntilWindow(t *testing.T) {
	st := tiny(t, eventGraph)
	s := st.NewState()
	// Spend the 22:20-22:40 window inside scenes: "early" must never fire afterwards.
	st.ChooseByID(&s, "wait10") // 22:10 hub
	st.ChooseByID(&s, "arm")    // 22:20 side
	st.ChooseByID(&s, "back")   // 22:30 hub: early (22:20-22:40) is still open and listed first
	if got := st.Nodes[s.Node].ID; got != "ev_early" {
		t.Fatalf("want ev_early inside its window, got %s", got)
	}
	s2 := st.NewState()
	st.ChooseByID(&s2, "wait")   // 22:30 -> ev_early fires
	st.ChooseByID(&s2, "ok")     // 22:40 -> ev_round
	st.ChooseByID(&s2, "ok")     // 22:50 hub
	st.ChooseByID(&s2, "wait10") // 23:00 hub
	if got := st.Nodes[s2.Node].ID; got != "hub" {
		t.Fatalf("no event should fire at 23:00, got %s", got)
	}
	s3 := st.NewState()
	s3.Minutes = 40 // jump past the window without visiting the hub
	st.ChooseByID(&s3, "wait10")
	if got := st.Nodes[s3.Node].ID; got != "ev_round" {
		t.Fatalf("past 22:40 the early event is skipped, want ev_round, got %s", got)
	}
}

func TestValidationRejectsBrokenGraphs(t *testing.T) {
	cases := map[string]string{
		"unknown target": `{"id": "a", "kind": "scene", "text_key": "a", "choices": [{"id": "x", "label_key": "x", "target": "nowhere", "cost_min": 10}]}`,
		"cost range":     `{"id": "a", "kind": "scene", "text_key": "a", "choices": [{"id": "x", "label_key": "x", "target": "end", "cost_min": 3}]}`,
		"no choices":     `{"id": "a", "kind": "scene", "text_key": "a", "choices": []}`,
		"flag never set": `{"id": "a", "kind": "scene", "text_key": "a", "choices": [{"id": "x", "label_key": "x", "target": "end", "cost_min": 10, "requires": {"flag": "ghost"}}]}`,
		"two kinds":      `{"id": "a", "kind": "scene", "text_key": "a", "choices": [{"id": "x", "label_key": "x", "target": "end", "cost_min": 10, "requires": {"before": "23:00", "after": "22:30"}}]}`,
		"duplicate ids":  `{"id": "a", "kind": "scene", "text_key": "a", "choices": [{"id": "x", "label_key": "x", "target": "end", "cost_min": 10}, {"id": "x", "label_key": "x", "target": "end", "cost_min": 10}]}`,
	}
	for name, node := range cases {
		src := `{"start": "a", "clock_start": "22:00", "deadline": "06:00", "deadline_node": "end",
		 "endings": [{"id": "E1", "node": "end"}],
		 "nodes": [` + node + `, {"id": "end", "kind": "ending", "text_key": "e", "late_ok": true, "on_enter": [{"ending": "E1"}]}]}`
		var g engine.GraphJSON
		if err := json.Unmarshal([]byte(src), &g); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if _, err := engine.Compile(&g, nil); err == nil {
			t.Errorf("%s: Compile accepted a broken graph", name)
		}
	}
	// Missing text keys are caught when texts are supplied.
	var g engine.GraphJSON
	json.Unmarshal([]byte(condGraph), &g)
	texts := map[string]map[string]string{"uz": {}, "ru": {}, "en": {}}
	if _, err := engine.Compile(&g, texts); err == nil || !strings.Contains(err.Error(), "missing or empty key") {
		t.Errorf("missing texts not reported: %v", err)
	}
}

func TestClock(t *testing.T) {
	for _, c := range []struct {
		hhmm string
		min  int
	}{{"22:00", 0}, {"23:30", 90}, {"00:00", 120}, {"04:00", 360}, {"06:00", 480}} {
		got, err := engine.ParseClock(c.hhmm, 22*60)
		if err != nil || got != c.min {
			t.Errorf("ParseClock(%s) = %d, %v; want %d", c.hhmm, got, err, c.min)
		}
		if back := engine.FormatClock(c.min, 22*60); back != c.hhmm {
			t.Errorf("FormatClock(%d) = %s, want %s", c.min, back, c.hhmm)
		}
	}
	if _, err := engine.ParseClock("25:00", 0); err == nil {
		t.Error("ParseClock accepted 25:00")
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	home := t.TempDir()
	t.Setenv(engine.HomeEnv, home)
	st, err := engine.Load(story.FS)
	if err != nil {
		t.Fatal(err)
	}
	s := st.NewState()
	for _, id := range endingPaths["E1"][:12] {
		if _, err := st.ChooseByID(&s, id); err != nil {
			t.Fatal(err)
		}
	}
	if engine.HasSave() {
		t.Fatal("fresh home already has a save")
	}
	if err := engine.WriteSave(st.Snapshot(&s, "ru", true, 42)); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, "save.json")); err != nil {
		t.Fatalf("save not written to NIGHT_EXPRESS_HOME: %v", err)
	}
	d, err := engine.ReadSave()
	if err != nil {
		t.Fatal(err)
	}
	if d.Lang != "ru" || !d.Hard || d.Seed != 42 {
		t.Fatalf("metadata lost: %+v", d)
	}
	back, err := st.Restore(d)
	if err != nil {
		t.Fatal(err)
	}
	if back != s {
		t.Fatalf("round trip changed the state:\n got %+v\nwant %+v", back, s)
	}
	// The restored night plays on identically.
	for _, id := range endingPaths["E1"][12:] {
		if _, err := st.ChooseByID(&back, id); err != nil {
			t.Fatal(err)
		}
	}
	if back.Ending != "E1" {
		t.Fatalf("restored game ended in %q", back.Ending)
	}
	if err := engine.RemoveSave(); err != nil || engine.HasSave() {
		t.Fatalf("RemoveSave: %v", err)
	}
	d.Node = "no_such_node"
	if _, err := st.Restore(d); err == nil {
		t.Error("Restore accepted an unknown node")
	}
}

func TestProgress(t *testing.T) {
	t.Setenv(engine.HomeEnv, t.TempDir())
	p, err := engine.ReadProgress()
	if err != nil || len(p.Endings) != 0 {
		t.Fatalf("empty progress: %+v %v", p, err)
	}
	if !p.Add("E5") || !p.Add("E1") || p.Add("E5") {
		t.Fatal("Add should report only new endings")
	}
	if err := engine.WriteProgress(p); err != nil {
		t.Fatal(err)
	}
	q, err := engine.ReadProgress()
	if err != nil || !reflect.DeepEqual(q.Endings, []string{"E1", "E5"}) {
		t.Fatalf("progress round trip: %+v %v", q, err)
	}
}
