package main

import (
	"os"
	"path/filepath"
	"strings"
)

// companyMembersGo is app/company-members.go — a new file. Just enough of
// Kiosk's company_users/roles tables to resolve "who's on this company" for
// the Settings page's Users tab. No JSON endpoint of its own in the
// baseline (nothing here needs one yet) — add one the way Repairs'
// assigned-technician picker does (fetch("/api/company-members")) if a
// later page needs to resolve a member by id.
const companyMembersGo = `package app

import "github.com/martin3zra/forge/routing"

// companyUserRow is a narrow, read-only shadow of Kiosk's company_users
// pivot — Kiosk owns that table, so this reads just the columns needed (who
// belongs to the current company, and their role) without pointing the
// module dependency the wrong way.
type companyUserRow struct {
	UserID int  ` + "`db:\"user_id\"`" + `
	RoleID *int ` + "`db:\"role_id\"`" + `
}

func (companyUserRow) TableName() string { return "company_users" }

// roleRow is the same kind of narrow shadow over Kiosk's roles table —
// enough to resolve a company_users.role_id into a display name.
type roleRow struct {
	ID   int    ` + "`db:\"id\"`" + `
	Name string ` + "`db:\"name\"`" + `
}

func (roleRow) TableName() string { return "roles" }

// companyMemberRow is what the Settings page's Users tab renders — a
// lowercase-tagged DTO (unlike most models in this app) since it's
// assembled from three tables and has no single backing model.
type companyMemberRow struct {
	ID    int    ` + "`json:\"id\"`" + `
	UUID  string ` + "`json:\"uuid\"`" + `
	Name  string ` + "`json:\"name\"`" + `
	Email string ` + "`json:\"email\"`" + `
	Role  string ` + "`json:\"role\"`" + `
}

// loadCompanyMembers is the query behind the Settings page's Users tab —
// three queries regardless of staff count (memberships, then users and
// roles batched by id) rather than one per member.
func loadCompanyMembers(ctx *routing.Context, companyID int) ([]companyMemberRow, error) {
	db := ctx.DB()
	var memberships []companyUserRow
	if err := db.Model(&companyUserRow{}).WhereEq("company_id", companyID).
		Get(ctx.Request.Context(), &memberships); err != nil {
		return nil, err
	}
	if len(memberships) == 0 {
		return []companyMemberRow{}, nil
	}

	userIDs := make([]int, len(memberships))
	var roleIDs []int
	for i, m := range memberships {
		userIDs[i] = m.UserID
		if m.RoleID != nil {
			roleIDs = append(roleIDs, *m.RoleID)
		}
	}

	var users []userModel
	if err := db.Model(&userModel{}).WhereIn("id", userIDs).Get(ctx.Request.Context(), &users); err != nil {
		return nil, err
	}
	userByID := make(map[int]userModel, len(users))
	for _, u := range users {
		userByID[u.ID] = u
	}

	roleByID := map[int]string{}
	if len(roleIDs) > 0 {
		var roles []roleRow
		if err := db.Model(&roleRow{}).WhereIn("id", roleIDs).Get(ctx.Request.Context(), &roles); err != nil {
			return nil, err
		}
		for _, r := range roles {
			roleByID[r.ID] = r.Name
		}
	}

	rows := make([]companyMemberRow, 0, len(memberships))
	for _, m := range memberships {
		user, ok := userByID[m.UserID]
		if !ok {
			continue
		}
		row := companyMemberRow{ID: user.ID, UUID: user.UUID, Name: user.Name, Email: user.Email}
		if m.RoleID != nil {
			row.Role = roleByID[*m.RoleID]
		}
		rows = append(rows, row)
	}
	return rows, nil
}
`

func wireKioskCompanyMembers(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "company-members.go")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, []byte(companyMembersGo), 0o644); err != nil {
		return false, err
	}
	return true, gofmtFile(path)
}

