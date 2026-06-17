package app

import "github.com/martin3zra/forge/support"

type LoginFormRequest struct {
	support.FormRequest
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (f *LoginFormRequest) Rules() map[string]any {
	return map[string]any{
		"email":    "required",
		"password": "required",
	}
}

type CreatePasswordForm struct {
	support.FormRequest
	Password string `json:"password"`
}

func (f *CreatePasswordForm) Rules() map[string]any {
	return map[string]any{
		"password": "required|min:8",
	}
}
