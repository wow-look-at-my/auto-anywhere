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

// maxSummaryText caps how much streamed text the logger retains. The log line
// is truncated to maxLogLen anyway, so there is no reason to keep the whole
// response in memory.
const maxSummaryText = maxLogLen * 4

// maxPendingEvent bounds the partial-event buffer in case the upstream never
// emits an event terminator, so a single response cannot grow memory without
// limit.
const maxPendingEvent = 64 * 1024

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
			switch block["type"] {
			case "text":
				if text, ok := block["text"].(string); ok {
					parts = append(parts, text)
				}
			case "tool_result":
				if s, ok := block["content"].(string); ok {
					parts = append(parts, s)
				} else if inner, ok := block["content"].([]any); ok {
					for _, ib := range inner {
						iblock, ok := ib.(map[string]any)
						if !ok {
							continue
						}
						if iblock["type"] == "text" {
							if text, ok := iblock["text"].(string); ok {
								parts = append(parts, text)
							}
						}
					}
				}
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

// sseSummary accumulates the handful of fields we log from a stream of SSE
// events.
type sseSummary struct {
	model        string
	stopReason   string
	inputTokens  int
	outputTokens int
	text         strings.Builder
}

// consume parses a single complete SSE event block (without its trailing blank
// line) and folds the relevant fields into the summary.
func (s *sseSummary) consume(event string) {
	var eventType, data string
	for _, line := range strings.Split(strings.TrimSpace(event), "\n") {
		if v, ok := strings.CutPrefix(line, "event: "); ok {
			eventType = v
		} else if v, ok := strings.CutPrefix(line, "data: "); ok {
			data = v
		}
	}
	if data == "" {
		return
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
			s.model = ev.Message.Model
			s.inputTokens = ev.Message.Usage.InputTokens
		}
	case "content_block_delta":
		var ev struct {
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
		}
		if json.Unmarshal([]byte(data), &ev) == nil && ev.Delta.Type == "text_delta" {
			if s.text.Len() < maxSummaryText {
				s.text.WriteString(ev.Delta.Text)
			}
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
			s.stopReason = ev.Delta.StopReason
			s.outputTokens = ev.Usage.OutputTokens
		}
	}
}

// sseLogger is a pass-through ReadCloser that logs a one-line summary of an SSE
// response once the stream ends. It parses events incrementally so it never
// holds more than a single partial event in memory, regardless of how long the
// response is.
type sseLogger struct {
	body    io.ReadCloser
	mu      sync.Mutex
	pending []byte
	summary sseSummary
	done    bool
}

func newSSELogger(body io.ReadCloser) *sseLogger {
	return &sseLogger{body: body}
}

func (l *sseLogger) Read(p []byte) (int, error) {
	n, err := l.body.Read(p)
	if n > 0 {
		l.mu.Lock()
		l.ingest(p[:n])
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

// ingest appends newly read bytes and folds any now-complete events into the
// summary, retaining only the trailing partial event. The caller holds l.mu.
func (l *sseLogger) ingest(b []byte) {
	l.pending = append(l.pending, b...)
	for {
		i := bytes.Index(l.pending, []byte("\n\n"))
		if i < 0 {
			break
		}
		l.summary.consume(string(l.pending[:i]))
		l.pending = l.pending[i+2:]
	}
	if len(l.pending) > maxPendingEvent {
		l.pending = l.pending[len(l.pending)-maxPendingEvent:]
	}
}

// snapshot returns the parsed summary so far. Intended for tests.
func (l *sseLogger) snapshot() sseSummary {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.summary
}

func (l *sseLogger) finish() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.done {
		return
	}
	l.done = true

	// Fold in any final event that wasn't terminated by a blank line.
	if len(l.pending) > 0 {
		l.summary.consume(string(l.pending))
		l.pending = nil
	}

	slog.Info("<-- response",
		"model", l.summary.model,
		"stop_reason", l.summary.stopReason,
		"input_tokens", l.summary.inputTokens,
		"output_tokens", l.summary.outputTokens,
		"content", truncate(l.summary.text.String(), maxLogLen),
	)
}

func logNonStreamResponse(body []byte) {
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return
	}

	if resp["type"] == "error" {
		if errObj, ok := resp["error"].(map[string]any); ok {
			errType, _ := errObj["type"].(string)
			errMsg, _ := errObj["message"].(string)
			slog.Warn("<-- error response",
				"error_type", errType,
				"message", truncate(errMsg, maxLogLen),
			)
			return
		}
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
