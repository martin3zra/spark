package app

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/foundation"
)

// AuthUser is the application's user identity. It owns the users-table schema and
// satisfies foundation.Authenticatable / foundation.MustVerifyPassword.
type AuthUser struct {
	Id                 int        `json:"id"`
	UUID               string     `json:"uuid"`
	Name               string     `json:"name"`
	Email              string     `json:"email"`
	Password           string     `json:"-"`
	Role               string     `json:"role"`
	EmailVerifiedAt    *time.Time `json:"email_verified_at"`
	MustChangePassword bool       `json:"must_change_password"`
	foundation.Timestamps
}

func (u *AuthUser) IsEmpty() bool                 { return *u == (AuthUser{}) }
func (u *AuthUser) GetAuthIdentifier() int        { return u.Id }
func (u *AuthUser) GetAuthIdentifierName() string { return "email" }
func (u *AuthUser) GetRole() string               { return u.Role }
func (u *AuthUser) SetRole(role string)           { u.Role = role }
func (u *AuthUser) GetAuthPassword() string       { return u.Password }
func (u *AuthUser) HasNotChangedPassword() bool   { return u.MustChangePassword }

func (u *AuthUser) MarkPasswordAsChanged(db *sql.DB, password string) error {
	_, err := db.Exec(
		"UPDATE users SET password = $2, must_change_password = FALSE, updated_at = NOW() WHERE id = $1",
		u.Id, foundation.NewHashable().Make(password),
	)
	return err
}

// AuthUserFromContext rebuilds the authenticated identity from the session value
// stored in the request context.
func AuthUserFromContext(ctx context.Context) *AuthUser {
	var user AuthUser
	if v := ctx.Value(auth.ContextUserID{}); v != nil {
		raw, _ := json.Marshal(v.(map[string]any))
		_ = json.Unmarshal(raw, &user)
	}
	return &user
}

// UserFromFoundationUser narrows a framework identity back to the concrete app user.
func UserFromFoundationUser(u foundation.Authenticatable) *AuthUser {
	au, ok := u.(*AuthUser)
	if !ok || au == nil {
		return &AuthUser{}
	}
	return au
}

const selectUserColumns = `id, uuid, name, email, password, role, email_verified_at, must_change_password, created_at, updated_at, deleted_at`

func scanUser(row *sql.Row) (*AuthUser, error) {
	u := new(AuthUser)
	err := row.Scan(
		&u.Id, &u.UUID, &u.Name, &u.Email, &u.Password, &u.Role,
		&u.EmailVerifiedAt, &u.MustChangePassword,
		&u.CreatedAt, &u.UpdatedAt, &u.DeletedAt,
	)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func init() {
	auth.SetCredentialResolver(func(db *sql.DB, column string, value any) (foundation.Authenticatable, error) {
		q := fmt.Sprintf("SELECT %s FROM users WHERE %s = $1", selectUserColumns, column)
		return scanUser(db.QueryRow(q, value))
	})

	auth.SetPasswordResolver(func(db *sql.DB, userID int) (string, error) {
		var password string
		err := db.QueryRow("SELECT password FROM users WHERE id = $1", userID).Scan(&password)
		return password, err
	})

	auth.SetUserDecoder(func(ctx context.Context) foundation.Authenticatable {
		return AuthUserFromContext(ctx)
	})
}
