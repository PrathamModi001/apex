package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"apex/api-gateway/internal/domain"
)

// UserRepo implements app.UserRepository using pgx v5.
type UserRepo struct {
	pool *pgxpool.Pool
}

// NewUserRepo creates a UserRepo backed by the given pool.
func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// FindOrCreate retrieves a user by email or inserts one with viewer role.
func (r *UserRepo) FindOrCreate(ctx context.Context, email, name string) (domain.User, error) {
	var u domain.User
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, name, role, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	if err == nil {
		return u, nil
	}

	// Not found — insert.
	u = domain.User{
		ID:        uuid.New().String(),
		Email:     email,
		Name:      name,
		Role:      domain.RoleViewer,
		CreatedAt: time.Now().UTC(),
	}
	_, err = r.pool.Exec(ctx,
		`INSERT INTO users (id, email, name, role, created_at) VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (email) DO NOTHING`,
		u.ID, u.Email, u.Name, string(u.Role), u.CreatedAt,
	)
	if err != nil {
		return domain.User{}, err
	}
	// Re-read in case of concurrent insert.
	_ = r.pool.QueryRow(ctx,
		`SELECT id, email, name, role, created_at FROM users WHERE email = $1`, email,
	).Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt)
	return u, nil
}

// UpdateRole updates the role of the user with the given ID.
func (r *UserRepo) UpdateRole(ctx context.Context, userID string, role domain.Role) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE users SET role = $1 WHERE id = $2`, string(role), userID)
	return err
}

// List returns all users ordered by creation date.
func (r *UserRepo) List(ctx context.Context) ([]domain.User, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT id, email, name, role, created_at FROM users ORDER BY created_at ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.CreatedAt); err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}
