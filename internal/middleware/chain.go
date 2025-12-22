package middleware

import "net/http"

// Middleware is the standard middleware signature
type Middleware func(http.Handler) http.Handler

// Chain holds a sequence of middleware to apply in order
type Chain struct {
	middlewares []Middleware
}

// New creates a new middleware chain
func New(middlewares ...Middleware) Chain {
	return Chain{middlewares: middlewares}
}

// Use appends middleware to the chain, returns new chain
func (c Chain) Use(m Middleware) Chain {
	newMiddlewares := make([]Middleware, len(c.middlewares)+1)
	copy(newMiddlewares, c.middlewares)
	newMiddlewares[len(c.middlewares)] = m
	return Chain{middlewares: newMiddlewares}
}

// Then applies the chain to a handler
// First middleware in chain is outermost (runs first on request, last on response)
func (c Chain) Then(h http.Handler) http.Handler {
	for i := len(c.middlewares) - 1; i >= 0; i-- {
		h = c.middlewares[i](h)
	}
	return h
}

// ThenFunc wraps a HandlerFunc and applies the chain
func (c Chain) ThenFunc(fn http.HandlerFunc) http.Handler {
	return c.Then(fn)
}
