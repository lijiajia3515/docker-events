package notifier

import (
	"testing"
	"time"

	"github.com/filippofinke/docker-events/internal/docker"
)

func TestFormatEvent_Minimal(t *testing.T) {
	e := docker.Event{
		Type:      "container",
		Action:    "start",
		Actor:     docker.Actor{ID: "actor"},
		Timestamp: time.Unix(0, 0),
	}
	subj, body := formatEvent("prefix", e, nil, 0)
	if subj == "" || body == "" {
		t.Fatalf("expected non-empty subject and body, got %q / %q", subj, body)
	}
}
