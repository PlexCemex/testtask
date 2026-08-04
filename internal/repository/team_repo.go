package repository

import (
	"context"
	"database/sql"
	"errors"

	"taskservice/internal/models"
)

type TeamRepository struct {
	db *sql.DB
}

func NewTeamRepository(db *sql.DB) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, tx *sql.Tx, name string, createdBy int64) (int64, error) {
	res, err := tx.ExecContext(ctx, `INSERT INTO teams (name, created_by) VALUES (?, ?)`, name, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (r *TeamRepository) AddMember(ctx context.Context, tx *sql.Tx, teamID, userID int64, role string) error {
	_, err := tx.ExecContext(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES (?, ?, ?)`,
		teamID, userID, role,
	)
	if err != nil && isDuplicateErr(err) {
		return ErrDuplicate
	}
	return err
}

func (r *TeamRepository) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}

func (r *TeamRepository) ListByUser(ctx context.Context, userID int64) ([]models.Team, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT t.id, t.name, t.created_by, t.created_at
		 FROM teams t
		 JOIN team_members tm ON tm.team_id = t.id
		 WHERE tm.user_id = ?
		 ORDER BY t.created_at DESC`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []models.Team
	for rows.Next() {
		var t models.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, rows.Err()
}

func (r *TeamRepository) GetMemberRole(ctx context.Context, teamID, userID int64) (string, error) {
	var role string
	err := r.db.QueryRowContext(ctx,
		`SELECT role FROM team_members WHERE team_id = ? AND user_id = ?`, teamID, userID,
	).Scan(&role)
	if errors.Is(err, sql.ErrNoRows) {
		return "", ErrNotFound
	}
	return role, err
}

func (r *TeamRepository) GetByID(ctx context.Context, id int64) (*models.Team, error) {
	t := &models.Team{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, created_by, created_at FROM teams WHERE id = ?`, id,
	).Scan(&t.ID, &t.Name, &t.CreatedBy, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return t, err
}

// TeamStats - JOIN 3+ таблиц + агрегация:
// название команды, количество участников, количество задач в статусе done за последние 7 дней
func (r *TeamRepository) TeamStats(ctx context.Context, userID int64) ([]models.TeamStats, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT
			t.id,
			t.name,
			COUNT(DISTINCT tm2.user_id) AS members_count,
			COUNT(DISTINCT CASE WHEN tk.status = 'done' AND tk.updated_at >= NOW() - INTERVAL 7 DAY THEN tk.id END) AS done_last_7d
		FROM teams t
		JOIN team_members tm ON tm.team_id = t.id AND tm.user_id = ?
		JOIN team_members tm2 ON tm2.team_id = t.id
		LEFT JOIN tasks tk ON tk.team_id = t.id
		GROUP BY t.id, t.name
		ORDER BY t.id`, userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.TeamStats
	for rows.Next() {
		var s models.TeamStats
		if err := rows.Scan(&s.TeamID, &s.TeamName, &s.MembersCount, &s.DoneLast7d); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

// TopCreators - оконная функция RANK() по количеству созданных задач в каждой команде за месяц.
func (r *TeamRepository) TopCreators(ctx context.Context, teamID int64) ([]models.TopCreator, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT team_id, user_id, user_name, tasks_created, rnk
		FROM (
			SELECT
				tk.team_id,
				u.id AS user_id,
				u.name AS user_name,
				COUNT(*) AS tasks_created,
				RANK() OVER (PARTITION BY tk.team_id ORDER BY COUNT(*) DESC) AS rnk
			FROM tasks tk
			JOIN users u ON u.id = tk.created_by
			WHERE tk.team_id = ? AND tk.created_at >= DATE_SUB(CURDATE(), INTERVAL 1 MONTH)
			GROUP BY tk.team_id, u.id, u.name
		) ranked
		WHERE rnk <= 3
		ORDER BY rnk`, teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.TopCreator
	for rows.Next() {
		var c models.TopCreator
		if err := rows.Scan(&c.TeamID, &c.UserID, &c.UserName, &c.TasksCount, &c.RankInTeam); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// OrphanAssigneeTasks - задачи, где assignee не является членом команды этой задачи.
func (r *TeamRepository) OrphanAssigneeTasks(ctx context.Context, teamID int64) ([]models.Task, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT tk.id, tk.team_id, tk.title, tk.description, tk.status, tk.assignee_id, tk.created_by, tk.created_at, tk.updated_at
		FROM tasks tk
		WHERE tk.team_id = ?
		  AND tk.assignee_id IS NOT NULL
		  AND NOT EXISTS (
		      SELECT 1 FROM team_members tm
		      WHERE tm.team_id = tk.team_id AND tm.user_id = tk.assignee_id
		  )`, teamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.TeamID, &t.Title, &t.Description, &t.Status, &t.AssigneeID, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}
