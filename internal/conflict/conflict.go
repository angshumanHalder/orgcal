package conflict

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Field struct {
	Name   string `json:"name"`
	Local  string `json:"local"`
	Remote string `json:"remote"`
}

type Conflict struct {
	GcalID     string  `json:"gcal_id"`
	Title      string  `json:"title"`
	File       string  `json:"file"`
	Line       int     `json:"line"`
	Fields     []Field `json:"fields"`
	Resolution string  `json:"resolution,omitempty"` // "local", "gcal", "skip"
}

func conflictsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "orgcal")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "conflicts.json"), nil
}

func Load() ([]*Conflict, error) {
	p, err := conflictsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var cs []*Conflict
	if err := json.Unmarshal(data, &cs); err != nil {
		return nil, err
	}
	return cs, nil
}

func Save(cs []*Conflict) error {
	p, err := conflictsPath()
	if err != nil {
		return err
	}
	if len(cs) == 0 {
		_ = os.Remove(p)
		return nil
	}
	data, err := json.MarshalIndent(cs, "", "  ")
	if err != nil {
		return err
	}
	tmp := p + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return err
	}
	return os.Rename(tmp, p)
}

// Merge adds new conflicts, deduplicating by GcalID (preserves existing resolutions).
func Merge(existing, incoming []*Conflict) []*Conflict {
	seen := make(map[string]*Conflict, len(existing))
	for _, c := range existing {
		seen[c.GcalID] = c
	}
	for _, c := range incoming {
		if _, ok := seen[c.GcalID]; !ok {
			seen[c.GcalID] = c
			existing = append(existing, c)
		}
	}
	return existing
}

func PendingCount(cs []*Conflict) int {
	n := 0
	for _, c := range cs {
		if c.Resolution == "" {
			n++
		}
	}
	return n
}
