package responses

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/phamtanminhtien/goroute/internal/openaiwire"
)

type state struct {
	ID         string
	CreatedAt  int64
	Status     openaiwire.ResponsesStatus
	Model      string
	Usage      *openaiwire.ResponseUsage
	Error      *openaiwire.ResponseError
	ItemsByIdx map[int]openaiwire.OutputItem
}

func ParseSSE(data []byte) (openaiwire.ResponsesResponse, error) {
	current := state{
		CreatedAt:  time.Now().Unix(),
		Status:     openaiwire.ResponsesStatusInProgress,
		Usage:      &openaiwire.ResponseUsage{},
		ItemsByIdx: make(map[int]openaiwire.OutputItem),
	}

	for _, raw := range sseDataEvents(data) {
		if raw == "" || raw == "[DONE]" {
			continue
		}

		var event openaiwire.ResponsesStreamEvent
		if err := json.Unmarshal([]byte(raw), &event); err != nil {
			return openaiwire.ResponsesResponse{}, fmt.Errorf("decode responses SSE event: %w", err)
		}
		processEvent(event, &current)
	}

	return finalize(current), nil
}

func processEvent(event openaiwire.ResponsesStreamEvent, current *state) {
	if current == nil {
		return
	}

	if event.Response != nil {
		mergeSnapshot(current, *event.Response)
	}

	switch event.Type {
	case openaiwire.ResponsesStreamEventTypeCreated:
		if event.Response != nil {
			mergeSnapshot(current, *event.Response)
		}
	case openaiwire.ResponsesStreamEventTypeOutputItemDone:
		if event.Item != nil {
			current.ItemsByIdx[event.OutputIndex] = *event.Item
		}
	case openaiwire.ResponsesStreamEventTypeCompleted:
		current.Status = openaiwire.ResponsesStatusCompleted
		if event.Response != nil {
			mergeSnapshot(current, *event.Response)
			if event.Response.Usage != nil {
				copied := *event.Response.Usage
				current.Usage = &copied
			}
		}
	case openaiwire.ResponsesStreamEventTypeFailed:
		current.Status = openaiwire.ResponsesStatusFailed
		if event.Response != nil && event.Response.Error != nil {
			copied := *event.Response.Error
			current.Error = &copied
		}
	}
}

func mergeSnapshot(current *state, response openaiwire.ResponsesResponse) {
	if response.ID != "" {
		current.ID = response.ID
	}
	if response.CreatedAt != 0 {
		current.CreatedAt = response.CreatedAt
	}
	if response.Status != "" {
		current.Status = response.Status
	}
	if response.Model != "" {
		current.Model = response.Model
	}
	if response.Usage != nil {
		copied := *response.Usage
		current.Usage = &copied
	}
	if response.Error != nil {
		copied := *response.Error
		current.Error = &copied
	}
	if len(response.Output) > 0 {
		for index, item := range response.Output {
			current.ItemsByIdx[index] = item
		}
	}
}

func finalize(current state) openaiwire.ResponsesResponse {
	indexes := make([]int, 0, len(current.ItemsByIdx))
	for index := range current.ItemsByIdx {
		indexes = append(indexes, index)
	}
	sort.Ints(indexes)

	output := make([]openaiwire.OutputItem, 0, len(indexes))
	for _, index := range indexes {
		output = append(output, current.ItemsByIdx[index])
	}

	return openaiwire.ResponsesResponse{
		ID:        current.ID,
		Object:    "response",
		CreatedAt: current.CreatedAt,
		Status:    current.Status,
		Model:     current.Model,
		Output:    output,
		Usage:     current.Usage,
		Error:     current.Error,
	}
}

func sseDataEvents(data []byte) []string {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 1024), 1024*1024)

	events := make([]string, 0, 8)
	var builder strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if builder.Len() > 0 {
				events = append(events, strings.TrimSuffix(builder.String(), "\n"))
				builder.Reset()
			}
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		builder.WriteString(payload)
		builder.WriteByte('\n')
	}
	if builder.Len() > 0 {
		events = append(events, strings.TrimSuffix(builder.String(), "\n"))
	}

	return events
}