// companySettingsGo is app/company-settings.go — a new file. The General
// tab's backend: read/edit the current company's name/tax id/theme preset.
// Deliberately excludes anything beyond that (NCF toggles, Locations, Tax
// Rates, Registers, ...) — those are natural extensions of this same
// pattern (a companyRow-backed handler, an owner-only edit route) that this
// baseline leaves for the app to add itself.
const companySettingsGo = `package app

import (
	"net/http"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
)

// companyRow is a narrow, read-only-ish shadow of Kiosk's companies table —
// Kiosk owns this table, so this reads (and, for name/identifier/theme,
// writes) just the columns this app's own settings page touches. Not
// tenant-scoped via Scopes() — company.id is itself the tenant key, so
// filtering is by hand (WhereEq "id"), the same pattern company-members.go
// uses for company_users/roles.
type companyRow struct {
	ID         int     ` + "`db:\"id\"`" + `
	UUID       string  ` + "`db:\"uuid\"`" + `
	Name       string  ` + "`db:\"name\"`" + `
	Status     string  ` + "`db:\"status\"`" + `
	Identifier *string ` + "`db:\"identifier\"`" + `
	PlanID     *int    ` + "`db:\"plan_id\"`" + `
	// One of themePresetOrder's ids (theme.go) — resolved separately, on
	// every request, by resolveTheme/resolveCompanyThemePreset; this copy
	// is read-only display data for the Appearance tab's picker.
	ThemePreset string ` + "`db:\"theme_preset\"`" + `
}

func (companyRow) TableName() string { return "companies" }

// planNameRow resolves a company's plan_id to a display name for the
// settings page — plan assignment itself stays Kiosk's job, this just reads.
type planNameRow struct {
	ID   int    ` + "`db:\"id\"`" + `
	Name string ` + "`db:\"name\"`" + `
}

func (planNameRow) TableName() string { return "plans" }

// currentCompanyRole resolves the signed-in user's role name within the
// current company — the one thing this app gates behavior on so far
// (editing company settings is owner-only).
func currentCompanyRole(ctx *routing.Context, companyID, userID int) (string, bool) {
	db := ctx.DB()
	var membership companyUserRow
	if err := db.Model(&companyUserRow{}).WhereEq("company_id", companyID).WhereEq("user_id", userID).
		First(ctx.Request.Context(), &membership); err != nil {
		return "", false
	}
	if membership.RoleID == nil {
		return "", false
	}
	var role roleRow
	if err := db.Model(&roleRow{}).WhereEq("id", *membership.RoleID).First(ctx.Request.Context(), &role); err != nil {
		return "", false
	}
	return role.Name, true
}

// companySettingsPayload is what the Settings page returns. can_edit is
// computed here, server-side, so the frontend never has to re-derive "is
// this user the owner" on its own.
type companySettingsPayload struct {
	UUID        string  ` + "`json:\"uuid\"`" + `
	Name        string  ` + "`json:\"name\"`" + `
	Status      string  ` + "`json:\"status\"`" + `
	Identifier  *string ` + "`json:\"identifier\"`" + `
	PlanName    *string ` + "`json:\"plan_name\"`" + `
	CanEdit     bool    ` + "`json:\"can_edit\"`" + `
	ThemePreset string  ` + "`json:\"theme_preset\"`" + `
}

func loadCompanySettings(ctx *routing.Context, companyID int) (companySettingsPayload, error) {
	db := ctx.DB()
	var company companyRow
	if err := db.Model(&companyRow{}).WhereEq("id", companyID).First(ctx.Request.Context(), &company); err != nil {
		return companySettingsPayload{}, err
	}

	var planName *string
	if company.PlanID != nil {
		var plan planNameRow
		if err := db.Model(&planNameRow{}).WhereEq("id", *company.PlanID).
			First(ctx.Request.Context(), &plan); err == nil {
			planName = &plan.Name
		}
	}

	user := AuthUserFromContext(ctx.Request.Context())
	role, _ := currentCompanyRole(ctx, companyID, user.Id)

	return companySettingsPayload{
		UUID:        company.UUID,
		Name:        company.Name,
		Status:      company.Status,
		Identifier:  company.Identifier,
		PlanName:    planName,
		CanEdit:     role == "owner",
		ThemePreset: company.ThemePreset,
	}, nil
}

// settingsPage renders the Settings page — a sub-nav (General/Users/
// Profile/Appearance, see resources/js/Pages/Settings/Index.tsx) over one
// Inertia page, each section's data loaded up front as its own prop. Add a
// new section (Locations, Tax Rates, ...) the same way: a loadX call here,
// a new prop, a new entry in the page's own SECTIONS list.
func settingsPage() routing.HandlerFunc {
	return func(ctx *routing.Context) {
		companyID, ok := companyIDFromSession(session.GetSession(ctx.Request))
		if !ok {
			ctx.Error(foundation.Unauthorized{})
			return
		}
		payload, err := loadCompanySettings(ctx, companyID)
		if err != nil {
			ctx.Error(err)
			return
		}
		members, err := loadCompanyMembers(ctx, companyID)
		if err != nil {
			ctx.Error(err)
			return
		}
		ctx.Render("Settings/Index", map[string]any{
			"company":      payload,
			"members":      members,
			"themePresets": themePresetCatalog(),
		})
	}
}

// companySettingsUpdate handles PUT /company — an Inertia form submit from
// the Settings page itself, not a JSON API call: on success it redirects
// back to /settings, which re-renders with the edit already in place. Edits
// name/identifier/theme_preset only, owner-only. Status and plan stay out
// of reach here on purpose: a company shouldn't be able to suspend itself,
// and plan changes are Kiosk's billing-adjacent job, not this page's.
func companySettingsUpdate() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *UpdateCompanySettingsForm) {
		companyID, ok := companyIDFromSession(session.GetSession(ctx.Request))
		if !ok {
			ctx.Error(foundation.Unauthorized{})
			return
		}

		user := AuthUserFromContext(ctx.Request.Context())
		role, _ := currentCompanyRole(ctx, companyID, user.Id)
		if role != "owner" {
			ctx.Error(foundation.Unauthorized{})
			return
		}

		if _, err := ctx.DB().Model(&companyRow{}).WhereEq("id", companyID).
			Update(ctx.Request.Context(), form.Validated()); err != nil {
			ctx.Error(err)
			return
		}

		ctx.Flash("success", "Settings updated")
		ctx.Redirect("/settings", http.StatusSeeOther)
	})
}
`

func wireKioskCompanySettings(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "company-settings.go")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, []byte(companySettingsGo), 0o644); err != nil {
		return false, err
	}
	return true, gofmtFile(path)
}

