package engine

// Exhaustive state-space search used by the validation tests.
//
// Every choice costs at least 5 minutes and time never runs backwards, so the
// state graph is a DAG layered by the clock. The explorer walks it minute by
// minute and memoises states per minute; once a minute has been processed its
// memo is dropped, which keeps memory bounded by the widest minute.
//
// States are compared through a canonical key that keeps exactly what can still
// change future choices (a sound reduction: two states with the same key have
// identical futures as far as choice availability is concerned):
//   - node, fired events, trust/duty/suspicion/money and flags;
//   - a flag is dropped once it can never again influence a condition: every
//     condition that reads it is guarded by a conjunct that has become
//     permanently false (a "killer" fact such as "it is past 01:00" or "flag X
//     is set" - flags are never cleared), or sits at a node that can no longer
//     be reached;
//   - a stat whose effects only ever increase it is capped at the highest
//     threshold any condition compares it with;
//   - an event past its "until" can never fire, so its fired bit is dropped;
//   - visit counts and text variants never affect choices and are ignored
//     unless some choice/event condition reads them.

import (
	"runtime"
	"runtime/debug"
	"sort"
	"sync"
	"sync/atomic"
)

// killer is a monotone fact: once true it stays true for the rest of the night.
type killer struct {
	kind  int // 0 time>=v, 1 flag set, 2 stat>=v (non-decreasing stat), 3 event fired, 4 visits>=v
	idx   int
	value int
}

type occurrence struct {
	flag    int
	node    int // -1 for event conditions
	killers [2]uint64
}

type explorer struct {
	st       *Story
	killers  []killer
	edgeFrom []int
	edgeTo   []int
	edgeKill [][2]uint64
	occs     []occurrence
	untilK   []int // per event: killer index of "time >= until", or -1
	statCap  [NumStats]int
	visNodes []int
	visCap   []uint8
	adj      [][]int    // edge indices by source node
	noPrune  bool       // test hook: keep every flag and event in the key
	mu       sync.Mutex // guards liveIDs
	liveIDs  map[liveInfo]uint32
}

type liveKey struct {
	node int
	kill [2]uint64
}

type liveInfo struct {
	flags  Bits
	events uint64
	id     uint32 // distinct per (flags, events) pair; part of the key so that
	// states are only merged when they agree on what is still live
}

func hasBit(m [2]uint64, i int) bool { return m[i>>6]&(1<<(uint(i)&63)) != 0 }

