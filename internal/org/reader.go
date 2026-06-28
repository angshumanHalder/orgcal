package org

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	headingRe    = regexp.MustCompile(`^(\*+)\s+(TODO|NEXT|DONE|CANCELLED)?\s*(?:\[#([ABC])\]\s*)?(.+)`)
	fileTagsRe   = regexp.MustCompile(`^#\+FILETAGS:\s+(.+)`)
	scheduledRe  = regexp.MustCompile(`SCHEDULED:\s+<([^>]+)>(?:--<([^>]+)>)?`)
	deadlineRe   = regexp.MustCompile(`DEADLINE:\s+<([^>]+)>`)
	gcalIDRe     = regexp.MustCompile(`:GCAL_ID:\s+(\S+)`)
	calendarIDRe = regexp.MustCompile(`:CALENDAR_ID:\s+(\S+)`)
	exportRe     = regexp.MustCompile(`:EXPORT_TO_GCAL:\s+(\S+)`)
	repeaterRe   = regexp.MustCompile(`([.+]{1,2})(\d+)([dwmy])`)
	tagsRe       = regexp.MustCompile(`\s+:([\w:]+):\s*$`)
	propBlockRe  = regexp.MustCompile(`^\s*:(PROPERTIES|END):`)
	propLineRe   = regexp.MustCompile(`^\s*:(\w+):\s+(.+)`)
)

func ReadTodos(dir string) ([]*Todo, error) {
	dir = expandHome(dir)
	var todos []*Todo

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".org") {
			return err
		}
		fileTodos, err := parseTodos(path)
		if err != nil {
			return err
		}
		todos = append(todos, fileTodos...)
		return nil
	})
	return todos, err
}

// ReadTodosFromFile reads todos from a single org file.
func ReadTodosFromFile(path string) ([]*Todo, error) {
	return parseTodos(path)
}

func parseTodos(path string) ([]*Todo, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(data), "\n")
	var todos []*Todo
	var current *Todo
	var fileTags []string
	inProps := false
	inBody := false

	flush := func() {
		if current == nil {
			return
		}
		current.Body = strings.TrimSpace(current.Body)
		current.FileTags = fileTags
		isDone := current.State == "DONE" || current.State == "CANCELLED"
		isActive := current.ExportToGcal || current.State == "TODO" || current.State == "NEXT"
		hasDate := !current.Scheduled.IsZero() || !current.Deadline.IsZero()
		if (isActive && hasDate) || (isDone && current.GcalID != "") {
			todos = append(todos, current)
		}
	}

	for i, line := range lines {
		if m := fileTagsRe.FindStringSubmatch(line); m != nil {
			fileTags = parseTags(m[1])
			continue
		}

		if m := headingRe.FindStringSubmatch(line); m != nil {
			flush()
			inProps = false
			inBody = false

			state := m[2]
			priority := m[3]
			rest := strings.TrimSpace(m[4])

			var tags []string
			if tm := tagsRe.FindStringSubmatch(rest); tm != nil {
				tags = parseTags(tm[1])
				rest = strings.TrimSpace(tagsRe.ReplaceAllString(rest, ""))
			}

			current = &Todo{
				Title:    rest,
				State:    state,
				Priority: priority,
				Tags:     tags,
				File:     path,
				Line:     i,
			}
			continue
		}

		if current == nil {
			continue
		}

		if propBlockRe.MatchString(line) {
			if strings.Contains(line, "PROPERTIES") {
				inProps = true
				inBody = false
			} else {
				inProps = false
				inBody = true
			}
			continue
		}

		if inProps {
			if m := propLineRe.FindStringSubmatch(line); m != nil {
				switch m[1] {
				case "GCAL_ID":
					current.GcalID = strings.TrimSpace(m[2])
				case "GCAL_ETAG":
					current.GcalEtag = strings.TrimSpace(m[2])
				case "CALENDAR_ID":
					current.CalendarID = strings.TrimSpace(m[2])
				case "EXPORT_TO_GCAL":
					v := strings.TrimSpace(m[2])
					current.ExportToGcal = v == "t" || v == "yes" || v == "true" || v == "1"
				}
			}
			continue
		}

		if m := scheduledRe.FindStringSubmatch(line); m != nil {
			current.Scheduled, current.AllDay, current.Repeater = parseOrgTimestamp(m[1])
			if m[2] != "" {
				current.ScheduledEnd, _, _ = parseOrgTimestamp(m[2])
			}
			continue
		}
		if m := deadlineRe.FindStringSubmatch(line); m != nil {
			current.Deadline, _, _ = parseOrgTimestamp(m[1])
			continue
		}

		if inBody && !strings.HasPrefix(strings.TrimSpace(line), ":") {
			current.Body += line + "\n"
		}
	}

	flush()
	return todos, nil
}

func parseOrgTimestamp(s string) (t time.Time, allDay bool, rep *Repeater) {
	var repeater *Repeater
	if m := repeaterRe.FindStringSubmatch(s); m != nil {
		val, _ := strconv.Atoi(m[2])
		repeater = &Repeater{Type: m[1], Value: val, Unit: m[3]}
	}

	clean := regexp.MustCompile(`\s+[A-Z][a-z]{2}`).ReplaceAllString(s, "")
	clean = repeaterRe.ReplaceAllString(clean, "")
	clean = strings.TrimSpace(clean)

	if parsed, err := time.ParseInLocation("2006-01-02 15:04", clean, time.Local); err == nil {
		return parsed, false, repeater
	}
	if parsed, err := time.ParseInLocation("2006-01-02", clean, time.Local); err == nil {
		return parsed, true, repeater
	}
	return time.Time{}, false, nil
}

func parseTags(s string) []string {
	var tags []string
	for _, t := range strings.Split(strings.Trim(s, ":"), ":") {
		if t = strings.TrimSpace(t); t != "" {
			tags = append(tags, t)
		}
	}
	return tags
}

// ReadCalendarEventIDs returns GCAL_IDs currently present in gcal/calendar.org.
func ReadCalendarEventIDs(dir string) ([]string, error) {
	path := filepath.Join(expandHome(dir), "gcal", "calendar.org")
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, line := range strings.Split(string(data), "\n") {
		if m := gcalIDRe.FindStringSubmatch(line); m != nil {
			ids = append(ids, m[1])
		}
	}
	return ids, nil
}

func expandHome(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}

// suppress unused variable warnings for regexes only used in writer.go
var _ = calendarIDRe
var _ = exportRe
