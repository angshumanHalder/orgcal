package sync

import (
	"errors"

	"google.golang.org/api/googleapi"

	"github.com/angshumanhalder/orgcal/internal/conflict"
	"github.com/angshumanhalder/orgcal/internal/gcal"
	"github.com/angshumanhalder/orgcal/internal/org"
	"github.com/angshumanhalder/orgcal/internal/state"
)

type Result struct {
	Imported  int
	Exported  int
	Deleted   int
	Conflicts int
}

func Run(dir string) (*Result, error) {
	client, err := gcal.NewClient()
	if err != nil {
		return nil, err
	}

	st, err := state.Load()
	if err != nil {
		return nil, err
	}

	prevImported := make(map[string]bool, len(st.ImportedIDs))
	for _, id := range st.ImportedIDs {
		prevImported[id] = true
	}

	currentOrgIDs, err := org.ReadCalendarEventIDs(dir)
	if err != nil {
		return nil, err
	}
	currentOrgIDSet := make(map[string]bool, len(currentOrgIDs))
	for _, id := range currentOrgIDs {
		currentOrgIDSet[id] = true
	}

	// IDs in prev state but missing from calendar.org = user deleted them
	deleted := 0
	for id := range prevImported {
		if !currentOrgIDSet[id] {
			if err := client.DeleteEvent(id); err != nil {
				var gErr *googleapi.Error
				if !errors.As(err, &gErr) || gErr.Code != 410 {
					return nil, err
				}
				// 410 = already deleted on GCal side, treat as success
			}
			deleted++
		}
	}

	todos, err := org.ReadTodos(dir)
	if err != nil {
		return nil, err
	}

	exported, gcalIDs, newConflicts, err := client.ExportTodos(todos)
	if err != nil {
		return nil, err
	}

	// Merge new conflicts with existing unresolved ones
	if len(newConflicts) > 0 {
		existing, err := conflict.Load()
		if err != nil {
			return nil, err
		}
		merged := conflict.Merge(existing, newConflicts)
		if err := conflict.Save(merged); err != nil {
			return nil, err
		}
	}

	currentExported := make(map[string]bool, len(gcalIDs))
	for _, id := range gcalIDs {
		currentExported[id] = true
	}

	// IDs in prev exported set but gone from org = user deleted the todo heading
	for _, id := range st.ExportedTodoIDs {
		if !currentExported[id] {
			if err := client.DeleteEvent(id); err != nil {
				var gErr *googleapi.Error
				if !errors.As(err, &gErr) || gErr.Code != 410 {
					return nil, err
				}
			}
			deleted++
		}
	}

	exportedSet := make(map[string]bool, len(gcalIDs))
	for _, id := range gcalIDs {
		exportedSet[id] = true
	}
	for _, id := range st.ExportedTodoIDs {
		exportedSet[id] = true
	}

	events, err := client.ListEvents()
	if err != nil {
		return nil, err
	}

	var calEvents []*org.Event
	for _, e := range events {
		if !exportedSet[e.ID] {
			calEvents = append(calEvents, e)
		}
	}
	if err := org.WriteEvents(dir, calEvents); err != nil {
		return nil, err
	}

	st.ImportedIDs = make([]string, 0, len(calEvents))
	for _, e := range calEvents {
		st.ImportedIDs = append(st.ImportedIDs, e.ID)
	}
	st.ExportedTodoIDs = gcalIDs

	if err := state.Save(st); err != nil {
		return nil, err
	}

	return &Result{
		Imported:  len(calEvents),
		Exported:  exported,
		Deleted:   deleted,
		Conflicts: len(newConflicts),
	}, nil
}
