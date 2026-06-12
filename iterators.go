// Copyright (c) 2026 Develeap
// SPDX-License-Identifier: MIT

package hyperping

import (
	"context"
	"iter"
)

// IterMonitors returns an iterator over all monitors.
// It calls ListMonitors once and yields each item. On error it yields (Monitor{}, err)
// and stops. If the caller returns false from yield, iteration stops immediately.
func (c *Client) IterMonitors(ctx context.Context) iter.Seq2[Monitor, error] {
	return func(yield func(Monitor, error) bool) {
		monitors, err := c.ListMonitors(ctx)
		if err != nil {
			yield(Monitor{}, err)
			return
		}
		for _, m := range monitors {
			if !yield(m, nil) {
				return
			}
		}
	}
}

// IterIncidents returns an iterator over all incidents.
// It calls ListIncidents once and yields each item. On error it yields (Incident{}, err)
// and stops. If the caller returns false from yield, iteration stops immediately.
func (c *Client) IterIncidents(ctx context.Context) iter.Seq2[Incident, error] {
	return func(yield func(Incident, error) bool) {
		incidents, err := c.ListIncidents(ctx)
		if err != nil {
			yield(Incident{}, err)
			return
		}
		for _, inc := range incidents {
			if !yield(inc, nil) {
				return
			}
		}
	}
}

// IterStatusPages returns an iterator over all status pages, following pagination.
// It calls ListStatusPages for each page (starting at 0), yielding items one by one.
// On error it yields (StatusPage{}, err) and stops. Context cancellation propagates
// as an error through ListStatusPages, so no explicit page cap is needed.
func (c *Client) IterStatusPages(ctx context.Context, search *string) iter.Seq2[StatusPage, error] {
	return func(yield func(StatusPage, error) bool) {
		for page := 0; ; page++ {
			resp, err := c.ListStatusPages(ctx, &page, search)
			if err != nil {
				yield(StatusPage{}, err)
				return
			}
			for _, sp := range resp.StatusPages {
				if !yield(sp, nil) {
					return
				}
			}
			if !resp.HasNextPage {
				return
			}
		}
	}
}