func newExplorer(st *Story) *explorer {
	x := &explorer{st: st, liveIDs: map[liveInfo]uint32{}}
	cleared := map[int]bool{}
	up := [NumStats]bool{true, true, true, true}
	scanEff := func(es []Effect) {
		for _, e := range es {
			switch {
			case e.kind == eClear:
				cleared[e.idx] = true
			case e.kind == eStat && e.value < 0:
				up[e.idx] = false
			case e.kind == eMoney && e.value < 0:
				up[StatMoney] = false
			}
		}
	}
	for _, n := range st.Nodes {
		scanEff(n.OnEnter)
		for _, c := range n.Choices {
			scanEff(c.Effects)
		}
	}
	thresholds := [NumStats]int{}
	visCaps := map[int]int{}
	scanCond := func(c *Cond) {
		c.walk(func(k *Cond) {
			switch k.kind {
			case cStat:
				b := k.value
				if k.op == opGT || k.op == opLE || k.op == opEQ || k.op == opNE {
					b = k.value + 1
				}
				thresholds[k.idx] = max(thresholds[k.idx], b)
			case cVisits:
				visCaps[k.idx] = max(visCaps[k.idx], k.value+1)
			}
		})
	}
	for _, n := range st.Nodes {
		for _, c := range n.Choices {
			scanCond(c.Requires)
		}
	}
	for _, e := range st.Events {
		scanCond(e.When)
	}
	for i := range x.statCap {
		x.statCap[i] = -1
		if up[i] {
			x.statCap[i] = thresholds[i]
		}
	}
	for n, c := range visCaps {
		x.visNodes = append(x.visNodes, n)
		x.visCap = append(x.visCap, uint8(min(c, 15)))
	}
	sort.Ints(x.visNodes) // deterministic packing (caps follow the same order below)
	x.visCap = x.visCap[:0]
	for _, n := range x.visNodes {
		x.visCap = append(x.visCap, uint8(min(visCaps[n], 15)))
	}

	kidx := map[killer]int{}
	addK := func(k killer, m *[2]uint64) int {
		i, ok := kidx[k]
		if !ok {
			if len(x.killers) >= 128 {
				return -1 // out of room: dropping a killer is always sound
			}
			i = len(x.killers)
			kidx[k] = i
			x.killers = append(x.killers, k)
		}
		m[i>>6] |= 1 << (uint(i) & 63)
		return i
	}
	// killersOf collects monotone facts that make c permanently false.
	var killersOf func(c *Cond, m *[2]uint64)
	killersOf = func(c *Cond, m *[2]uint64) {
		if c == nil {
			return
		}
		switch c.kind {
		case cBefore:
			addK(killer{0, 0, c.value}, m)
		case cNotFlag:
			if !cleared[c.idx] {
				addK(killer{1, c.idx, 0}, m)
			}
		case cStat:
			if up[c.idx] {
				switch c.op {
				case opLT:
					addK(killer{2, c.idx, c.value}, m)
				case opLE, opEQ:
					addK(killer{2, c.idx, c.value + 1}, m)
				}
			}
		case cVisits:
			switch c.op {
			case opLT:
				addK(killer{4, c.idx, c.value}, m)
			case opLE, opEQ:
				addK(killer{4, c.idx, c.value + 1}, m)
			}
		case cAll:
			for _, s := range c.subs {
				killersOf(s, m)
			}
		}
	}
	flagsIn := func(c *Cond) []int {
		set := map[int]bool{}
		c.walk(func(k *Cond) {
			switch k.kind {
			case cFlag, cNotFlag:
				set[k.idx] = true
			case cCount:
				for _, g := range k.flags {
					set[g] = true
				}
			}
		})
		out := make([]int, 0, len(set))
		for f := range set {
			out = append(out, f)
		}
		sort.Ints(out)
		return out
	}
	mentions := func(c *Cond, f int) bool {
		for _, g := range flagsIn(c) {
			if g == f {
				return true
			}
		}
		return false
	}
	// Occurrence killers: the top-level conjuncts that do not read the flag itself.
	occKill := func(c *Cond, f int, m *[2]uint64) {
		if c == nil || c.kind != cAll {
			return
		}
		for _, s := range c.subs {
			if !mentions(s, f) {
				killersOf(s, m)
			}
		}
	}
	for ni, n := range st.Nodes {
		for _, c := range n.Choices {
			var em [2]uint64
			killersOf(c.Requires, &em)
			x.edgeFrom = append(x.edgeFrom, ni)
			x.edgeTo = append(x.edgeTo, c.Target)
			x.edgeKill = append(x.edgeKill, em)
			for _, f := range flagsIn(c.Requires) {
				o := occurrence{flag: f, node: ni}
				occKill(c.Requires, f, &o.killers)
				x.occs = append(x.occs, o)
			}
		}
		// The deadline can redirect any transition to the arrival node.
		x.edgeFrom = append(x.edgeFrom, ni)
		x.edgeTo = append(x.edgeTo, st.DeadlineNode)
		x.edgeKill = append(x.edgeKill, [2]uint64{})
	}
	for ei, e := range st.Events {
		var em [2]uint64
		addK(killer{3, ei, 0}, &em)
		u := -1
		if e.Until > 0 {
			u = addK(killer{0, 0, e.Until}, &em)
		}
		x.untilK = append(x.untilK, u)
		killersOf(e.When, &em)
		for ni, n := range st.Nodes {
			if n.Kind == KindHub {
				x.edgeFrom = append(x.edgeFrom, ni)
				x.edgeTo = append(x.edgeTo, e.Target)
				x.edgeKill = append(x.edgeKill, em)
			}
		}
		for _, f := range flagsIn(e.When) {
			o := occurrence{flag: f, node: -1}
			addK(killer{3, ei, 0}, &o.killers)
			if e.Until > 0 {
				addK(killer{0, 0, e.Until}, &o.killers)
			}
			occKill(e.When, f, &o.killers)
			x.occs = append(x.occs, o)
		}
	}
	x.adj = make([][]int, len(st.Nodes))
	for i, from := range x.edgeFrom {
		x.adj[from] = append(x.adj[from], i)
	}
	return x
}

