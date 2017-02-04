package crawler

import "net/url"

// CheckFetchFunc is used to check whether a page should be fetched during the
// crawl or not
type CheckFetchFunc func(*url.URL) bool

// CheckFetchStack is a stack of CheckFetchFunc types where all have to pass
// for the fetch to happen.
type CheckFetchStack []CheckFetchFunc

// CheckFetch will return true if all funcs in the stack return true. false otherwise.
func (s CheckFetchStack) CheckFetch(u *url.URL) bool {
	for _, fn := range s {
		if !fn(u) {
			return false
		}
	}
	return true
}
