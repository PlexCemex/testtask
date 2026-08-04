package service

import (
	"context"
	"testing"
	"time"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"taskservice/internal/repository"
)

func TestTaskService_Create_Forbidden(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tasks := repository.NewTaskRepository(db, nil)
	teams := repository.NewTeamRepository(db)
	svc := NewTaskService(tasks, teams)

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(1), int64(99)).
		WillReturnError(sqlNoRowsErr)

	_, err = svc.Create(context.Background(), 99, &taskFixture)
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestTaskService_Create_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tasks := repository.NewTaskRepository(db, nil)
	teams := repository.NewTeamRepository(db)
	svc := NewTaskService(tasks, teams)

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("member"))

	mock.ExpectExec("INSERT INTO tasks").
		WillReturnResult(sqlmock.NewResult(7, 1))

	mock.ExpectQuery("SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at FROM tasks WHERE id").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}).
			AddRow(7, 1, "Task 1", "", "todo", nil, 1, time.Now(), time.Now()))

	created, err := svc.Create(context.Background(), 1, &taskFixture)
	require.NoError(t, err)
	assert.Equal(t, int64(7), created.ID)
}

func TestTaskService_Update_StatusChangeRecordsHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tasks := repository.NewTaskRepository(db, nil)
	teams := repository.NewTeamRepository(db)
	svc := NewTaskService(tasks, teams)

	mock.ExpectQuery("SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at FROM tasks WHERE id").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}).
			AddRow(7, 1, "Task 1", "", "todo", nil, 1, time.Now(), time.Now()))

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("member"))

	mock.ExpectExec("INSERT INTO task_history").
		WillReturnResult(sqlmock.NewResult(1, 1))

	mock.ExpectExec("UPDATE tasks SET").
		WillReturnResult(sqlmock.NewResult(0, 1))

	mock.ExpectQuery("SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at FROM tasks WHERE id").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}).
			AddRow(7, 1, "Task 1", "", "in_progress", nil, 1, time.Now(), time.Now()))

	newStatus := "in_progress"
	updated, err := svc.Update(context.Background(), 1, 7, TaskUpdateInput{Status: &newStatus})
	require.NoError(t, err)
	assert.Equal(t, "in_progress", updated.Status)
}

func TestTaskService_List_Forbidden(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tasks := repository.NewTaskRepository(db, nil)
	teams := repository.NewTeamRepository(db)
	svc := NewTaskService(tasks, teams)

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(1), int64(99)).
		WillReturnError(sqlNoRowsErr)

	_, err = svc.List(context.Background(), 99, repository.TaskFilter{TeamID: 1})
	assert.ErrorIs(t, err, ErrForbidden)
}

func TestTaskService_List_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tasks := repository.NewTaskRepository(db, nil)
	teams := repository.NewTeamRepository(db)
	svc := NewTaskService(tasks, teams)

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("member"))

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tasks WHERE").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

	mock.ExpectQuery("SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at").
		WithArgs(int64(1), 20, 0).
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}))

	result, err := svc.List(context.Background(), 1, repository.TaskFilter{TeamID: 1})
	require.NoError(t, err)
	assert.Equal(t, 0, result.Total)
}

func TestTaskService_History(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	tasks := repository.NewTaskRepository(db, nil)
	teams := repository.NewTeamRepository(db)
	svc := NewTaskService(tasks, teams)

	mock.ExpectQuery("SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at FROM tasks WHERE id").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}).
			AddRow(7, 1, "Task 1", "", "todo", nil, 1, time.Now(), time.Now()))

	mock.ExpectQuery("SELECT role FROM team_members").
		WithArgs(int64(1), int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"role"}).AddRow("member"))

	mock.ExpectQuery("SELECT id, task_id, changed_by, field, old_value, new_value, created_at FROM task_history").
		WithArgs(int64(7)).
		WillReturnRows(sqlmock.NewRows([]string{"id", "task_id", "changed_by", "field", "old_value", "new_value", "created_at"}).
			AddRow(1, 7, 1, "status", "todo", "done", time.Now()))

	history, err := svc.History(context.Background(), 1, 7)
	require.NoError(t, err)
	require.Len(t, history, 1)
}
