package proxy

import (
	"bytes"
	"compress/gzip"
	"io"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"github.com/wow-look-at-my/auto-anywhere/rewrite"
)

type reverseProxy struct {
	rp *httputil.ReverseProxy
}

func newReverseProxy(upstream string) (*reverseProxy, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, err
	}

	rp := httputil.NewSingleHostReverseProxy(u)
	rp.FlushInterval = -1 // flush immediately for SSE streaming

	origDirector := rp.Director
	rp.Director = func(r *http.Request) {
		origDirector(r)
		r.Host = u.Host
		rewriteRequest(r)
	}

	rp.ModifyResponse = rewriteResponse

	return &reverseProxy{rp: rp}, nil
}

func (rp *reverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rp.rp.ServeHTTP(w, r)
}

func rewriteRequest(r *http.Request) {
	if r.Method != http.MethodPost {
		return
	}
	if !strings.HasPrefix(r.URL.Path, "/v1/messages") {
		return
	}

	body, err := io.ReadAll(r.Body)
	r.Body.Close()
	if err != nil {
		slog.Error("reading request body", "error", err)
		r.Body = io.NopCloser(bytes.NewReader(body))
		return
	}

	out, changed, err := rewrite.InjectThinking(body)
	if err != nil {
		slog.Warn("thinking rewrite failed", "error", err)
		r.Body = io.NopCloser(bytes.NewReader(body))
		r.ContentLength = int64(len(body))
		return
	}

	if changed {
		slog.Info("injected thinking", "path", r.URL.Path)
	}

	r.Body = io.NopCloser(bytes.NewReader(out))
	r.ContentLength = int64(len(out))
}

func rewriteResponse(resp *http.Response) error {
	if resp.Request == nil {
		return nil
	}
	path := resp.Request.URL.Path
	if !isGrowthBookPath(path) {
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil
	}

	var body []byte
	var err error

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gr, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil
		}
		body, err = io.ReadAll(gr)
		gr.Close()
		if err != nil {
			return nil
		}
		resp.Header.Del("Content-Encoding")
	} else {
		body, err = io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil
		}
	}

	out, changed, err := rewrite.InjectAutoMode(body)
	if err != nil {
		slog.Warn("auto mode rewrite failed", "error", err)
		resp.Body = io.NopCloser(bytes.NewReader(body))
		resp.ContentLength = int64(len(body))
		return nil
	}

	if changed {
		slog.Info("injected auto mode config", "path", path)
	}

	resp.Body = io.NopCloser(bytes.NewReader(out))
	resp.ContentLength = int64(len(out))
	resp.Header.Set("Content-Length", "")
	resp.Header.Del("Content-Length")
	return nil
}

func isGrowthBookPath(path string) bool {
	return strings.HasPrefix(path, "/api/eval/") ||
		strings.HasPrefix(path, "/api/features/") ||
		strings.HasPrefix(path, "/sub/")
}
