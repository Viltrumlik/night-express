package engine

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"regexp"
	"sort"
	"strings"
)

// Languages supported by the game, in title-screen order.
var Languages = []string{"uz", "ru", "en"}

// Load reads graph.json and text/<lang>.json from fsys, compiles and validates them.
func Load(fsys fs.FS) (*Story, error) {
	raw, err := fs.ReadFile(fsys, "graph.json")
	if err != nil {
		return nil, err
	}
	var g GraphJSON
	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&g); err != nil {
		return nil, fmt.Errorf("graph.json: %w", err)
	}
	texts := map[string]map[string]string{}
	for _, lang := range Languages {
		b, err := fs.ReadFile(fsys, "text/"+lang+".json")
		if err != nil {
			return nil, err
		}
		m := map[string]string{}
		if err := json.Unmarshal(b, &m); err != nil {
			return nil, fmt.Errorf("text/%s.json: %w", lang, err)
		}
		texts[lang] = m
	}
	st, err := Compile(&g, texts)
	if err != nil {
		return nil, err
	}
	return st, nil
}

type compiler struct {
	st   *Story
	errs []string
}

func (c *compiler) errf(format string, a ...any) {
	c.errs = append(c.errs, fmt.Sprintf(format, a...))
}

func (c *compiler) flag(name string) int {
	if i, ok := c.st.FlagIdx[name]; ok {
		return i
	}
	if name == "" {
		c.errf("empty flag name")
		return 0
	}
	i := len(c.st.Flags)
	c.st.Flags = append(c.st.Flags, name)
	c.st.FlagIdx[name] = i
	return i
}

func (c *compiler) clock(hhmm, where string) int {
	m, err := ParseClock(hhmm, c.st.ClockStart)
	if err != nil {
		c.errf("%s: %v", where, err)
	}
	return m
}

func (c *compiler) cond(j *CondJSON, where string) *Cond {
	if j == nil {
		return nil
	}
	kinds := 0
	var out Cond
	needCmp := false
	if j.Flag != "" {
		kinds++
		out.kind, out.idx = cFlag, c.flag(j.Flag)
	}
	if j.NotFlag != "" {
		kinds++
		out.kind, out.idx = cNotFlag, c.flag(j.NotFlag)
	}
	if j.Stat != "" {
		kinds++
		i, ok := statIndex(j.Stat)
		if !ok {
			c.errf("%s: unknown stat %q", where, j.Stat)
		}
		out.kind, out.idx, needCmp = cStat, i, true
	}
	if j.Visits != "" {
		kinds++
		i, ok := c.st.NodeIdx[j.Visits]
		if !ok {
			c.errf("%s: visits of unknown node %q", where, j.Visits)
		}
		out.kind, out.idx, needCmp = cVisits, i, true
	}
	if len(j.Count) > 0 {
		kinds++
		out.kind, needCmp = cCount, true
		for _, f := range j.Count {
			out.flags = append(out.flags, c.flag(f))
		}
	}
	if j.Before != "" {
		kinds++
		out.kind, out.value = cBefore, c.clock(j.Before, where)
	}
	if j.After != "" {
		kinds++
		out.kind, out.value = cAfter, c.clock(j.After, where)
	}
	if len(j.All) > 0 {
		kinds++
		out.kind = cAll
		for _, s := range j.All {
			out.subs = append(out.subs, c.cond(s, where))
		}
	}
	if len(j.Any) > 0 {
		kinds++
		out.kind = cAny
		for _, s := range j.Any {
			out.subs = append(out.subs, c.cond(s, where))
		}
	}
	if j.Not != nil {
		kinds++
		out.kind = cNot
		out.subs = []*Cond{c.cond(j.Not, where)}
	}
	if kinds != 1 {
		c.errf("%s: condition must have exactly one kind, has %d", where, kinds)
		return nil
	}
	if needCmp {
		if j.Op == "" || j.Value == nil {
			c.errf("%s: comparison needs op and value", where)
		} else {
			op, err := parseOp(j.Op)
			if err != nil {
				c.errf("%s: %v", where, err)
			}
			out.op, out.value = op, *j.Value
		}
	} else if j.Op != "" || j.Value != nil {
		c.errf("%s: op/value only allowed with stat, visits or count", where)
	}
	return &out
}

