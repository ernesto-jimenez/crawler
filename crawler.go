package crawler

import "context"

// Simple is responsible of running a crawl, allowing you to queue new URLs to
// be crawled and build requests to be crawled.
type Simple struct {
	opts []Option
}

// New initialises a new crawl runner
func New(opts ...Option) (*Simple, error) {
	return &Simple{
		opts: opts,
	}, nil
}

// Crawl will fetch all the linked websites starting from startURL and invoking
// crawlFn for each fetched url with either the response or the error.
//
// It will return an error if the crawl was prematurely stopped or could not be
// started.
//
// Crawl will always add WithOneRequestPerURL to the options of the worker to
// avoid infinite loops.
func (s *Simple) Crawl(startURL string, crawlFn CrawlFunc) error {
	req, err := NewRequest(startURL)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// initialise the queue
	queue := NewInMemoryQueue(ctx)
	queue.PushBack(req)

	s.opts = append(s.opts, WithOneRequestPerURL())

	w, err := NewWorker(crawlFn, s.opts...)
	if err != nil {
		return err
	}

	return w.Run(ctx, queue)
}
