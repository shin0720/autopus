package cli

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowStreamBrokerPublish(t *testing.T) {
	t.Parallel()

	broker := newWorkflowStreamBroker()
	ch := broker.subscribe()
	defer broker.unsubscribe(ch)

	event := broker.publish("started", "planner", "Planner", "started")
	assert.Equal(t, "started", event.Type)
	assert.NotEmpty(t, event.ID)

	received := <-ch
	assert.Equal(t, event.ID, received.ID)
	assert.Equal(t, "planner", received.AgentID)
}

func TestWriteWorkflowStreamEvent(t *testing.T) {
	t.Parallel()

	recorder := httptest.NewRecorder()
	event := workflowStreamEvent{
		ID:        "evt-000001",
		Type:      "completed",
		AgentID:   "planner",
		AgentName: "Planner",
		Message:   "done",
		Timestamp: "2026-04-29T10:15:00+09:00",
	}

	require.NoError(t, writeWorkflowStreamEvent(t.Context(), recorder, event))

	body := recorder.Body.String()
	assert.Contains(t, body, "id: evt-000001")
	assert.Contains(t, body, "event: completed")

	idx := strings.Index(body, "data: ")
	require.GreaterOrEqual(t, idx, 0)

	var parsed workflowStreamEvent
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(body[idx+6:])), &parsed))
	assert.Equal(t, "Planner", parsed.AgentName)
}
