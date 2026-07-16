package hawksdk

import (
	"context"
	"iter"
)

// defaultIterPageSize is used when the caller's ListOptions has no Limit set.
const defaultIterPageSize = 50

// AllMessages returns an iterator over every message in a session, fetching
// additional pages from the daemon transparently as the caller ranges past
// the end of each page. Iteration stops when the daemon reports no more
// pages (HasMore == false) or when a request fails, in which case the error
// is yielded and iteration stops.
//
// opts may be nil; a copy is used internally so the caller's ListOptions is
// never mutated. If opts.Limit is unset (<= 0), defaultIterPageSize is used.
//
//	for msg, err := range hawksdk.AllMessages(ctx, client, sessionID, nil) {
//		if err != nil {
//			// handle err, stop iterating
//			break
//		}
//		// use msg
//	}
func AllMessages(ctx context.Context, c *Client, sessionID string, opts *ListOptions) iter.Seq2[Message, error] {
	return func(yield func(Message, error) bool) {
		limit := defaultIterPageSize
		offset := 0
		if opts != nil {
			if opts.Limit > 0 {
				limit = opts.Limit
			}
			offset = opts.Offset
		}

		for {
			page, err := c.Messages(ctx, sessionID, &ListOptions{Offset: offset, Limit: limit})
			if err != nil {
				yield(Message{}, err)
				return
			}

			for _, msg := range page.Data {
				if !yield(msg, nil) {
					return
				}
			}

			if !page.HasMore {
				return
			}
			offset += limit
		}
	}
}
