package app

import (
	"net/http"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
)

func (s *Server) bootRoutes() {
	// Built frontend assets — served with no middleware. Registered first, in
	// its own group, so the snapshot middleware chain is empty.
	s.route.Group(func(route *routing.Router) {
		uiAssets := foundation.GetBuildAssets(s.assets, "public/build")
		route.FileServer("/build/", http.FS(uiAssets))
	})

	s.route.WithMiddleware(s.SharedProps)

	// Health check — open. Registered inside its own group so the auth
	// middleware added by later groups is not inherited (forge accumulates
	// router middleware in place; groups snapshot it at registration time).
	s.route.Group(func(route *routing.Router) {
		route.GET("/health", func(ctx *routing.Context) {
			ctx.JSON(http.StatusOK, map[string]string{"status": "ok"})
		})
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
