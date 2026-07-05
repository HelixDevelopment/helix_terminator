package model

import (
	"time"
)

// User represents a platform user
type User struct {
	ID            string     `json:"id" db:"id"`
	Email         string     `json:"email" db:"email"`
	DisplayName   string     `json:"displayName" db:"display_name"`
	AvatarURL     string     `json:"avatarUrl,omitempty" db:"avatar_url"`
	Role          string     `json:"role" db:"role"`
	Permissions   []string   `json:"permissions,omitempty" db:"permissions"`
	OrgID         *string    `json:"orgId,omitempty" db:"org_id"`
	EmailVerified bool       `json:"emailVerified" db:"email_verified"`
	LastLoginAt   *time.Time `json:"lastLoginAt,omitempty" db:"last_login_at"`
	CreatedAt     time.Time  `json:"createdAt" db:"created_at"`
	UpdatedAt     time.Time  `json:"updatedAt" db:"updated_at"`
	DeletedAt     *time.Time `json:"deletedAt,omitempty" db:"deleted_at"`
}

// UserProfile extends User with additional profile fields
type UserProfile struct {
	User
	Bio          string            `json:"bio,omitempty" db:"bio"`
	Timezone     string            `json:"timezone,omitempty" db:"timezone"`
	Locale       string            `json:"locale,omitempty" db:"locale"`
	Preferences  map[string]string `json:"preferences,omitempty" db:"preferences"`
	SSHPublicKey string            `json:"sshPublicKey,omitempty" db:"ssh_public_key"`
	GitHubID     string            `json:"githubId,omitempty" db:"github_id"`
	GitLabID     string            `json:"gitlabId,omitempty" db:"gitlab_id"`
}

// CreateUserRequest represents a request to create a user
type CreateUserRequest struct {
	Email       string   `json:"email" binding:"required,email"`
	DisplayName string   `json:"displayName" binding:"required,min=1,max=255"`
	Role        string   `json:"role" binding:"required,oneof=user admin superadmin"`
	OrgID       *string  `json:"orgId,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

// UpdateUserRequest represents a request to update a user
type UpdateUserRequest struct {
	DisplayName *string   `json:"displayName,omitempty"`
	AvatarURL   *string   `json:"avatarUrl,omitempty"`
	Role        *string   `json:"role,omitempty" binding:"omitempty,oneof=user admin superadmin"`
	Permissions []string  `json:"permissions,omitempty"`
	OrgID       *string   `json:"orgId,omitempty"`
	Bio         *string   `json:"bio,omitempty"`
	Timezone    *string   `json:"timezone,omitempty"`
	Locale      *string   `json:"locale,omitempty"`
}

// ListUsersRequest represents a request to list users
type ListUsersRequest struct {
	OrgID  string `form:"orgId"`
	Role   string `form:"role"`
	Search string `form:"search"`
	Limit  int    `form:"limit,default=20"`
	Offset int    `form:"offset,default=0"`
}

// UserResponse is the API response for a user
type UserResponse struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	DisplayName   string     `json:"displayName"`
	AvatarURL     string     `json:"avatarUrl,omitempty"`
	Role          string     `json:"role"`
	Permissions   []string   `json:"permissions,omitempty"`
	OrgID         *string    `json:"orgId,omitempty"`
	EmailVerified bool       `json:"emailVerified"`
	LastLoginAt   *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}

// UserProfileResponse is the API response for a user profile
type UserProfileResponse struct {
	UserResponse
	Bio          string            `json:"bio,omitempty"`
	Timezone     string            `json:"timezone,omitempty"`
	Locale       string            `json:"locale,omitempty"`
	Preferences  map[string]string `json:"preferences,omitempty"`
	SSHPublicKey string            `json:"sshPublicKey,omitempty"`
	GitHubID     string            `json:"githubId,omitempty"`
	GitLabID     string            `json:"gitlabId,omitempty"`
}

// ListUsersResponse is the API response for listing users
type ListUsersResponse struct {
	Users  []*UserResponse `json:"users"`
	Total  int             `json:"total"`
	Limit  int             `json:"limit"`
	Offset int             `json:"offset"`
}

// UpdateProfileRequest represents a request to update user profile
type UpdateProfileRequest struct {
	DisplayName  *string           `json:"displayName,omitempty"`
	Bio          *string           `json:"bio,omitempty"`
	Timezone     *string           `json:"timezone,omitempty"`
	Locale       *string           `json:"locale,omitempty"`
	Preferences  map[string]string `json:"preferences,omitempty"`
	SSHPublicKey *string           `json:"sshPublicKey,omitempty"`
	AvatarURL    *string           `json:"avatarUrl,omitempty"`
}

// PreferencesResponse represents user preferences
type PreferencesResponse struct {
	Preferences map[string]string `json:"preferences"`
}
