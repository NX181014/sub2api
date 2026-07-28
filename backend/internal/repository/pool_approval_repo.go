package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/lib/pq"
)

type poolApprovalRepository struct{ client *dbent.Client }

func NewPoolApprovalRepository(client *dbent.Client) service.PoolApprovalRepository {
	return &poolApprovalRepository{client: client}
}

func (r *poolApprovalRepository) executor(ctx context.Context) *dbent.Client {
	return clientFromContext(ctx, r.client)
}

const poolApprovalSelect = `
SELECT r.id, r.action_type, r.account_id, a.name, r.status, r.reason,
       r.base_revision, r.requested_by_user_id, requester.email,
       r.decided_by_user_id, decider.email, r.decision_reason,
       r.requested_at, r.expires_at, r.decided_at, r.reveal_expires_at,
       r.revealed_at, r.primary_bypass, r.change_summary, r.payload
FROM pool_approval_requests r
JOIN accounts a ON a.id = r.account_id
JOIN users requester ON requester.id = r.requested_by_user_id
LEFT JOIN users decider ON decider.id = r.decided_by_user_id`

func scanPoolApproval(scanner interface{ Scan(...any) error }) (*service.PoolApproval, error) {
	item := &service.PoolApproval{}
	var summaryRaw, payloadRaw []byte
	if err := scanner.Scan(
		&item.ID, &item.ActionType, &item.AccountID, &item.AccountName, &item.Status, &item.Reason,
		&item.BaseRevision, &item.RequestedByUserID, &item.RequestedByEmail,
		&item.DecidedByUserID, &item.DecidedByEmail, &item.DecisionReason,
		&item.RequestedAt, &item.ExpiresAt, &item.DecidedAt, &item.RevealExpiresAt,
		&item.RevealedAt, &item.PrimaryBypass, &summaryRaw, &payloadRaw,
	); err != nil {
		return nil, err
	}
	if len(summaryRaw) > 0 {
		if err := json.Unmarshal(summaryRaw, &item.Changes); err != nil {
			return nil, fmt.Errorf("decode approval summary: %w", err)
		}
	}
	item.Payload = append(json.RawMessage(nil), payloadRaw...)
	return item, nil
}

func (r *poolApprovalRepository) ExpireStale(ctx context.Context, now time.Time) error {
	_, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status='expired',
    decision_reason=COALESCE(decision_reason, 'approval window expired'),
    decided_at=COALESCE(decided_at, $1), updated_at=$1
WHERE (status='pending' AND expires_at <= $1)
   OR (action_type='VIEW_CREDENTIAL' AND status='approved'
       AND revealed_at IS NULL AND reveal_expires_at <= $1)`, now.UTC())
	return err
}

func (r *poolApprovalRepository) CreateApproval(ctx context.Context, item *service.PoolApproval) (*service.PoolApproval, error) {
	if item == nil {
		return nil, fmt.Errorf("nil approval request")
	}
	summary, err := json.Marshal(item.Changes)
	if err != nil {
		return nil, err
	}
	rows, err := r.executor(ctx).QueryContext(ctx, `
INSERT INTO pool_approval_requests(
    action_type,account_id,status,payload,change_summary,reason,base_revision,
    requested_by_user_id,requested_at,expires_at,primary_bypass)
VALUES($1,$2,$3,$4::jsonb,$5::jsonb,$6,$7,$8,$9,$10,$11)
RETURNING id`, item.ActionType, item.AccountID, item.Status, []byte(item.Payload), summary,
		item.Reason, item.BaseRevision, item.RequestedByUserID, item.RequestedAt, item.ExpiresAt, item.PrimaryBypass)
	if err != nil {
		if isPoolApprovalUniqueViolation(err) {
			return nil, service.ErrPoolApprovalConflict
		}
		return nil, fmt.Errorf("create pool approval: %w", err)
	}
	if !rows.Next() {
		_ = rows.Close()
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("create pool approval returned no id")
	}
	var id int64
	if err := rows.Scan(&id); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	return r.GetApproval(ctx, id, false)
}

func isPoolApprovalUniqueViolation(err error) bool {
	var pqErr *pq.Error
	return errors.As(err, &pqErr) && pqErr.Code == "23505"
}

func (r *poolApprovalRepository) ListApprovals(ctx context.Context, filter service.PoolApprovalFilter, limit, offset int) ([]service.PoolApproval, int64, error) {
	clauses := []string{"1=1"}
	args := make([]any, 0, 6)
	add := func(clause string, value any) {
		args = append(args, value)
		clauses = append(clauses, fmt.Sprintf(clause, len(args)))
	}
	if filter.Status != "" {
		add("r.status=$%d", filter.Status)
	}
	if filter.ActionType != "" {
		add("r.action_type=$%d", filter.ActionType)
	}
	if filter.AccountID != nil {
		add("r.account_id=$%d", *filter.AccountID)
	}
	if filter.RequestedByUserID != nil {
		add("r.requested_by_user_id=$%d", *filter.RequestedByUserID)
	}
	where := " WHERE " + strings.Join(clauses, " AND ")

	countRows, err := r.executor(ctx).QueryContext(ctx, `SELECT COUNT(*) FROM pool_approval_requests r`+where, args...)
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if countRows.Next() {
		err = countRows.Scan(&total)
	} else if countRows.Err() != nil {
		err = countRows.Err()
	} else {
		err = sql.ErrNoRows
	}
	_ = countRows.Close()
	if err != nil {
		return nil, 0, err
	}

	args = append(args, limit, offset)
	query := poolApprovalSelect + where + fmt.Sprintf(" ORDER BY r.requested_at DESC, r.id DESC LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	rows, err := r.executor(ctx).QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	items := make([]service.PoolApproval, 0, limit)
	for rows.Next() {
		item, scanErr := scanPoolApproval(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, *item)
	}
	return items, total, rows.Err()
}

func (r *poolApprovalRepository) GetApproval(ctx context.Context, id int64, forUpdate bool) (*service.PoolApproval, error) {
	query := poolApprovalSelect + ` WHERE r.id=$1`
	if forUpdate {
		query += ` FOR UPDATE OF r`
	}
	rows, err := r.executor(ctx).QueryContext(ctx, query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPoolApprovalNotFound
	}
	item, err := scanPoolApproval(rows)
	if err != nil {
		return nil, err
	}
	return item, rows.Err()
}

func (r *poolApprovalRepository) LockAccount(ctx context.Context, accountID int64) error {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT id FROM accounts WHERE id=$1 AND deleted_at IS NULL FOR UPDATE`, accountID)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return service.ErrPoolAccountNotFound
	}
	return nil
}

