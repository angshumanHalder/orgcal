package org

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// findHeadingIdx returns the line index of the todo's heading.
// Uses todo.Line as primary locator; falls back to title scan.
func findHeadingIdx(lines []string, todo *Todo) int {
	if todo.Line > 0 && todo.Line < len(lines) && headingRe.MatchString(lines[todo.Line]) {
		return todo.Line
	}
	for i, line := range lines {
		if m := headingRe.FindStringSubmatch(line); m != nil {
			title := strings.TrimSpace(m[4])
			if tm := tagsRe.FindStringSubmatch(title); tm != nil {
				title = strings.TrimSpace(tagsRe.ReplaceAllString(title, ""))
			}
			if title == todo.Title {
				return i
			}
		}
	}
	return -1
}

// WriteGcalProps writes :GCAL_ID: and :GCAL_ETAG: into the todo's PROPERTIES drawer.
// Replaces WriteGcalID and handles both insert and update.
func WriteGcalProps(todo *Todo) error {
	path := expandHome(todo.File)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")

	headingIdx := findHeadingIdx(lines, todo)
	if headingIdx == -1 {
		return nil
	}

	propsStart, propsEnd := -1, -1
	gcalIDLine, gcalEtagLine := -1, -1

	for i := headingIdx + 1; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if strings.HasPrefix(trimmed, "*") {
			break
		}
		switch {
		case trimmed == ":PROPERTIES:":
			propsStart = i
		case propsStart >= 0 && strings.HasPrefix(trimmed, ":GCAL_ID:"):
			gcalIDLine = i
		case propsStart >= 0 && strings.HasPrefix(trimmed, ":GCAL_ETAG:"):
			gcalEtagLine = i
		case trimmed == ":END:" && propsStart >= 0:
			propsEnd = i
		}
		if propsEnd >= 0 {
			break
		}
	}

	var out []string

	if propsStart >= 0 && propsEnd >= 0 {
		for i, line := range lines {
			switch {
			case i == gcalIDLine && todo.GcalID != "":
				out = append(out, "  :GCAL_ID: "+todo.GcalID)
			case i == gcalEtagLine:
				if todo.GcalEtag != "" {
					out = append(out, "  :GCAL_ETAG: "+todo.GcalEtag)
				} else {
					out = append(out, line)
				}
			case i == propsEnd:
				if gcalIDLine == -1 && todo.GcalID != "" {
					out = append(out, "  :GCAL_ID: "+todo.GcalID)
				}
				if gcalEtagLine == -1 && todo.GcalEtag != "" {
					out = append(out, "  :GCAL_ETAG: "+todo.GcalEtag)
				}
				out = append(out, line)
			default:
				out = append(out, line)
			}
		}
	} else {
		insertAt := headingIdx + 1
		for insertAt < len(lines) {
			t := strings.TrimSpace(lines[insertAt])
			if strings.HasPrefix(t, "SCHEDULED:") || strings.HasPrefix(t, "DEADLINE:") {
				insertAt++
			} else {
				break
			}
		}
		props := []string{"  :PROPERTIES:"}
		if todo.GcalID != "" {
			props = append(props, "  :GCAL_ID: "+todo.GcalID)
		}
		if todo.GcalEtag != "" {
			props = append(props, "  :GCAL_ETAG: "+todo.GcalEtag)
		}
		props = append(props, "  :END:")
		out = append(out, lines[:insertAt]...)
		out = append(out, props...)
		out = append(out, lines[insertAt:]...)
	}

	return os.WriteFile(path, []byte(strings.Join(out, "\n")), 0644)
}

// WriteScheduled updates the SCHEDULED: timestamp for a todo (used in gcal-wins resolution).
func WriteScheduled(todo *Todo) error {
	path := expandHome(todo.File)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")

	headingIdx := findHeadingIdx(lines, todo)
	if headingIdx == -1 {
		return nil
	}

	var newLine string
	if todo.AllDay {
		newLine = fmt.Sprintf("  SCHEDULED: <%s>", todo.Scheduled.Format("2006-01-02 Mon"))
	} else {
		newLine = fmt.Sprintf("  SCHEDULED: <%s>", todo.Scheduled.Format("2006-01-02 Mon 15:04"))
	}

	for i := headingIdx + 1; i < len(lines); i++ {
		if headingRe.MatchString(lines[i]) {
			break
		}
		if scheduledRe.MatchString(lines[i]) {
			lines[i] = newLine
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		}
	}
	return nil
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
