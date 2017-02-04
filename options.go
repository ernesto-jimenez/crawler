package crawler

import (
	"fmt"
	"net/http"
	"sync"
)

// Option is used to provide optional configuration to a crawler
type Option func(*options) error

type options struct {
	maxDepth   int
	client     *http.Client
	checkFetch []CheckFetchFunc
}

// WithHTTPClient sets the optional http client
func WithHTTPClient(client *http.Client) Option {
	return func(opts *options) error {
		opts.client = client
		return nil
	}
}

// WithMaxDepth sets the max depth of the crawl
func WithMaxDepth(depth int) Option {
	return func(opts *options) error {
		if depth < 0 {
			return fmt.Errorf("depth must be greater or equal than zero. was: %d", depth)
		}
		opts.maxDepth = depth
		return nil
	}
}

// WithCheckFetch takes CheckFetchFunc that will be run before fetching each page to check whether it should be fetched or not
func WithCheckFetch(fn CheckFetchFunc) Option {
	return func(opts *options) error {
		opts.checkFetch = append(opts.checkFetch, fn)
		return nil
	}
}

// WithOneRequestPerURL uses WithCheckFetch to register a function that will pass once per URL
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
