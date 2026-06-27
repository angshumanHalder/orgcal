package org

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteGcalID writes the GcalID back into the todo's source org file.
// Finds the heading by title and inserts/updates :GCAL_ID: in its PROPERTIES drawer.
func WriteGcalID(todo *Todo) error {
	path := expandHome(todo.File)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	lines := strings.Split(string(data), "\n")

	// find first heading matching this title
	headingIdx := -1
	for i, line := range lines {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			title := strings.TrimSpace(m[4])
			if tm := tagsRe.FindStringSubmatch(title); tm != nil {
				title = strings.TrimSpace(tagsRe.ReplaceAllString(title, ""))
			}
			if title == todo.Title {
				headingIdx = i
				break
			}
		}
	}
	if headingIdx == -1 {
		return nil
	}

	propsStart, propsEnd, gcalLine := -1, -1, -1
	for i := headingIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "*") {
			break
		}
		if trimmed == ":PROPERTIES:" {
			propsStart = i
		}
		if propsStart >= 0 && strings.HasPrefix(trimmed, ":GCAL_ID:") {
			gcalLine = i
		}
		if trimmed == ":END:" && propsStart >= 0 {
			propsEnd = i
			break
		}
	}

	var out []string
	switch {
	case gcalLine >= 0:
		// update existing line
		for i, line := range lines {
			if i == gcalLine {
				out = append(out, "  :GCAL_ID: "+todo.GcalID)
			} else {
				out = append(out, line)
			}
		}
	case propsStart >= 0 && propsEnd >= 0:
		// insert before :END:
		for i, line := range lines {
			if i == propsEnd {
				out = append(out, "  :GCAL_ID: "+todo.GcalID)
			}
			out = append(out, line)
		}
	default:
		// no PROPERTIES block — find insert point after SCHEDULED/DEADLINE lines
		insertAt := headingIdx + 1
		for insertAt < len(lines) {
			t := strings.TrimSpace(lines[insertAt])
			if strings.HasPrefix(t, "SCHEDULED:") || strings.HasPrefix(t, "DEADLINE:") {
				insertAt++
			} else {
				break
			}
		}
		out = append(out, lines[:insertAt]...)
		out = append(out, "  :PROPERTIES:", "  :GCAL_ID: "+todo.GcalID, "  :END:")
		out = append(out, lines[insertAt:]...)
	}

	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

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
