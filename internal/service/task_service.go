package service

import (
	"context"
	"strconv"

	"taskservice/internal/models"
	"taskservice/internal/repository"
)

type TaskService struct {
	tasks *repository.TaskRepository
	teams *repository.TeamRepository
}

func NewTaskService(tasks *repository.TaskRepository, teams *repository.TeamRepository) *TaskService {
	return &TaskService{tasks: tasks, teams: teams}
}

func (s *TaskService) Create(ctx context.Context, userID int64, t *models.Task) (*models.Task, error) {
	if _, err := s.teams.GetMemberRole(ctx, t.TeamID, userID); err != nil {
		return nil, ErrForbidden
	}

	if t.Status == "" {
		t.Status = models.StatusTodo
	}
	t.CreatedBy = userID

	id, err := s.tasks.Create(ctx, t)
	if err != nil {
		return nil, err
	}

	return s.tasks.GetByID(ctx, id)
}

func (s *TaskService) List(ctx context.Context, userID int64, f repository.TaskFilter) (*repository.TaskListResult, error) {
	if _, err := s.teams.GetMemberRole(ctx, f.TeamID, userID); err != nil {
		return nil, ErrForbidden
	}
	return s.tasks.List(ctx, f)
}

type TaskUpdateInput struct {
	Title       *string
	Description *string
	Status      *string
	AssigneeID  *int64
}

func (s *TaskService) Update(ctx context.Context, userID, taskID int64, in TaskUpdateInput) (*models.Task, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	if _, err := s.teams.GetMemberRole(ctx, task.TeamID, userID); err != nil {
		return nil, ErrForbidden
	}

	if in.Title != nil && *in.Title != task.Title {
		s.recordHistory(ctx, taskID, userID, "title", task.Title, *in.Title)
		task.Title = *in.Title
	}
	if in.Description != nil && *in.Description != task.Description {
		s.recordHistory(ctx, taskID, userID, "description", task.Description, *in.Description)
		task.Description = *in.Description
	}
	if in.Status != nil && *in.Status != task.Status {
		s.recordHistory(ctx, taskID, userID, "status", task.Status, *in.Status)
		task.Status = *in.Status
	}
	if in.AssigneeID != nil {
		old := ""
		if task.AssigneeID != nil {
			old = strconv.FormatInt(*task.AssigneeID, 10)
		}
		s.recordHistory(ctx, taskID, userID, "assignee_id", old, strconv.FormatInt(*in.AssigneeID, 10))
		task.AssigneeID = in.AssigneeID
	}

	if err := s.tasks.Update(ctx, task); err != nil {
		return nil, err
	}

	return s.tasks.GetByID(ctx, taskID)
}

func (s *TaskService) recordHistory(ctx context.Context, taskID, userID int64, field, oldVal, newVal string) {
	_ = s.tasks.AddHistory(ctx, &models.TaskHistory{
		TaskID:    taskID,
		ChangedBy: userID,
		Field:     field,
		OldValue:  oldVal,
		NewValue:  newVal,
	})
}

func (s *TaskService) History(ctx context.Context, userID, taskID int64) ([]models.TaskHistory, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}
	if _, err := s.teams.GetMemberRole(ctx, task.TeamID, userID); err != nil {
		return nil, ErrForbidden
	}
	return s.tasks.GetHistory(ctx, taskID)
}
