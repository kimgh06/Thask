package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/thask/backend/internal/dbgen"
	"github.com/thask/backend/internal/model"
)

type TeamRepo struct {
	pool *pgxpool.Pool
	q    *dbgen.Queries
}

func NewTeamRepo(pool *pgxpool.Pool) *TeamRepo {
	return &TeamRepo{pool: pool, q: dbgen.New(pool)}
}

// teamFromRow converts a dbgen.Team row to *model.Team.
func teamFromRow(r dbgen.Team) *model.Team {
	return &model.Team{
		ID:        r.ID,
		Name:      r.Name,
		Slug:      r.Slug,
		CreatedBy: r.CreatedBy,
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// ============================================================================
// Static methods — sqlc-generated wrappers
// ============================================================================

func (r *TeamRepo) Create(ctx context.Context, name, slug, createdBy string) (*model.Team, error) {
	row, err := r.q.TeamCreate(ctx, dbgen.TeamCreateParams{
		Name:      name,
		Slug:      slug,
		CreatedBy: createdBy,
	})
	if err != nil {
		return nil, err
	}
	return teamFromRow(row), nil
}

func (r *TeamRepo) FindBySlug(ctx context.Context, slug string) (*model.Team, error) {
	row, err := r.q.TeamFindBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return teamFromRow(row), nil
}

func (r *TeamRepo) FindByUserID(ctx context.Context, userID string) ([]model.Team, error) {
	rows, err := r.q.TeamFindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]model.Team, len(rows))
	for i, row := range rows {
		out[i] = *teamFromRow(row)
	}
	return out, nil
}

func (r *TeamRepo) Delete(ctx context.Context, id string) error {
	return r.q.TeamDelete(ctx, id)
}

func (r *TeamRepo) Update(ctx context.Context, id, name string) (*model.Team, error) {
	row, err := r.q.TeamUpdate(ctx, dbgen.TeamUpdateParams{Name: name, ID: id})
	if err != nil {
		return nil, err
	}
	return teamFromRow(row), nil
}

func (r *TeamRepo) AddMember(ctx context.Context, teamID, userID string, role model.TeamRole) error {
	return r.q.TeamMemberAdd(ctx, dbgen.TeamMemberAddParams{
		TeamID: teamID,
		UserID: userID,
		Role:   string(role),
	})
}

func (r *TeamRepo) IsMember(ctx context.Context, teamID, userID string) (bool, error) {
	return r.q.TeamMemberExists(ctx, dbgen.TeamMemberExistsParams{TeamID: teamID, UserID: userID})
}

func (r *TeamRepo) RemoveMember(ctx context.Context, teamID, userID string) error {
	return r.q.TeamMemberRemove(ctx, dbgen.TeamMemberRemoveParams{TeamID: teamID, UserID: userID})
}

func (r *TeamRepo) GetMemberRole(ctx context.Context, teamID, userID string) (model.TeamRole, error) {
	role, err := r.q.TeamMemberGetRole(ctx, dbgen.TeamMemberGetRoleParams{TeamID: teamID, UserID: userID})
	if err != nil {
		return "", err
	}
	return model.TeamRole(role), nil
}

func (r *TeamRepo) UpdateMemberRole(ctx context.Context, teamID, userID string, role model.TeamRole) error {
	rows, err := r.q.TeamMemberUpdateRole(ctx, dbgen.TeamMemberUpdateRoleParams{
		TeamID: teamID,
		UserID: userID,
		Role:   string(role),
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("member not found")
	}
	return nil
}

func (r *TeamRepo) CountByRole(ctx context.Context, teamID string, role model.TeamRole) (int, error) {
	n, err := r.q.TeamMemberCountByRole(ctx, dbgen.TeamMemberCountByRoleParams{
		TeamID: teamID,
		Role:   string(role),
	})
	if err != nil {
		return 0, err
	}
	return int(n), nil
}

// ============================================================================
// Dynamic / multi-statement methods — hand-written pgx
// ============================================================================

// GetMembers JOINs team_members with users into a flat TeamMemberWithUser.
// Kept as raw pgx because sqlc cannot split a JOIN row across two structs.
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

// TransferOwnership atomically demotes fromUserID to admin and promotes
// toUserID to owner. Kept as raw pgx because it is a multi-statement tx.
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
