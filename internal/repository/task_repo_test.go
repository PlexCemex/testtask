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

func TestTaskRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTaskRepository(db, nil)

	mock.ExpectExec("INSERT INTO tasks").
		WillReturnResult(sqlmock.NewResult(10, 1))

	id, err := repo.Create(context.Background(), &models.Task{
		TeamID: 1, Title: "Task 1", Status: models.StatusTodo, CreatedBy: 1,
	})
	require.NoError(t, err)
	assert.Equal(t, int64(10), id)
}

func TestTaskRepository_List_NoCache(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTaskRepository(db, nil)

	mock.ExpectQuery("SELECT COUNT\\(\\*\\) FROM tasks WHERE").
		WithArgs(int64(1)).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

	rows := sqlmock.NewRows([]string{"id", "team_id", "title", "description", "status", "assignee_id", "created_by", "created_at", "updated_at"}).
		AddRow(1, 1, "Task 1", "", "todo", nil, 1, time.Now(), time.Now())

	mock.ExpectQuery("SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at").
		WithArgs(int64(1), 20, 0).
		WillReturnRows(rows)

	result, err := repo.List(context.Background(), TaskFilter{TeamID: 1})
	require.NoError(t, err)
	assert.Equal(t, 1, result.Total)
	assert.Len(t, result.Tasks, 1)
}

func TestTaskRepository_GetByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTaskRepository(db, nil)

	mock.ExpectQuery("SELECT (.+) FROM tasks WHERE id").
		WithArgs(int64(99)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetByID(context.Background(), 99)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestTaskRepository_Update(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTaskRepository(db, nil)

	mock.ExpectExec("UPDATE tasks SET").
		WithArgs("New title", "desc", "done", nil, int64(1)).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.Update(context.Background(), &models.Task{
		ID: 1, TeamID: 1, Title: "New title", Description: "desc", Status: "done",
	})
	require.NoError(t, err)
}

func TestTaskRepository_AddAndGetHistory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewTaskRepository(db, nil)

	mock.ExpectExec("INSERT INTO task_history").
		WithArgs(int64(1), int64(1), "status", "todo", "done").
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = repo.AddHistory(context.Background(), &models.TaskHistory{
		TaskID: 1, ChangedBy: 1, Field: "status", OldValue: "todo", NewValue: "done",
	})
	require.NoError(t, err)

	rows := sqlmock.NewRows([]string{"id", "task_id", "changed_by", "field", "old_value", "new_value", "created_at"}).
		AddRow(1, 1, 1, "status", "todo", "done", time.Now())

	mock.ExpectQuery("SELECT id, task_id, changed_by, field, old_value, new_value, created_at FROM task_history").
		WithArgs(int64(1)).
		WillReturnRows(rows)

	history, err := repo.GetHistory(context.Background(), 1)
	require.NoError(t, err)
	require.Len(t, history, 1)
	assert.Equal(t, "status", history[0].Field)
}
