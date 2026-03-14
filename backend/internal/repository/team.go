package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/model"
)

type TeamRepo struct {
	pool *pgxpool.Pool
}

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool}
}

func (r *TeamRepo) Create(ctx context.Context, name, slug, createdBy string) (*model.Team, error) {
	var t model.Team
	err := r.pool.QueryRow(ctx,
		`INSERT INTO teams (name, slug, created_by) VALUES ($1, $2, $3)
		 RETURNING id, name, slug, created_by, created_at, updated_at`,
		name, slug, createdBy,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepo) FindBySlug(ctx context.Context, slug string) (*model.Team, error) {
	var t model.Team
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, created_by, created_at, updated_at FROM teams WHERE slug = $1`,
		slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepo) FindByUserID(ctx context.Context, userID string) ([]model.Team, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT t.id, t.name, t.slug, t.created_by, t.created_at, t.updated_at
		 FROM teams t
		 INNER JOIN team_members tm ON tm.team_id = t.id
		 WHERE tm.user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var teams []model.Team
	for rows.Next() {
		var t model.Team
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		teams = append(teams, t)
	}
	return teams, nil
}

func (r *TeamRepo) Delete(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM teams WHERE id = $1`, id)
	return err
}

func (r *TeamRepo) Update(ctx context.Context, id, name string) (*model.Team, error) {
	var t model.Team
	err := r.pool.QueryRow(ctx,
		`UPDATE teams SET name = $1, updated_at = now() WHERE id = $2
		 RETURNING id, name, slug, created_by, created_at, updated_at`,
		name, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.CreatedBy, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *TeamRepo) AddMember(ctx context.Context, teamID, userID string, role model.TeamRole) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO team_members (team_id, user_id, role) VALUES ($1, $2, $3)`,
		teamID, userID, role,
	)
	return err
}

func (r *TeamRepo) GetMembers(ctx context.Context, teamID string) ([]model.TeamMemberWithUser, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT tm.id, tm.team_id, tm.user_id, tm.role, tm.joined_at,
		        u.id, u.email, u.display_name, u.created_at, u.updated_at
		 FROM team_members tm
		 INNER JOIN users u ON tm.user_id = u.id
		 WHERE tm.team_id = $1`, teamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []model.TeamMemberWithUser
	for rows.Next() {
		var m model.TeamMemberWithUser
		u := &model.User{}
		if err := rows.Scan(
			&m.ID, &m.TeamID, &m.UserID, &m.Role, &m.JoinedAt,
			&u.ID, &u.Email, &u.DisplayName, &u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, err
		}
		m.User = u
		members = append(members, m)
	}
	return members, nil
}

func (r *TeamRepo) IsMember(ctx context.Context, teamID, userID string) (bool, error) {
	var exists bool
	err := r.pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM team_members WHERE team_id = $1 AND user_id = $2)`,
		teamID, userID,
	).Scan(&exists)
	return exists, err
}

func (r *TeamRepo) RemoveMember(ctx context.Context, teamID, userID string) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM team_members WHERE team_id = $1 AND user_id = $2`, teamID, userID)
	return err
}
