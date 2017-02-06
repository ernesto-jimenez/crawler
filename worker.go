package crawler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// CrawlFunc is the type of the function called for each webpage visited by
// Crawl. The incoming url specifies which url was fetched, while res contains
// the response of the fetched URL if it was successful. If the fetch failed,
// the incoming error will specify the reason and res will be nil.
//
// Returning ErrSkipURL will avoid queing up the resources links to be crawled.
//
// Returning any other error from the function will immediately stop the crawl.
type CrawlFunc func(url string, res *Response, err error) error

// ErrSkipURL can be returned by CrawlFunc to avoid crawling the links from the given url
var ErrSkipURL = errors.New("skip URL")

// Runner defines the interface requred to run a crawl
type Runner interface {
	Run(context.Context, Queue) error
}

// Worker is used to run a crawl on a single goroutine
type Worker struct {
	client     *http.Client
	fn         CrawlFunc
	checkFetch CheckFetchStack
	maxRedirs  int
}

// NewWorker initialises a goroutine
func NewWorker(fn CrawlFunc, opts ...Option) (*Worker, error) {
	o := options{
		transport: http.DefaultTransport,
	}

	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return nil, err
		}
	}

	return &Worker{
		client: &http.Client{
			Transport:     o.transport,
			CheckRedirect: skipRedirects,
		},
		checkFetch: CheckFetchStack(o.checkFetch),
		fn:         fn,
	}, nil
}

// Run starts processing requests from the queue
func (w *Worker) Run(ctx context.Context, q Queue) error {
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		req, err := q.PopFront()
		if err != nil {
			return err
		}
		if req == nil {
			return nil
		}
		if !w.checkFetch.CheckFetch(req) {
			continue
		}
		res, err := fetch(ctx, w.client, req)
		if err := ctx.Err(); err != nil {
			return err
		}
		// call the CrawlFunc for each fetched url
		// note this err is scoped to the if and does not override the previous declaration
		if err := w.fn(req.URL.String(), res, err); err == ErrSkipURL {
			continue
		} else if err != nil {
			return err
		}
		// continue if there was an error crawlking
		if err != nil {
			continue
		}
		// Mark the response as visited since it might be different to the original URL due to redirects
		if w.maxDepth > 0 && w.maxDepth <= req.depth {
			continue
		}
		if req, err := nextRequest(res, res.RedirectTo); err == nil {
			q.PushBack(req)
		}
		for _, link := range res.Links {
			if req, err := nextRequest(res, link.URL); err == nil {
				q.PushBack(req)
			}
		}
	}
}

func skipRedirects(req *http.Request, via []*http.Request) error {
	return http.ErrUseLastResponse
}

func fetch(ctx context.Context, c *http.Client, req *Request) (*Response, error) {
	uri := req.URL.String()
	httpReq, err := http.NewRequest(http.MethodGet, uri, nil)
	if err != nil {
		return nil, err
	}
	httpReq = httpReq.WithContext(ctx)

	httpRes, err := c.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer httpRes.Body.Close()
	res := Response{
		request: req,
	}
	switch httpRes.StatusCode {
	case http.StatusOK:
	case http.StatusMovedPermanently, http.StatusFound:
		res.URL = httpRes.Request.URL.String()
		loc, err := url.Parse(httpRes.Header.Get("Location"))
		if err != nil {
			return nil, err
		}
		res.RedirectTo = httpRes.Request.URL.ResolveReference(loc).String()
		return &res, nil
	default:
		return nil, fmt.Errorf("%s for %s", httpRes.Status, uri)
	}
	err = ReadResponse(httpRes.Request.URL, httpRes.Body, &res)
	return &res, nil
}

func nextRequest(res *Response, href string) (*Request, error) {
	if href == "" {
		return nil, ErrSkipURL
	}
	req, err := NewRequest(href)
	if err != nil {
		return nil, err
	}
	if res.RedirectTo == "" {
		req.depth = res.request.depth + 1
		req.redirects = 0
	} else {
		req.redirects = res.request.redirects + 1
	}
	return req, nil
}
