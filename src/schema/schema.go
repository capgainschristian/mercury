package schema

import "time"

type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityMedium   Severity = "MEDIUM"
	SeverityLow      Severity = "LOW"
)

type EventType string

const (
	EventTypeUpgrade     EventType = "UPGRADE"
	EventTypeMaintenace  EventType = "MAINTENANCE"
	EventTypeIncident    EventType = "INCIDENT"
	EventTypeResolution  EventType = "RESOLUTION"
	EventTypeDeprecation EventType = "DEPRECATION"
)

type NotificationEvent struct {
	EventID       string `json:"event_id"`
	CorrelationID string `json:"correlation_id"`
	SchemaVersion string `json:"schema_version"`

	EventType EventType `json:"event_type"`
	Severity  Severity  `json:"severity"`

	Title   string `json:"title"`
	Message string `json:"message"`

	TemplateID   string                 `json:"template_id,omitempty"`
	TemplateData map[string]interface{} `json:"template_data,omitempty"`

	AffectedServices []string `json:"affected_services"`
	// Empty means broadcast all
	TargetCustomerIDs []string `json:"customer_ids,omitempty"`

	PublishedAt          time.Time  `json:"published_at"`
	ScheduledWindowStart *time.Time `json:"scheduled_window_start,omitempty"`
	ScheduledWindowEnd   *time.Time `json:"scheduled_window_end,omitempty"`
	ExpiresAt            *time.Time `json:"expires_at,omitempty"` // TTL

	PublisherService string `json:"publisher_service"`
	Environment      string `json:"environment"`
}
