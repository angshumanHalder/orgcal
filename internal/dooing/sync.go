package dooing

import (
	"encoding/json"
	"errors"
	"os"
	"time"

	"github.com/angshumanhalder/orgcal/internal/gcal"
	googleapi "google.golang.org/api/googleapi"
)

type Result struct {
	Exported int
	Deleted  int
}

func Sync(filePath string) (*Result, error) {
	data, err := os.ReadFile(filePath)
	if os.IsNotExist(err) {
		return &Result{}, nil
	}
	if err != nil {
		return nil, err
	}

	var todos []Todo
	if err := json.Unmarshal(data, &todos); err != nil {
		return nil, err
	}

	client, err := gcal.NewClient()
	if err != nil {
		return nil, err
	}

	res := &Result{}
	modified := false

	for i := range todos {
		t := &todos[i]

		if t.Done && t.GcalID != "" {
			if err := client.DeleteEvent(t.GcalID); err != nil {
				var gErr *googleapi.Error
				if !errors.As(err, &gErr) || gErr.Code != 410 {
					return nil, err
				}
			}
			t.GcalID = ""
			res.Deleted++
			modified = true
			continue
		}

		if t.DueAt == nil || t.Done {
			continue
		}

		due := time.Unix(*t.DueAt, 0)
		newID, err := client.UpsertDooingEvent(t.GcalID, t.Text, due)
		if err != nil {
			return nil, err
		}
		if newID != t.GcalID {
			t.GcalID = newID
			modified = true
		}
		res.Exported++
	}

	if modified {
		out, err := json.Marshal(todos)
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(filePath, out, 0644); err != nil {
			return nil, err
		}
	}

	return res, nil
}
