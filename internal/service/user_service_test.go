package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskservice/internal/auth"
	"taskservice/internal/repository"
)

func TestUserService_RegisterAndLogin(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	users := repository.NewUserRepository(db)
	jwtManager := auth.NewJWTManager("secret", 60)
	svc := NewUserService(users, jwtManager)

	mock.ExpectExec("INSERT INTO users").
		WillReturnResult(sqlmock.NewResult(1, 1))

	u, err := svc.Register(context.Background(), "a@b.com", "password123", "Alice")
	require.NoError(t, err)
	assert.Equal(t, int64(1), u.ID)

	hash, _ := auth.HashPassword("password123")
	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at"}).
		AddRow(1, "a@b.com", hash, "Alice", time.Now())

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("a@b.com").
		WillReturnRows(rows)

	token, err := svc.Login(context.Background(), "a@b.com", "password123")
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestUserService_Login_WrongPassword(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	users := repository.NewUserRepository(db)
	jwtManager := auth.NewJWTManager("secret", 60)
	svc := NewUserService(users, jwtManager)

	hash, _ := auth.HashPassword("correct")
	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at"}).
		AddRow(1, "a@b.com", hash, "Alice", time.Now())

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("a@b.com").
		WillReturnRows(rows)

	_, err = svc.Login(context.Background(), "a@b.com", "wrong")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserService_Login_UserNotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	users := repository.NewUserRepository(db)
	jwtManager := auth.NewJWTManager("secret", 60)
	svc := NewUserService(users, jwtManager)

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("missing@b.com").
		WillReturnError(sql.ErrNoRows)

	_, err = svc.Login(context.Background(), "missing@b.com", "pw")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUserService_Register_EmailTaken(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	users := repository.NewUserRepository(db)
	jwtManager := auth.NewJWTManager("secret", 60)
	svc := NewUserService(users, jwtManager)

	mock.ExpectExec("INSERT INTO users").
		WillReturnError(&dupErr{})

	_, err = svc.Register(context.Background(), "a@b.com", "password123", "Alice")
	assert.ErrorIs(t, err, ErrEmailTaken)
}

type dupErr struct{}

func (e *dupErr) Error() string { return "Error 1062: Duplicate entry 'a@b.com' for key 'email'" }
