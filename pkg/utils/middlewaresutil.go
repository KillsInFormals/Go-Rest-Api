package utils

import "net/http"


type Middlewares func(http.Handler) http.Handler

func ApplyMiddlewares(handler http.Handler, middlewares ...Middlewares) http.Handler {
	for _, middleware := range middlewares {
		handler = middleware(handler)
	}
	return handler
}