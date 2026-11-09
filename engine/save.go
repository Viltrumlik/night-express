package engine

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// HomeEnv overrides the directory used for saves and progress.
const HomeEnv = "NIGHT_EXPRESS_HOME"

// HomeDir returns the data directory (~/.night-express unless NIGHT_EXPRESS_HOME is set).
func HomeDir() (string, error) {
	if d := os.Getenv(HomeEnv); d != "" {
		return d, nil
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(h, ".night-express"), nil
}

func dataPath(name string) (string, error) {
	d, err := HomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, name), nil
}

// SaveData is the on-disk save format (names, not indices, so saves survive graph edits).
type SaveData struct {
	Version   int            `json:"version"`
	Node      string         `json:"node"`
	Minutes   int            `json:"minutes"`
	Trust     int            `json:"trust"`
	Duty      int            `json:"duty"`
	Suspicion int            `json:"suspicion"`
	Money     int            `json:"money"`
	Evidence  int            `json:"evidence"`
	Flags     []string       `json:"flags"`
	Fired     []string       `json:"fired"`
	Visits    map[string]int `json:"visits"`
	Ending    string         `json:"ending,omitempty"`
	Lang      string         `json:"lang"`
	Hard      bool           `json:"hard"`
	Seed      int64          `json:"seed"`
}

// Snapshot converts a state into save data.
func (st *Story) Snapshot(s *State, lang string, hard bool, seed int64) SaveData {
	d := SaveData{
		Version: 1, Node: st.Nodes[s.Node].ID, Minutes: s.Minutes,
		Trust: s.Stats[StatTrust], Duty: s.Stats[StatDuty], Suspicion: s.Stats[StatSuspicion], Money: s.Stats[StatMoney],
		Evidence: s.Stats[StatEvidence],
		Flags:    []string{}, Fired: []string{}, Visits: map[string]int{}, Ending: s.Ending,
		Lang: lang, Hard: hard, Seed: seed,
	}
	for i, name := range st.Flags {
		if s.Flags.Has(i) {
			d.Flags = append(d.Flags, name)
		}
	}
	sort.Strings(d.Flags)
	for i, e := range st.Events {
		if s.Fired&(1<<uint(i)) != 0 {
			d.Fired = append(d.Fired, e.ID)
		}
	}
	for i, n := range st.Nodes {
		if s.Visits[i] > 0 {
			d.Visits[n.ID] = int(s.Visits[i])
		}
	}
	return d
}

// Restore converts save data back into a state.
func (st *Story) Restore(d SaveData) (State, error) {
	var s State
	if d.Version != 1 {
		return s, fmt.Errorf("unsupported save version %d", d.Version)
	}
	n, ok := st.NodeIdx[d.Node]
	if !ok {
		return s, fmt.Errorf("save refers to unknown node %q", d.Node)
	}
	s.Node, s.Minutes, s.Ending = n, d.Minutes, d.Ending
	s.Stats = [NumStats]int{
		clamp(d.Trust, 0, StatMax), clamp(d.Duty, 0, StatMax), clamp(d.Suspicion, 0, StatMax), max(d.Money, 0),
		clamp(d.Evidence, 0, StatMax),
	}
	for _, f := range d.Flags {
		i, ok := st.FlagIdx[f]
		if !ok {
			return s, fmt.Errorf("save refers to unknown flag %q", f)
		}
		s.Flags.Set(i)
	}
	for _, id := range d.Fired {
		found := false
		for i, e := range st.Events {
			if e.ID == id {
				s.Fired |= 1 << uint(i)
				found = true
			}
		}
		if !found {
			return s, fmt.Errorf("save refers to unknown event %q", id)
		}
	}
	for id, v := range d.Visits {
		i, ok := st.NodeIdx[id]
		if !ok {
			return s, fmt.Errorf("save refers to unknown node %q", id)
		}
		s.Visits[i] = uint8(clamp(v, 0, 255))
	}
	return s, nil
}

func writeJSON(path string, v any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// WriteSave stores the save file.
func WriteSave(d SaveData) error {
	p, err := dataPath("save.json")
	if err != nil {
		return err
	}
	return writeJSON(p, d)
}

// ReadSave loads the save file.
func ReadSave() (SaveData, error) {
	var d SaveData
	p, err := dataPath("save.json")
	if err != nil {
		return d, err
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return d, err
	}
	err = json.Unmarshal(b, &d)
	return d, err
}

// HasSave reports whether a save file exists.
func HasSave() bool {
	p, err := dataPath("save.json")
	if err != nil {
		return false
	}
	_, err = os.Stat(p)
	return err == nil
}

// RemoveSave deletes the save file (used once a night has ended).
func RemoveSave() error {
	p, err := dataPath("save.json")
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

// Progress records which endings have been discovered across runs.
type Progress struct {
	Endings []string `json:"endings"`
}

// Has reports whether the ending was already discovered.
func (p *Progress) Has(id string) bool {
	for _, e := range p.Endings {
		if e == id {
			return true
		}
	}
	return false
}

// Add records an ending; it returns true if it was new.
func (p *Progress) Add(id string) bool {
	if p.Has(id) {
		return false
	}
	p.Endings = append(p.Endings, id)
	sort.Strings(p.Endings)
	return true
}

// ReadProgress loads progress; a missing file yields empty progress.
func ReadProgress() (Progress, error) {
	p := Progress{Endings: []string{}}
	path, err := dataPath("progress.json")
	if err != nil {
		return p, err
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return p, nil
	}
	if err != nil {
		return p, err
	}
	if err := json.Unmarshal(b, &p); err != nil {
		return Progress{Endings: []string{}}, err
	}
	if p.Endings == nil {
		p.Endings = []string{}
	}
	return p, nil
}

// WriteProgress stores progress.
func WriteProgress(p Progress) error {
	path, err := dataPath("progress.json")
	if err != nil {
		return err
	}
	return writeJSON(path, p)
}
