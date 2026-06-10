package gateway

import (
	"sort"
	"strings"

	"github.com/user/just-llm-gateway/config"
)

type Router struct {
	routes []config.RouteConfig
}

func NewRouter(routes []config.RouteConfig) *Router {
	sorted := make([]config.RouteConfig, len(routes))
	copy(sorted, routes)
	sort.Slice(sorted, func(i, j int) bool {
		return len(sorted[i].Prefix) > len(sorted[j].Prefix)
	})
	return &Router{routes: sorted}
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
	return nil, "", false
}
