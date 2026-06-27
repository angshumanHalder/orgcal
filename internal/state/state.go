package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type State struct {
	ImportedIDs     []string `json:"imported_ids"`
	ExportedTodoIDs []string `json:"exported_todo_ids"`
}

func statePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".local", "share", "orgcal")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(dir, "state.json"), nil
}

func Load() (*State, error) {
	p, err := statePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return &State{}, nil
	}
	if err != nil {
		return nil, err
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return &State{}, nil
	}
	return &s, nil
}

func Save(s *State) error {
	p, err := statePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}
