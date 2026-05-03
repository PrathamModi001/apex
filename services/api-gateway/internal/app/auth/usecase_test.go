package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"apex/api-gateway/internal/app/auth"
	"apex/api-gateway/internal/domain"
)

// --- fake repo ---

type fakeUserRepo struct {
	users map[string]domain.User
	calls int
}

func newFakeRepo() *fakeUserRepo {
	return &fakeUserRepo{users: make(map[string]domain.User)}
}

func (r *fakeUserRepo) FindOrCreate(_ context.Context, email, name string) (domain.User, error) {
	r.calls++
	if u, ok := r.users[email]; ok {
		return u, nil
	}
	u := domain.User{
		ID:        "uid-" + email,
		Email:     email,
		Name:      name,
		Role:      domain.RoleViewer,
		CreatedAt: time.Now(),
	}
	r.users[email] = u
	return u, nil
}

func (r *fakeUserRepo) UpdateRole(_ context.Context, userID string, role domain.Role) error {
	for k, u := range r.users {
		if u.ID == userID {
			u.Role = role
			r.users[k] = u
			return nil
		}
	}
	return errors.New("user not found")
}

func (r *fakeUserRepo) List(_ context.Context) ([]domain.User, error) {
	users := make([]domain.User, 0, len(r.users))
	for _, u := range r.users {
		users = append(users, u)
	}
	return users, nil
}

// --- helpers ---

func mustGenerateKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return priv
}

// --- tests ---

func TestIssueToken_ValidClaims(t *testing.T) {
	priv := mustGenerateKey(t)
	uc := auth.NewWithKeys(newFakeRepo(), priv, &priv.PublicKey)

	user := domain.User{ID: "u1", Email: "test@example.com", Role: domain.RoleAdmin}
	tok, err := uc.IssueToken(user)
	if err != nil {
		t.Fatalf("IssueToken: %v", err)
	}
	if tok == "" {
		t.Fatal("expected non-empty token")
	}

	claims, err := uc.VerifyToken(tok)
	if err != nil {
		t.Fatalf("VerifyToken: %v", err)
	}
	if claims.UserID != "u1" {
		t.Errorf("want sub=u1, got %q", claims.UserID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("want email=test@example.com, got %q", claims.Email)
	}
	if claims.Role != string(domain.RoleAdmin) {
		t.Errorf("want role=admin, got %q", claims.Role)
	}
}

func TestVerifyToken_Valid(t *testing.T) {
	priv := mustGenerateKey(t)
	uc := auth.NewWithKeys(newFakeRepo(), priv, &priv.PublicKey)

	user := domain.User{ID: "u2", Email: "a@b.com", Role: domain.RoleReviewer}
	tok, _ := uc.IssueToken(user)

	claims, err := uc.VerifyToken(tok)
	if err != nil {
		t.Fatalf("VerifyToken returned error: %v", err)
	}
	if claims.UserID != "u2" {
		t.Errorf("unexpected UserID: %q", claims.UserID)
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	priv := mustGenerateKey(t)

	// Manually craft an already-expired token
	claims := auth.Claims{
		UserID: "u3",
		Email:  "x@y.com",
		Role:   "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign expired token: %v", err)
	}

	uc := auth.NewWithKeys(newFakeRepo(), priv, &priv.PublicKey)
	_, err = uc.VerifyToken(tok)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestVerifyToken_Tampered(t *testing.T) {
	priv := mustGenerateKey(t)
	uc := auth.NewWithKeys(newFakeRepo(), priv, &priv.PublicKey)

	user := domain.User{ID: "u4", Email: "t@t.com", Role: domain.RoleViewer}
	tok, _ := uc.IssueToken(user)

	// Tamper: corrupt a char in the middle of the signature segment
	// (last char may only affect base64 padding bits and round-trip unchanged)
	mid := len(tok) / 2
	flip := "X"
	if tok[mid] == 'X' {
		flip = "Y"
	}
	tampered := tok[:mid] + flip + tok[mid+1:]

	_, err := uc.VerifyToken(tampered)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestVerifyToken_WrongAlgorithm(t *testing.T) {
	priv := mustGenerateKey(t)
	uc := auth.NewWithKeys(newFakeRepo(), priv, &priv.PublicKey)

	// Sign with HMAC (wrong algorithm for RSA use case)
	claims := auth.Claims{
		UserID: "u5",
		Email:  "hs@256.com",
		Role:   "viewer",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tok, err := token.SignedString([]byte("somesecret"))
	if err != nil {
		t.Fatalf("sign HS256 token: %v", err)
	}

	_, err = uc.VerifyToken(tok)
	if err == nil {
		t.Fatal("expected error for wrong algorithm, got nil")
	}
}

func TestFindOrCreate_NewUser(t *testing.T) {
	repo := newFakeRepo()
	uc := auth.NewWithHMAC(repo, []byte("secret"))

	user, err := uc.FindOrCreateUser(context.Background(), "new@test.com", "New User")
	if err != nil {
		t.Fatalf("FindOrCreateUser: %v", err)
	}
	if user.Email != "new@test.com" {
		t.Errorf("want email=new@test.com, got %q", user.Email)
	}
	if user.Role != domain.RoleViewer {
		t.Errorf("new user should have viewer role, got %q", user.Role)
	}
	if repo.calls != 1 {
		t.Errorf("expected 1 repo call, got %d", repo.calls)
	}
}

func TestFindOrCreate_ExistingUser(t *testing.T) {
	repo := newFakeRepo()
	existing := domain.User{
		ID:    "existing-id",
		Email: "exist@test.com",
		Name:  "Existing",
		Role:  domain.RoleAdmin,
	}
	repo.users["exist@test.com"] = existing

	uc := auth.NewWithHMAC(repo, []byte("secret"))

	user, err := uc.FindOrCreateUser(context.Background(), "exist@test.com", "Existing")
	if err != nil {
		t.Fatalf("FindOrCreateUser: %v", err)
	}
	if user.ID != "existing-id" {
		t.Errorf("want ID=existing-id, got %q", user.ID)
	}
	if user.Role != domain.RoleAdmin {
		t.Errorf("existing user role should be admin, got %q", user.Role)
	}
}
