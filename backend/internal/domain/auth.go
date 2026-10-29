package domain

type SessionUser struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
	Source   string `json:"source,omitempty"`
}

type AuthLoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Mode     string `json:"mode"`
}

type AuthLoginResponse struct {
	Token string      `json:"token"`
	User  SessionUser `json:"user"`
}

type ChangePasswordBody struct {
	OldPassword     string `json:"old_password"`
	NewPassword     string `json:"new_password"`
	ConfirmPassword string `json:"confirm_password"`
}
