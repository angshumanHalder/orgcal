package gcal

import (
	"context"
	"fmt"
	"time"

	"github.com/angshumanhalder/orgcal/internal/auth"
	"github.com/angshumanhalder/orgcal/internal/org"
	googlecalendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
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

func (c *Client) ExportTodos(todos []*org.Todo) (int, error) {
	count := 0
	for _, todo := range todos {
		calID := c.calendarID
		if todo.CalendarID != "" {
			calID = todo.CalendarID
		}

		// DONE/CANCELLED with GCAL_ID → delete from GCal
		if todo.GcalID != "" && (todo.State == "DONE" || todo.State == "CANCELLED") {
			if err := c.svc.Events.Delete(calID, todo.GcalID).Do(); err != nil {
				return count, err
			}
			count++
			continue
		}

		if todo.GcalID != "" {
			if err := c.updateEvent(todo, calID); err != nil {
				return count, err
			}
		} else {
			if err := c.createEvent(todo, calID); err != nil {
				return count, err
			}
		}
		count++
	}
	return count, nil
}

func buildEventDateTime(t time.Time, allDay bool) *googlecalendar.EventDateTime {
	if allDay {
		return &googlecalendar.EventDateTime{Date: t.Format("2006-01-02")}
	}
	return &googlecalendar.EventDateTime{DateTime: t.Format(time.RFC3339)}
}

func (c *Client) createEvent(todo *org.Todo, calID string) error {
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
	event := &googlecalendar.Event{
		Summary:     todo.Title,
		Description: todo.Body,
		Start:       buildEventDateTime(start, todo.AllDay),
		End:         buildEventDateTime(end, todo.AllDay),
	}
	_, err := c.svc.Events.Insert(calID, event).Do()
	return err
}

func (c *Client) updateEvent(todo *org.Todo, calID string) error {
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
	event := &googlecalendar.Event{
		Summary:     todo.Title,
		Description: todo.Body,
		Start:       buildEventDateTime(start, todo.AllDay),
		End:         buildEventDateTime(end, todo.AllDay),
	}
	_, err := c.svc.Events.Update(calID, todo.GcalID, event).Do()
	return err
}
