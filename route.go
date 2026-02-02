package goose

import "context"

type routeInfoKey struct{}

// RouteInfo is a struct that holds information about a route.
type RouteInfo struct {
	// HttpMethod is the HTTP method of the route.
	HttpMethod string
	// Pattern is the path pattern of the route.
	Pattern string
	// FullMethod is the full RPC method string, i.e., /package.service/method.
	FullMethod string
}

// ExtractRouteInfo extracts the route information from the context.
func ExtractRouteInfo(ctx context.Context) (*RouteInfo, bool) {
	val, ok := ctx.Value(routeInfoKey{}).(*RouteInfo)
	return val, ok
}

// InjectRouteInfo injects the route information into the context.
func InjectRouteInfo(ctx context.Context, routeInfo *RouteInfo) context.Context {
	return context.WithValue(ctx, routeInfoKey{}, routeInfo)
}
