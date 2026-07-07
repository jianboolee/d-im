package model

// EventEnvelope 事件总线标准信封
type EventEnvelope struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Subject       string                 `json:"subject"`
	AggregateType string                 `json:"aggregate_type"`
	AggregateID   string                 `json:"aggregate_id"`
	OccurredAt    string                 `json:"occurred_at"`
	Source        string                 `json:"source"`
	Data          map[string]interface{} `json:"data"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}