func (r *poolApprovalRepository) GetApprovalAccountState(ctx context.Context, accountID int64) (*service.PoolApprovalAccountState, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `
SELECT provider_identity,contributor_user_id,created_by_user_id,cost_sharing_enabled
FROM accounts WHERE id=$1 AND deleted_at IS NULL`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPoolAccountNotFound
	}
	item := &service.PoolApprovalAccountState{}
	if err := rows.Scan(&item.ProviderIdentity, &item.ContributorUserID, &item.CreatedByUserID, &item.CostSharingEnabled); err != nil {
		return nil, err
	}
	return item, rows.Err()
}

func (r *poolApprovalRepository) UpdatePoolAccountApproved(ctx context.Context, id int64, input service.UpdatePoolAccountInput) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE accounts SET
    provider_identity=CASE WHEN $2 THEN NULLIF($3,'') ELSE provider_identity END,
    contributor_user_id=CASE WHEN $4 THEN NULLIF($5,0) ELSE contributor_user_id END,
    created_by_user_id=CASE WHEN $6 THEN NULLIF($7,0) ELSE created_by_user_id END,
    cost_sharing_enabled=CASE WHEN $8 THEN $9 ELSE cost_sharing_enabled END,
    updated_at=NOW()
WHERE id=$1 AND deleted_at IS NULL`, id,
		input.ProviderIdentity != nil, stringValue(input.ProviderIdentity),
		input.ContributorUserID != nil, int64Value(input.ContributorUserID),
		input.CreatedByUserID != nil, int64Value(input.CreatedByUserID),
		input.CostSharingEnabled != nil, boolValue(input.CostSharingEnabled))
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolAccountNotFound
	}
	return nil
}

func (r *poolApprovalRepository) SetDecision(ctx context.Context, id int64, status string, actorID int64, reason *string, revealExpiresAt *time.Time) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status=$2,decided_by_user_id=$3,decision_reason=$4,decided_at=NOW(),
    reveal_expires_at=$5,updated_at=NOW()
WHERE id=$1 AND status='pending'`, id, status, actorID, reason, revealExpiresAt)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolApprovalConflict
	}
	return nil
}

func (r *poolApprovalRepository) MarkExpired(ctx context.Context, id int64, reason string) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status='expired',decision_reason=$2,decided_at=COALESCE(decided_at,NOW()),updated_at=NOW()
WHERE id=$1 AND status='pending'`, id, reason)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolApprovalConflict
	}
	return nil
}

func (r *poolApprovalRepository) LoadCredentials(ctx context.Context, accountID int64) (map[string]any, error) {
	rows, err := r.executor(ctx).QueryContext(ctx, `SELECT credentials FROM accounts WHERE id=$1 AND deleted_at IS NULL`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, err
		}
		return nil, service.ErrPoolAccountNotFound
	}
	var raw []byte
	if err := rows.Scan(&raw); err != nil {
		return nil, err
	}
	out := make(map[string]any)
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &out); err != nil {
			return nil, err
		}
	}
	return out, rows.Err()
}

func (r *poolApprovalRepository) ConsumeReveal(ctx context.Context, id int64, revealedAt time.Time) error {
	result, err := r.executor(ctx).ExecContext(ctx, `
UPDATE pool_approval_requests
SET status='consumed',revealed_at=$2,updated_at=$2
WHERE id=$1 AND action_type='VIEW_CREDENTIAL' AND status='approved'
  AND revealed_at IS NULL AND reveal_expires_at > $2`, id, revealedAt.UTC())
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return service.ErrPoolApprovalConflict
	}
	return nil
}
