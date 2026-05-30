package proxy

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wow-look-at-my/testify/require"
)

func makeResp(method, path string, status int, header http.Header, body string) *http.Response {
	req := httptest.NewRequest(method, "http://upstream"+path, nil)
	if header == nil {
		header = http.Header{}
	}
	return &http.Response{
		StatusCode: status,
		Header:     header,
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}

func TestRewriteResponseNonStreamMessages(t *testing.T) {
	body := `{"model":"m","stop_reason":"end_turn","content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":1,"output_tokens":2}}`
	resp := makeResp(http.MethodPost, "/v1/messages", 200, http.Header{"Content-Type": []string{"application/json"}}, body)

	require.NoError(t, rewriteResponse(resp))

	got, _ := io.ReadAll(resp.Body)
	require.Equal(t, body, string(got), "non-stream body must be preserved")
	require.Equal(t, int64(len(body)), resp.ContentLength)
}

func TestRewriteResponseStreamMessages(t *testing.T) {
	raw := "event: message_stop\ndata: {}\n\n"
	resp := makeResp(http.MethodPost, "/v1/messages", 200, http.Header{"Content-Type": []string{"text/event-stream"}}, raw)

	require.NoError(t, rewriteResponse(resp))

	_, isLogger := resp.Body.(*sseLogger)
	require.True(t, isLogger, "streaming body should be wrapped in sseLogger")
	got, _ := io.ReadAll(resp.Body)
	require.Equal(t, raw, string(got), "streaming bytes must pass through unchanged")
}

func TestRewriteResponseGrowthBookPlain(t *testing.T) {
	resp := makeResp(http.MethodGet, "/api/eval/abc", 200, http.Header{"Content-Type": []string{"application/json"}}, `{"features":{}}`)

	require.NoError(t, rewriteResponse(resp))

	got, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(got), "tengu_auto_mode_config", "auto mode config should be injected")
}

func TestRewriteResponseGrowthBookGzip(t *testing.T) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	_, _ = gw.Write([]byte(`{"features":{}}`))
	gw.Close()

	resp := makeResp(http.MethodGet, "/api/features/x", 200, http.Header{"Content-Encoding": []string{"gzip"}}, "")
	resp.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))

	require.NoError(t, rewriteResponse(resp))

	require.Empty(t, resp.Header.Get("Content-Encoding"), "gzip encoding should be stripped after decompression")
	got, _ := io.ReadAll(resp.Body)
	require.Contains(t, string(got), "tengu_auto_mode_config")
}

func TestRewriteResponseGrowthBookNon200(t *testing.T) {
	body := `{"features":{}}`
	resp := makeResp(http.MethodGet, "/api/eval/abc", 500, http.Header{}, body)

	require.NoError(t, rewriteResponse(resp))

	got, _ := io.ReadAll(resp.Body)
	require.Equal(t, body, string(got), "non-200 GrowthBook responses pass through unchanged")
}

func TestRewriteResponsePassthrough(t *testing.T) {
	body := `irrelevant`
	resp := makeResp(http.MethodGet, "/v1/models", 200, http.Header{}, body)

	require.NoError(t, rewriteResponse(resp))

	got, _ := io.ReadAll(resp.Body)
	require.Equal(t, body, string(got))
}

func TestRewriteResponseNilRequest(t *testing.T) {
	resp := &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader("x"))}
	require.NoError(t, rewriteResponse(resp))
}

func TestIsGrowthBookPath(t *testing.T) {
	require.True(t, isGrowthBookPath("/api/eval/x"))
	require.True(t, isGrowthBookPath("/api/features/y"))
	require.True(t, isGrowthBookPath("/sub/z"))
	require.False(t, isGrowthBookPath("/v1/messages"))
	require.False(t, isGrowthBookPath("/"))
}

func TestRewriteRequestBranches(t *testing.T) {
	// Non-POST requests are left untouched.
	r := httptest.NewRequest(http.MethodGet, "http://upstream/v1/messages", strings.NewReader("x"))
	rewriteRequest(r)

	// Non-/v1/messages POSTs are left untouched.
	r = httptest.NewRequest(http.MethodPost, "http://upstream/v1/other", strings.NewReader("x"))
	rewriteRequest(r)

	// Invalid JSON on the messages path: InjectThinking errors, the body is
	// preserved verbatim, and GetBody is still installed.
	r = httptest.NewRequest(http.MethodPost, "http://upstream/v1/messages", strings.NewReader("not json"))
	r.GetBody = nil
	rewriteRequest(r)
	require.NotNil(t, r.GetBody)
	rc, err := r.GetBody()
	require.NoError(t, err)
	data, _ := io.ReadAll(rc)
	require.Equal(t, "not json", string(data))

	// Valid request that does not need changes (already adaptive 4-7) still
	// gets a rewindable body.
	r = httptest.NewRequest(http.MethodPost, "http://upstream/v1/messages",
		strings.NewReader(`{"model":"claude-opus-4-7","thinking":{"type":"adaptive","display":"summarized"},"messages":[]}`))
	r.GetBody = nil
	rewriteRequest(r)
	require.NotNil(t, r.GetBody)
}