// profileGo is app/profile.go — a new file, copied as-is: it's already
// fully generic (ctx.DB(), s.mailer, s.config, AuthUserFromContext are all
// present in a bare kiosk-installed app once wireKioskConfig/wireKioskServer
// have run), so unlike company-settings.go/company-members.go it needs no
// trimming for the kiosk baseline.
const profileGo = `package app

import (
	"net/http"
	"strconv"
	"time"

	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/forge/mailer"
	"github.com/martin3zra/forge/routing"
	"github.com/martin3zra/forge/session"
)

// emailConfirmExpiry bounds how long a signed /email/confirm link stays
// valid — both the plain first-verification case and the email-change case
// share this window.
const emailConfirmExpiry = time.Hour

// userByEmail looks up a user by email — used by profileUpdateEmail and
// emailConfirmHandler to hand-check uniqueness excluding the acting user's
// own row (see UpdateEmailForm's doc comment for why this isn't a "unique"
// validator rule).
func userByEmail(ctx *routing.Context, email string) (userModel, error) {
	var row userModel
	err := ctx.DB().Model(&userModel{}).WhereEq("email", email).First(ctx.Request.Context(), &row)
	return row, err
}

// profileUpdateName handles PUT /profile — an Inertia form submit from the
// Settings page's Profile section, redirecting back there on success. The
// session's cached auth.user is refreshed in place so the new name shows up
// immediately without a re-login — SharedProps (app/middleware.go) reads
// sess.Get("user") verbatim on every request.
func profileUpdateName() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *UpdateProfileForm) {
		user := AuthUserFromContext(ctx.Request.Context())
		if _, err := ctx.DB().Model(&userModel{}).WhereEq("id", user.Id).
			Update(ctx.Request.Context(), map[string]any{"name": form.Name}); err != nil {
			ctx.Error(err)
			return
		}
		user.Name = form.Name
		session.GetSession(ctx.Request).Put("user", user)
		ctx.Flash("success", "Name updated")
		ctx.Redirect("/settings?section=profile", http.StatusSeeOther)
	})
}

// profileUpdateAppearance handles PUT /profile/appearance — same pattern as
// profileUpdateName (session refresh so resolveTheme/userAppearanceFromSession,
// app/theme.go, sees the change on the very next request without a re-login).
func profileUpdateAppearance() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *UpdateAppearanceForm) {
		user := AuthUserFromContext(ctx.Request.Context())
		if _, err := ctx.DB().Model(&userModel{}).WhereEq("id", user.Id).
			Update(ctx.Request.Context(), map[string]any{"appearance": form.Appearance}); err != nil {
			ctx.Error(err)
			return
		}
		user.Appearance = form.Appearance
		session.GetSession(ctx.Request).Put("user", user)
		ctx.Flash("success", "Appearance updated")
		ctx.Redirect("/settings?section=profile", http.StatusSeeOther)
	})
}

// profileUpdatePassword handles PUT /profile/password — current-password-
// gated via UpdatePasswordForm's "current_password" rule, then reuses
// AuthUser.MarkPasswordAsChanged the same way passwords/create does.
func profileUpdatePassword() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *UpdatePasswordForm) {
		user := AuthUserFromContext(ctx.Request.Context())
		if err := user.MarkPasswordAsChanged(ctx.DB(), form.Password); err != nil {
			ctx.BackWith("password", err.Error())
			return
		}
		ctx.Flash("success", "Password updated")
		ctx.Redirect("/settings?section=profile", http.StatusSeeOther)
	})
}

// profileUpdateEmail handles POST /profile/email — does not write the new
// email yet: it mails a signed confirmation link to the *new* address, and
// only GET /email/confirm (emailConfirmHandler, Signed middleware) actually
// swaps the column, which simultaneously proves ownership and verifies it.
func (s *Server) profileUpdateEmail() routing.HandlerFunc {
	return routing.WithRequest(func(ctx *routing.Context, form *UpdateEmailForm) {
		user := AuthUserFromContext(ctx.Request.Context())
		if existing, err := userByEmail(ctx, form.Email); err == nil && existing.ID != user.Id {
			ctx.BackWith("email", "That email is already in use")
			return
		}
		if err := s.sendEmailConfirmation(user.Id, user.Name, form.Email, form.Email); err != nil {
			ctx.Error(err)
			return
		}
		ctx.Flash("success", "Check "+form.Email+" for a confirmation link")
		ctx.Redirect("/settings?section=profile", http.StatusSeeOther)
	})
}

// verifyEmailPrompt renders the page Verified (app/middleware.go) redirects
// an unverified user to.
func (s *Server) verifyEmailPrompt(ctx *routing.Context) {
	ctx.Render("Auth/VerifyEmail", map[string]any{})
}

// profileResendVerification handles POST /email/verify — mails a signed
// link to the user's *current* email with no new_email param, so
// emailConfirmHandler treats it as a plain first-verification rather than an
// email change.
func (s *Server) profileResendVerification() routing.HandlerFunc {
	return func(ctx *routing.Context) {
		user := AuthUserFromContext(ctx.Request.Context())
		if err := s.sendEmailConfirmation(user.Id, user.Name, user.Email, ""); err != nil {
			ctx.Error(err)
			return
		}
		ctx.Flash("success", "Check "+user.Email+" to verify your address")
		ctx.Redirect("/verify-email", http.StatusSeeOther)
	}
}

// emailConfirmHandler handles GET /email/confirm, behind the Signed
// middleware (app/middleware.go) — the signature is the credential, so this
// route carries no auth requirement of its own and works from a fresh
// browser tab. newEmail's presence distinguishes an email-change
// confirmation from a plain first-verification (see profileUpdateEmail vs.
// profileResendVerification, the two callers of sendEmailConfirmation).
func (s *Server) emailConfirmHandler(ctx *routing.Context) {
	userID, err := strconv.Atoi(ctx.Query("user_id"))
	if err != nil {
		ctx.Error(foundation.Unauthorized{})
		return
	}
	newEmail := ctx.Query("new_email")

	updates := map[string]any{"email_verified_at": time.Now()}
	if newEmail != "" {
		// Someone else may have claimed the address in the time between the
		// link being sent and clicked — re-check, don't trust the link alone.
		if existing, err := userByEmail(ctx, newEmail); err == nil && existing.ID != userID {
			ctx.Flash("error", "That email is no longer available")
			ctx.Redirect("/settings?section=profile", http.StatusSeeOther)
			return
		}
		updates["email"] = newEmail
	}

	if _, err := ctx.DB().Model(&userModel{}).WhereEq("id", userID).
		Update(ctx.Request.Context(), updates); err != nil {
		ctx.Error(err)
		return
	}

	// Refresh the session's cached auth.user if this is the same browser
	// that made the change — the link may well be opened elsewhere (e.g. a
	// phone that never had a session here), in which case there's nothing to
	// refresh and the redirect below just bounces through /login. This route
	// carries only the Signed middleware (see routes.go), not
	// AuthenticatedMiddleware, so AuthUserFromContext's ctx value is never
	// populated here — the session itself has to be read directly, tolerating
	// both shapes user_id arrives in the same way companyIDFromSession does.
	sess := session.GetSession(ctx.Request)
	sessionUserID, sessionHasUser := 0, false
	switch v := sess.Get("user_id").(type) {
	case int:
		sessionUserID, sessionHasUser = v, true
	case float64:
		sessionUserID, sessionHasUser = int(v), true
	}
	if sessionHasUser && sessionUserID == userID {
		var row userModel
		if err := ctx.DB().Model(&userModel{}).WhereEq("id", userID).
			First(ctx.Request.Context(), &row); err == nil {
			sess.Put("user", row.toAuthUser())
		}
	}

	ctx.Flash("success", "Email confirmed")
	ctx.Redirect("/settings?section=profile", http.StatusSeeOther)
}

// sendEmailConfirmation signs a /email/confirm link and mails it to toEmail
// — the new address for an email change (newEmail set), or the user's
// current address for a plain resend (newEmail empty, see
// profileResendVerification).
func (s *Server) sendEmailConfirmation(userID int, name, toEmail, newEmail string) error {
	params := map[string]string{"user_id": strconv.Itoa(userID)}
	isChange := newEmail != ""
	if isChange {
		params["new_email"] = newEmail
	}

	confirmURL, err := routing.TemporarySignedURL(s.config.host+"/email/confirm", params, string(s.config.secretKey), emailConfirmExpiry)
	if err != nil {
		return err
	}

	s.mailer.To(toEmail, name).Send(emailConfirmMailable{
		confirmURL: confirmURL,
		isChange:   isChange,
		appName:    s.config.appName,
		expiresAt:  time.Now().Add(emailConfirmExpiry),
	})
	return nil
}

// emailConfirmMailable renders resources/views/mail/confirm-email.html for
// both cases sendEmailConfirmation handles.
type emailConfirmMailable struct {
	confirmURL string
	isChange   bool
	appName    string
	expiresAt  time.Time
}

func (m emailConfirmMailable) Subject() string {
	if m.isChange {
		return "Confirm your new email address"
	}
	return "Verify your email address"
}

// To returns nil, not the recipient — sendEmailConfirmation already sets
// them as the primary recipient via mailer.Mailer's own .To(...).
func (m emailConfirmMailable) To() []mailer.Individual { return nil }

func (m emailConfirmMailable) Content() string {
	return "resources/views/mail/confirm-email.html"
}

func (m emailConfirmMailable) Data() map[string]any {
	message := "Please verify your email address to finish setting up your account."
	if m.isChange {
		message = "Click below to confirm this is your new email address."
	}
	return map[string]any{
		"Subject":     m.Subject(),
		"Title":       m.Subject(),
		"Message":     message,
		"CTAText":     "Confirm email",
		"CTAURL":      m.confirmURL,
		"ExpiresAt":   m.expiresAt.Format("January 2, 2006 3:04 PM"),
		"Year":        time.Now().Year(),
		"CompanyName": m.appName,
	}
}

func (m emailConfirmMailable) Attachments() []mailer.Attachment { return nil }
`

