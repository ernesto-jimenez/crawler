package crawler

import (
	"container/list"
	"context"
	"log"
	"sync"
)

// Queue is used by workers to keep track of the urls that need to be fetched.
// Queue must be safe to use concurrently.
type Queue interface {
	PushBack(*Request) error
	PopFront() (*Request, error)
}

// InMemoryQueue holds a queue of items to be crawled in memory
type InMemoryQueue struct {
	ctx  context.Context
	in   chan *Request
	out  chan *Request
	done chan struct{}

	inFlight int64
	mut      sync.Mutex
}

// NewInMemoryQueue returns an in memory queue ready to be used by different workers
func NewInMemoryQueue(ctx context.Context) *InMemoryQueue {
	q := &InMemoryQueue{
		ctx:  ctx,
		in:   make(chan *Request),
		out:  make(chan *Request),
		done: make(chan struct{}),
	}
	go q.run()
	return q
}

func (q *InMemoryQueue) run() {
	queue := list.New()

	for {
		var (
			out  = q.out
			next *Request
		)
		front := queue.Front()
		if front == nil {
			out = nil
		} else {
			next = front.Value.(*Request)
		}

		select {
		case req := <-q.in:
			queue.PushBack(req)
		case out <- next:
			queue.Remove(front)
		case <-q.ctx.Done():
			return
		case <-q.done:
			return
		}
	}
}

// PushBack adds a request to the queue
func (q *InMemoryQueue) PushBack(req *Request) error {
	if req.finished {
		panic("requeueing finished request is forbidden")
	}
	q.mut.Lock()
	defer q.mut.Unlock()

	req.onFinish = func() {
		q.mut.Lock()
		defer q.mut.Unlock()
		q.inFlight--
		log.Println(q.inFlight)
		if q.inFlight == 0 {
			close(q.done)
		}
	}

	select {
	case <-q.ctx.Done():
		return q.ctx.Err()
	case <-q.done:
		panic("cannot push after queue was exhausted")
	case q.in <- req:
		q.inFlight++
	}
	return nil
}

// PopFront gets the next request from the queue.
// It will return a nil request and a nil error if the queue is empty.
func (q *InMemoryQueue) PopFront() (*Request, error) {
	select {
	case req := <-q.out:
		if req.finished {
			panic("popped message had already been finished")
		}
		return req, nil
	case <-q.done:
		return nil, nil
	case <-q.ctx.Done():
		return nil, q.ctx.Err()
	}
}
