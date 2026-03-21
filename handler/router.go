package handler

import "GPTBot/api/telegram"

// Router dispatches updates to the first matching handler,
// in the order they were registered.
type Router struct {
	handlers []UpdateHandler
}

// NewRouter creates an empty Router. Register handlers via Register().
func NewRouter() *Router {
	return &Router{}
}

// Register adds a handler to the router.
func (r *Router) Register(h UpdateHandler) {
	r.handlers = append(r.handlers, h)
}

// Route returns the first handler whose Match returns true, or nil.
func (r *Router) Route(ctx *telegram.UpdateContext) UpdateHandler {
	for _, h := range r.handlers {
		if h.Match(ctx) {
			return h
		}
	}
	return nil
}
