package app

import "github.com/martin3zra/forge/routing"

func (s *Server) welcomeHandler(ctx *routing.Context) {
	ctx.Render("Welcome", map[string]any{
		"translations": trans("global"),
	})
}

func (s *Server) homeHandler(ctx *routing.Context) {
	ctx.Render("Dashboard", map[string]any{
		"translations": trans("global"),
	})
}
