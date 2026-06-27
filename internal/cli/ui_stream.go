package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

// sessionNonce is a 6-hex-char prefix derived from startup time.
// It makes every server-restart's event IDs globally unique,
// preventing seenEventIds from filtering new heartbeats as duplicates.
var sessionNonce = fmt.Sprintf("%06x", time.Now().UnixNano()&0xFFFFFF)

type workflowStreamBroker struct {
	mu          sync.Mutex
	subscribers map[chan workflowStreamEvent]struct{}
	nextID      atomic.Uint64
}

func newWorkflowStreamBroker() *workflowStreamBroker {
	return &workflowStreamBroker{
		subscribers: make(map[chan workflowStreamEvent]struct{}),
	}
}

func (b *workflowStreamBroker) subscribe() chan workflowStreamEvent {
	ch := make(chan workflowStreamEvent, 16)
	b.mu.Lock()
	b.subscribers[ch] = struct{}{}
	b.mu.Unlock()
	return ch
}

func (b *workflowStreamBroker) unsubscribe(ch chan workflowStreamEvent) {
	b.mu.Lock()
	if _, ok := b.subscribers[ch]; ok {
		delete(b.subscribers, ch)
		close(ch)
	}
	b.mu.Unlock()
}

func (b *workflowStreamBroker) publish(eventType, agentID, agentName, message string) workflowStreamEvent {
	event := workflowStreamEvent{
		ID:        fmt.Sprintf("e%s-%06d", sessionNonce, b.nextID.Add(1)),
		Type:      eventType,
		AgentID:   agentID,
		AgentName: agentName,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	b.mu.Lock()
	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	b.mu.Unlock()

	return event
}

func (b *workflowStreamBroker) publishWithResult(eventType, agentID, agentName, message string, result *workflowAgentResult) workflowStreamEvent {
	event := workflowStreamEvent{
		ID:        fmt.Sprintf("e%s-%06d", sessionNonce, b.nextID.Add(1)),
		Type:      eventType,
		AgentID:   agentID,
		AgentName: agentName,
		Message:   message,
		Timestamp: time.Now().Format(time.RFC3339),
		Result:    result,
	}
	b.mu.Lock()
	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	b.mu.Unlock()
	return event
}

func handleWorkflowStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := uiWorkflowBroker.subscribe()
	defer uiWorkflowBroker.unsubscribe(ch)

	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(20 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-heartbeat.C:
			if _, err := fmt.Fprint(w, ": ping\n\n"); err != nil {
				return
			}
			flusher.Flush()
		case event, ok := <-ch:
			if !ok {
				return
			}
			if err := writeWorkflowStreamEvent(r.Context(), w, event); err != nil {
				return
			}
			flusher.Flush()
		}
	}
}

func writeWorkflowStreamEvent(ctx context.Context, w http.ResponseWriter, event workflowStreamEvent) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "id: %s\nevent: %s\ndata: %s\n\n", event.ID, event.Type, data); err != nil {
		return err
	}
	return nil
}

func (b *workflowStreamBroker) publishChecklist(agentID, agentName string, items []checklistItem) workflowStreamEvent {
	doneCount := 0
	for _, it := range items {
		if it.Done {
			doneCount++
		}
	}
	msg := fmt.Sprintf("체크리스트 업데이트: %d/%d 완료", doneCount, len(items))
	event := workflowStreamEvent{
		ID:        fmt.Sprintf("e%s-%06d", sessionNonce, b.nextID.Add(1)),
		Type:      "checklist",
		AgentID:   agentID,
		AgentName: agentName,
		Message:   msg,
		Timestamp: time.Now().Format(time.RFC3339),
		Checklist: items,
	}
	b.mu.Lock()
	for ch := range b.subscribers {
		select {
		case ch <- event:
		default:
		}
	}
	b.mu.Unlock()
	return event
}

var uiWorkflowBroker = newWorkflowStreamBroker()
