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

	// Public landing page. Own group so it carries SharedProps (for auth.user)
	// but none of the auth/redirect middleware — both guests and logged-in
	// users can see it.
	s.route.Group(func(route *routing.Router) {
		route.GET("/", s.welcomeHandler)
	})

	// Guest-only routes. GroupWithMiddleware (not WithMiddleware(...).Group(...))
	// scopes RedirectIfAuthenticated to just this block — WithMiddleware mutates
	// s.route in place with nothing to restore it, so it would otherwise leak
	// onto every route registered after this one (e.g. Kiosk's routes, if
	// installed — see the "Optional: Kiosk ops console" section below).
	s.route.GroupWithMiddleware([]routing.Middleware{RedirectIfAuthenticated}, func(route *routing.Router) {
		route.GET("/login", s.login)
		route.POST("/login", s.authHandler)
	})

	// Theme customizer (app/theme-handlers.go). Its own group, without
	// Verified, so the theme can be changed from any signed-in page.
	s.route.GroupWithMiddleware([]routing.Middleware{s.AuthenticatedMiddleware}, func(route *routing.Router) {
		route.GET("/settings/theme/catalog", s.themeCatalogHandler)
		route.PUT("/settings/theme", s.updateThemeHandler)
	})

	// Authenticated routes.
	s.route.GroupWithMiddleware([]routing.Middleware{s.AuthenticatedMiddleware, Verified}, func(route *routing.Router) {
		route.GET("/home", s.homeHandler).Can("view:dashboard")
		route.POST("/logout", s.logoutHandler)

		route.GET("/passwords/create", s.createPasswordPrompt).
			WithoutMiddleware(Verified)
		route.POST("/passwords", s.createPasswordHandler()).
			WithoutMiddleware(Verified)
	})
}
