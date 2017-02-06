package crawler

import (
	"net/http"
	"strings"
	"sync"
)

// Option is used to provide optional configuration to a crawler
type Option func(*options) error

type options struct {
	maxDepth   int
	transport  http.RoundTripper
	checkFetch []CheckFetchFunc
}

// WithHTTPTransport sets the optional http client
func WithHTTPTransport(rt http.RoundTripper) Option {
	return func(opts *options) error {
		opts.transport = rt
		return nil
	}
}

// WithMaxDepth sets the max depth of the crawl. It must be over zero or
// the call will panic.
func WithMaxDepth(depth int) Option {
	if depth <= 0 {
		panic("depth should always be greater or than zero")
	}
	return WithCheckFetch(func(req *Request) bool {
		return req.depth <= depth
	})
}

// WithCheckFetch takes CheckFetchFunc that will be run before fetching each page to check whether it should be fetched or not
func WithCheckFetch(fn CheckFetchFunc) Option {
	return func(opts *options) error {
		opts.checkFetch = append(opts.checkFetch, fn)
		return nil
	}
}

// WithOneRequestPerURL adds a check to only allow URLs once
func WithOneRequestPerURL() Option {
	var mut sync.Mutex
	v := make(map[string]struct{})
	return WithCheckFetch(func(req *Request) bool {
		mut.Lock()
		defer mut.Unlock()
		_, ok := v[req.URL.String()]
		if ok {
			return false
		}
		v[req.URL.String()] = struct{}{}
		return true
	})
}

// WithAllowedHosts adds a check to only allow URLs with the given hosts
func WithAllowedHosts(hosts ...string) Option {
	m := make(map[string]struct{})
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		m[h] = struct{}{}
	}
	return WithCheckFetch(func(req *Request) bool {
		_, ok := m[req.URL.Host]
		return ok
	})
}

// WithExcludedHosts adds a check to only allow URLs with hosts other than the given ones
func WithExcludedHosts(hosts ...string) Option {
	m := make(map[string]struct{})
	for _, h := range hosts {
		h = strings.TrimSpace(h)
		m[h] = struct{}{}
	}
	return WithCheckFetch(func(req *Request) bool {
		_, ok := m[req.URL.Host]
		return !ok
	})
}
