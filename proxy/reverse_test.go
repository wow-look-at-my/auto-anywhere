package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/wow-look-at-my/testify/require"
)

const streamReqBody = `{"model":"claude-opus-4-8","stream":true,"max_tokens":1000,"messages":[{"role":"user","content":"hi"}]}`

// TestSingleStreamingRequestForwardsOnce verifies that one client request
// produces exactly one upstream request on the happy path.
func TestSingleStreamingRequestForwardsOnce(t *testing.T) {
	var count int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "event: message_start\ndata: {\"message\":{\"model\":\"claude-opus-4-8\",\"usage\":{\"input_tokens\":5}}}\n\n")
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		_, _ = io.WriteString(w, "event: message_stop\ndata: {}\n\n")
	}))
	defer upstream.Close()

	p, err := New(Config{Upstream: upstream.URL})
	require.NoError(t, err)
	front := httptest.NewServer(p)
	defer front.Close()

	resp, err := http.Post(front.URL+"/v1/messages", "application/json", strings.NewReader(streamReqBody))
	require.NoError(t, err)
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()

	require.Equal(t, int64(1), atomic.LoadInt64(&count), "exactly one upstream request expected")
}

// recordingRT captures the request the reverse proxy hands to the transport,
// then returns a minimal SSE response.
type recordingRT struct {
	req *http.Request
}

func (rt *recordingRT) RoundTrip(r *http.Request) (*http.Response, error) {
	rt.req = r
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{"Content-Type": []string{"text/event-stream"}},
		Body:       io.NopCloser(strings.NewReader("event: message_stop\ndata: {}\n\n")),
		Request:    r,
	}, nil
}

// TestRewrittenRequestIsRewindable is the core regression test for the
// duplicate-request bug. After the body is rewritten, the outgoing request
// MUST expose GetBody so net/http can transparently retry a stale keepalive
// connection itself (see persistConn.shouldRetryRequest: for a reused
// connection with nothingWrittenError it retries iff
// outgoingLength()==0 || GetBody!=nil). Without GetBody the transport refuses
// to retry the body-bearing POST, the failure surfaces to Claude Code as a
// 502, and the Anthropic SDK re-sends the identical request -- the duplicate.
func TestRewrittenRequestIsRewindable(t *testing.T) {
	rp, err := newReverseProxy("https://api.anthropic.com")
	require.NoError(t, err)

	rec := &recordingRT{}
	rp.rp.Transport = rec

	req := httptest.NewRequest(http.MethodPost, "http://proxy/v1/messages", strings.NewReader(streamReqBody))
	req.Header.Set("Content-Type", "application/json")
	rp.ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, rec.req, "transport must be called")
	require.NotNil(t, rec.req.GetBody,
		"outgoing request must expose GetBody so net/http can retry stale-connection failures internally")

	read := func() []byte {
		rc, err := rec.req.GetBody()
		require.NoError(t, err)
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		rc.Close()
		return b
	}
	b1, b2 := read(), read()
	require.Equal(t, b1, b2, "GetBody must reproduce the body deterministically")
	require.Equal(t, int64(len(b1)), rec.req.ContentLength, "ContentLength must match the rewritten body")
	require.Contains(t, string(b1), `"thinking"`, "GetBody must return the rewritten (thinking-injected) body")
}

// TestHealthEndpointNotProxied verifies the local health endpoint answers 200
// without forwarding upstream, so the container healthcheck stops reporting
// unhealthy.
func TestHealthEndpointNotProxied(t *testing.T) {
	var count int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{Upstream: upstream.URL})
	require.NoError(t, err)
	front := httptest.NewServer(p)
	defer front.Close()

	for _, path := range []string{"/", "/healthz"} {
		resp, err := http.Get(front.URL + path)
		require.NoError(t, err)
		resp.Body.Close()
		require.Equal(t, http.StatusOK, resp.StatusCode, "GET %s should be 200", path)
	}
	require.Equal(t, int64(0), atomic.LoadInt64(&count), "health checks must not be proxied upstream")
}

// TestNonHealthRootMethodsStillProxy ensures only GET/HEAD on the health paths
// are short-circuited; other methods/paths continue to the upstream.
func TestNonHealthRootMethodsStillProxy(t *testing.T) {
	var count int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&count, 1)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	p, err := New(Config{Upstream: upstream.URL})
	require.NoError(t, err)
	front := httptest.NewServer(p)
	defer front.Close()

	// POST / is not a health check and must be proxied.
	resp, err := http.Post(front.URL+"/", "application/json", strings.NewReader("{}"))
	require.NoError(t, err)
	resp.Body.Close()
	require.Equal(t, int64(1), atomic.LoadInt64(&count), "POST / must be proxied upstream")
}

// TestSSELoggerPassthroughAndSummary verifies the streaming logger passes bytes
// through unchanged while parsing the summary incrementally across arbitrary
// chunk boundaries (so it never buffers the whole response in memory).
func TestSSELoggerPassthroughAndSummary(t *testing.T) {
	raw := "event: message_start\ndata: {\"message\":{\"model\":\"claude-opus-4-8\",\"usage\":{\"input_tokens\":7}}}\n\n" +
		"event: content_block_delta\ndata: {\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello \"}}\n\n" +
		"event: content_block_delta\ndata: {\"delta\":{\"type\":\"text_delta\",\"text\":\"world\"}}\n\n" +
		"event: message_delta\ndata: {\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":3}}\n\n"

	l := newSSELogger(io.NopCloser(oneByteReader{strings.NewReader(raw)}))
	got, err := io.ReadAll(l)
	require.NoError(t, err)
	require.Equal(t, raw, string(got), "SSE bytes must pass through unchanged")

	s := l.snapshot()
	require.Equal(t, "claude-opus-4-8", s.model)
	require.Equal(t, "end_turn", s.stopReason)
	require.Equal(t, 7, s.inputTokens)
	require.Equal(t, 3, s.outputTokens)
	require.Equal(t, "Hello world", s.text.String())
}

// oneByteReader forces Read to return a single byte at a time, exercising the
// incremental SSE parser across event boundaries.
type oneByteReader struct{ r io.Reader }

func (o oneByteReader) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	return o.r.Read(p[:1])
}
