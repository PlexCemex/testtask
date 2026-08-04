package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeamRepository_Create_AddMember(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTeamRepository(db)

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO teams").
		WithArgs("Dev Team", int64(1)).
		WillReturnResult(sqlmock.NewResult(5, 1))
	mock.ExpectExec("INSERT INTO team_members").
		WithArgs(int64(5), int64(1), "owner").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	var teamID int64
	err = repo.WithTx(context.Background(), func(tx *sql.Tx) error {
		id, err := repo.Create(context.Background(), tx, "Dev Team", 1)
		if err != nil {
			return err
		}
		teamID = id
		return repo.AddMember(context.Background(), tx, id, 1, "owner")
	})

	require.NoError(t, err)
	assert.Equal(t, int64(5), teamID)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestTeamRepository_TeamStats(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTeamRepository(db)

	rows := sqlmock.NewRows([]string{"id", "name", "members_count", "done_last_7d"}).
		AddRow(1, "Dev Team", 3, 2)

	mock.ExpectQuery("SELECT(.|\\n)+FROM teams t").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	stats, err := repo.TeamStats(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	assert.Equal(t, "Dev Team", stats[0].TeamName)
	assert.Equal(t, 3, stats[0].MembersCount)
	assert.Equal(t, 2, stats[0].DoneLast7d)
}

func TestTeamRepository_TopCreators(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTeamRepository(db)

	rows := sqlmock.NewRows([]string{"team_id", "user_id", "user_name", "tasks_created", "rnk"}).
		AddRow(1, 2, "Alice", 5, 1)

	mock.ExpectQuery("SELECT(.|\\n)+FROM \\(").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	top, err := repo.TopCreators(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, top, 1)
	assert.Equal(t, "Alice", top[0].UserName)
	assert.Equal(t, 1, top[0].RankInTeam)
}

func TestTeamRepository_GetMemberRole_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTeamRepository(db)

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(1), int64(1)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetMemberRole(context.Background(), 1, 1)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestTeamRepository_ListByUser(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTeamRepository(db)

	rows := sqlmock.NewRows([]string{"id", "name", "created_by", "created_at"}).
		AddRow(1, "Dev Team", 1, time.Now())

	mock.ExpectQuery("SELECT t.id, t.name, t.created_by, t.created_at").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	teams, err := repo.ListByUser(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, teams, 1)
	assert.Equal(t, "Dev Team", teams[0].Name)
}

func TestTeamRepository_OrphanAssigneeTasks(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTeamRepository(db)

	rows := sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}).
		AddRow(1, 1, "Orphan Task", "", "todo", 99, 1, time.Now(), time.Now())

	mock.ExpectQuery("SELECT(.|\\n)+FROM tasks tk").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	tasks, err := repo.OrphanAssigneeTasks(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, int64(99), *tasks[0].AssigneeID)
}