func (c *compiler) effects(list []EffectJSON, where string) []Effect {
	var out []Effect
	for _, j := range list {
		kinds := 0
		var e Effect
		if j.Set != "" {
			kinds++
			e.kind, e.idx = eSet, c.flag(j.Set)
		}
		if j.Clear != "" {
			kinds++
			e.kind, e.idx = eClear, c.flag(j.Clear)
		}
		if j.Stat != "" {
			kinds++
			i, ok := statIndex(j.Stat)
			if !ok || i == StatMoney {
				c.errf("%s: bad stat effect %q (use money for money)", where, j.Stat)
			}
			if j.Add == 0 {
				c.errf("%s: stat effect with add 0", where)
			}
			e.kind, e.idx, e.value = eStat, i, j.Add
		} else if j.Add != 0 {
			c.errf("%s: add without stat", where)
		}
		if j.Money != 0 {
			kinds++
			e.kind, e.value = eMoney, j.Money
		}
		if j.Time != 0 {
			kinds++
			if j.Time < 0 {
				c.errf("%s: time cannot go backwards", where)
			}
			e.kind, e.value = eTime, j.Time
		}
		if j.Ending != "" {
			kinds++
			e.kind, e.ending = eEnding, j.Ending
		}
		if kinds != 1 {
			c.errf("%s: effect must have exactly one kind, has %d", where, kinds)
			continue
		}
		out = append(out, e)
	}
	return out
}

