package app

import (
	"net/http"

	"github.com/martin3zra/forge/routing"
)

func (s *Server) bootRoutes() {
	s.route.WithMiddleware(s.SharedProps)

	// Health check — open.
	s.route.GET("/health", func(ctx *routing.Context) {
		ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
	})

	// Guest-only routes.
	s.route.
		WithMiddleware(RedirectIfAuthenticated).
		Group(func(route *routing.Router) {
			route.GET("/login", s.login)
			route.POST("/login", s.authHandler)
		})

	// Authenticated routes.
	s.route.
		WithMiddleware(s.AuthenticatedMiddleware, Verified).
		WithoutGroupMiddleware(RedirectIfAuthenticated).
		Group(func(route *routing.Router) {
			route.GET("/home", s.homeHandler).Can("view:dashboard")
			route.POST("/logout", s.logoutHandler)

			route.GET("/passwords/create", s.createPasswordPrompt).
				WithoutMiddleware(Verified)
			route.POST("/passwords", s.createPasswordHandler()).
				WithoutMiddleware(Verified)
		})
}
