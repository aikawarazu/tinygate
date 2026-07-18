package gateway

import (
	"sort"
	"strings"

	"github.com/user/tinygate/config"
)

type Router struct {
	routes       []config.RouteConfig
	defaultRoute *config.RouteConfig
}

func NewRouter(routes []config.RouteConfig) *Router {
	sorted := make([]config.RouteConfig, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Prefix) > len(sorted[j].Prefix)
	})
	return &Router{routes: sorted}
}

// SetDefault configures a fallback route used when no explicit prefix matches.
// The route pointer is returned as-is from Match, so callers can compare
// against DefaultRoute() to detect the fallback case.
func (r *Router) SetDefault(route *config.RouteConfig) {
	r.defaultRoute = route
}

// DefaultRoute returns the configured default route, or nil if none is set.
func (r *Router) DefaultRoute() *config.RouteConfig {
	return r.defaultRoute
}

func (r *Router) Match(path string) (*config.RouteConfig, string, bool) {
	for i := range r.routes {
		prefix := r.routes[i].Prefix
		if strings.HasPrefix(path, prefix) {
			remaining := strings.TrimPrefix(path, prefix)
			if remaining == "" || strings.HasPrefix(remaining, "/") {
				return &r.routes[i], remaining, true
			}
		}
	}
	// No prefix matched — fall back to the default route, forwarding the
	// original path unchanged (the default has no prefix to strip).
	if r.defaultRoute != nil {
		return r.defaultRoute, path, true
	}
	return nil, "", false
}
