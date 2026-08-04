//go:build integration

package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mysql"
	"github.com/testcontainers/testcontainers-go/wait"

	"taskservice/internal/models"
)

func setupMySQL(t *testing.T) *sql.DB {
	ctx := context.Background()

	container, err := mysql.Run(ctx, "mysql:8.0",
		mysql.WithDatabase("taskservice"),
		mysql.WithUsername("root"),
		mysql.WithPassword("root"),
		mysql.WithScripts("../../migrations/001_init.sql"),
		testcontainers.WithWaitStrategy(wait.ForLog("ready for connections").WithOccurrence(2)),
	)
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "parseTime=true")
	require.NoError(t, err)

	db, err := sql.Open("mysql", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	for i := 0; i < 10; i++ {
		if err := db.Ping(); err == nil {
			break
		}
		time.Sleep(time.Second)
	}

	return db
}

func TestIntegration_UserTeamTaskFlow(t *testing.T) {
	db := setupMySQL(t)
	ctx := context.Background()

	userRepo := NewUserRepository(db)
	teamRepo := NewTeamRepository(db)
	taskRepo := NewTaskRepository(db, nil)

	userID, err := userRepo.Create(ctx, &models.User{Email: "int@test.com", PasswordHash: "hash", Name: "Integration"})
	require.NoError(t, err)

	var teamID int64
	err = teamRepo.WithTx(ctx, func(tx *sql.Tx) error {
		id, err := teamRepo.Create(ctx, tx, "Integration Team", userID)
		if err != nil {
			return err
		}
		teamID = id
		return teamRepo.AddMember(ctx, tx, teamID, userID, models.RoleOwner)
	})
	require.NoError(t, err)

	taskID, err := taskRepo.Create(ctx, &models.Task{
		TeamID: teamID, Title: "Integration Task", Status: models.StatusTodo, CreatedBy: userID,
	})
	require.NoError(t, err)

	result, err := taskRepo.List(ctx, TaskFilter{TeamID: teamID})
	require.NoError(t, err)
	require.Len(t, result.Tasks, 1)
	require.Equal(t, taskID, result.Tasks[0].ID)

	stats, err := teamRepo.TeamStats(ctx, userID)
	require.NoError(t, err)
	require.Len(t, stats, 1)
	require.Equal(t, 1, stats[0].MembersCount)
}
