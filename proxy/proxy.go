package proxy

import (
	"io"
	"net/http"
)

type Config struct {
	Upstream string
}

type Proxy struct {
	reverse *reverseProxy
}

func New(cfg Config) (*Proxy, error) {
	rp, err := newReverseProxy(cfg.Upstream)
	if err != nil {
		return nil, err
	}
	return &Proxy{reverse: rp}, nil
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if isHealthCheck(r) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "ok\n")
		return
	}
	p.reverse.ServeHTTP(w, r)
}

// isHealthCheck reports whether r is a local liveness probe that must be
// answered directly instead of being forwarded upstream. The proxy otherwise
// forwards every path verbatim, so a probe against "/" would be relayed to
// api.anthropic.com (which returns a 404 for the root), making the container
// healthcheck fail even while the proxy serves traffic normally.
func isHealthCheck(r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	return r.URL.Path == "/" || r.URL.Path == "/healthz"
}
