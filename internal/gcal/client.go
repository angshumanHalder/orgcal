package gcal

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/angshumanhalder/orgcal/internal/conflict"
	"github.com/angshumanhalder/orgcal/internal/org"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"

	"github.com/angshumanhalder/orgcal/internal/auth"
)

type Client struct {
	svc        *googlecalendar.Service
	calendarID string
}

func NewClient() (*Client, error) {
	ctx := context.Background()
	httpClient, err := auth.GetHTTPClient(ctx)
	if err != nil {
		return nil, err
	}
	svc, err := googlecalendar.NewService(ctx, option.WithHTTPClient(httpClient))
	if err != nil {
		return nil, fmt.Errorf("failed to create calendar service: %w", err)
	}
	return &Client{svc: svc, calendarID: "primary"}, nil
}

func (c *Client) ListEvents() ([]*org.Event, error) {
	now := time.Now()
	tMin := now.Format(time.RFC3339)
	tMax := now.AddDate(0, 1, 0).Format(time.RFC3339)

	resp, err := c.svc.Events.List(c.calendarID).
		TimeMin(tMin).
		TimeMax(tMax).
		SingleEvents(true).
		OrderBy("startTime").
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to list events: %w", err)
	}

	var events []*org.Event
	for _, item := range resp.Items {
		e := &org.Event{
			ID:       item.Id,
			Etag:     item.Etag,
			Title:    item.Summary,
			Location: item.Location,
			Notes:    item.Description,
		}
		if item.Start.DateTime != "" {
			e.Start, _ = time.Parse(time.RFC3339, item.Start.DateTime)
		} else {
			e.Start, _ = time.Parse("2006-01-02", item.Start.Date)
			e.AllDay = true
		}
		if item.End.DateTime != "" {
			e.End, _ = time.Parse(time.RFC3339, item.End.DateTime)
		}
		events = append(events, e)
	}
	return events, nil
}

// GcalEvent is a simplified view of a Google Calendar event for use in conflict resolution.
type GcalEvent struct {
	Etag      string
	Summary   string
	Start     time.Time
	AllDay    bool
	Desc      string
}

func (c *Client) GetEvent(calID, gcalID string) (*GcalEvent, error) {
	ev, err := c.svc.Events.Get(calID, gcalID).Do()
	if err != nil {
		return nil, err
	}
	ge := &GcalEvent{Etag: ev.Etag, Summary: ev.Summary, Desc: ev.Description}
	if ev.Start.DateTime != "" {
		ge.Start, _ = time.Parse(time.RFC3339, ev.Start.DateTime)
	} else {
		ge.Start, _ = time.Parse("2006-01-02", ev.Start.Date)
		ge.AllDay = true
	}
	return ge, nil
}

// ExportTodos pushes org todos to GCal. Returns count, all active GcalIDs, conflicts, and error.
func (c *Client) ExportTodos(todos []*org.Todo) (int, []string, []*conflict.Conflict, error) {
	count := 0
	var gcalIDs []string
	var conflicts []*conflict.Conflict

	for _, todo := range todos {
		calID := c.calendarID
		if todo.CalendarID != "" {
			calID = todo.CalendarID
		}

		// DONE/CANCELLED with GCAL_ID → delete from GCal
		if todo.GcalID != "" && (todo.State == "DONE" || todo.State == "CANCELLED") {
			if err := c.svc.Events.Delete(calID, todo.GcalID).Do(); err != nil {
				var gErr *googleapi.Error
				if !errors.As(err, &gErr) || gErr.Code != 410 {
					return count, gcalIDs, conflicts, err
				}
				// 410 = already deleted on GCal, treat as success
			}
			count++
			continue
		}

		if todo.GcalID != "" {
			newEtag, cf, err := c.updateEvent(todo, calID)
			if err != nil {
				return count, gcalIDs, conflicts, err
			}
			if cf != nil {
				conflicts = append(conflicts, cf)
				// still track the gcalID so it's filtered from imports
				gcalIDs = append(gcalIDs, todo.GcalID)
				continue
			}
			todo.GcalEtag = newEtag
			gcalIDs = append(gcalIDs, todo.GcalID)
			if err := org.WriteGcalProps(todo); err != nil {
				return count, gcalIDs, conflicts, err
			}
		} else {
			newID, newEtag, err := c.createEvent(todo, calID)
			if err != nil {
				return count, gcalIDs, conflicts, err
			}
			todo.GcalID = newID
			todo.GcalEtag = newEtag
			gcalIDs = append(gcalIDs, newID)
			if err := org.WriteGcalProps(todo); err != nil {
				return count, gcalIDs, conflicts, err
			}
		}
		count++
	}
	return count, gcalIDs, conflicts, nil
}