func (x *explorer) trueKillers(s *State) [2]uint64 {
	var m [2]uint64
	for i, k := range x.killers {
		var t bool
		switch k.kind {
		case 0:
			t = s.Minutes >= k.value
		case 1:
			t = s.Flags.Has(k.idx)
		case 2:
			t = s.Stats[k.idx] >= k.value
		case 3:
			t = s.Fired&(1<<uint(k.idx)) != 0
		case 4:
			t = int(s.Visits[k.idx]) >= k.value
		}
		if t {
			m[i>>6] |= 1 << (uint(i) & 63)
		}
	}
	return m
}

// live returns which flags/events can still matter, given the current node
// and the set of killer facts that hold. It is a pure function of its inputs,
// cached per worker in local; only the id assignment is shared (and locked).
func (x *explorer) live(local map[liveKey]liveInfo, node int, kill [2]uint64) liveInfo {
	if x.noPrune {
		var all liveInfo
		for i := range all.flags {
			all.flags[i] = ^uint64(0)
		}
		all.events = ^uint64(0)
		return all
	}
	lk := liveKey{node, kill}
	if li, ok := local[lk]; ok {
		return li
	}
	dead := func(m [2]uint64) bool { return m[0]&kill[0] != 0 || m[1]&kill[1] != 0 }
	var reach [MaxNodes]bool
	reach[node] = true
	queue := make([]int, 1, len(x.st.Nodes))
	queue[0] = node
	for len(queue) > 0 {
		n := queue[len(queue)-1]
		queue = queue[:len(queue)-1]
		for _, ei := range x.adj[n] {
			to := x.edgeTo[ei]
			if reach[to] || dead(x.edgeKill[ei]) {
				continue
			}
			reach[to] = true
			queue = append(queue, to)
		}
	}
	var li liveInfo
	for _, o := range x.occs {
		if dead(o.killers) || (o.node >= 0 && !reach[o.node]) {
			continue
		}
		li.flags.Set(o.flag)
	}
	for i := range x.st.Events {
		if u := x.untilK[i]; u >= 0 && hasBit(kill, u) {
			continue
		}
		li.events |= 1 << uint(i)
	}
	x.mu.Lock()
	id, ok := x.liveIDs[li]
	if !ok {
		id = uint32(len(x.liveIDs))
		x.liveIDs[li] = id
	}
	x.mu.Unlock()
	li.id = id
	if len(local) >= 1<<16 {
		clear(local) // bounded per-worker cache; recomputing is cheap
	}
	local[lk] = li
	return li
}

type exploreKey struct {
	a, b, c, d uint64
	live       uint32
}

