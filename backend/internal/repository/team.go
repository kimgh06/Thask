package repository

import (
	"context"
	"fmt"

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

func (r *TeamRepo) GetMemberRole(ctx context.Context, teamID, userID string) (model.TeamRole, error) {
	var role model.TeamRole
	err := r.pool.QueryRow(ctx,
		`SELECT role FROM team_members WHERE team_id = $1 AND user_id = $2`,
		teamID, userID,
	).Scan(&role)
	return role, err
}

func (r *TeamRepo) UpdateMemberRole(ctx context.Context, teamID, userID string, role model.TeamRole) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE team_members SET role = $3 WHERE team_id = $1 AND user_id = $2`,
		teamID, userID, role,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *TeamRepo) CountByRole(ctx context.Context, teamID string, role model.TeamRole) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM team_members WHERE team_id = $1 AND role = $2`,
		teamID, role,
	).Scan(&count)
	return count, err
}

func (r *TeamRepo) TransferOwnership(ctx context.Context, teamID, fromUserID, toUserID string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx,
		`UPDATE team_members SET role = 'admin' WHERE team_id = $1 AND user_id = $2 AND role = 'owner'`,
		teamID, fromUserID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("caller is not the current owner")
	}

	if _, err := tx.Exec(ctx,
		`UPDATE team_members SET role = 'owner' WHERE team_id = $1 AND user_id = $2`,
		teamID, toUserID,
	); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
