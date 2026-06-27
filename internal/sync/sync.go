package sync

import (
	"github.com/angshumanhalder/orgcal/internal/gcal"
	"github.com/angshumanhalder/orgcal/internal/org"
)

type Result struct {
	Imported int
	Exported int
}

func Run(dir string) (*Result, error) {
	client, err := gcal.NewClient()
	if err != nil {
		return nil, err
	}

	events, err := client.ListEvents()
	if err != nil {
		return nil, err
	}
	if err := org.WriteEvents(dir, events); err != nil {
		return nil, err
	}

	todos, err := org.ReadTodos(dir)
	if err != nil {
		return nil, err
	}
	exported, err := client.ExportTodos(todos)
	if err != nil {
		return nil, err
	}

	return &Result{Imported: len(events), Exported: exported}, nil
}
