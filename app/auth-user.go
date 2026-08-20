package app

import (
	"context"
	"encoding/json"
	"time"

	"github.com/martin3zra/forge/auth"
	"github.com/martin3zra/forge/foundation"
	"github.com/martin3zra/playsql"
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

func (u *AuthUser) MarkPasswordAsChanged(db *playsql.DB, password string) error {
	_, err := db.Model(&userModel{}).WhereEq("id", u.Id).Update(context.Background(), map[string]any{
		"password":              foundation.NewHashable().Make(password),
		"must_change_password": false,
	})
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

// userModel is the on-disk shape of a row in the users table. Kept separate
// from AuthUser (the domain/JSON type) so playsql's db-tag-driven metadata
// never has to interpret AuthUser's json tags — Password uses json:"-", which
// playsql would otherwise read as a literal "-" column name.
type userModel struct {
	ID                 int        `db:"id"`
	UUID               string     `db:"uuid"`
	Name               string     `db:"name"`
	Email              string     `db:"email"`
	Password           string     `db:"password"`
	Role               string     `db:"role"`
	EmailVerifiedAt    *time.Time `db:"email_verified_at"`
	MustChangePassword bool       `db:"must_change_password"`
	CreatedAt          *time.Time `db:"created_at"`
	UpdatedAt          *time.Time `db:"updated_at"`
	DeletedAt          *time.Time `db:"deleted_at"`
}

func (userModel) TableName() string { return "users" }

func (m userModel) toAuthUser() *AuthUser {
	return &AuthUser{
		Id:                 m.ID,
		UUID:               m.UUID,
		Name:               m.Name,
		Email:              m.Email,
		Password:           m.Password,
		Role:               m.Role,
		EmailVerifiedAt:    m.EmailVerifiedAt,
		MustChangePassword: m.MustChangePassword,
		Timestamps: foundation.Timestamps{
			CreatedAt: m.CreatedAt,
			UpdatedAt: m.UpdatedAt,
			DeletedAt: m.DeletedAt,
		},
	}
}

func init() {
	auth.SetCredentialResolver(func(db *playsql.DB, column string, value any) (foundation.Authenticatable, error) {
		var row userModel
		if err := db.Model(&userModel{}).WhereEq(column, value).First(context.Background(), &row); err != nil {
			return nil, err
		}
		return row.toAuthUser(), nil
	})

	auth.SetPasswordResolver(func(db *playsql.DB, userID int) (string, error) {
		var row userModel
		if err := db.Model(&userModel{}).WhereEq("id", userID).First(context.Background(), &row); err != nil {
			return "", err
		}
		return row.Password, nil
	})

	auth.SetUserDecoder(func(ctx context.Context) foundation.Authenticatable {
		return AuthUserFromContext(ctx)
	})
}