func wireKioskProfile(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "profile.go")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.WriteFile(path, []byte(profileGo), 0o644); err != nil {
		return false, err
	}
	return true, gofmtFile(path)
}

// verifyEmailTSX is resources/js/Pages/Auth/VerifyEmail.tsx — a new file,
// copied as-is (no theming/settings-specific content, just the resend
// prompt verifyEmailPrompt renders).
const verifyEmailTSX = `import { router } from "@inertiajs/react";
import { useState } from "react";

import { Button } from "@/components/ui/button";
import AuthLayout from "@/layouts/auth-layout";
import { useCsrfHeaders } from "@/lib/csrf";

// Rendered by verifyEmailPrompt (app/profile.go) — where the Verified
// middleware (app/middleware.go) redirects a logged-in user whose
// email_verified_at is still null. Resending posts to /email/verify
// (profileResendVerification), which mails a signed confirmation link
// (emailConfirmHandler) to the account's current email; clicking it sets
// email_verified_at and this page stops appearing on future visits.
export default function VerifyEmail() {
  const headers = useCsrfHeaders();
  const [sending, setSending] = useState(false);

  function resend() {
    setSending(true);
    router.post(
      "/email/verify",
      {},
      { headers, preserveScroll: true, onFinish: () => setSending(false) },
    );
  }

  return (
    <AuthLayout
      title="Verify your email"
      description="We need to confirm your email address before you can continue."
    >
      <div className="flex flex-col gap-6">
        <p className="text-muted-foreground text-sm">
          Click the button below to receive a confirmation link.
        </p>
        <Button onClick={resend} disabled={sending} className="w-full">
          {sending ? "Sending…" : "Resend confirmation email"}
        </Button>
      </div>
    </AuthLayout>
  );
}
`

func wireKioskVerifyEmailPage(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "Pages", "Auth", "VerifyEmail.tsx")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(verifyEmailTSX), 0o644)
}

