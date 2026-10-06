package app

import (
	"encoding/json"
	"net/http"

	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
)

// themeCatalogHandler handles GET /settings/theme/catalog: every preset (with
// its colors, for live preview) and every font. The customizer fetches it
// once, when it first opens, so pages don't carry it on every load.
func (s *Server) themeCatalogHandler(ctx *routing.Context) {
	ctx.JSON(http.StatusOK, map[string]any{
		"presets": themePresetCatalog(),
		"fonts":   themeFontCatalog(),
	})
}

// themeUpdate is the body of PUT /settings/theme: the whole customizer state.
type themeUpdate struct {
	ThemeSettings
	Appearance string `json:"appearance"`
}

// updateThemeHandler handles PUT /settings/theme, sent with fetch by the
// customizer (resources/js/components/theme-customizer.tsx) on every change.
// Unknown values fall back to the defaults instead of failing: the panel
// only offers valid ones, and a bad value must never break a page. The
// session's cached user is refreshed so the next page renders the new theme,
// and the response is the theme now in effect (an empty body resets to the
// defaults, so the client needs the result).
func (s *Server) updateThemeHandler(ctx *routing.Context) {
	var body themeUpdate
	if err := json.NewDecoder(ctx.Request.Body).Decode(&body); err != nil {
		ctx.JSON(http.StatusBadRequest, map[string]string{"message": "invalid JSON request body"})
		return
	}
	settings := body.ThemeSettings.normalize()
	settings.V = newThemeVersion()
	appearance := normalizeAppearance(body.Appearance)

	user := AuthUserFromContext(ctx.Request.Context())
	if _, err := s.play.Model(&userModel{}).WhereEq("id", user.Id).Update(ctx.Request.Context(), map[string]any{
		"appearance":     appearance,
		"theme_settings": settings.encode(),
	}); err != nil {
		ctx.Error(err)
		return
	}
	user.Appearance = appearance
	user.ThemeSettings = settings
	sess := session.GetSession(ctx.Request)
	sess.Put("user", user)

	ctx.JSON(http.StatusOK, s.currentTheme(ctx.Request.Context(), sess))
}