// key builds the canonical key. In symbolic mode duty and suspicion are left
// out: they travel alongside the key as a dsSet.
func (x *explorer) key(local map[liveKey]liveInfo, s *State, symbolic bool) exploreKey {
	li := x.live(local, s.Node, x.trueKillers(s))
	var sv [NumStats]uint64
	for i := range sv {
		v := s.Stats[i]
		if x.statCap[i] >= 0 && v > x.statCap[i] {
			v = x.statCap[i]
		}
		sv[i] = uint64(min(v, 4095))
	}
	if symbolic {
		sv[StatDuty], sv[StatSuspicion] = 0, 0
	}
	a := uint64(s.Node) | sv[StatTrust]<<8 | sv[StatDuty]<<12 | sv[StatSuspicion]<<16 | sv[StatEvidence]<<20 |
		sv[StatMoney]<<24 | (s.Fired&li.events&0xffffffff)<<32
	var d uint64
	for i, n := range x.visNodes {
		v := min(s.Visits[n], x.visCap[i])
		d |= uint64(v) << (4 * uint(i))
	}
	return exploreKey{a, s.Flags[0] & li.flags[0], s.Flags[1] & li.flags[1], d, li.id}
}

// dsSet is a set of (duty, suspicion) pairs: row = duty 0..10, bit = suspicion 0..10.
type dsSet [StatMax + 1]uint16

const dsRow = uint16(1<<(StatMax+1) - 1)

func dsFull() dsSet {
	var m dsSet
	for i := range m {
		m[i] = dsRow
	}
	return m
}

func dsPoint(duty, susp int) dsSet {
	var m dsSet
	m[duty] = 1 << uint(susp)
	return m
}

func (m dsSet) and(o dsSet) dsSet {
	for i := range m {
		m[i] &= o[i]
	}
	return m
}

func (m dsSet) or(o dsSet) dsSet {
	for i := range m {
		m[i] |= o[i]
	}
	return m
}

func (m dsSet) andNot(o dsSet) dsSet {
	for i := range m {
		m[i] &^= o[i]
	}
	return m
}

func (m dsSet) empty() bool {
	for _, r := range m {
		if r != 0 {
			return false
		}
	}
	return true
}

func (m dsSet) count() int {
	n := 0
	for _, r := range m {
		for ; r != 0; r &= r - 1 {
			n++
		}
	}
	return n
}

// first returns some (duty, suspicion) in the set.
func (m dsSet) first() (int, int) {
	for d, r := range m {
		for b := 0; b <= StatMax; b++ {
			if r&(1<<uint(b)) != 0 {
				return d, b
			}
		}
	}
	return -1, -1
}

// addDuty shifts every pair by k duty, clamping to 0..10.
func (m dsSet) addDuty(k int) dsSet {
	var out dsSet
	for d, r := range m {
		out[clamp(d+k, 0, StatMax)] |= r
	}
	return out
}

// addSusp shifts every pair by k suspicion, clamping to 0..10.
func (m dsSet) addSusp(k int) dsSet {
	for i, r := range m {
		if k > 0 {
			top := r >> uint(StatMax+1-k) // bits that would overflow
			r = (r << uint(k)) & dsRow
			if top != 0 {
				r |= 1 << StatMax
			}
		} else if k < 0 {
			low := r & (1<<uint(-k) - 1)
			r >>= uint(-k)
			if low != 0 {
				r |= 1
			}
		}
		m[i] = r
	}
	return m
}

// evalDS returns the (duty, suspicion) pairs for which c holds, everything
// else being taken from s.
func evalDS(c *Cond, s *State) dsSet {
	if c == nil {
		return dsFull()
	}
	switch c.kind {
	case cStat:
		switch c.idx {
		case StatDuty:
			var m dsSet
			for d := range m {
				if compare(d, c.op, c.value) {
					m[d] = dsRow
				}
			}
			return m
		case StatSuspicion:
			var row uint16
			for b := 0; b <= StatMax; b++ {
				if compare(b, c.op, c.value) {
					row |= 1 << uint(b)
				}
			}
			var m dsSet
			for d := range m {
				m[d] = row
			}
			return m
		}
	case cAll:
		m := dsFull()
		for _, sub := range c.subs {
			m = m.and(evalDS(sub, s))
		}
		return m
	case cAny:
		var m dsSet
		for _, sub := range c.subs {
			m = m.or(evalDS(sub, s))
		}
		return m
	case cNot:
		return dsFull().andNot(evalDS(c.subs[0], s))
	}
	if c.Eval(s) {
		return dsFull()
	}
	return dsSet{}
}

