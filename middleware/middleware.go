package middleware

import "net/http"

// compose takes two functions and returns a new function that is the composition of the input functions.
func compose[A any, B any, C any](f func(A) B, g func(B) C) func(A) C {
	return func(a A) C {
		return g(f(a))
	}
}

// ComposeFunc takes a list of middlewares and returns a new middleware that is the composition of the input middlewares.
// The first middleware in the list is the outermost middleware, and the last middleware in the list is the innermost middleware.
func ComposeFunc(middlewares ...func(http.HandlerFunc) http.HandlerFunc) func(http.HandlerFunc) http.HandlerFunc {
	composition := func(h http.HandlerFunc) http.HandlerFunc {
		return h
	}
	for _, m := range middlewares {
		composition = compose(composition, m)
	}
	return composition
}

func Compose(middlewares ...func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	composition := func(h http.Handler) http.Handler {
		return h
	}
	for _, m := range middlewares {
		composition = compose(composition, m)
	}
	return composition
}
