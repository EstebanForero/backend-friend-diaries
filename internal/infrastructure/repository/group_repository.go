package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/friend-diaries/backend/internal/domain/group"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupRepository struct {
	pool *pgxpool.Pool
}

func NewGroupRepository(pool *pgxpool.Pool) *GroupRepository {
	return &GroupRepository{pool: pool}
}

func (r *GroupRepository) Create(ctx context.Context, g *group.Group) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO diary_groups (id, name, owner_id, timezone, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, g.ID, g.Name, g.OwnerID, g.Timezone, g.CreatedAt, g.UpdatedAt)
	return err
}

func (r *GroupRepository) FindByID(ctx context.Context, id uuid.UUID) (*group.Group, error) {
	g := &group.Group{}
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, owner_id, timezone, created_at, updated_at FROM diary_groups WHERE id=$1
	`, id).Scan(&g.ID, &g.Name, &g.OwnerID, &g.Timezone, &g.CreatedAt, &g.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("group not found")
		}
		return nil, err
	}
	return g, nil
}

func (r *GroupRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]*group.Group, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT g.id, g.name, g.owner_id, g.timezone, g.created_at, g.updated_at
		FROM diary_groups g
		JOIN group_members gm ON gm.group_id = g.id
		WHERE gm.user_id=$1 AND gm.invitation_status='accepted'
		ORDER BY g.created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []*group.Group
	for rows.Next() {
		g := &group.Group{}
		if err := rows.Scan(&g.ID, &g.Name, &g.OwnerID, &g.Timezone, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (r *GroupRepository) Update(ctx context.Context, g *group.Group) error {
	g.UpdatedAt = time.Now().UTC()
	_, err := r.pool.Exec(ctx, `
		UPDATE diary_groups SET name=$1, timezone=$2, updated_at=$3 WHERE id=$4
	`, g.Name, g.Timezone, g.UpdatedAt, g.ID)
	return err
}

func (r *GroupRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM diary_groups WHERE id=$1`, id)
	return err
}

func (r *GroupRepository) AddMember(ctx context.Context, m *group.Member) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO group_members (group_id, user_id, role, invitation_status, invited_at, joined_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (group_id, user_id) DO UPDATE
		SET invitation_status=EXCLUDED.invitation_status, joined_at=EXCLUDED.joined_at
	`, m.GroupID, m.UserID, m.Role, m.InvitationStatus, m.InvitedAt, m.JoinedAt)
	return err
}

func (r *GroupRepository) FindMember(ctx context.Context, groupID, userID uuid.UUID) (*group.Member, error) {
	m := &group.Member{}
	err := r.pool.QueryRow(ctx, `
		SELECT group_id, user_id, role, invitation_status, invited_at, joined_at
		FROM group_members WHERE group_id=$1 AND user_id=$2
	`, groupID, userID).Scan(&m.GroupID, &m.UserID, &m.Role, &m.InvitationStatus, &m.InvitedAt, &m.JoinedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("member not found")
		}
		return nil, err
	}
	return m, nil
}

func (r *GroupRepository) UpdateMember(ctx context.Context, m *group.Member) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE group_members SET invitation_status=$1, joined_at=$2 WHERE group_id=$3 AND user_id=$4
	`, m.InvitationStatus, m.JoinedAt, m.GroupID, m.UserID)
	return err
}

func (r *GroupRepository) RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM group_members WHERE group_id=$1 AND user_id=$2`, groupID, userID)
	return err
}

func (r *GroupRepository) CountActiveMembers(ctx context.Context, groupID uuid.UUID) (int, error) {
	var count int
	err := r.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM group_members WHERE group_id=$1 AND invitation_status='accepted'
	`, groupID).Scan(&count)
	return count, err
}

func (r *GroupRepository) FindActiveMembers(ctx context.Context, groupID uuid.UUID) ([]*group.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT group_id, user_id, role, invitation_status, invited_at, joined_at
		FROM group_members WHERE group_id=$1 AND invitation_status='accepted'
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*group.Member
	for rows.Next() {
		m := &group.Member{}
		if err := rows.Scan(&m.GroupID, &m.UserID, &m.Role, &m.InvitationStatus, &m.InvitedAt, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}

func (r *GroupRepository) FindActiveMembersWithDetails(ctx context.Context, groupID uuid.UUID) ([]*group.MemberDetail, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT gm.user_id::text, u.display_name, u.app_id, gm.role::text,
		       to_char(gm.joined_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"')
		FROM group_members gm
		JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1 AND gm.invitation_status = 'accepted' AND u.deleted_at IS NULL
		ORDER BY gm.joined_at ASC NULLS LAST
	`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []*group.MemberDetail
	for rows.Next() {
		m := &group.MemberDetail{}
		if err := rows.Scan(&m.UserID, &m.DisplayName, &m.AppID, &m.Role, &m.JoinedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *GroupRepository) FindPendingInvitationsByUser(ctx context.Context, userID uuid.UUID) ([]*group.Member, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT group_id, user_id, role, invitation_status, invited_at, joined_at
		FROM group_members WHERE user_id=$1 AND invitation_status='pending'
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*group.Member
	for rows.Next() {
		m := &group.Member{}
		if err := rows.Scan(&m.GroupID, &m.UserID, &m.Role, &m.InvitationStatus, &m.InvitedAt, &m.JoinedAt); err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, rows.Err()
}