// applyDS applies effects: duty/suspicion changes go to the set, the rest to s.
func applyDS(es []Effect, s *State, m dsSet) dsSet {
	for _, e := range es {
		if e.kind == eStat && e.idx == StatDuty {
			m = m.addDuty(e.value)
		} else if e.kind == eStat && e.idx == StatSuspicion {
			m = m.addSusp(e.value)
		} else {
			e.apply(s)
		}
	}
	return m
}

// ExploreReport summarises an exhaustive search.
type ExploreReport struct {
	States      int            // canonical states expanded (duty x suspicion handled as sets)
	Concrete    int            // concrete states covered: sum over canonical states of their duty/suspicion pairs
	Endings     map[string]int // ending id -> number of canonical states reaching it
	NodeReached []bool         // indexed like Story.Nodes
	DeadEnds    []State        // non-ending states with no available choice (first few)
	WidestLayer int            // largest number of canonical states sharing one minute
	Truncated   bool           // the state budget ran out before the search finished
	LiveSets    int            // distinct "what can still matter" sets seen
	LiveKeys    int            // cached (node, facts) liveness lookups, summed over workers
}

// symEntry is a compact canonical state: raw flags (never masked - masking
// would lose which conditions are already dead), fired events, stats, node,
// the visit counts that conditions read, and the duty x suspicion set.
type symEntry struct {
	flags Bits
	fired uint64
	vis   uint64
	stats [NumStats]int16
	node  uint16
	m     dsSet
}

func (x *explorer) pack(s *State, m dsSet) symEntry {
	e := symEntry{flags: s.Flags, fired: s.Fired, node: uint16(s.Node), m: m}
	for i, v := range s.Stats {
		e.stats[i] = int16(min(v, 30000))
	}
	for i, n := range x.visNodes {
		e.vis |= uint64(min(s.Visits[n], x.visCap[i])) << (4 * uint(i))
	}
	return e
}

func (x *explorer) unpack(e *symEntry, minutes int) State {
	s := State{Node: int(e.node), Minutes: minutes, Flags: e.flags, Fired: e.fired}
	for i, v := range e.stats {
		s.Stats[i] = int(v)
	}
	for i, n := range x.visNodes {
		s.Visits[n] = uint8(e.vis>>(4*uint(i))) & 15
	}
	if x.st.Nodes[s.Node].Kind == KindEnding {
		s.Ending = x.st.Nodes[s.Node].Ending
	}
	return s
}

// Explore walks every reachable state from the start of the night.
func (st *Story) Explore() ExploreReport { return st.ExploreLimit(0) }

const exploreShards = 64

type shard struct {
	mu      sync.Mutex
	index   map[exploreKey]int32
	entries []symEntry
}

type layer struct{ shards [exploreShards]shard }

func shardOf(k exploreKey) int {
	h := k.a*0x9E3779B97F4A7C15 ^ k.b*0xC2B2AE3D27D4EB4F ^ k.c*0x165667B19E3779F9 ^ k.d ^ uint64(k.live)*0x27D4EB2F165667C5
	return int((h >> 32) % exploreShards)
}

// ExploreLimit is Explore with a budget of canonical states (0 = unlimited).
//
// The search is exact: duty and suspicion (the two stats that both rise and
// fall) are tracked as an 11x11 set of possible values per canonical state;
// conditions and effects are applied to the whole set at once. Each minute's
// states are expanded in parallel; successors always lie in later minutes.
func (st *Story) ExploreLimit(limit int) ExploreReport {
	return st.explore(limit, exploreOpts{})
}

// exploreOpts are test hooks: noPrune disables every state-merging reduction
// except exact equality, horizon stops before that minute, observe sees every
// canonical state (it is called concurrently).
type exploreOpts struct {
	noPrune bool
	horizon int
	observe func(minute int, s *State, m dsSet)
}

