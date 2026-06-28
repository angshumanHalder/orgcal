package cmd

import (
	"errors"
	"fmt"

	"github.com/angshumanhalder/orgcal/internal/conflict"
	"github.com/angshumanhalder/orgcal/internal/gcal"
	"github.com/angshumanhalder/orgcal/internal/org"
	"github.com/spf13/cobra"
	googleapi "google.golang.org/api/googleapi"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Apply pending conflict resolutions from conflicts.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		cs, err := conflict.Load()
		if err != nil {
			return err
		}
		if len(cs) == 0 {
			fmt.Println("No conflicts.")
			return nil
		}

		hasWork := false
		for _, c := range cs {
			if c.Resolution != "" {
				hasWork = true
				break
			}
		}
		if !hasWork {
			fmt.Println("No pending conflicts.")
			return nil
		}

		client, err := gcal.NewClient()
		if err != nil {
			return err
		}

		var remaining []*conflict.Conflict
		resolved := 0

		for _, c := range cs {
			switch c.Resolution {
			case "gcal":
				if err := applyGcalWins(client, c); err != nil {
					fmt.Printf("warn: gcal resolution failed for %q: %v\n", c.Title, err)
					remaining = append(remaining, c)
					continue
				}
				resolved++
			case "local":
				if err := applyLocalWins(client, c); err != nil {
					fmt.Printf("warn: local resolution failed for %q: %v\n", c.Title, err)
					remaining = append(remaining, c)
					continue
				}
				resolved++
			case "skip":
				resolved++
			default:
				remaining = append(remaining, c)
			}
		}

		if err := conflict.Save(remaining); err != nil {
			return err
		}
		fmt.Printf("Resolved: %d  Pending: %d\n", resolved, len(remaining))
		return nil
	},
}

func applyGcalWins(client *gcal.Client, c *conflict.Conflict) error {
	ev, err := client.GetEvent("primary", c.GcalID)
	if err != nil {
		var gErr *googleapi.Error
		if errors.As(err, &gErr) && gErr.Code == 410 {
			return nil // already deleted on GCal, nothing to update
		}
		return err
	}

	todo := &org.Todo{
		GcalID:    c.GcalID,
		GcalEtag:  ev.Etag,
		File:      c.File,
		Line:      c.Line,
		Title:     ev.Summary,
		Scheduled: ev.Start,
		AllDay:    ev.AllDay,
	}
	if err := org.WriteGcalProps(todo); err != nil {
		return err
	}
	if !todo.Scheduled.IsZero() {
		return org.WriteScheduled(todo)
	}
	return nil
}

func applyLocalWins(client *gcal.Client, c *conflict.Conflict) error {
	todos, err := org.ReadTodosFromFile(c.File)
	if err != nil {
		return err
	}
	var todo *org.Todo
	for _, t := range todos {
		if t.GcalID == c.GcalID {
			todo = t
			break
		}
	}
	if todo == nil {
		return fmt.Errorf("todo with gcal_id %s not found in %s", c.GcalID, c.File)
	}
	newEtag, err := client.ForceUpdateEvent(todo, "primary")
	if err != nil {
		return err
	}
	todo.GcalEtag = newEtag
	return org.WriteGcalProps(todo)
}
