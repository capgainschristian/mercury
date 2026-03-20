package mattermost

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/capgainschristian/mercury/schema"
)

var severityEmoji = map[schema.Severity]string{
	schema.SeverityCritical: ":red_circle",
	schema.SeverityHigh:     ":large_orange_circle",
	schema.SeverityMedium:   ":large_yellow_circle",
	schema.SeverityLow:      ":large_blue_circle",
}

type MattermostPayload struct {
	Text        string       `json:"text"`
	Username    string       `json:"username,omitempty"`
	IconEmoji   string       `json:"icon_emoji,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color  string `json:"color"`
	Title  string `json:"title"`
	Text   string `json:"text"`
	Footer string `json:"footer"`
}

// Mattermost notifier delivers notifications via webhook
type Notifier struct {
	webookURL  string
	httpClient *http.Client
	logger     *slog.Logger
}

func NewNotifier(webookURL string, logger *slog.Logger) *Notifier {
	return &Notifier{
		webookURL:  webookURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
		logger:     logger,
	}
}

func (n *Notifier) Send(ctx context.Context, evt schema.NotificationEvent) error {
	payload := n.buildPayload(evt)

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("mattermost: marshal payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.webookURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("mattermost: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("mattermost: http post: %w", err)
	}
	defer res.Body.Close()

	// Take only 200 status code - treat everything as failure so consumer can retry.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("mattermost: unexpected status %d", resp.StatusCode)
	}

	n.logger.InfoContext(ctx, "mattermost notification sent", 
	"event_id", evt.EventID,
	"severity", evt.Severity,
	"correlation_id", evt.CorrelationID, )
	
	return nil
}

func (n *Notifier) buildPayload(evt schema.NotificationEvent) MattermostPayload {
	emoji := severityEmoji[evt.Severity]
	color := severityColor(evt.Severity)

	footer := fmt.Sprintf("Published by %s | %s | Event ID: %s",
		evt.PublisherService,
		evt.PublishedAt.Format(time.RFC822),
		evt.EventID)

	var serviceNote string
	if len(evt.AffectedServices) > 0 {
		serviceNote = fmt.Sprintf("\n**Affected Services:** %s", strings.Join(evt.AffectedServices, ", "))
	}

	var windowNote string
	if evt.ScheduledWindowStart != nil && evt.ScheduledWindowEnd != nil {
		windowNote = fmt.Sprintf("\n**Maintenance Window:** %s – %s",
			evt.ScheduledWindowStart.Format(time.RFC822),
			evt.ScheduledWindowEnd.Format(time.RFC822),
		)
	}

	return MattermostPayload{
		Username: "Platform Notifier",
		IconEmoji: ":loudspeaker:",
		Attachments: []Attachment{
			Color: color,
			Title: fmt.Sprintf("%s %s", emoji, evt.Title),
			Text: evt.Message + serviceNote + windowNote,
			Footer: footer,
		}
	}
}

func severityColor(s schema.Severity) string {
	switch s {
	case schema.SeverityCritical:
		return "#FF0000"
	case schema.SeverityHigh:
		return "#FF8800"
	case schema.SeverityMedium:
		return "#FFCC00"
	default:
		return "#0088FF"
	}
}
