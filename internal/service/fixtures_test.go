package service

import (
	"database/sql"

	"taskservice/internal/models"
)

var sqlNoRowsErr = sql.ErrNoRows

var taskFixture = models.Task{
	TeamID: 1,
	Title:  "Task 1",
	Status: models.StatusTodo,
}
