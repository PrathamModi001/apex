package domain

import "time"

// Role represents RBAC role for a user.
type Role string

const (
	RoleAdmin    Role = "admin"
	RoleReviewer Role = "reviewer"
	RoleViewer   Role = "viewer"
)

// User is the core user entity.
type User struct {
	ID        string
	Email     string
	Name      string
	Role      Role
	CreatedAt time.Time
}