// Compile turns parsed JSON into a validated Story.
func Compile(g *GraphJSON, texts map[string]map[string]string) (*Story, error) {
	st := &Story{Raw: g, NodeIdx: map[string]int{}, FlagIdx: map[string]int{}, Texts: texts}
	c := &compiler{st: st}
	start := 0
	if g.ClockStart != "" {
		m, err := ParseClock(g.ClockStart, 0)
		if err != nil {
			c.errf("clock_start: %v", err)
		}
		start = m
	} else {
		c.errf("clock_start missing")
	}
	st.ClockStart = start
	st.Deadline = c.clock(g.Deadline, "deadline")
	if len(g.Nodes) > MaxNodes {
		c.errf("too many nodes: %d > %d", len(g.Nodes), MaxNodes)
	}
	for i, n := range g.Nodes {
		if _, dup := st.NodeIdx[n.ID]; dup {
			c.errf("duplicate node id %q", n.ID)
		}
		st.NodeIdx[n.ID] = i
	}
	node := func(id, where string) int {
		i, ok := st.NodeIdx[id]
		if !ok {
			c.errf("%s: unknown node %q", where, id)
		}
		return i
	}
	st.Start = node(g.Start, "start")
	st.DeadlineNode = node(g.DeadlineNode, "deadline_node")
	for name, v := range g.Initial {
		i, ok := statIndex(name)
		if !ok {
			c.errf("initial: unknown stat %q", name)
			continue
		}
		st.Initial[i] = v
	}
	st.KeyClueGoal = g.KeyClueGoal
	for _, n := range g.Nodes {
		where := "node " + n.ID
		cn := &Node{ID: n.ID, TextKey: n.TextKey, LateOK: n.LateOK, Ambient: n.Ambient}
		switch n.Kind {
		case "scene":
			cn.Kind = KindScene
		case "hub":
			cn.Kind = KindHub
		case "ending":
			cn.Kind = KindEnding
		default:
			c.errf("%s: unknown kind %q", where, n.Kind)
		}
		for _, v := range n.Variants {
			if v.When == nil {
				c.errf("%s: variant %s without condition", where, v.TextKey)
			}
			cn.Variants = append(cn.Variants, Variant{When: c.cond(v.When, where+" variant"), TextKey: v.TextKey})
		}
		cn.OnEnter = c.effects(n.OnEnter, where+" on_enter")
		for _, e := range cn.OnEnter {
			if e.kind == eEnding {
				cn.Ending = e.ending
			}
		}
		seen := map[string]bool{}
		for _, ch := range n.Choices {
			cw := where + " choice " + ch.ID
			if seen[ch.ID] || ch.ID == "" {
				c.errf("%s: duplicate or empty choice id", cw)
			}
			seen[ch.ID] = true
			if ch.CostMin < 5 || ch.CostMin > 40 {
				c.errf("%s: cost %d outside 5..40", cw, ch.CostMin)
			}
			effs := c.effects(ch.Effects, cw)
			cn.Choices = append(cn.Choices, &Choice{
				ID: ch.ID, LabelKey: ch.LabelKey, Target: node(ch.Target, cw), Cost: ch.CostMin,
				Requires: c.cond(ch.Requires, cw), Effects: effs, ResultKey: ch.ResultKey,
			})
			for _, e := range effs {
				if e.kind == eEnding {
					c.errf("%s: endings are set by ending nodes, not choices", cw)
				}
			}
		}
		switch cn.Kind {
		case KindEnding:
			if len(cn.Choices) > 0 {
				c.errf("%s: ending with choices", where)
			}
			if cn.Ending == "" {
				c.errf("%s: ending node must set an ending in on_enter", where)
			}
		default:
			if len(cn.Choices) == 0 {
				c.errf("%s: non-ending node without choices", where)
			}
			if cn.Ending != "" {
				c.errf("%s: only ending nodes may set an ending", where)
			}
		}
		st.Nodes = append(st.Nodes, cn)
	}
	if len(st.Nodes) > 0 && st.DeadlineNode < len(st.Nodes) && !st.Nodes[st.DeadlineNode].LateOK {
		c.errf("deadline node %s must be late_ok", g.DeadlineNode)
	}
	if len(g.Events) > MaxEvents {
		c.errf("too many events: %d > %d", len(g.Events), MaxEvents)
	}
	evSeen := map[string]bool{}
	for _, e := range g.Events {
		where := "event " + e.ID
		if evSeen[e.ID] || e.ID == "" {
			c.errf("%s: duplicate or empty id", where)
		}
		evSeen[e.ID] = true
		ce := &Event{ID: e.ID, At: c.clock(e.At, where), When: c.cond(e.When, where), Target: node(e.Target, where)}
		if e.Until != "" {
			ce.Until = c.clock(e.Until, where)
			if ce.Until <= ce.At {
				c.errf("%s: until must be after at", where)
			}
		}
		st.Events = append(st.Events, ce)
	}
	for _, cl := range g.Clues {
		st.Clues = append(st.Clues, Clue{Flag: cl.Flag, Idx: c.flag(cl.Flag), Key: cl.Key})
	}
	for _, n := range g.Notes {
		st.Notes = append(st.Notes, Note{Flag: n.Flag, Idx: c.flag(n.Flag)})
	}
	endingIDs := map[string]bool{}
	for _, e := range g.Endings {
		i := node(e.Node, "ending "+e.ID)
		endingIDs[e.ID] = true
		st.Endings = append(st.Endings, Ending{ID: e.ID, Node: i})
		if i < len(st.Nodes) && st.Nodes[i].Ending != e.ID {
			c.errf("ending %s: node %s sets ending %q", e.ID, e.Node, st.Nodes[i].Ending)
		}
	}
	for _, n := range st.Nodes {
		if n.Kind == KindEnding && !endingIDs[n.Ending] {
			c.errf("node %s: ending %q is not declared", n.ID, n.Ending)
		}
	}
	if len(st.Flags) > MaxFlags {
		c.errf("too many flags: %d > %d", len(st.Flags), MaxFlags)
	}
	c.checkFlagsUsed()
	c.checkTexts()
	if len(c.errs) > 0 {
		sort.Strings(c.errs)
		return nil, fmt.Errorf("story validation failed:\n  %s", strings.Join(c.errs, "\n  "))
	}
	return st, nil
}

