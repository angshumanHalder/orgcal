package org

import "time"

type Event struct {
	ID       string
	Title    string
	Location string
	Notes    string
	Start    time.Time
	End      time.Time
	AllDay   bool
}

type Repeater struct {
	Type     string // "+", ".+", "++"
	Value    int
	Unit     string // d, w, m, y
}

type Todo struct {
	Title        string
	Body         string
	State        string    // TODO, NEXT, DONE, CANCELLED
	Priority     string    // A, B, C
	Tags         []string
	FileTags     []string
	Scheduled    time.Time
	ScheduledEnd time.Time
	Deadline     time.Time
	AllDay       bool
	Repeater     *Repeater
	GcalID       string
	CalendarID   string
	ExportToGcal bool
	File         string
}
