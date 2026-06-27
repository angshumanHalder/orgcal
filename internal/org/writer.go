package org

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func WriteEvents(dir string, events []*Event) error {
	dir = expandHome(dir)
	gcalDir := filepath.Join(dir, "gcal")
	if err := os.MkdirAll(gcalDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(gcalDir, "calendar.org")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "#+TITLE: Google Calendar Events\n#+STARTUP: overview\n\n")

	for _, e := range events {
		fmt.Fprintf(f, "* %s\n", e.Title)
		if e.AllDay {
			fmt.Fprintf(f, "  SCHEDULED: <%s>\n", e.Start.Format("2006-01-02 Mon"))
		} else {
			fmt.Fprintf(f, "  SCHEDULED: <%s>--<%s>\n",
				e.Start.Format("2006-01-02 Mon 15:04"),
				e.End.Format("2006-01-02 Mon 15:04"),
			)
		}
		fmt.Fprintf(f, "  :PROPERTIES:\n")
		fmt.Fprintf(f, "  :GCAL_ID: %s\n", e.ID)
		if e.Location != "" {
			fmt.Fprintf(f, "  :LOCATION: %s\n", e.Location)
		}
		fmt.Fprintf(f, "  :GCAL_UPDATED: %s\n", time.Now().Format(time.RFC3339))
		fmt.Fprintf(f, "  :END:\n")
		if e.Notes != "" {
			fmt.Fprintf(f, "\n  %s\n", e.Notes)
		}
		fmt.Fprintln(f)
	}
	return nil
}
