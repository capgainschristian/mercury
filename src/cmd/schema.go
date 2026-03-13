package schema

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
