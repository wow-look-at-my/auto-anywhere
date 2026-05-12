package proxy

import "net/http"

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
	p.reverse.ServeHTTP(w, r)
}
