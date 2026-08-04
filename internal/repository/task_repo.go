package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"taskservice/internal/models"
)

const taskListCacheTTL = 5 * time.Minute

type TaskRepository struct {
	db    *sql.DB
	cache *redis.Client
}

func NewTaskRepository(db *sql.DB, cache *redis.Client) *TaskRepository {
	return &TaskRepository{db: db, cache: cache}
}

func (r *TaskRepository) Create(ctx context.Context, t *models.Task) (int64, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO tasks (team_id, title, description, status, assignee_id, created_by) VALUES (?, ?, ?, ?, ?, ?)`,
		t.TeamID, t.Title, t.Description, t.Status, t.AssigneeID, t.CreatedBy,
	)
	if err != nil {
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}
	r.invalidateListCache(ctx, t.TeamID)
	return id, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*models.Task, error) {
	t := &models.Task{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at FROM tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status, &t.AssigneeID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

type TaskFilter struct {
	TeamID     int64
	Status     string
	AssigneeID int64
	Page       int
	PageSize   int
}

type TaskListResult struct {
	Tasks []models.Task `json:"tasks"`
	Total int           `json:"total"`
	Page  int           `json:"page"`
}

func (r *TaskRepository) cacheKey(f TaskFilter) string {
	return fmt.Sprintf("tasks:team:%d:status:%s:assignee:%d:page:%d:size:%d",
		f.TeamID, f.Status, f.AssigneeID, f.Page, f.PageSize)
}

func (r *TaskRepository) invalidateListCache(ctx context.Context, teamID int64) {
	if r.cache == nil {
		return
	}
	iter := r.cache.Scan(ctx, 0, fmt.Sprintf("tasks:team:%d:*", teamID), 100).Iterator()
	for iter.Next(ctx) {
		r.cache.Del(ctx, iter.Val())
	}
}

func (r *TaskRepository) List(ctx context.Context, f TaskFilter) (*TaskListResult, error) {
	if f.Page <= 0 {
		f.Page = 1
	}
	if f.PageSize <= 0 || f.PageSize > 100 {
		f.PageSize = 20
	}
	// пока offset-пагинация, на больших командах стоит перейти на cursor-based

	if r.cache != nil {
		cached, err := r.cache.Get(ctx, r.cacheKey(f)).Result()
		if err == nil {
			var result TaskListResult
			if json.Unmarshal([]byte(cached), &result) == nil {
				return &result, nil
			}
		}
	}

	where := []string{"team_id = ?"}
	args := []interface{}{f.TeamID}

	if f.Status != "" {
		where = append(where, "status = ?")
		args = append(args, f.Status)
	}
	if f.AssigneeID != 0 {
		where = append(where, "assignee_id = ?")
		args = append(args, f.AssigneeID)
	}
	whereClause := strings.Join(where, " AND ")

	var total int
	countQuery := "SELECT COUNT(*) FROM tasks WHERE " + whereClause
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	offset := (f.Page - 1) * f.PageSize
	query := fmt.Sprintf(
		`SELECT id, team_id, title, description, status, assignee_id, created_by, created_at, updated_at
		 FROM tasks WHERE %s ORDER BY created_at DESC LIMIT ? OFFSET ?`, whereClause,
	)
	args = append(args, f.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tasks := []models.Task{}
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status, &t.AssigneeID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	result := &TaskListResult{Tasks: tasks, Total: total, Page: f.Page}

	if r.cache != nil {
		if data, err := json.Marshal(result); err == nil {
			r.cache.Set(ctx, r.cacheKey(f), data, taskListCacheTTL)
		}
	}

	return result, nil
}

func (r *TaskRepository) Update(ctx context.Context, t *models.Task) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE tasks SET title = ?, description = ?, status = ?, assignee_id = ? WHERE id = ?`,
		t.Title, t.Description, t.Status, t.AssigneeID, t.ID,
	)
	if err != nil {
		return err
	}
	r.invalidateListCache(ctx, t.TeamID)
	return nil
}

func (r *TaskRepository) AddHistory(ctx context.Context, h *models.TaskHistory) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO task_history (task_id, changed_by, field, old_value, new_value) VALUES (?, ?, ?, ?, ?)`,
		h.TaskID, h.ChangedBy, h.Field, h.OldValue, h.NewValue,
	)
	return err
}

func (r *TaskRepository) GetHistory(ctx context.Context, taskID int64) ([]models.TaskHistory, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, task_id, changed_by, field, old_value, new_value, created_at
		 FROM task_history WHERE task_id = ? ORDER BY created_at DESC`, taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	history := []models.TaskHistory{}
	for rows.Next() {
		var h models.TaskHistory
		if err := rows.Scan(&h.ID, &h.TaskID, &h.ChangedBy, &h.Field, &h.OldValue, &h.NewValue, &h.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, rows.Err()
}
