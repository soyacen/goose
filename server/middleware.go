package server

import (
	"net/http"

	"github.com/soyacen/goose"
)

// Middleware defines a function type for HTTP middleware.
// It receives an http.ResponseWriter, an http.Request, and the next handler (invoker) in the chain.
//
// Parameters:
//
//	response - http.ResponseWriter to write the HTTP response
//	request  - *http.Request containing the HTTP request data
//	invoker  - http.HandlerFunc representing the next handler in the middleware chain
type Middleware func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc)

// Chain combines multiple Middleware functions into a single Middleware.
// If no middlewares are provided, returns nil.
// If one middleware is provided, returns that middleware.
// If multiple middlewares are provided, returns a middleware that executes them in chain.
//
// Parameters:
//
//	middlewares - variadic list of Middleware functions to chain together
//
// Returns:
//
//	Middleware - a single middleware function representing the entire chain
func Chain(middlewares ...Middleware) Middleware {
	var mdw Middleware
	if len(middlewares) == 0 {
		mdw = nil
	} else if len(middlewares) == 1 {
		mdw = middlewares[0]
	} else {
		mdw = func(response http.ResponseWriter, request *http.Request, invoker http.HandlerFunc) {
			middlewares[0](response, request, getInvoker(middlewares, 0, invoker))
		}
	}
	return mdw
}

// getInvoker recursively builds the invoker chain for executing middlewares in sequence.
//
// Parameters:
//
//	interceptors   - slice of Middleware functions to be executed
//	curr           - current index in the interceptors slice
//	finalInvoker   - the final handler function to be executed after all middlewares
//
// Returns:
//
//	http.HandlerFunc - a handler function that continues the middleware chain
func getInvoker(interceptors []Middleware, curr int, finalInvoker http.HandlerFunc) http.HandlerFunc {
	if curr == len(interceptors)-1 {
		return finalInvoker
	}
	return func(response http.ResponseWriter, request *http.Request) {
		interceptors[curr+1](response, request, getInvoker(interceptors, curr+1, finalInvoker))
	}
}

// Invoke wraps a Middleware and a final handler function into an http.Handler.
// If the middleware is nil, directly calls the final handler.
// Otherwise, executes the middleware chain ending with the final handler.
//
// Parameters:
//
//	middleware - Middleware function to execute (can be nil)
//	invoke - http.HandlerFunc representing the final handler
//	response - http.ResponseWriter to write the HTTP response
//	request - *http.Request representing the incoming HTTP request
//	routeInfo - *goose.RouteInfo representing the route information
func Invoke(middleware Middleware, response http.ResponseWriter, request *http.Request, invoke http.HandlerFunc, routeInfo *goose.RouteInfo) {
	request = request.WithContext(goose.InjectRouteInfo(request.Context(), routeInfo))
	if middleware == nil {
		invoke(response, request)
		return
	}
	middleware(response, request, invoke)
}
