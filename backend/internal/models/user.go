package models

import "time"

type User struct {
	ID           string     `db:"id" json:"id"`
	Email        string     `db:"email" json:"email"`
	Username     string     `db:"username" json:"username"`
	PasswordHash *string    `db:"password_hash" json:"-"`
	DisplayName  *string    `db:"display_name" json:"display_name"`
	AvatarURL    *string    `db:"avatar_url" json:"avatar_url"`
	Role         string     `db:"role" json:"role"`
	TOTPSecret   *string    `db:"totp_secret" json:"-"`
	TOTPEnabled  int        `db:"totp_enabled" json:"totp_enabled"`
	IsActive     int        `db:"is_active" json:"is_active"`
	CreatedAt    time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at" json:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

type UserCreateInput struct {
	Email       string `json:"email" validate:"required,email"`
	Username    string `json:"username" validate:"required,min=3,max=50"`
	Password    string `json:"password" validate:"required,min=8"`
	DisplayName string `json:"display_name"`
	Role        string `json:"role" validate:"omitempty,oneof=owner admin developer viewer"`
}

type UserUpdateInput struct {
	Email       *string `json:"email" validate:"omitempty,email"`
	Username    *string `json:"username" validate:"omitempty,min=3,max=50"`
	DisplayName *string `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	Role        *string `json:"role" validate:"omitempty,oneof=owner admin developer viewer"`
}
