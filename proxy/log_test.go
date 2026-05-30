package proxy

import (
	"testing"

	"github.com/wow-look-at-my/testify/require"
)

func TestExtractTextContent(t *testing.T) {
	// Plain string content.
	require.Equal(t, "hello", extractTextContent(map[string]any{"content": "hello"}))

	// Text blocks are joined with spaces.
	require.Equal(t, "a b", extractTextContent(map[string]any{"content": []any{
		map[string]any{"type": "text", "text": "a"},
		map[string]any{"type": "text", "text": "b"},
	}}))

	// tool_result with string content.
	require.Equal(t, "result-str", extractTextContent(map[string]any{"content": []any{
		map[string]any{"type": "tool_result", "content": "result-str"},
	}}))

	// tool_result with nested content array (text blocks extracted, others skipped).
	require.Equal(t, "inner1 inner2", extractTextContent(map[string]any{"content": []any{
		map[string]any{"type": "tool_result", "content": []any{
			map[string]any{"type": "text", "text": "inner1"},
			map[string]any{"type": "text", "text": "inner2"},
			map[string]any{"type": "image"},
			"not-a-map",
		}},
	}}))

	// Non-map blocks and unknown block types are ignored.
	require.Equal(t, "kept", extractTextContent(map[string]any{"content": []any{
		"not-a-map",
		map[string]any{"type": "thinking", "text": "ignored"},
		map[string]any{"type": "text", "text": "kept"},
	}}))

	// No content.
	require.Equal(t, "", extractTextContent(map[string]any{}))
}

func TestLogRequest(t *testing.T) {
	// Block content, tools, stream flag.
	logRequest([]byte(`{"model":"claude-opus-4-8","stream":true,"tools":[{"name":"a"},{"name":"b"}],"messages":[{"role":"user","content":[{"type":"text","text":"hi"}]}]}`))
	// String content.
	logRequest([]byte(`{"model":"m","messages":[{"role":"user","content":"hey"}]}`))
	// No messages.
	logRequest([]byte(`{"model":"m"}`))
	// Invalid JSON returns early without panicking.
	logRequest([]byte(`not json`))
}

func TestLogNonStreamResponse(t *testing.T) {
	// API error response is logged at WARN.
	logNonStreamResponse([]byte(`{"type":"error","error":{"type":"overloaded_error","message":"slow down"}}`))
	// Normal response with content + usage.
	logNonStreamResponse([]byte(`{"model":"m","stop_reason":"end_turn","usage":{"input_tokens":3,"output_tokens":5},"content":[{"type":"text","text":"hi"},{"type":"thinking"}]}`))
	// "error" type but malformed error object falls through to normal logging.
	logNonStreamResponse([]byte(`{"type":"error","error":"oops"}`))
	// Invalid JSON returns early.
	logNonStreamResponse([]byte(`nope`))
}

func TestTruncate(t *testing.T) {
	require.Equal(t, "abc", truncate("abc", 10))
	require.Equal(t, "a b c", truncate("a   b\n\tc", 10))
	require.Equal(t, "xxxxx...", truncate("xxxxxxxxxxxx", 5))
}

func TestSSESummaryConsumeEdgeCases(t *testing.T) {
	var s sseSummary
	s.consume("event: ping\ndata: ") // empty data, ignored
	s.consume(": just a comment")     // no data line
	s.consume("event: message_start\ndata: {bad json")
	require.Equal(t, "", s.model)

	// Non-text delta is ignored.
	s.consume(`event: content_block_delta` + "\n" + `data: {"delta":{"type":"input_json_delta","partial_json":"x"}}`)
	require.Equal(t, "", s.text.String())

	// Unknown event type with valid data is ignored.
	s.consume(`event: ping` + "\n" + `data: {"foo":"bar"}`)
	require.Equal(t, "", s.model)
}
