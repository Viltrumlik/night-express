package engine

import (
	"errors"
	"fmt"
)

// Bits is a fixed-size flag set.
type Bits [MaxFlags / 64]uint64

// Set turns flag i on.
func (b *Bits) Set(i int) { b[i>>6] |= 1 << (uint(i) & 63) }

// Clear turns flag i off.
func (b *Bits) Clear(i int) { b[i>>6] &^= 1 << (uint(i) & 63) }

// Has reports whether flag i is on.
func (b Bits) Has(i int) bool { return b[i>>6]&(1<<(uint(i)&63)) != 0 }

// State is the complete logical game state. It is a plain value: copying it
// copies everything, which the solver relies on.
type State struct {
	Node    int
	Minutes int
	Stats   [NumStats]int
	Flags   Bits
	Fired   uint64
	Visits  [MaxNodes]uint8
	Ending  string
}

// Story is a loaded, compiled and validated story.
type Story struct {
	Raw          *GraphJSON
	Nodes        []*Node
	NodeIdx      map[string]int
	Flags        []string
	FlagIdx      map[string]int
	Events       []*Event
	Clues        []Clue
	Notes        []Note
	Endings      []Ending
	KeyClueGoal  int
	Start        int
	ClockStart   int // minutes after midnight
	Deadline     int // minutes since ClockStart
	DeadlineNode int
	Initial      [NumStats]int
	Texts        map[string]map[string]string
}

// Step describes what happened during one Choose call.
type Step struct {
	ResultKey string // text to show right after the choice ("" if none)
	EventID   string // timed event that redirected the player ("" if none)
	Arrival   bool   // the 06:00 deadline forced the arrival sequence
}

// ErrUnavailable is returned when a choice's condition does not hold.
var ErrUnavailable = errors.New("choice not available")

// NewState returns the state at the start of a new night.
func (st *Story) NewState() State {
	var s State
	s.Stats = st.Initial
	st.enter(&s, st.Start)
	return s
}

// Node returns the node the state is at.
func (st *Story) Node(s *State) *Node { return st.Nodes[s.Node] }

// Available returns the indices (into the node's Choices) of the choices
// whose conditions hold, in graph order.
func (st *Story) Available(s *State) []int {
	n := st.Nodes[s.Node]
	out := make([]int, 0, len(n.Choices))
	for i, c := range n.Choices {
		if c.Requires.Eval(s) {
			out = append(out, i)
		}
	}
	return out
}

// Choose applies choice ci (an index into the node's Choices) to s.
func (st *Story) Choose(s *State, ci int) (Step, error) {
	if s.Ending != "" {
		return Step{}, fmt.Errorf("the story has ended (%s)", s.Ending)
	}
	n := st.Nodes[s.Node]
	if ci < 0 || ci >= len(n.Choices) {
		return Step{}, fmt.Errorf("node %s has no choice #%d", n.ID, ci)
	}
	c := n.Choices[ci]
	if !c.Requires.Eval(s) {
		return Step{}, ErrUnavailable
	}
	s.Minutes += c.Cost
	for _, e := range c.Effects {
		e.apply(s)
	}
	step := Step{ResultKey: c.ResultKey}
	target := st.redirect(s, c.Target, &step)
	st.enter(s, target)
	return step, nil
}

// ChooseByID applies the choice with the given id.
func (st *Story) ChooseByID(s *State, id string) (Step, error) {
	for i, c := range st.Nodes[s.Node].Choices {
		if c.ID == id {
			return st.Choose(s, i)
		}
	}
	return Step{}, fmt.Errorf("node %s has no choice %q", st.Nodes[s.Node].ID, id)
}

// redirect applies the deadline rule and timed events to a transition.
func (st *Story) redirect(s *State, target int, step *Step) int {
	t := st.Nodes[target]
	if t.Kind != KindEnding && !t.LateOK && s.Minutes >= st.Deadline {
		// The train arrives on time, whatever you were doing.
		s.Minutes = st.Deadline
		step.Arrival = true
		return st.DeadlineNode
	}
	if t.Kind != KindHub {
		return target
	}
	for i, ev := range st.Events {
		if s.Fired&(1<<uint(i)) != 0 || s.Minutes < ev.At {
			continue
		}
		if ev.Until > 0 && s.Minutes >= ev.Until {
			continue
		}
		if !ev.When.Eval(s) {
			continue
		}
		s.Fired |= 1 << uint(i)
		step.EventID = ev.ID
		return ev.Target
	}
	return target
}

func (st *Story) enter(s *State, idx int) {
	s.Node = idx
	if s.Visits[idx] < 255 {
		s.Visits[idx]++
	}
	for _, e := range st.Nodes[idx].OnEnter {
		e.apply(s)
	}
}

// TextKey returns the text key to show for the current node (variants applied).
func (st *Story) TextKey(s *State) string {
	n := st.Nodes[s.Node]
	for _, v := range n.Variants {
		if v.When.Eval(s) {
			return v.TextKey
		}
	}
	return n.TextKey
}

// HasFlag reports whether the named flag is set (false for unknown flags).
func (st *Story) HasFlag(s *State, name string) bool {
	i, ok := st.FlagIdx[name]
	return ok && s.Flags.Has(i)
}

// CluesFound counts discovered clues.
func (st *Story) CluesFound(s *State) int {
	n := 0
	for _, c := range st.Clues {
		if s.Flags.Has(c.Idx) {
			n++
		}
	}
	return n
}

// KeyCluesFound counts discovered key clues (evidence for the militia).
func (st *Story) KeyCluesFound(s *State) int {
	n := 0
	for _, c := range st.Clues {
		if c.Key && s.Flags.Has(c.Idx) {
			n++
		}
	}
	return n
}

// Clock formats the state's time as HH:MM.
func (st *Story) Clock(s *State) string { return FormatClock(s.Minutes, st.ClockStart) }

// EndingIndex returns the 1-based number of an ending id ("E3" -> 3), or 0.
func (st *Story) EndingIndex(id string) int {
	for i, e := range st.Endings {
		if e.ID == id {
			return i + 1
		}
	}
	return 0
}
