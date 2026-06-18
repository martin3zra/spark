package app

import (
	"context"
	"net/http"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/database"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/romsar/gonertia/v2"
)

// BindMiddleware injects per-request dependencies into the context.
func (s *Server) BindMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), database.ConnectionKey{}, s.db)
		ctx = context.WithValue(ctx, ConfigKey{}, s.config)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SharedProps exposes props available to every Inertia page.
func (s *Server) SharedProps(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		sess := ctx.Request.Context().Value(session.SessionContextKey{}).(*session.Session)

		props := map[string]any{
			"auth": map[string]any{
				"user": sess.Get("user"),
			},
			"csrf_token":   sess.Get("csrf_token"),
			"errors":       sess.Get("errors"),
			"flash":        sess.Get("flash"),
			"translations": s.translator.Translations,
		}

		s.sessionManager.AgeFlash(sess)

		ctxWithProps := gonertia.SetProps(ctx.Request.Context(), props)
		next(ctx.WithContext(ctxWithProps))
	}
}

// AuthenticatedMiddleware requires a logged-in user and seeds the identity context.
func (s *Server) AuthenticatedMiddleware(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		sess := session.GetSession(ctx.Request)

		userId := sess.Get("user_id")
		if userId == nil || userId.(float64) == 0 {
			if ctx.Request.RequestURI == "/" {
				next(ctx)
				return
			}
			sess.Put("intended", ctx.Request.RequestURI)
			ctx.Redirect("/login", http.StatusSeeOther)
			return
		}

		userCtx := context.WithValue(ctx.Request.Context(), auth.ContextUserID{}, sess.Get("user"))

		// Seed the permission set so route .Can(...) guards can resolve.
		user := AuthUserFromContext(userCtx)
		permCtx := context.WithValue(userCtx, routing.PermissionKey{}, permissions(user.GetRole()))

		next(ctx.WithContext(permCtx))
	}
}

// RedirectIfAuthenticated sends logged-in users away from guest-only pages.
func RedirectIfAuthenticated(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		sess := session.GetSession(ctx.Request)
		userId := sess.Get("user_id")
		if userId != nil && userId.(float64) != 0 {
			ctx.Redirect("/home", http.StatusSeeOther)
			return
		}
		next(ctx)
	}
}

// Verified blocks users that have not confirmed their email.
func Verified(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		user := AuthUserFromContext(ctx.Request.Context())
		if !user.IsEmpty() && user.EmailVerifiedAt == nil {
			ctx.Redirect("/verify-email")
			return
		}
		next(ctx)
	}
}

// EnsurePasswordHasBeenChanged forces a first-login password rotation.
func EnsurePasswordHasBeenChanged(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		var user any = auth.User(ctx.Request.Context())
		if u, ok := user.(foundation.MustVerifyPassword); ok && u.HasNotChangedPassword() {
			ctx.Redirect("/passwords/create", http.StatusFound)
			return
		}
		next(ctx)
	}
}

// Signed verifies a temporary signed URL.
func (s *Server) Signed(next routing.HandlerFunc) routing.HandlerFunc {
	return func(ctx *routing.Context) {
		if routing.VerifyRequest(ctx.Request, string(s.config.secretKey)) {
			next(ctx)
			return
		}
		ctx.Error(foundation.Unauthorized{})
	}
}
