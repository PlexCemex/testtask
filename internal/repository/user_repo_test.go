package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskservice/internal/models"
)

func TestUserRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("INSERT INTO users").
		WithArgs("a@b.com", "hash", "Alice").
		WillReturnResult(sqlmock.NewResult(1, 1))

	id, err := repo.Create(context.Background(), &models.User{Email: "a@b.com", PasswordHash: "hash", Name: "Alice"})
	require.NoError(t, err)
	assert.Equal(t, int64(1), id)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestUserRepository_Create_Duplicate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectExec("INSERT INTO users").
		WillReturnError(&mockMySQLDuplicateError{})

	_, err = repo.Create(context.Background(), &models.User{Email: "a@b.com", PasswordHash: "hash", Name: "Alice"})
	assert.ErrorIs(t, err, ErrDuplicate)
}

func TestUserRepository_GetByEmail_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("missing@b.com").
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByEmail(context.Background(), "missing@b.com")
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestUserRepository_GetByEmail_Found(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewUserRepository(db)

	rows := sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at"}).
		AddRow(1, "a@b.com", "hash", "Alice", time.Now())

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("a@b.com").
		WillReturnRows(rows)

	u, err := repo.GetByEmail(context.Background(), "a@b.com")
	require.NoError(t, err)
	assert.Equal(t, "Alice", u.Name)
}

type mockMySQLDuplicateError struct{}

func (e *mockMySQLDuplicateError) Error() string {
	return "Error 1062: Duplicate entry 'a@b.com' for key 'email'"
}
