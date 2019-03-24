package gql

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/tengen-io/server/models"
	"golang.org/x/crypto/bcrypt"
	"strconv"
	"time"
)

type AuthRepository struct {
	db         *sqlx.DB
	signingKey []byte
	lifetime   time.Duration
}

func NewAuthRepository(db *sqlx.DB, signingKey []byte, lifetime time.Duration) *AuthRepository {
	return &AuthRepository{
		db:         db,
		signingKey: signingKey,
		lifetime:   lifetime,
	}
}

func (p *AuthRepository) ValidateJWT(tokenString string) (*jwt.Token, error) {
	token, err := jwt.ParseWithClaims(tokenString, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("invalid signing method")
		}

		return p.signingKey, nil
	})

	return token, err
}

func (p *AuthRepository) SignJWT(identity models.Identity) (string, error) {
	// TODO(eac): reintroduce custom claims for ID
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Id:        identity.Id,
		NotBefore: time.Now().Unix(),
		ExpiresAt: time.Now().Add(p.lifetime * time.Second).Unix(),
		Issuer:    "tengen.io",
	})

	ss, err := token.SignedString(p.signingKey)
	if err != nil {
		return "", err
	}

	return ss, nil
}

// TODO(eac): Figure out how to use dbx structs for nested structures
func (p *AuthRepository) CheckPasswordByEmail(email, password string) (*models.Identity, error) {
	var passwordHash string
	err := p.db.Get(&passwordHash, "SELECT password_hash FROM identities WHERE email = $1", email)
	if err != nil {
		return nil, err
	}

	passwordBytes := []byte(password)
	hashBytes := []byte(passwordHash)

	err = bcrypt.CompareHashAndPassword(hashBytes, passwordBytes)
	if err != nil {
		return nil, err
	}

	var rv models.Identity
	row := p.db.QueryRowx("SELECT i.id, i.email, u.id, u.name FROM identities i, users u WHERE i.id = u.identity_id AND email = $1", email)
	err = row.Scan(&rv.Id, &rv.Email, &rv.User.Id, &rv.User.Name)

	if err != nil {
		return nil, err
	}

	return &rv, nil
}

type IdentityRepository struct {
	db         *sqlx.DB
	BcryptCost int
}

func NewIdentityRepository(db *sqlx.DB, bcryptCost int) *IdentityRepository {
	return &IdentityRepository{
		db,
		bcryptCost,
	}
}

func (p *IdentityRepository) CreateIdentity(email string, password string, name string) (*models.Identity, error) {
	// TODO(eac): re-add validation
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), p.BcryptCost)
	tx, err := p.db.Beginx()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var rv models.Identity
	ts := pq.FormatTimestamp(time.Now().UTC())

	// TODO(eac): do a precondition check for duplicate users to save autoincrement IDs
	identity := tx.QueryRowx("INSERT INTO identities (email, password_hash, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id, email", email, passwordHash, ts, ts)
	err = identity.Scan(&rv.Id, &rv.Email)
	if err != nil {
		return nil, err
	}

	user := tx.QueryRowx("INSERT INTO users (identity_id, name, created_at, updated_at) VALUES ($1, $2, $3, $4) RETURNING id, name", rv.Id, name, ts, ts)
	err = user.Scan(&rv.User.Id, &rv.User.Name)
	if err != nil {
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		return nil, err
	}

	return &rv, nil
}

// TODO(eac): switch to sqlx.get
func (p *IdentityRepository) GetIdentityById(id int32) (*models.Identity, error) {
	var identity models.Identity
	row := p.db.QueryRowx("SELECT i.id, i.email, u.id, u.name FROM identities i, users u WHERE i.id = u.identity_id AND i.id = $1", id)
	err := row.Scan(&identity.Id, &identity.Email, &identity.User.Id, &identity.User.Name)
	if err != nil {
		return nil, err
	}
	return &identity, nil
}

type UserRepository struct {
	db *sqlx.DB
}

func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{
		db,
	}
}

// TODO(eac): switch to sqlx get
func (p *UserRepository) GetUserById(id string) (*models.User, error) {
	idInt, err := strconv.Atoi(id)
	if err != nil {
		return nil, err
	}

	var rvId int
	var user models.User
	row := p.db.QueryRow("SELECT id, name FROM users WHERE id = $1", idInt)
	err = row.Scan(&rvId, &user.Name)
	user.Id = strconv.Itoa(rvId)
	if err != nil {
		return nil, err
	}

	return &user, nil
}
