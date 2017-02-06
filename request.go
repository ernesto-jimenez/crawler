package crawler

import (
	"net/url"
)

// Request is used to fetch a page and informoation about its resources
type Request struct {
	URL *url.URL

	depth     int
	redirects int
	finished  bool
	onFinish  func()
}

// NewRequest initialises a new crawling request to extract information from a single URL
func NewRequest(uri string) (*Request, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}
	u.Fragment = ""
	if u.Path == "" {
		u.Path = "/"
	}
	return &Request{
		URL: u,
	}, nil
}

// Finish should be called once the request has been completed
func (r *Request) Finish() {
	if r.onFinish != nil {
		r.onFinish()
	}
	r.finished = true
}