// checkFlagsUsed catches typos: every flag read by a condition must be set somewhere.
func (c *compiler) checkFlagsUsed() {
	set := map[int]bool{}
	read := map[int]bool{}
	mark := func(cond *Cond) {
		cond.walk(func(x *Cond) {
			switch x.kind {
			case cFlag, cNotFlag:
				read[x.idx] = true
			case cCount:
				for _, f := range x.flags {
					read[f] = true
				}
			}
		})
	}
	for _, n := range c.st.Nodes {
		for _, e := range n.OnEnter {
			if e.kind == eSet {
				set[e.idx] = true
			}
		}
		for _, v := range n.Variants {
			mark(v.When)
		}
		for _, ch := range n.Choices {
			mark(ch.Requires)
			for _, e := range ch.Effects {
				if e.kind == eSet {
					set[e.idx] = true
				}
			}
		}
	}
	for _, e := range c.st.Events {
		mark(e.When)
	}
	for _, cl := range c.st.Clues {
		read[cl.Idx] = true
	}
	for _, n := range c.st.Notes {
		read[n.Idx] = true
	}
	for i := range read {
		if !set[i] {
			c.errf("flag %q is read but never set", c.st.Flags[i])
		}
	}
}

// TextKeys returns every text key the graph refers to.
func (st *Story) TextKeys() []string {
	keys := map[string]bool{}
	add := func(k string) {
		if k != "" {
			keys[k] = true
		}
	}
	for _, n := range st.Nodes {
		add(n.TextKey)
		for _, v := range n.Variants {
			add(v.TextKey)
		}
		for _, a := range n.Ambient {
			add(a)
		}
		for _, ch := range n.Choices {
			add(ch.LabelKey)
			add(ch.ResultKey)
		}
	}
	for _, cl := range st.Clues {
		add("clue." + cl.Flag + ".name")
		add("clue." + cl.Flag + ".hint")
	}
	for _, n := range st.Notes {
		add("note." + n.Flag)
	}
	for _, e := range st.Endings {
		add("ending." + e.ID + ".title")
	}
	out := make([]string, 0, len(keys))
	for k := range keys {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func (c *compiler) checkTexts() {
	for _, n := range c.st.Nodes {
		if n.TextKey == "" {
			c.errf("node %s: empty text_key", n.ID)
		}
		for _, ch := range n.Choices {
			if ch.LabelKey == "" {
				c.errf("node %s choice %s: empty label_key", n.ID, ch.ID)
			}
		}
	}
	if c.st.Texts == nil {
		return
	}
	for _, lang := range Languages {
		m, ok := c.st.Texts[lang]
		if !ok {
			c.errf("missing language %s", lang)
			continue
		}
		for _, k := range c.st.TextKeys() {
			if strings.TrimSpace(m[k]) == "" {
				c.errf("text %s: missing or empty key %q", lang, k)
			}
		}
	}
}

var placeholderRE = regexp.MustCompile(`\{[a-z_]+\}`)

// Placeholders returns the sorted, de-duplicated {placeholders} in s.
func Placeholders(s string) []string {
	found := placeholderRE.FindAllString(s, -1)
	set := map[string]bool{}
	for _, f := range found {
		set[f] = true
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// Text returns the text for key in lang, falling back to English, then to the key itself.
func (st *Story) Text(lang, key string) string {
	if t, ok := st.Texts[lang][key]; ok && t != "" {
		return t
	}
	if t, ok := st.Texts["en"][key]; ok && t != "" {
		return t
	}
	return key
}

// Format replaces {name} placeholders with values from vars; unknown ones are left as is.
func Format(s string, vars map[string]string) string {
	return placeholderRE.ReplaceAllStringFunc(s, func(p string) string {
		if v, ok := vars[p[1:len(p)-1]]; ok {
			return v
		}
		return p
	})
}