func (c *Client) DeleteEvent(gcalID string) error {
	return c.svc.Events.Delete(c.calendarID, gcalID).Do()
}

func buildEventDateTime(t time.Time, allDay bool) *googlecalendar.EventDateTime {
	if allDay {
		return &googlecalendar.EventDateTime{Date: t.Format("2006-01-02")}
	}
	return &googlecalendar.EventDateTime{DateTime: t.Format(time.RFC3339)}
}

func buildEvent(todo *org.Todo) *googlecalendar.Event {
	start := todo.Scheduled
	if start.IsZero() {
		start = todo.Deadline
	}
	end := todo.ScheduledEnd
	if end.IsZero() {
		if todo.AllDay {
			end = start.AddDate(0, 0, 1)
		} else {
			end = start.Add(30 * time.Minute)
		}
	}
	return &googlecalendar.Event{
		Summary:     todo.Title,
		Description: todo.Body,
		Start:       buildEventDateTime(start, todo.AllDay),
		End:         buildEventDateTime(end, todo.AllDay),
	}
}

func (c *Client) createEvent(todo *org.Todo, calID string) (id, etag string, err error) {
	created, err := c.svc.Events.Insert(calID, buildEvent(todo)).Do()
	if err != nil {
		return "", "", err
	}
	return created.Id, created.Etag, nil
}

// updateEvent checks etag conflict, skips update if nothing changed, returns new etag or Conflict.
func (c *Client) updateEvent(todo *org.Todo, calID string) (newEtag string, cf *conflict.Conflict, err error) {
	existing, err := c.svc.Events.Get(calID, todo.GcalID).Do()
	if err != nil {
		return "", nil, err
	}

	if todo.GcalEtag != "" && existing.Etag != todo.GcalEtag {
		cf := &conflict.Conflict{
			GcalID: todo.GcalID,
			Title:  todo.Title,
			File:   todo.File,
			Line:   todo.Line,
		}
		if existing.Summary != todo.Title {
			cf.Fields = append(cf.Fields, conflict.Field{
				Name: "title", Local: todo.Title, Remote: existing.Summary,
			})
		}
		localTime := formatTime(todo.Scheduled, todo.AllDay)
		remoteTime := existing.Start.DateTime
		if remoteTime == "" {
			remoteTime = existing.Start.Date
		}
		if localTime != remoteTime {
			cf.Fields = append(cf.Fields, conflict.Field{
				Name: "scheduled", Local: localTime, Remote: remoteTime,
			})
		}
		return "", cf, nil
	}

	// skip update if nothing changed
	remoteTime := existing.Start.DateTime
	if remoteTime == "" {
		remoteTime = existing.Start.Date
	}
	if existing.Summary == todo.Title &&
		existing.Description == todo.Body &&
		remoteTime == formatTime(todo.Scheduled, todo.AllDay) {
		return existing.Etag, nil, nil
	}

	updated, err := c.svc.Events.Update(calID, todo.GcalID, buildEvent(todo)).Do()
	if err != nil {
		return "", nil, err
	}
	return updated.Etag, nil, nil
}

// ForceUpdateEvent pushes todo to GCal without etag conflict check (for local-wins resolution).
func (c *Client) ForceUpdateEvent(todo *org.Todo, calID string) (string, error) {
	updated, err := c.svc.Events.Update(calID, todo.GcalID, buildEvent(todo)).Do()
	if err != nil {
		return "", err
	}
	return updated.Etag, nil
}

func formatTime(t time.Time, allDay bool) string {
	if t.IsZero() {
		return ""
	}
	if allDay {
		return t.Format("2006-01-02")
	}
	return t.Format(time.RFC3339)
}
