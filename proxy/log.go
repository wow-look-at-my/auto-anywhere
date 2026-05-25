package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"log/slog"
	"regexp"
	"strings"
	"sync"
)

var whitespaceRun = regexp.MustCompile(`\s+`)

const maxLogLen = 200

func logRequest(body []byte) {
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		return
	}

	model, _ := req["model"].(string)
	stream, _ := req["stream"].(bool)

	var toolCount int
	if tools, ok := req["tools"].([]any); ok {
		toolCount = len(tools)
	}

	var role, content string
	if msgs, ok := req["messages"].([]any); ok && len(msgs) > 0 {
		if last, ok := msgs[len(msgs)-1].(map[string]any); ok {
			role, _ = last["role"].(string)
			content = extractTextContent(last)
		}
	}

	slog.Info("--> request",
		"model", model,
		"stream", stream,
		"tools", toolCount,
		"role", role,
		"content", truncate(content, maxLogLen),
	)
}

func extractTextContent(msg map[string]any) string {
	if s, ok := msg["content"].(string); ok {
		return s
	}
	if blocks, ok := msg["content"].([]any); ok {
		var parts []string
		for _, b := range blocks {
			block, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if block["type"] == "text" {
				if text, ok := block["text"].(string); ok {
					parts = append(parts, text)
				}
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

type sseLogger struct {
	body io.ReadCloser
	buf  bytes.Buffer
	mu   sync.Mutex
	done bool
}

func newSSELogger(body io.ReadCloser) *sseLogger {
	return &sseLogger{body: body}
}

func (l *sseLogger) Read(p []byte) (int, error) {
	n, err := l.body.Read(p)
	if n > 0 {
		l.mu.Lock()
		l.buf.Write(p[:n])
		l.mu.Unlock()
	}
	if err != nil {
		l.finish()
	}
	return n, err
}

func (l *sseLogger) Close() error {
	l.finish()
	return l.body.Close()
}

func (l *sseLogger) finish() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.done {
		return
	}
	l.done = true

	text, model, stopReason, inputTokens, outputTokens := parseSSEEvents(l.buf.String())
	slog.Info("<-- response",
		"model", model,
		"stop_reason", stopReason,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"content", truncate(text, maxLogLen),
	)
}

func parseSSEEvents(raw string) (text, model, stopReason string, inputTokens, outputTokens int) {
	var textParts []string

	for _, event := range strings.Split(raw, "\n\n") {
		lines := strings.Split(strings.TrimSpace(event), "\n")
		var eventType, data string
		for _, line := range lines {
			if strings.HasPrefix(line, "event: ") {
				eventType = strings.TrimPrefix(line, "event: ")
			} else if strings.HasPrefix(line, "data: ") {
				data = strings.TrimPrefix(line, "data: ")
			}
		}
		if data == "" {
			continue
		}

		switch eventType {
		case "message_start":
			var ev struct {
				Message struct {
					Model string `json:"model"`
					Usage struct {
						InputTokens int `json:"input_tokens"`
					} `json:"usage"`
				} `json:"message"`
			}
			if json.Unmarshal([]byte(data), &ev) == nil {
				model = ev.Message.Model
				inputTokens = ev.Message.Usage.InputTokens
			}
		case "content_block_delta":
			var ev struct {
				Delta struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
			}
			if json.Unmarshal([]byte(data), &ev) == nil && ev.Delta.Type == "text_delta" {
				textParts = append(textParts, ev.Delta.Text)
			}
		case "message_delta":
			var ev struct {
				Delta struct {
					StopReason string `json:"stop_reason"`
				} `json:"delta"`
				Usage struct {
					OutputTokens int `json:"output_tokens"`
				} `json:"usage"`
			}
			if json.Unmarshal([]byte(data), &ev) == nil {
				stopReason = ev.Delta.StopReason
				outputTokens = ev.Usage.OutputTokens
			}
		}
	}

	text = strings.Join(textParts, "")
	return
}

func logNonStreamResponse(body []byte) {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return
	}

	model, _ := resp["model"].(string)
	stopReason, _ := resp["stop_reason"].(string)

	var inputTokens, outputTokens int
	if usage, ok := resp["usage"].(map[string]any); ok {
		if v, ok := usage["input_tokens"].(float64); ok {
			inputTokens = int(v)
		}
		if v, ok := usage["output_tokens"].(float64); ok {
			outputTokens = int(v)
		}
	}

	var text string
	if content, ok := resp["content"].([]any); ok {
		for _, b := range content {
			block, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if block["type"] == "text" {
				if t, ok := block["text"].(string); ok {
					text += t
				}
			}
		}
	}

	slog.Info("<-- response",
		"model", model,
		"stop_reason", stopReason,
		"input_tokens", inputTokens,
		"output_tokens", outputTokens,
		"content", truncate(text, maxLogLen),
	)
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(whitespaceRun.ReplaceAllString(s, " "))
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
