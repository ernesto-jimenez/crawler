package crawler

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestInMemoryQueue(t *testing.T) {
	r := require.New(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	q := NewInMemoryQueue(ctx)

	var (
		req *Request
		err error
	)

	err = q.PushBack(&Request{depth: 1})
	r.NoError(err)

	req, err = q.PopFront()
	r.NoError(err)
	r.Equal(1, req.depth)

	err = q.PushBack(&Request{depth: 2})
	r.NoError(err)

	req.Finish()

	err = q.PushBack(&Request{depth: 3})
	r.NoError(err)

	req, err = q.PopFront()
	r.NoError(err)
	r.Equal(2, req.depth)
	req.Finish()

	req, err = q.PopFront()
	r.NoError(err)
	r.Equal(3, req.depth)
	req.Finish()

	req, err = q.PopFront()
	r.NoError(err)
	r.Nil(req)
}
