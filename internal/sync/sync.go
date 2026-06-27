package sync

import (
	"errors"

	"google.golang.org/api/googleapi"

	"github.com/angshumanhalder/orgcal/internal/gcal"
	"github.com/angshumanhalder/orgcal/internal/org"
	"github.com/angshumanhalder/orgcal/internal/state"
)

type Result struct {
	Imported int
	Exported int
	Deleted  int
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

	// build set of IDs imported on previous sync
	prevImported := make(map[string]bool, len(st.ImportedIDs))
	for _, id := range st.ImportedIDs {
		prevImported[id] = true
	}

	// read IDs currently in calendar.org before overwriting
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
			}
			deleted++
		}
	}

	// read todos before WriteEvents so we can filter out exported IDs
	todos, err := org.ReadTodos(dir)
	if err != nil {
		return nil, err
	}

	// export todos first to capture new GcalIDs written back to org files
	exported, gcalIDs, err := client.ExportTodos(todos)
	if err != nil {
		return nil, err
	}

	// build set of GcalIDs that belong to org todos — exclude from calendar import
	exportedSet := make(map[string]bool, len(gcalIDs))
	for _, id := range gcalIDs {
		exportedSet[id] = true
	}
	// also exclude previously tracked exported todo IDs
	for _, id := range st.ExportedTodoIDs {
		exportedSet[id] = true
	}

	events, err := client.ListEvents()
	if err != nil {
		return nil, err
	}

	// filter out events that are org todos — prevents duplicates in calendar.org
	var calEvents []*org.Event
	for _, e := range events {
		if !exportedSet[e.ID] {
			calEvents = append(calEvents, e)
		}
	}
	if err := org.WriteEvents(dir, calEvents); err != nil {
		return nil, err
	}

	// update state
	st.ImportedIDs = make([]string, 0, len(calEvents))
	for _, e := range calEvents {
		st.ImportedIDs = append(st.ImportedIDs, e.ID)
	}
	st.ExportedTodoIDs = gcalIDs

	if err := state.Save(st); err != nil {
		return nil, err
	}

	return &Result{Imported: len(calEvents), Exported: exported, Deleted: deleted}, nil
}
