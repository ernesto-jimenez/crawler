package crawler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

//go:generate goautomock Queue

func TestWorkerStopsOnContextCancelation(t *testing.T) {
	r := require.New(t)

	s := httptest.NewServer(http.FileServer(http.Dir("testdata")))
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())

	q := NewQueueMock()
	q.PopFrontFunc = func() (*Request, error) {
		return NewRequest(s.URL)
	}
	q.PushBackFunc = func(*Request) error { return nil }

	w, err := NewWorker(func(url string, res *Response, err error) error {
		if q.PopFrontTotalCalls() == 2 {
			cancel()
		}
		return err
	})
	r.NoError(err)

	err = w.Run(ctx, q)
	r.Equal(context.Canceled, err)
	r.Equal(2, q.PopFrontTotalCalls())
}

func TestWorkerCancelsHTTPRequest(t *testing.T) {
	r := require.New(t)

	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		ctx := req.Context()
		select {
		case <-ctx.Done():
			r.Equal(context.Canceled, ctx.Err())
		case <-time.After(time.Second):
			http.Error(w, "should have cancelled", http.StatusInternalServerError)
		}
	}))
	defer s.Close()

	ctx, cancel := context.WithCancel(context.Background())

	q := NewQueueMock()
	q.PopFrontFunc = func() (*Request, error) {
		cancel()
		return NewRequest(s.URL)
	}
	q.PushBackFunc = func(*Request) error { return nil }

	w, err := NewWorker(func(url string, res *Response, err error) error {
		return err
	})
	r.NoError(err)

	err = w.Run(ctx, q)
	r.Equal(context.Canceled, err)
	r.Equal(1, q.PopFrontTotalCalls())
}

func TestFetch(t *testing.T) {
	sf := http.FileServer(http.Dir("testdata"))
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Log(r.URL.Path)
		if r.URL.Path == "/redirect" {
			t.Log("redirect")
			http.Redirect(w, r, "/ok", http.StatusFound)
		} else {
			sf.ServeHTTP(w, r)
		}
	}))
	defer s.Close()

	c := &http.Client{
		CheckRedirect: skipRedirects,
	}

	tests := []struct {
		path             string
		expectedURL      string
		expectedRedirect string
	}{
		{path: "", expectedURL: s.URL + "/"},
		{path: "/", expectedURL: s.URL + "/"},
		{path: "/index.html", expectedURL: s.URL + "/index.html", expectedRedirect: s.URL + "/"},
		{path: "/#with-fragment", expectedURL: s.URL + "/"},
		{path: "/redirect", expectedURL: s.URL + "/redirect", expectedRedirect: s.URL + "/ok"},
	}

	for _, test := range tests {
		t.Run(test.path, func(t *testing.T) {
			req, err := NewRequest(s.URL + test.path)
			require.NoError(t, err)

			res, err := fetch(context.Background(), c, req)
			require.NoError(t, err)

			require.Equal(t, test.expectedURL, res.URL)
			require.Equal(t, test.expectedRedirect, res.RedirectTo)
		})
	}
}
