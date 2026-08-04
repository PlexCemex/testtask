package service

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskservice/internal/repository"
)

func newTeamService(t *testing.T) (*TeamService, sqlmock.Sqlmock, func()) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	teams := repository.NewTeamRepository(db)
	users := repository.NewUserRepository(db)
	svc := NewTeamService(teams, users, NewEmailService())

	return svc, mock, func() { db.Close() }
}

func TestTeamService_Create(t *testing.T) {
	svc, mock, closeDB := newTeamService(t)
	defer closeDB()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO teams").
		WithArgs("Dev Team", int64(1)).
		WillReturnResult(sqlmock.NewResult(3, 1))
	mock.ExpectExec("INSERT INTO team_members").
		WithArgs(int64(3), int64(1), "owner").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectQuery("SELECT id, name, created_by, created_at FROM teams WHERE id").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_by", "created_at"}).
			AddRow(3, "Dev Team", 1, time.Now()))

	team, err := svc.Create(context.Background(), "Dev Team", 1)
	require.NoError(t, err)
	assert.Equal(t, "Dev Team", team.Name)
}

func TestTeamService_Invite_Forbidden(t *testing.T) {
	svc, mock, closeDB := newTeamService(t)
	defer closeDB()

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(3), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("member"))

	err := svc.Invite(context.Background(), 3, 1, "new@user.com")
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestTeamService_Invite_Success(t *testing.T) {
	svc, mock, closeDB := newTeamService(t)
	defer closeDB()

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(3), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("owner"))

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("new@user.com").
		WillReturnRows(sqlmock.NewRows([]string{"id", "email", "password_hash", "name", "created_at"}).
			AddRow(2, "new@user.com", "hash", "Bob", time.Now()))

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO team_members").
		WithArgs(int64(3), int64(2), "member").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	mock.ExpectQuery("SELECT id, name, created_by, created_at FROM teams WHERE id").
		WithArgs(int64(3)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_by", "created_at"}).
			AddRow(3, "Dev Team", 1, time.Now()))

	err := svc.Invite(context.Background(), 3, 1, "new@user.com")
	require.NoError(t, err)
}

func TestTeamService_Invite_UserNotFound(t *testing.T) {
	svc, mock, closeDB := newTeamService(t)
	defer closeDB()

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(3), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("owner"))

	mock.ExpectQuery("SELECT (.+) FROM users WHERE email").
		WithArgs("missing@user.com").
		WillReturnError(sql.ErrNoRows)

	err := svc.Invite(context.Background(), 3, 1, "missing@user.com")
	assert.ErrorIs(t, err, repository.ErrNotFound)
}

func TestTeamService_ListForUser(t *testing.T) {
	svc, mock, closeDB := newTeamService(t)
	defer closeDB()

	mock.ExpectQuery("SELECT t.id, t.name, t.created_by, t.created_at").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "created_by", "created_at"}).
			AddRow(1, "Dev Team", 1, time.Now()))

	teams, err := svc.ListForUser(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, teams, 1)
}

func TestTeamService_TeamStatsTopCreatorsOrphans(t *testing.T) {
	svc, mock, closeDB := newTeamService(t)
	defer closeDB()

	mock.ExpectQuery("SELECT(.|\\n)+FROM teams t").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name", "members_count", "done_last_7d"}).
			AddRow(1, "Dev Team", 2, 1))
	stats, err := svc.TeamStats(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, stats, 1)

	mock.ExpectQuery("SELECT(.|\\n)+FROM \\(").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"team_id", "user_id", "user_name", "tasks_created", "rnk"}).
			AddRow(1, 1, "Alice", 3, 1))
	top, err := svc.TopCreators(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, top, 1)

	mock.ExpectQuery("SELECT(.|\\n)+FROM tasks tk").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}))
	orphans, err := svc.OrphanAssignees(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, orphans, 0)
}