// wireKioskSettingsForms appends the Settings/Profile form types (plus the
// clearEmptyStrings helper they share) to app/forms.go.
func wireKioskSettingsForms(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "forms.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), "UpdateCompanySettingsForm") {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `type CreatePasswordForm struct {
	support.FormRequest
	Password string ` + "`json:\"password\"`" + `
}

func (f *CreatePasswordForm) Rules() map[string]any {
	return map[string]any{
		"password": "required|min:8",
	}
}`,
			replacement: `type CreatePasswordForm struct {
	support.FormRequest
	Password string ` + "`json:\"password\"`" + `
}

func (f *CreatePasswordForm) Rules() map[string]any {
	return map[string]any{
		"password": "required|min:8",
	}
}

// clearEmptyStrings sets validated[key] to nil for every key whose value is
// an empty string — mirrors an emptied field on a fully pre-filled edit
// form meaning "clear this column to NULL," not "leave it untouched." A
// present-but-empty string field comes through Validated() as a literal
// "", not nil, so any form that wants an edit form's empty field to clear
// the column calls this from PassedValidation.
func clearEmptyStrings(validated map[string]any, keys ...string) {
	for _, k := range keys {
		if s, ok := validated[k].(string); ok && s == "" {
			validated[k] = nil
		}
	}
}

// UpdateCompanySettingsForm edits the company's own name/tax id/theme —
// status and plan stay out of reach here (see companySettingsUpdate,
// app/company-settings.go).
type UpdateCompanySettingsForm struct {
	support.FormRequest
	Name string ` + "`json:\"name\"`" + `
	// A normal pre-filled edit-form field: an emptied Identifier clears the
	// stored tax id to NULL rather than leaving the old value in place —
	// see PassedValidation below.
	Identifier string ` + "`json:\"identifier\"`" + `
	// One of themePresetOrder's ids (app/theme.go) — the Appearance tab's
	// picker always sends this alongside the rest of the form, so it's
	// effectively always present even though the rule itself is "sometimes".
	ThemePreset string ` + "`json:\"theme_preset\"`" + `
}

func (f *UpdateCompanySettingsForm) Rules() map[string]any {
	return map[string]any{
		"name":         "required",
		"identifier":   "sometimes",
		"theme_preset": "sometimes|in:" + themePresetIDs(),
	}
}

// PassedValidation mutates the map companySettingsUpdate reads back via
// f.Validated() — an emptied Identifier clears the stored tax id to NULL
// rather than leaving the old value in place.
func (f *UpdateCompanySettingsForm) PassedValidation(validated map[string]any) {
	clearEmptyStrings(validated, "identifier")
}

// UpdateProfileForm changes the logged-in user's own display name — see
// profileUpdateName (app/profile.go).
type UpdateProfileForm struct {
	support.FormRequest
	Name string ` + "`json:\"name\"`" + `
}

func (f *UpdateProfileForm) Rules() map[string]any {
	return map[string]any{
		"name": "required",
	}
}

// UpdatePasswordForm changes the logged-in user's own password.
// "current_password" is a forge-builtin rule — it checks CurrentPassword
// against the logged-in user's stored hash via the resolvers registered in
// auth-user.go, so there's no separate hand-rolled check needed here.
type UpdatePasswordForm struct {
	support.FormRequest
	CurrentPassword string ` + "`json:\"current_password\"`" + `
	Password        string ` + "`json:\"password\"`" + `
}

func (f *UpdatePasswordForm) Rules() map[string]any {
	return map[string]any{
		"current_password": "required|current_password",
		"password":         "required|min:8",
	}
}

// UpdateEmailForm starts an email change — see profileUpdateEmail
// (app/profile.go). Uniqueness (excluding the acting user's own row) is
// hand-checked in the handler rather than via a "unique" rule, since that
// rule's dynamic ignore-id needs the fluent Rule{}.Unique(...).Ignore(...)
// builder and Rules() here has no access to the acting user.
type UpdateEmailForm struct {
	support.FormRequest
	Email string ` + "`json:\"email\"`" + `
}

func (f *UpdateEmailForm) Rules() map[string]any {
	return map[string]any{
		"email": "required|email",
	}
}

// UpdateAppearanceForm changes the logged-in user's own light/dark/system
// preference — see profileUpdateAppearance (app/profile.go). Independent
// of the company's theme_preset (UpdateCompanySettingsForm), which is the
// separate, owner-only, shared-brand-color setting.
type UpdateAppearanceForm struct {
	support.FormRequest
	Appearance string ` + "`json:\"appearance\"`" + `
}

func (f *UpdateAppearanceForm) Rules() map[string]any {
	return map[string]any{
		"appearance": "required|in:light,dark,system",
	}
}`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return false, err
	}
	if err := gofmtFile(path); err != nil {
		return false, err
	}
	return true, nil
}

// wireKioskSettingsRoutes registers /settings, /company, /profile*,
// /verify-email, /email/verify, and the signed /email/confirm route. Must
// run after wireKioskRoutes and wireKioskRoutesAccessControl — its anchor
// matches app/routes.go's shape once both of those have already patched it.
func wireKioskSettingsRoutes(root string) (applied bool, err error) {
	path := filepath.Join(root, "app", "routes.go")
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	if strings.Contains(string(data), `route.GET("/settings"`) {
		return false, nil
	}

	patches := []wirePatch{
		{
			anchor: `	// Authenticated routes.
	s.route.GroupWithMiddleware([]routing.Middleware{s.AuthenticatedMiddleware, Verified, s.RedirectDeveloperToKiosk}, func(route *routing.Router) {
		route.GET("/home", s.homeHandler).Can("view:dashboard")
		route.POST("/logout", s.logoutHandler).
			WithoutMiddleware(s.RedirectDeveloperToKiosk)

		route.GET("/passwords/create", s.createPasswordPrompt).
			WithoutMiddleware(Verified)
		route.POST("/passwords", s.createPasswordHandler()).
			WithoutMiddleware(Verified)
	})

	kiosk.Register(s.route, kiosk.Options{`,
			replacement: `	// Authenticated routes.
	s.route.GroupWithMiddleware([]routing.Middleware{s.AuthenticatedMiddleware, Verified, s.RedirectDeveloperToKiosk}, func(route *routing.Router) {
		route.GET("/home", s.homeHandler).Can("view:dashboard")
		route.POST("/logout", s.logoutHandler).
			WithoutMiddleware(s.RedirectDeveloperToKiosk)

		route.GET("/passwords/create", s.createPasswordPrompt).
			WithoutMiddleware(Verified)
		route.POST("/passwords", s.createPasswordHandler()).
			WithoutMiddleware(Verified)

		route.GET("/verify-email", s.verifyEmailPrompt).
			WithoutMiddleware(Verified)
		route.POST("/email/verify", s.profileResendVerification()).
			WithoutMiddleware(Verified)

		route.GET("/settings", settingsPage())
		route.PUT("/company", companySettingsUpdate())
		route.PUT("/profile", profileUpdateName())
		route.PUT("/profile/password", profileUpdatePassword())
		route.POST("/profile/email", s.profileUpdateEmail())
		route.PUT("/profile/appearance", profileUpdateAppearance())
	})

	// Signed email-confirmation link — handles both first-time signup
	// verification and an email change (see emailConfirmHandler, distinguished
	// by ?new_email). No auth middleware: the link's signature is the
	// credential, and it must work from a fresh browser tab that never had a
	// session here.
	s.route.GroupWithMiddleware([]routing.Middleware{s.Signed}, func(route *routing.Router) {
		route.GET("/email/confirm", s.emailConfirmHandler)
	})

	kiosk.Register(s.route, kiosk.Options{`,
		},
	}

	patched, err := applyPatches(path, string(data), patches)
	if err != nil {
		return false, err
	}
	if err := os.WriteFile(path, []byte(patched), 0o644); err != nil {
		return false, err
	}
	if err := gofmtFile(path); err != nil {
		return false, err
	}
	return true, nil
}

