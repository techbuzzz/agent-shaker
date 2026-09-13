package middleware

import "net/http"

// Middleware is a function that wraps an http.Handler with extra behaviour.
// Composition is right-to-left: Chain(A, B, C)(h) yields A(B(C(h))).
type Middleware func(http.Handler) http.Handler

// Chain composes the supplied middlewares into a single Middleware. Order
// matches Go's variadic call order — Chain(A, B, C) applies A first, then B,
// then C, on the way down to the wrapped handler.
func Chain(hs ...Middleware) Middleware {
	return func(next http.Handler) http.Handler {
		h := next
		for i := len(hs) - 1; i >= 0; i-- {
			h = hs[i](h)
		}
		return h
	}
}

// Apply composes middlewares around h. Shorthand for Chain(...)(h).
func Apply(h http.Handler, hs ...Middleware) http.Handler {
	return Chain(hs...)(h)
}