func (st *Story) explore(limit int, opts exploreOpts) ExploreReport {
	defer debug.SetGCPercent(debug.SetGCPercent(50))
	x := newExplorer(st)
	if opts.noPrune {
		x.noPrune = true
		for i := range x.statCap {
			x.statCap[i] = -1
		}
	}
	rep := ExploreReport{Endings: map[string]int{}, NodeReached: make([]bool, len(st.Nodes))}

	// The furthest any single step can move the clock.
	maxStep := 0
	timeIn := func(es []Effect) int {
		t := 0
		for _, e := range es {
			if e.kind == eTime {
				t += e.value
			}
		}
		return t
	}
	maxEnter := 0
	for _, n := range st.Nodes {
		maxEnter = max(maxEnter, timeIn(n.OnEnter))
		for _, c := range n.Choices {
			maxStep = max(maxStep, c.Cost+timeIn(c.Effects))
		}
	}
	maxStep += maxEnter

	var layers []*layer
	ensure := func(upto int) {
		for len(layers) <= upto {
			layers = append(layers, nil)
		}
		for m := 0; m <= upto; m++ {
			if layers[m] == nil {
				l := &layer{}
				for i := range l.shards {
					l.shards[i].index = map[exploreKey]int32{}
				}
				layers[m] = l
			}
		}
	}
	push := func(local map[liveKey]liveInfo, s State, m dsSet) {
		if m.empty() {
			return
		}
		s.Stats[StatDuty], s.Stats[StatSuspicion] = 0, 0
		k := x.key(local, &s, true)
		sh := &layers[s.Minutes].shards[shardOf(k)] // pre-created: no lock needed for the lookup
		sh.mu.Lock()
		if i, ok := sh.index[k]; ok {
			sh.entries[i].m = sh.entries[i].m.or(m)
		} else {
			sh.index[k] = int32(len(sh.entries))
			sh.entries = append(sh.entries, x.pack(&s, m))
		}
		sh.mu.Unlock()
	}
	enter := func(local map[liveKey]liveInfo, s State, node int, m dsSet) {
		s.Node = node
		if s.Visits[node] < 255 {
			s.Visits[node]++
		}
		m = applyDS(st.Nodes[node].OnEnter, &s, m)
		push(local, s, m)
	}
	type partial struct {
		states, concrete int
		endings          map[string]int
		reached          []bool
		dead             []State
	}
	expand := func(local map[liveKey]liveInfo, e *symEntry, minute int, out *partial) {
		cur := x.unpack(e, minute)
		s, m := &cur, e.m
		if opts.observe != nil {
			opts.observe(minute, s, m)
		}
		out.states++
		out.concrete += m.count()
		out.reached[s.Node] = true
		if s.Ending != "" {
			out.endings[s.Ending]++
			return
		}
		var covered dsSet
		for _, c := range st.Nodes[s.Node].Choices {
			avail := m.and(evalDS(c.Requires, s))
			if avail.empty() {
				continue
			}
			covered = covered.or(avail)
			ns := *s
			ns.Minutes += c.Cost
			am := applyDS(c.Effects, &ns, avail)
			t := st.Nodes[c.Target]
			switch {
			case t.Kind != KindEnding && !t.LateOK && ns.Minutes >= st.Deadline:
				ns.Minutes = st.Deadline
				enter(local, ns, st.DeadlineNode, am)
			case t.Kind == KindHub:
				rest := am
				for ei, ev := range st.Events {
					if rest.empty() {
						break
					}
					if ns.Fired&(1<<uint(ei)) != 0 || ns.Minutes < ev.At || (ev.Until > 0 && ns.Minutes >= ev.Until) {
						continue
					}
					em := rest.and(evalDS(ev.When, &ns))
					if em.empty() {
						continue
					}
					es := ns
					es.Fired |= 1 << uint(ei)
					enter(local, es, ev.Target, em)
					rest = rest.andNot(em)
				}
				enter(local, ns, c.Target, rest)
			default:
				enter(local, ns, c.Target, am)
			}
		}
		if dead := m.andNot(covered); !dead.empty() && len(out.dead) < 10 {
			d, sp := dead.first()
			ds := *s
			ds.Stats[StatDuty], ds.Stats[StatSuspicion] = d, sp
			out.dead = append(out.dead, ds)
		}
	}
	workers := max(1, runtime.GOMAXPROCS(0))
	locals := make([]map[liveKey]liveInfo, workers)
	for i := range locals {
		locals[i] = map[liveKey]liveInfo{}
	}
	ensure(maxStep)
	startState := st.NewState()
	push(locals[0], startState, dsPoint(startState.Stats[StatDuty], startState.Stats[StatSuspicion]))
	pending := func(from int) bool {
		for m := from; m < len(layers); m++ {
			if l := layers[m]; l != nil {
				for i := range l.shards {
					if len(l.shards[i].entries) > 0 {
						return true
					}
				}
			}
		}
		return false
	}
	for minute := 0; minute < len(layers); minute++ {
		if opts.horizon > 0 && minute >= opts.horizon {
			break
		}
		l := layers[minute]
		layers[minute] = nil // free it once processed
		ensure(minute + maxStep)
		var all []*symEntry
		for i := range l.shards {
			sh := &l.shards[i]
			sh.index = nil // nothing can reach this minute any more
			for j := range sh.entries {
				all = append(all, &sh.entries[j])
			}
		}
		if len(all) == 0 {
			if !pending(minute + 1) {
				break
			}
			continue
		}
		rep.WidestLayer = max(rep.WidestLayer, len(all))
		if limit > 0 && rep.States > limit {
			rep.Truncated = true
			break
		}
		parts := make([]partial, workers)
		var next atomic.Int64
		var wg sync.WaitGroup
		const block = 512
		for w := 0; w < workers; w++ {
			parts[w] = partial{endings: map[string]int{}, reached: make([]bool, len(st.Nodes))}
			wg.Add(1)
			go func(w int) {
				defer wg.Done()
				for {
					lo := int(next.Add(block)) - block
					if lo >= len(all) {
						return
					}
					for _, e := range all[lo:min(lo+block, len(all))] {
						expand(locals[w], e, minute, &parts[w])
					}
				}
			}(w)
		}
		wg.Wait()
		for _, p := range parts {
			rep.States += p.states
			rep.Concrete += p.concrete
			for k, v := range p.endings {
				rep.Endings[k] += v
			}
			for i, r := range p.reached {
				if r {
					rep.NodeReached[i] = true
				}
			}
			for _, d := range p.dead {
				if len(rep.DeadEnds) < 10 {
					rep.DeadEnds = append(rep.DeadEnds, d)
				}
			}
		}
	}
	rep.LiveSets = len(x.liveIDs)
	for _, l := range locals {
		rep.LiveKeys += len(l)
	}
	return rep
}

// ScriptNumbers converts a path of choice ids into the 1-based numbers a
// player would type (choices are numbered among the available ones).
func (st *Story) ScriptNumbers(ids []string) ([]int, error) {
	s := st.NewState()
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		num := 0
		for i, ci := range st.Available(&s) {
			if st.Nodes[s.Node].Choices[ci].ID == id {
				num = i + 1
			}
		}
		if num == 0 {
			return nil, &PathError{Node: st.Nodes[s.Node].ID, Choice: id, Clock: st.Clock(&s)}
		}
		if _, err := st.ChooseByID(&s, id); err != nil {
			return nil, err
		}
		out = append(out, num)
	}
	return out, nil
}

// PathError reports a choice id that is not available where a path expects it.
type PathError struct{ Node, Choice, Clock string }

func (e *PathError) Error() string {
	return "choice " + e.Choice + " not available at node " + e.Node + " (" + e.Clock + ")"
}