// settingsIndexTSX is resources/js/Pages/Settings/Index.tsx — a new file.
// General/Users/Profile/Appearance sections only. Add a new section
// (Locations, Tax Rates, ...) by following this same shape: a new Section
// literal, a SECTIONS entry, a prop from settingsPage, and a component.
const settingsIndexTSX = `import { router, usePage } from "@inertiajs/react";
import { type FormEvent, useState } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import AppLayout from "@/layouts/app-layout";
import { useCsrfHeaders } from "@/lib/csrf";
import { applyTheme, type Appearance, type ThemeColors } from "@/lib/theme";
import { type BreadcrumbItem, type SharedProps, type User } from "@/types";

// A lowercase-tagged DTO from companySettingsPayload, not a raw model — see
// app/company-settings.go.
interface CompanySettings {
  uuid: string;
  name: string;
  status: string;
  identifier: string | null;
  plan_name: string | null;
  can_edit: boolean;
  theme_preset: string;
}

// A lowercase-tagged DTO from themePresetSummary, not a raw model — see
// app/theme.go.
interface ThemePresetSummary {
  id: string;
  name: string;
  light: ThemeColors;
  dark: ThemeColors;
}

// A lowercase-tagged DTO from companyMemberRow, not a raw model — see
// app/company-members.go.
interface CompanyMember {
  id: number;
  uuid: string;
  name: string;
  email: string;
  role: string;
}

interface SettingsIndexProps {
  company: CompanySettings;
  members: CompanyMember[];
  themePresets: ThemePresetSummary[];
}

const breadcrumbs: BreadcrumbItem[] = [{ title: "Settings", href: "/settings" }];

type Section = "general" | "users" | "appearance" | "profile";

const SECTIONS: { key: Section; label: string }[] = [
  { key: "general", label: "General" },
  { key: "users", label: "Users" },
  { key: "appearance", label: "Appearance" },
  { key: "profile", label: "Profile" },
];

const selectClass =
  "h-9 w-full rounded-md border border-input bg-transparent px-3 text-sm shadow-xs outline-none focus-visible:border-ring focus-visible:ring-[3px] focus-visible:ring-ring/50 dark:bg-input/30";

function currentSection(url: string): Section {
  const query = url.split("?")[1] ?? "";
  const value = new URLSearchParams(query).get("section");
  return SECTIONS.some((s) => s.key === value) ? (value as Section) : "general";
}

// Settings is a sub-nav over one Inertia page — every section's data is
// loaded up front by settingsPage as unconditional props, so switching
// sections is just a URL-driven visit to this same page (?section=...), not
// local view state.
export default function SettingsIndex({ company, members, themePresets }: SettingsIndexProps) {
  const headers = useCsrfHeaders();
  const page = usePage<SharedProps>();
  const section = currentSection(page.url);
  const path = page.url.split("?")[0];

  function goTo(next: Section) {
    const qs = next === "general" ? "" : ` + "`?section=${next}`" + `;
    router.visit(` + "`${path}${qs}`" + `, { preserveScroll: true, preserveState: true });
  }

  return (
    <AppLayout breadcrumbs={breadcrumbs}>
      <div className="flex flex-1 gap-8 p-6">
        <nav className="flex w-40 shrink-0 flex-col gap-1">
          {SECTIONS.map((s) => (
            <button
              key={s.key}
              type="button"
              onClick={() => goTo(s.key)}
              className={
                "rounded-md px-3 py-2 text-left text-sm font-medium " +
                (section === s.key
                  ? "bg-accent text-accent-foreground"
                  : "text-muted-foreground hover:bg-accent/50")
              }
            >
              {s.label}
            </button>
          ))}
        </nav>

        <div className="min-w-0 flex-1">
          {section === "general" && <GeneralSection company={company} headers={headers} />}
          {section === "users" && <UsersSection members={members} />}
          {section === "appearance" && (
            <AppearanceSection
              company={company}
              themePresets={themePresets}
              appearance={page.props.theme.appearance}
              headers={headers}
            />
          )}
          {section === "profile" && page.props.auth.user && (
            <ProfileSection user={page.props.auth.user} headers={headers} />
          )}
        </div>
      </div>
    </AppLayout>
  );
}

function GeneralSection({
  company,
  headers,
}: {
  company: CompanySettings;
  headers: Record<string, string>;
}) {
  return (
    <div>
      <div>
        <h1 className="text-2xl font-semibold">Company settings</h1>
        <p className="text-muted-foreground text-sm">
          {company.can_edit
            ? "Only the owner can make changes here."
            : "Only the owner can make changes here — you have read access."}
        </p>
      </div>

      <div className="mt-4 max-w-md rounded-md border p-4">
        <dl className="mb-4 space-y-2 text-sm">
          <div className="flex justify-between">
            <dt className="text-muted-foreground">Plan</dt>
            <dd className="font-medium">{company.plan_name ?? "—"}</dd>
          </div>
          <div className="flex justify-between">
            <dt className="text-muted-foreground">Status</dt>
            <dd className="font-medium capitalize">{company.status}</dd>
          </div>
        </dl>

        {company.can_edit ? (
          <SettingsForm headers={headers} company={company} />
        ) : (
          <dl className="space-y-2 text-sm">
            <div>
              <dt className="text-muted-foreground">Name</dt>
              <dd className="font-medium">{company.name}</dd>
            </div>
            <div>
              <dt className="text-muted-foreground">Tax ID</dt>
              <dd className="font-medium">{company.identifier ?? "—"}</dd>
            </div>
          </dl>
        )}
      </div>
    </div>
  );
}

function UsersSection({ members }: { members: CompanyMember[] }) {
  return (
    <div>
      <div>
        <h2 className="text-xl font-semibold">Users</h2>
        <p className="text-muted-foreground text-sm">
          Everyone with access to this company.
        </p>
      </div>

      <div className="mt-4 rounded-md border">
        <table className="w-full text-sm">
          <thead className="text-muted-foreground text-left">
            <tr>
              <th className="px-4 py-2 font-medium">Name</th>
              <th className="font-medium">Email</th>
              <th className="font-medium">Role</th>
            </tr>
          </thead>
          <tbody>
            {members.map((m) => (
              <tr key={m.uuid} className="border-t align-top">
                <td className="px-4 py-2 font-medium">{m.name}</td>
                <td className="text-muted-foreground">{m.email}</td>
                <td>
                  {m.role ? (
                    <span className="bg-muted rounded px-2 py-0.5 text-xs font-medium capitalize">
                      {m.role.replace(/_/g, " ")}
                    </span>
                  ) : (
                    <span className="text-muted-foreground">—</span>
                  )}
                </td>
              </tr>
            ))}
            {members.length === 0 && (
              <tr>
                <td colSpan={3} className="text-muted-foreground py-6 text-center">
                  No users yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}

function SettingsForm({
  headers,
  company,
}: {
  headers: Record<string, string>;
  company: CompanySettings;
}) {
  const [name, setName] = useState(company.name);
  const [identifier, setIdentifier] = useState(company.identifier ?? "");
  const [error, setError] = useState<string | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function submit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    setNotice(null);
    router.put(
      "/company",
      { name, identifier },
      {
        headers,
        preserveScroll: true,
        onSuccess: () => setNotice("Saved."),
        onError: (errors) => {
          // WithRequest (app/company-settings.go) flashes validation
          // failures as the session's real Record<string, string[]> shape —
          // Inertia's own callback type assumes a bare string per field.
          const messages = errors as unknown as Record<string, string[]>;
          setError(messages.name?.[0] ?? "Could not save settings.");
        },
        onFinish: () => setSubmitting(false),
      },
    );
  }

  return (
    <form onSubmit={submit} className="grid gap-3">
      <div className="grid gap-1">
        <Label htmlFor="settings-name">Name</Label>
        <Input
          id="settings-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
      </div>
      <div className="grid gap-1">
        <Label htmlFor="settings-identifier">Tax ID</Label>
        <Input
          id="settings-identifier"
          value={identifier}
          onChange={(e) => setIdentifier(e.target.value)}
        />
      </div>
      {error && <p className="text-destructive text-sm">{error}</p>}
      {notice && <p className="text-sm text-emerald-600">{notice}</p>}
      <Button type="submit" disabled={submitting} className="justify-self-start">
        {submitting ? "Saving…" : "Save"}
      </Button>
    </form>
  );
}

// AppearanceSection — the company's shared brand color preset (owner-only,
// same can_edit gate as General). A single-select swatch grid that saves
// immediately on click, reusing the existing PUT /company endpoint —
// theme_preset is just one more field on UpdateCompanySettingsForm, so
// every existing field has to ride along in the same request. Clicking a
// swatch also calls applyTheme (lib/theme.ts) immediately, before the save
// round-trip completes, for instant feedback — router.on("navigate")
// (app.tsx) reconciles with the server's own resolved value once the
// redirect lands, including reverting the pick if the save actually failed.
function AppearanceSection({
  company,
  themePresets,
  appearance,
  headers,
}: {
  company: CompanySettings;
  themePresets: ThemePresetSummary[];
  appearance: Appearance;
  headers: Record<string, string>;
}) {
  const [selected, setSelected] = useState(company.theme_preset);
  const [submitting, setSubmitting] = useState(false);

  function choose(preset: ThemePresetSummary) {
    if (!company.can_edit || submitting) return;
    const previous = selected;
    setSelected(preset.id);
    applyTheme({
      preset: preset.id,
      light: preset.light,
      dark: preset.dark,
      appearance,
    });
    setSubmitting(true);
    router.put(
      "/company",
      {
        name: company.name,
        identifier: company.identifier ?? "",
        theme_preset: preset.id,
      },
      {
        headers,
        preserveScroll: true,
        preserveState: true,
        onError: () => setSelected(previous),
        onFinish: () => setSubmitting(false),
      },
    );
  }

  return (
    <div>
      <div>
        <h2 className="text-xl font-semibold">Appearance</h2>
        <p className="text-muted-foreground text-sm">
          {company.can_edit
            ? "A color preset for your whole team — light/dark/system stays a personal choice (see Profile)."
            : "Only the owner can change this."}
        </p>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3">
        {themePresets.map((preset) => (
          <button
            key={preset.id}
            type="button"
            disabled={!company.can_edit || submitting}
            onClick={() => choose(preset)}
            className={
              "flex flex-col items-start gap-2 rounded-md border p-3 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-60 " +
              (selected === preset.id
                ? "border-primary ring-primary ring-1"
                : "hover:bg-accent/50")
            }
          >
            <div className="flex gap-1">
              <span
                className="size-5 rounded-full border"
                style={{ background: preset.light.Primary }}
              />
              <span
                className="size-5 rounded-full border"
                style={{ background: preset.dark.Primary }}
              />
            </div>
            <span className="text-sm font-medium">{preset.name}</span>
          </button>
        ))}
      </div>
    </div>
  );
}

// ProfileSection — the logged-in user's own account settings (as opposed to
// every other section here, which is company-wide). Reads "user" from
// SharedProps' auth.user rather than a dedicated prop from settingsPage,
// since every page already gets that for free. Four independent forms;
// each posts to its own backend route (app/profile.go) and redirects back
// to ?section=profile.
function ProfileSection({
  user,
  headers,
}: {
  user: User;
  headers: Record<string, string>;
}) {
  return (
    <div className="flex flex-col gap-8">
      <div>
        <h1 className="text-2xl font-semibold">Profile</h1>
        <p className="text-muted-foreground text-sm">Your own account settings.</p>
      </div>

      <div className="max-w-md rounded-md border p-4">
        <h2 className="mb-3 text-sm font-medium">Name</h2>
        <NameForm user={user} headers={headers} />
      </div>

      <div className="max-w-md rounded-md border p-4">
        <h2 className="mb-3 text-sm font-medium">Password</h2>
        <PasswordForm headers={headers} />
      </div>

      <div className="max-w-md rounded-md border p-4">
        <h2 className="mb-3 text-sm font-medium">Email</h2>
        <EmailForm user={user} headers={headers} />
      </div>

      <div className="max-w-md rounded-md border p-4">
        <h2 className="mb-3 text-sm font-medium">Appearance</h2>
        <AppearanceForm user={user} headers={headers} />
      </div>
    </div>
  );
}

function NameForm({ user, headers }: { user: User; headers: Record<string, string> }) {
  const [name, setName] = useState(user.name);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function submit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    router.put(
      "/profile",
      { name },
      {
        headers,
        preserveScroll: true,
        onError: (errors) => {
          const messages = errors as unknown as Record<string, string[]>;
          setError(messages.name?.[0] ?? "Could not save name.");
        },
        onFinish: () => setSubmitting(false),
      },
    );
  }

  return (
    <form onSubmit={submit} className="grid gap-3">
      <div className="grid gap-1">
        <Label htmlFor="profile-name">Name</Label>
        <Input
          id="profile-name"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
        />
      </div>
      {error && <p className="text-destructive text-sm">{error}</p>}
      <Button type="submit" disabled={submitting} className="justify-self-start">
        {submitting ? "Saving…" : "Save"}
      </Button>
    </form>
  );
}

function PasswordForm({ headers }: { headers: Record<string, string> }) {
  const [currentPassword, setCurrentPassword] = useState("");
  const [password, setPassword] = useState("");
  const [confirm, setConfirm] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function submit(e: FormEvent) {
    e.preventDefault();
    if (password !== confirm) {
      setError("Passwords do not match.");
      return;
    }
    setSubmitting(true);
    setError(null);
    router.put(
      "/profile/password",
      { current_password: currentPassword, password },
      {
        headers,
        preserveScroll: true,
        onSuccess: () => {
          setCurrentPassword("");
          setPassword("");
          setConfirm("");
        },
        onError: (errors) => {
          const messages = errors as unknown as Record<string, string[]>;
          setError(
            messages.current_password?.[0] ??
              messages.password?.[0] ??
              "Could not change password.",
          );
        },
        onFinish: () => setSubmitting(false),
      },
    );
  }

  return (
    <form onSubmit={submit} className="grid gap-3">
      <div className="grid gap-1">
        <Label htmlFor="profile-current-password">Current password</Label>
        <Input
          id="profile-current-password"
          type="password"
          value={currentPassword}
          onChange={(e) => setCurrentPassword(e.target.value)}
          autoComplete="current-password"
          required
        />
      </div>
      <div className="grid gap-1">
        <Label htmlFor="profile-new-password">New password</Label>
        <Input
          id="profile-new-password"
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          autoComplete="new-password"
          minLength={8}
          required
        />
      </div>
      <div className="grid gap-1">
        <Label htmlFor="profile-confirm-password">Confirm new password</Label>
        <Input
          id="profile-confirm-password"
          type="password"
          value={confirm}
          onChange={(e) => setConfirm(e.target.value)}
          autoComplete="new-password"
          minLength={8}
          required
        />
      </div>
      {error && <p className="text-destructive text-sm">{error}</p>}
      <Button type="submit" disabled={submitting} className="justify-self-start">
        {submitting ? "Saving…" : "Change password"}
      </Button>
    </form>
  );
}

function EmailForm({ user, headers }: { user: User; headers: Record<string, string> }) {
  const [email, setEmail] = useState("");
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  function submit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    setError(null);
    router.post(
      "/profile/email",
      { email },
      {
        headers,
        preserveScroll: true,
        onSuccess: () => setEmail(""),
        onError: (errors) => {
          const messages = errors as unknown as Record<string, string[]>;
          setError(messages.email?.[0] ?? "Could not start email change.");
        },
        onFinish: () => setSubmitting(false),
      },
    );
  }

  return (
    <div className="grid gap-3">
      <dl className="text-sm">
        <div className="flex justify-between">
          <dt className="text-muted-foreground">Current email</dt>
          <dd className="font-medium">{user.email}</dd>
        </div>
        <div className="flex justify-between">
          <dt className="text-muted-foreground">Verified</dt>
          <dd className="font-medium">{user.email_verified_at ? "Yes" : "No"}</dd>
        </div>
      </dl>
      <form onSubmit={submit} className="grid gap-3">
        <div className="grid gap-1">
          <Label htmlFor="profile-email">New email</Label>
          <Input
            id="profile-email"
            type="email"
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            placeholder={user.email}
            required
          />
          <p className="text-muted-foreground text-xs">
            We&apos;ll email a confirmation link to the new address before it
            takes effect.
          </p>
        </div>
        {error && <p className="text-destructive text-sm">{error}</p>}
        <Button type="submit" disabled={submitting} className="justify-self-start">
          {submitting ? "Sending…" : "Send confirmation"}
        </Button>
      </form>
    </div>
  );
}

// AppearanceForm — light/dark/system, personal to this user, independent of
// whichever company's theme_preset is active (see AppearanceSection's own
// doc comment for the split). Reads the current preset's colors from
// page.props.theme so switching this selector previews instantly, the same
// applyTheme + router.on("navigate") reconciliation pattern
// AppearanceSection uses.
function AppearanceForm({
  user,
  headers,
}: {
  user: User;
  headers: Record<string, string>;
}) {
  const theme = usePage<SharedProps>().props.theme;
  const [appearance, setAppearance] = useState<Appearance>(user.appearance);
  const [submitting, setSubmitting] = useState(false);

  function submit(e: FormEvent) {
    e.preventDefault();
    setSubmitting(true);
    router.put(
      "/profile/appearance",
      { appearance },
      {
        headers,
        preserveScroll: true,
        onError: () => setAppearance(user.appearance),
        onFinish: () => setSubmitting(false),
      },
    );
  }

  function change(next: Appearance) {
    setAppearance(next);
    applyTheme({ ...theme, appearance: next });
  }

  return (
    <form onSubmit={submit} className="grid gap-3">
      <div className="grid gap-1">
        <Label htmlFor="profile-appearance">Appearance</Label>
        <select
          id="profile-appearance"
          value={appearance}
          onChange={(e) => change(e.target.value as Appearance)}
          className={selectClass}
        >
          <option value="system">System</option>
          <option value="light">Light</option>
          <option value="dark">Dark</option>
        </select>
      </div>
      <Button type="submit" disabled={submitting} className="justify-self-start">
        {submitting ? "Saving…" : "Save"}
      </Button>
    </form>
  );
}
`

func wireKioskSettingsPage(root string) (applied bool, err error) {
	path := filepath.Join(root, "resources", "js", "Pages", "Settings", "Index.tsx")
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(settingsIndexTSX), 0o644)
}
