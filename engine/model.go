package engine

import (
	"fmt"
	"strconv"
	"strings"
)

// Kind is the node kind.
type Kind int

// Node kinds.
const (
	KindScene Kind = iota
	KindHub
	KindEnding
)

// Stat indices into State.Stats.
const (
	StatTrust = iota
	StatDuty
	StatSuspicion
	StatMoney
	StatEvidence
	NumStats
)

// StatNames are the JSON names of the stats, indexed like State.Stats.
// Evidence counts key clues (the militia needs three of the four).
var StatNames = [NumStats]string{"trust", "duty", "suspicion", "money", "evidence"}

// StatMax is the clamp ceiling for every stat except money (which is unbounded above).
const StatMax = 10

// Limits of the compact state representation.
const (
	MaxFlags  = 128
	MaxNodes  = 96
	MaxEvents = 32
)

func statIndex(name string) (int, bool) {
	for i, n := range StatNames {
		if n == name {
			return i, true
		}
	}
	return 0, false
}

type condKind int

const (
	cFlag condKind = iota
	cNotFlag
	cStat
	cVisits
	cCount
	cBefore
	cAfter
	cAll
	cAny
	cNot
)

type opKind int

const (
	opGE opKind = iota
	opGT
	opLE
	opLT
	opEQ
	opNE
)

func parseOp(s string) (opKind, error) {
	switch s {
	case ">=":
		return opGE, nil
	case ">":
		return opGT, nil
	case "<=":
		return opLE, nil
	case "<":
		return opLT, nil
	case "==", "=":
		return opEQ, nil
	case "!=":
		return opNE, nil
	}
	return 0, fmt.Errorf("unknown comparison operator %q", s)
}

func compare(a int, op opKind, b int) bool {
	switch op {
	case opGE:
		return a >= b
	case opGT:
		return a > b
	case opLE:
		return a <= b
	case opLT:
		return a < b
	case opEQ:
		return a == b
	case opNE:
		return a != b
	}
	return false
}

// Cond is a compiled condition. A nil *Cond is always true.
type Cond struct {
	kind  condKind
	idx   int // flag, stat or node index
	flags []int
	op    opKind
	value int
	subs  []*Cond
}

// Eval reports whether the condition holds in s.
func (c *Cond) Eval(s *State) bool {
	if c == nil {
		return true
	}
	switch c.kind {
	case cFlag:
		return s.Flags.Has(c.idx)
	case cNotFlag:
		return !s.Flags.Has(c.idx)
	case cStat:
		return compare(s.Stats[c.idx], c.op, c.value)
	case cVisits:
		return compare(int(s.Visits[c.idx]), c.op, c.value)
	case cCount:
		n := 0
		for _, f := range c.flags {
			if s.Flags.Has(f) {
				n++
			}
		}
		return compare(n, c.op, c.value)
	case cBefore:
		return s.Minutes < c.value
	case cAfter:
		return s.Minutes >= c.value
	case cAll:
		for _, sub := range c.subs {
			if !sub.Eval(s) {
				return false
			}
		}
		return true
	case cAny:
		for _, sub := range c.subs {
			if sub.Eval(s) {
				return true
			}
		}
		return false
	case cNot:
		return !c.subs[0].Eval(s)
	}
	return false
}

// walk calls fn for every condition in the tree (including c itself).
func (c *Cond) walk(fn func(*Cond)) {
	if c == nil {
		return
	}
	fn(c)
	for _, s := range c.subs {
		s.walk(fn)
	}
}

type effKind int

const (
	eSet effKind = iota
	eClear
	eStat
	eMoney
	eTime
	eEnding
)

// Effect is a compiled effect.
type Effect struct {
	kind   effKind
	idx    int
	value  int
	ending string
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func (e Effect) apply(s *State) {
	switch e.kind {
	case eSet:
		s.Flags.Set(e.idx)
	case eClear:
		s.Flags.Clear(e.idx)
	case eStat:
		s.Stats[e.idx] = clamp(s.Stats[e.idx]+e.value, 0, StatMax)
	case eMoney:
		s.Stats[StatMoney] += e.value
		if s.Stats[StatMoney] < 0 {
			s.Stats[StatMoney] = 0
		}
	case eTime:
		s.Minutes += e.value
	case eEnding:
		s.Ending = e.ending
	}
}

// Choice is a compiled choice.
type Choice struct {
	ID        string
	LabelKey  string
	Target    int
	Cost      int
	Requires  *Cond
	Effects   []Effect
	ResultKey string
}

// Variant is a compiled text variant.
type Variant struct {
	When    *Cond
	TextKey string
}

// Node is a compiled node.
type Node struct {
	ID       string
	Kind     Kind
	TextKey  string
	Variants []Variant
	LateOK   bool
	Ambient  []string
	OnEnter  []Effect
	Choices  []*Choice
	Ending   string // ending id for KindEnding nodes
}

// Event is a compiled timed event.
type Event struct {
	ID     string
	At     int
	Until  int // 0 = no upper bound
	When   *Cond
	Target int
}

// Clue is a journal clue.
type Clue struct {
	Flag string
	Idx  int
	Key  bool
}

// Note is a journal note.
type Note struct {
	Flag string
	Idx  int
}

// Ending is a declared ending.
type Ending struct {
	ID   string
	Node int
}

// ParseClock converts "HH:MM" into minutes since clockStart (wrapping past midnight).
func ParseClock(hhmm string, clockStart int) (int, error) {
	parts := strings.Split(hhmm, ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("bad time %q (want HH:MM)", hhmm)
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, fmt.Errorf("bad time %q (want HH:MM)", hhmm)
	}
	return ((h*60+m)-clockStart+24*60)%(24*60) + 0, nil
}

// FormatClock converts minutes since clockStart into "HH:MM".
func FormatClock(minutes, clockStart int) string {
	t := (clockStart + minutes) % (24 * 60)
	return fmt.Sprintf("%02d:%02d", t/60, t%60)
}
