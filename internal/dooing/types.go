package dooing

type Todo struct {
	ID          string   `json:"id"`
	Text        string   `json:"text"`
	Done        bool     `json:"done"`
	InProgress  bool     `json:"in_progress"`
	Category    string   `json:"category"`
	CreatedAt   int64    `json:"created_at"`
	Priorities  []string `json:"priorities,omitempty"`
	DueAt       *int64   `json:"due_at,omitempty"`
	ParentID    *string  `json:"parent_id,omitempty"`
	Depth       int      `json:"depth,omitempty"`
	ECT         *float64 `json:"ect,omitempty"`
	CompletedAt *int64   `json:"completed_at,omitempty"`
	GcalID      string   `json:"gcal_id,omitempty"`
}
