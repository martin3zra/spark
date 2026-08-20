package app

import (
	"net/http"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
	"github.com/martin3zra/forge/support"
)

func (s *Server) login(ctx *routing.Context) {
	ctx.Render("Auth/Login", map[string]any{
		"translations": trans("auth"),
	})
}

func (s *Server) authHandler(ctx *routing.Context) {
	sess := session.GetSession(ctx.Request)

	var form LoginFormRequest
	if err := support.ParseRequest(ctx.Request, &form); err != nil {
		ctx.Back(http.StatusSeeOther)
		return
	}

	user, err := auth.NewAuth(ctx.Request.Context()).Authenticate(form.Email, form.Password)
	if err != nil {
		sess.Errors("email", s.trans("auth.credentialsDoNotMatch"))
		ctx.Back(http.StatusSeeOther)
		return
	}

	if err := s.sessionManager.ReGenerate(ctx.Request, user, map[string]any{}); err != nil {
		ctx.Error(err)
		return
	}

	if intended := sess.Get("intended"); intended != nil {
		sess.Delete("intended")
		ctx.Redirect(intended.(string), http.StatusSeeOther)
		return
	}
	ctx.Redirect("/home", http.StatusSeeOther)
}

func (s *Server) logoutHandler(ctx *routing.Context) {
	if err := s.sessionManager.Invalidate(ctx.Request); err != nil {
		ctx.Error(err)
		return
	}
	ctx.Redirect("/login", http.StatusSeeOther)
}

func (s *Server) createPasswordPrompt(ctx *routing.Context) {
	ctx.Render("Auth/CreatePassword", map[string]any{
		"translations": trans("auth"),
	})
}

func (s *Server) createPasswordHandler() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *CreatePasswordForm) {
		user := AuthUserFromContext(ctx.Request.Context())
		if err := user.MarkPasswordAsChanged(s.play, form.Password); err != nil {
			ctx.BackWith("password", err.Error())
			return
		}
		ctx.Redirect("/home", http.StatusSeeOther)
	})
}
