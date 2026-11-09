// Package engine loads, validates and runs the Night Express story graph.
//
// The story is split into a language-independent graph (story/graph.json)
// and per-language text tables (story/text/{uz,ru,en}.json). The engine is
// deterministic: the same state and the same choice always produce the same
// next state, which is what makes the exhaustive solver in explore.go possible.
package engine

// The JSON shapes below mirror story/graph.json one to one.

// CondJSON is a condition. Exactly one of the kinds must be set:
// flag, not_flag, stat (+op, value), visits (+op, value), count (+op, value),
// before, after, all, any, not.
type CondJSON struct {
	Flag    string      `json:"flag,omitempty"`
	NotFlag string      `json:"not_flag,omitempty"`
	Stat    string      `json:"stat,omitempty"`
	Visits  string      `json:"visits,omitempty"`
	Count   []string    `json:"count,omitempty"`
	Op      string      `json:"op,omitempty"`
	Value   *int        `json:"value,omitempty"`
	Before  string      `json:"before,omitempty"`
	After   string      `json:"after,omitempty"`
	All     []*CondJSON `json:"all,omitempty"`
	Any     []*CondJSON `json:"any,omitempty"`
	Not     *CondJSON   `json:"not,omitempty"`
}

// EffectJSON is one effect. Exactly one kind must be set:
// set, clear, stat (+add), money, time, ending.
type EffectJSON struct {
	Set    string `json:"set,omitempty"`
	Clear  string `json:"clear,omitempty"`
	Stat   string `json:"stat,omitempty"`
	Add    int    `json:"add,omitempty"`
	Money  int    `json:"money,omitempty"`
	Time   int    `json:"time,omitempty"`
	Ending string `json:"ending,omitempty"`
}

// ChoiceJSON is a numbered option shown to the player.
type ChoiceJSON struct {
	ID        string       `json:"id"`
	LabelKey  string       `json:"label_key"`
	Target    string       `json:"target"`
	CostMin   int          `json:"cost_min"`
	Requires  *CondJSON    `json:"requires,omitempty"`
	Effects   []EffectJSON `json:"effects,omitempty"`
	ResultKey string       `json:"result_key,omitempty"`
}

// VariantJSON replaces a node's text when its condition holds (first match wins).
type VariantJSON struct {
	When    *CondJSON `json:"when"`
	TextKey string    `json:"text_key"`
}

// NodeJSON is a scene, hub or ending.
type NodeJSON struct {
	ID       string        `json:"id"`
	Kind     string        `json:"kind"`
	TextKey  string        `json:"text_key"`
	Variants []VariantJSON `json:"variants,omitempty"`
	LateOK   bool          `json:"late_ok,omitempty"`
	Ambient  []string      `json:"ambient,omitempty"`
	OnEnter  []EffectJSON  `json:"on_enter,omitempty"`
	Choices  []ChoiceJSON  `json:"choices,omitempty"`
}

// EventJSON is a timed interruption: once the clock reaches At (and is still
// before Until, if set) and When holds, the next hub visit is redirected to
// Target. Each event fires at most once.
type EventJSON struct {
	ID     string    `json:"id"`
	At     string    `json:"at"`
	Until  string    `json:"until,omitempty"`
	When   *CondJSON `json:"when,omitempty"`
	Target string    `json:"target"`
}

// ClueJSON declares a journal clue. Text keys: clue.<flag>.name, clue.<flag>.hint.
type ClueJSON struct {
	Flag string `json:"flag"`
	Key  bool   `json:"key"`
}

// NoteJSON declares a journal note. Text key: note.<flag>.
type NoteJSON struct {
	Flag string `json:"flag"`
}

// EndingJSON declares an ending. Text keys: ending.<id>.title.
type EndingJSON struct {
	ID   string `json:"id"`
	Node string `json:"node"`
}

// GraphJSON is the whole of story/graph.json.
type GraphJSON struct {
	Start        string         `json:"start"`
	ClockStart   string         `json:"clock_start"`
	Deadline     string         `json:"deadline"`
	DeadlineNode string         `json:"deadline_node"`
	Initial      map[string]int `json:"initial"`
	KeyClueGoal  int            `json:"key_clue_goal"`
	Clues        []ClueJSON     `json:"clues"`
	Notes        []NoteJSON     `json:"notes"`
	Endings      []EndingJSON   `json:"endings"`
	Events       []EventJSON    `json:"events"`
	Nodes        []NodeJSON     `json:"nodes"`
}
