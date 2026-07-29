package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/shopspring/decimal"
)

type accountDeleteTxStarter interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

type accountDeleteExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func scanAccountDeleteRow(ctx context.Context, exec accountDeleteExecutor, query string, args []any, dest ...any) error {
	rows, err := exec.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	return rows.Scan(dest...)
}

type accountDeleteState struct {
	id              int64
	deleted         bool
	history         bool
	remaining       int64
	remainingTokens int64
}

type accountDeleteCostBucket struct {
	payerID     int64
	sourceID    sql.NullInt64
	balance     int64
	tokens      int64
	serviceFrom time.Time
	serviceTo   time.Time
}

func (r *accountRepository) DeleteAccountWithLifecycle(ctx context.Context, accountID int64, options service.AccountDeleteOptions) (*service.AccountDeleteResult, error) {
	if accountID <= 0 {
		return nil, service.ErrAccountNotFound
	}
	options.CostDisposition = strings.ToLower(strings.TrimSpace(options.CostDisposition))
	options.Reason = strings.TrimSpace(options.Reason)
	if len(options.Reason) > 1000 {
		return nil, infraerrors.BadRequest("ACCOUNT_DELETE_REASON_TOO_LONG", "delete reason must not exceed 1000 characters")
	}
	if options.CostDisposition != "" && options.CostDisposition != "write_off" && options.CostDisposition != "refund" && options.CostDisposition != "transfer" {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_DELETE_COST_DISPOSITION", "cost_disposition must be write_off, refund, or transfer")
	}
	if options.RefundAmountMinor != nil && (options.CostDisposition != "refund" || *options.RefundAmountMinor <= 0) {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_DELETE_REFUND_AMOUNT", "refund_amount_minor must be positive and is only valid for refund disposition")
	}
	if options.ReplacementAccountID != nil && options.CostDisposition != "transfer" {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_DELETE_REPLACEMENT", "replacement_account_id is only valid for transfer disposition")
	}
	if options.CostDisposition == "transfer" && (options.ReplacementAccountID == nil || *options.ReplacementAccountID <= 0) {
		return nil, infraerrors.BadRequest("INVALID_ACCOUNT_DELETE_REPLACEMENT", "replacement_account_id is required for transfer disposition")
	}
	var exec accountDeleteExecutor
	var tx *sql.Tx
	if dbent.TxFromContext(ctx) != nil {
		exec = clientFromContext(ctx, r.client)
	} else {
		starter, ok := r.sql.(accountDeleteTxStarter)
		if !ok {
			return nil, fmt.Errorf("account repository SQL transaction support is not configured")
		}
		var err error
		tx, err = starter.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("begin account lifecycle delete: %w", err)
		}
		defer func() { _ = tx.Rollback() }()
		exec = tx
	}

	states, err := lockAccountDeleteFamily(ctx, exec, accountID)
	if err != nil {
		return nil, err
	}
	if len(states) == 0 {
		return nil, service.ErrAccountNotFound
	}
	mainFound := false
	for i := range states {
		if states[i].id == accountID {
			mainFound = true
			if states[i].deleted {
				return &service.AccountDeleteResult{AccountID: accountID, AlreadyDeleted: true}, nil
			}
		}
	}
	if !mainFound {
		return nil, service.ErrAccountNotFound
	}

	active := states[:0]
	archiveAll := false
	for _, state := range states {
		if state.deleted {
			continue
		}
		state.history, state.remaining, state.remainingTokens, err = loadAccountDeleteHistory(ctx, exec, state.id)
		if err != nil {
			return nil, err
		}
		archiveAll = archiveAll || state.history
		active = append(active, state)
	}
	states = active
	if len(states) == 0 {
		return &service.AccountDeleteResult{AccountID: accountID, AlreadyDeleted: true}, nil
	}
	if options.CostDisposition == "transfer" && options.ReplacementAccountID != nil {
		for _, state := range states {
			if state.id == *options.ReplacementAccountID {
				return nil, infraerrors.BadRequest("ACCOUNT_DELETE_REPLACEMENT_IN_DELETE_SET", "replacement account cannot be part of the account family being deleted")
			}
		}
	}

	var mainRemaining, totalRemaining int64
	for _, state := range states {
		if state.id == accountID {
			mainRemaining = state.remaining
		}
		if state.remaining > 0 {
			totalRemaining += state.remaining
		}
	}
	refundTotal := int64(0)
	writtenOffTotal := int64(0)
	if options.CostDisposition == "refund" {
		if options.RefundAmountMinor != nil {
			refundTotal = *options.RefundAmountMinor
		} else {
			refundTotal = totalRemaining
		}
		if refundTotal > totalRemaining {
			return nil, infraerrors.BadRequest("ACCOUNT_DELETE_REFUND_EXCEEDS_REMAINING", "refund amount cannot exceed remaining account cost")
		}
		writtenOffTotal = totalRemaining - refundTotal
	} else if options.CostDisposition == "write_off" {
		writtenOffTotal = totalRemaining
	}
	stateRefunds := allocateAccountDeleteAmounts(refundTotal, stateDeleteBalances(states))
	for i, state := range states {
		if state.remaining <= 0 {
			continue
		}
		if options.CostDisposition == "" {
			return nil, infraerrors.Conflict("ACCOUNT_DELETE_COST_DISPOSITION_REQUIRED", "remaining account cost must be written off, refunded, or transferred before deletion")
		}
		if options.ActorUserID <= 0 {
			return nil, infraerrors.BadRequest("ACCOUNT_DELETE_ACTOR_REQUIRED", "an administrator identity is required to dispose remaining account cost")
		}
		stateOptions := options
		if options.CostDisposition == "refund" {
			stateOptions.RefundAmountMinor = &stateRefunds[i]
		}
		if err := applyAccountDeleteCostDisposition(ctx, exec, state.id, state.remaining, state.remainingTokens, stateOptions); err != nil {
			return nil, err
		}
		archiveAll = true
	}
	if archiveAll && options.ActorUserID <= 0 {
		return nil, infraerrors.BadRequest("ACCOUNT_DELETE_ACTOR_REQUIRED", "an administrator identity is required to archive an account with history")
	}

	affected := make([]int64, 0, len(states))
	for _, state := range states {
		if err := cleanAccountDeleteBindings(ctx, exec, state.id); err != nil {
			return nil, err
		}
		if archiveAll {
			if err := archiveAccountForDelete(ctx, exec, state.id); err != nil {
				return nil, err
			}
			if state.remaining <= 0 {
				if err := recordAccountRetired(ctx, exec, state.id, options); err != nil {
					return nil, err
				}
			}
		} else if err := physicallyDeleteEmptyAccount(ctx, exec, state.id); err != nil {
			return nil, err
		}
		affected = append(affected, state.id)
	}
	if tx != nil {
		if err := tx.Commit(); err != nil {
			return nil, err
		}
	}
	if tx != nil {
		for _, id := range affected {
			r.deleteSchedulerAccountSnapshot(ctx, id)
		}
	}
	return &service.AccountDeleteResult{
		AccountID: accountID, Archived: archiveAll, RemainingCostMinor: mainRemaining,
		CostDisposition: options.CostDisposition, RefundAmountMinor: refundTotal,
		WrittenOffMinor: writtenOffTotal, AffectedAccountIDs: affected,
	}, nil
}

func stateDeleteBalances(states []accountDeleteState) []int64 {
	balances := make([]int64, len(states))
	for i := range states {
		if states[i].remaining > 0 {
			balances[i] = states[i].remaining
		}
	}
	return balances
}

func lockAccountDeleteFamily(ctx context.Context, exec accountDeleteExecutor, accountID int64) ([]accountDeleteState, error) {
	rows, err := exec.QueryContext(ctx, `
SELECT id, deleted_at IS NOT NULL
FROM accounts
WHERE id=$1 OR parent_account_id=$1
ORDER BY CASE WHEN id=$1 THEN 1 ELSE 0 END, id
FOR UPDATE`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	states := make([]accountDeleteState, 0, 2)
	for rows.Next() {
		var state accountDeleteState
		if err := rows.Scan(&state.id, &state.deleted); err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	return states, rows.Err()
}

func loadAccountDeleteHistory(ctx context.Context, exec accountDeleteExecutor, accountID int64) (bool, int64, int64, error) {
	var usage, costs, lifecycle, decidedApproval, locked, unpriced bool
	var costBasis, disposed, usedTokens int64
	var trancheJSON []byte
	err := scanAccountDeleteRow(ctx, exec, `
SELECT
  EXISTS(SELECT 1 FROM usage_logs WHERE account_id=$1),
  EXISTS(SELECT 1 FROM account_cost_entries WHERE account_id=$1 OR related_account_id=$1),
  EXISTS(SELECT 1 FROM account_lifecycle_events WHERE account_id=$1 OR replacement_account_id=$1),
  EXISTS(SELECT 1 FROM pool_approval_requests WHERE account_id=$1),
  EXISTS(
    SELECT 1 FROM pool_settlements s
    CROSS JOIN LATERAL jsonb_array_elements(COALESCE(s.cost_snapshot,'[]'::jsonb)) item
	WHERE s.status IN ('locked','paid') AND item->>'account_id'=$1::text
  ),
	EXISTS(SELECT 1 FROM account_cost_entries WHERE account_id=$1 AND cny_amount_minor>0 AND expected_token_count IS NULL),
  COALESCE((SELECT SUM(cny_amount_minor)::bigint FROM account_cost_entries WHERE account_id=$1 AND entry_type NOT IN ('refund','replacement_out','write_off')),0),
  COALESCE((SELECT -SUM(cny_amount_minor)::bigint FROM account_cost_entries WHERE account_id=$1 AND entry_type IN ('refund','replacement_out','write_off') AND cny_amount_minor<0),0),
  COALESCE((SELECT SUM(input_tokens::bigint+output_tokens::bigint+cache_creation_tokens::bigint+
    cache_read_tokens::bigint+image_output_tokens::bigint+image_input_tokens::bigint) FROM usage_logs WHERE account_id=$1),0),
	COALESCE((SELECT jsonb_agg(jsonb_build_object(
	  'id',priced.id,'cost_minor',priced.cny_amount_minor,'expected_tokens',priced.expected_token_count,
	  'paid_at',priced.paid_at,'payer_user_id',priced.payer_user_id,'purchase_source_id',priced.purchase_source_id,
	  'service_start',priced.service_start,'service_end',priced.service_end,
	  'usage_tokens',COALESCE((SELECT SUM(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
	    ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint)
	    FROM usage_logs ul WHERE ul.account_id=$1 AND ul.created_at>=priced.paid_at
	      AND (priced.next_paid_at IS NULL OR ul.created_at<priced.next_paid_at)),0)) ORDER BY priced.paid_at,priced.id)
	  FROM (SELECT c.*,LEAD(c.paid_at) OVER (ORDER BY c.paid_at,c.id) next_paid_at
	        FROM account_cost_entries c WHERE c.account_id=$1 AND c.cny_amount_minor>0 AND c.expected_token_count>0) priced),'[]'::jsonb)`, []any{accountID},
		&usage, &costs, &lifecycle, &decidedApproval, &locked, &unpriced, &costBasis, &disposed, &usedTokens, &trancheJSON)
	if err != nil {
		return false, 0, 0, err
	}
	tranches, err := decodePoolCostTranches(trancheJSON)
	if err != nil {
		return false, 0, 0, err
	}
	states, _, _, remainingTokens := service.RecognizeAccountCostTranches(usedTokens, tranches)
	_, remaining, _ := service.CalculateAccountCostRecognitionByTranches(costBasis, disposed, usedTokens, tranches)
	if unpriced && remaining > 0 {
		return false, 0, 0, infraerrors.Conflict("ACCOUNT_DELETE_UNPRICED_COST", "repair expected token counts before disposing this account")
	}
	var unrecognized int64
	for _, state := range states {
		unrecognized += state.RemainingCostMinor
	}
	if unrecognized > 0 && remaining < unrecognized {
		remainingTokens = decimal.NewFromInt(remainingTokens).Mul(decimal.NewFromInt(remaining)).Div(decimal.NewFromInt(unrecognized)).Floor().IntPart()
	}
	return usage || costs || lifecycle || decidedApproval || locked, remaining, remainingTokens, nil
}

func applyAccountDeleteCostDisposition(ctx context.Context, exec accountDeleteExecutor, accountID, amountMinor, expectedTokenCount int64, options service.AccountDeleteOptions) error {
	buckets, err := loadAccountDeleteCostBuckets(ctx, exec, accountID)
	if err != nil {
		return err
	}
	balances := make([]int64, len(buckets))
	for i := range buckets {
		balances[i] = buckets[i].balance
	}
	amounts := allocateAccountDeleteAmounts(amountMinor, balances)
	tokenAmounts := make([]int64, len(buckets))
	for i := range buckets {
		tokenAmounts[i] = buckets[i].tokens
	}
	note := options.Reason
	if note == "" {
		note = "account deletion " + options.CostDisposition
	}

	if options.CostDisposition == "transfer" {
		if options.ReplacementAccountID == nil || *options.ReplacementAccountID <= 0 || *options.ReplacementAccountID == accountID {
			return infraerrors.BadRequest("ACCOUNT_DELETE_REPLACEMENT_REQUIRED", "a different replacement account is required for cost transfer")
		}
		var targetID int64
		err = scanAccountDeleteRow(ctx, exec, `
SELECT id FROM accounts
WHERE id=$1 AND deleted_at IS NULL AND parent_account_id IS NULL
  AND status='active' AND schedulable=TRUE AND cost_sharing_enabled=TRUE
	AND (expires_at IS NULL OR expires_at>NOW())
FOR UPDATE`, []any{*options.ReplacementAccountID}, &targetID)
		if errors.Is(err, sql.ErrNoRows) {
			return infraerrors.BadRequest("ACCOUNT_DELETE_REPLACEMENT_NOT_AVAILABLE", "replacement account must be an active, schedulable, unexpired shared-pool primary account")
		}
		if err != nil {
			return err
		}
	}

	refundAmount := int64(0)
	if options.CostDisposition == "refund" {
		refundAmount = amountMinor
		if options.RefundAmountMinor != nil {
			refundAmount = *options.RefundAmountMinor
		}
	}
	refunds := allocateAccountDeleteAmounts(refundAmount, amounts)
	for i, bucket := range buckets {
		amount := amounts[i]
		if amount == 0 {
			continue
		}
		switch options.CostDisposition {
		case "refund":
			if refunds[i] > 0 {
				if err := insertAccountDeleteCostEntry(ctx, exec, accountID, bucket, "refund", -refunds[i], nil, nil, options.ActorUserID, note); err != nil {
					return err
				}
			}
			if loss := amount - refunds[i]; loss > 0 {
				if err := insertAccountDeleteCostEntry(ctx, exec, accountID, bucket, "write_off", -loss, nil, nil, options.ActorUserID, note); err != nil {
					return err
				}
			}
		case "transfer":
			replacementID := *options.ReplacementAccountID
			if err := insertAccountDeleteCostEntry(ctx, exec, accountID, bucket, "replacement_out", -amount, &replacementID, nil, options.ActorUserID, note); err != nil {
				return err
			}
			var transferredTokens *int64
			if tokenAmounts[i] > 0 {
				transferredTokens = &tokenAmounts[i]
			}
			if err := insertAccountDeleteCostEntry(ctx, exec, replacementID, bucket, "replacement_in", amount, &accountID, transferredTokens, options.ActorUserID, note); err != nil {
				return err
			}
		default:
			if err := insertAccountDeleteCostEntry(ctx, exec, accountID, bucket, "write_off", -amount, nil, nil, options.ActorUserID, note); err != nil {
				return err
			}
		}
	}

	if options.CostDisposition == "refund" {
		if refundAmount > 0 {
			if _, err := exec.ExecContext(ctx, `
INSERT INTO account_lifecycle_events(account_id,event_type,occurred_at,reason,transferred_cost_minor,created_by_user_id,source)
VALUES($1,'refund',NOW(),$2,0,$3,'manual')`, accountID, note, options.ActorUserID); err != nil {
				return err
			}
		}
		_, err = exec.ExecContext(ctx, `
INSERT INTO account_lifecycle_events(account_id,event_type,occurred_at,reason,transferred_cost_minor,created_by_user_id,source)
VALUES($1,'retired',NOW(),$2,0,$3,'manual')`, accountID, note, options.ActorUserID)
		return err
	}
	if options.CostDisposition == "transfer" {
		if expectedTokenCount > 0 {
			if _, err = exec.ExecContext(ctx, `UPDATE accounts SET expected_token_count=COALESCE(expected_token_count,0)+$2,updated_at=NOW() WHERE id=$1`, *options.ReplacementAccountID, expectedTokenCount); err != nil {
				return err
			}
		}
		_, err = exec.ExecContext(ctx, `
INSERT INTO account_lifecycle_events(account_id,event_type,occurred_at,reason,replacement_account_id,transferred_cost_minor,created_by_user_id,source)
VALUES($1,'replaced',NOW(),$2,$3,$4,$5,'manual')`, accountID, note, *options.ReplacementAccountID, amountMinor, options.ActorUserID)
		return err
	}
	_, err = exec.ExecContext(ctx, `
INSERT INTO account_lifecycle_events(account_id,event_type,occurred_at,reason,transferred_cost_minor,created_by_user_id,source)
VALUES($1,'retired',NOW(),$2,0,$3,'manual')`, accountID, note, options.ActorUserID)
	return err
}

func loadAccountDeleteCostBuckets(ctx context.Context, exec accountDeleteExecutor, accountID int64) ([]accountDeleteCostBucket, error) {
	rows, err := exec.QueryContext(ctx, `
WITH priced AS (
  SELECT c.*,LEAD(c.paid_at) OVER (ORDER BY c.paid_at,c.id) next_paid_at
  FROM account_cost_entries c
  WHERE c.account_id=$1 AND c.cny_amount_minor>0 AND c.expected_token_count>0
)
SELECT priced.id,priced.payer_user_id,priced.purchase_source_id,priced.cny_amount_minor,
       priced.expected_token_count,priced.paid_at,priced.service_start,priced.service_end,
       COALESCE((SELECT SUM(ul.input_tokens::bigint+ul.output_tokens::bigint+ul.cache_creation_tokens::bigint+
         ul.cache_read_tokens::bigint+ul.image_output_tokens::bigint+ul.image_input_tokens::bigint)
         FROM usage_logs ul WHERE ul.account_id=$1 AND ul.created_at>=priced.paid_at
           AND (priced.next_paid_at IS NULL OR ul.created_at<priced.next_paid_at)),0)::bigint
FROM priced ORDER BY priced.paid_at,priced.id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("load account cost buckets: %w", err)
	}
	defer rows.Close()
	var tranches []service.AccountCostTranche
	for rows.Next() {
		var tranche service.AccountCostTranche
		var sourceID sql.NullInt64
		if err := rows.Scan(&tranche.ID, &tranche.PayerUserID, &sourceID, &tranche.CostMinor, &tranche.ExpectedTokens,
			&tranche.PaidAt, &tranche.ServiceStart, &tranche.ServiceEnd, &tranche.UsageTokens); err != nil {
			return nil, err
		}
		if sourceID.Valid {
			tranche.PurchaseSourceID = &sourceID.Int64
		}
		tranches = append(tranches, tranche)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(tranches) == 0 {
		return nil, infraerrors.Conflict("ACCOUNT_DELETE_COST_BASIS_MISSING", "remaining account cost has no positive payer/source balance")
	}
	states, _, _, _ := service.RecognizeAccountCostTranches(0, tranches)
	bucketByKey := make(map[string]*accountDeleteCostBucket)
	for _, state := range states {
		if state.RemainingCostMinor <= 0 {
			continue
		}
		sourceID := int64(0)
		if state.Tranche.PurchaseSourceID != nil {
			sourceID = *state.Tranche.PurchaseSourceID
		}
		key := fmt.Sprintf("%d:%d", state.Tranche.PayerUserID, sourceID)
		bucket := bucketByKey[key]
		if bucket == nil {
			bucket = &accountDeleteCostBucket{payerID: state.Tranche.PayerUserID, serviceFrom: state.Tranche.ServiceStart, serviceTo: state.Tranche.ServiceEnd}
			if state.Tranche.PurchaseSourceID != nil {
				bucket.sourceID = sql.NullInt64{Int64: sourceID, Valid: true}
			}
			bucketByKey[key] = bucket
		}
		bucket.balance += state.RemainingCostMinor
		bucket.tokens += state.RemainingTokens
		if state.Tranche.ServiceStart.Before(bucket.serviceFrom) {
			bucket.serviceFrom = state.Tranche.ServiceStart
		}
		if state.Tranche.ServiceEnd.After(bucket.serviceTo) {
			bucket.serviceTo = state.Tranche.ServiceEnd
		}
	}
	rows, err = exec.QueryContext(ctx, `
SELECT payer_user_id,purchase_source_id,-SUM(cny_amount_minor)::bigint
FROM account_cost_entries
WHERE account_id=$1 AND entry_type IN ('refund','replacement_out','write_off') AND cny_amount_minor<0
GROUP BY payer_user_id,purchase_source_id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("load account cost dispositions: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var payerID, disposed int64
		var sourceID sql.NullInt64
		if err := rows.Scan(&payerID, &sourceID, &disposed); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%d:%d", payerID, nullInt64Value(sourceID))
		if bucket := bucketByKey[key]; bucket != nil && disposed > 0 {
			before := bucket.balance
			bucket.balance -= disposed
			if bucket.balance < 0 {
				bucket.balance = 0
			}
			if before > 0 {
				bucket.tokens = decimal.NewFromInt(bucket.tokens).Mul(decimal.NewFromInt(bucket.balance)).Div(decimal.NewFromInt(before)).Floor().IntPart()
			}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	buckets := make([]accountDeleteCostBucket, 0, len(bucketByKey))
	for _, bucket := range bucketByKey {
		if bucket.balance > 0 {
			buckets = append(buckets, *bucket)
		}
	}
	sort.Slice(buckets, func(i, j int) bool {
		if buckets[i].payerID == buckets[j].payerID {
			return buckets[i].sourceID.Int64 < buckets[j].sourceID.Int64
		}
		return buckets[i].payerID < buckets[j].payerID
	})
	if len(buckets) == 0 {
		return nil, infraerrors.Conflict("ACCOUNT_DELETE_COST_BASIS_MISSING", "remaining account cost has no positive payer/source balance")
	}
	return buckets, nil
}

func insertAccountDeleteCostEntry(ctx context.Context, exec accountDeleteExecutor, accountID int64, bucket accountDeleteCostBucket, entryType string, amountMinor int64, relatedAccountID *int64, expectedTokenCount *int64, actorID int64, note string) error {
	sourceKey := int64(0)
	if bucket.sourceID.Valid {
		sourceKey = bucket.sourceID.Int64
	}
	operationKey := fmt.Sprintf("account-delete:%d:%s:%d:%d", accountID, entryType, bucket.payerID, sourceKey)
	if relatedAccountID != nil {
		operationKey += fmt.Sprintf(":%d", *relatedAccountID)
	}
	originalAmount := decimal.NewFromInt(amountMinor).Div(decimal.NewFromInt(100)).StringFixed(2)
	_, err := exec.ExecContext(ctx, `
	INSERT INTO account_cost_entries(
	  account_id,payer_user_id,purchase_source_id,entry_type,currency,original_amount,cny_amount_minor,
	  fx_rate,service_start,service_end,paid_at,note,related_account_id,expected_token_count,created_by_user_id,operation_key)
	VALUES($1,$2,$3,$4,'CNY',$5,$6,'1',$7,$8,NOW(),$9,$10,$11,$12,$13)`,
		accountID, bucket.payerID, nullInt64Value(bucket.sourceID), entryType, originalAmount, amountMinor,
		bucket.serviceFrom, bucket.serviceTo, note, relatedAccountID, expectedTokenCount, actorID, operationKey)
	return err
}

func allocateAccountDeleteAmounts(total int64, weights []int64) []int64 {
	result := make([]int64, len(weights))
	if total <= 0 {
		return result
	}
	weightTotal := int64(0)
	for _, weight := range weights {
		if weight > 0 {
			weightTotal += weight
		}
	}
	if weightTotal <= 0 {
		return result
	}
	type fraction struct {
		index int
		value decimal.Decimal
	}
	fractions := make([]fraction, 0, len(weights))
	assigned := int64(0)
	for i, weight := range weights {
		if weight <= 0 {
			continue
		}
		exact := decimal.NewFromInt(total).Mul(decimal.NewFromInt(weight)).Div(decimal.NewFromInt(weightTotal))
		result[i] = exact.Floor().IntPart()
		assigned += result[i]
		fractions = append(fractions, fraction{index: i, value: exact.Sub(decimal.NewFromInt(result[i]))})
	}
	sort.SliceStable(fractions, func(i, j int) bool {
		return fractions[i].value.GreaterThan(fractions[j].value)
	})
	for i := int64(0); i < total-assigned; i++ {
		result[fractions[int(i)%len(fractions)].index]++
	}
	return result
}

func cleanAccountDeleteBindings(ctx context.Context, exec accountDeleteExecutor, accountID int64) error {
	queries := []string{
		`DELETE FROM account_groups WHERE account_id=$1`,
		`DELETE FROM scheduled_test_plans WHERE account_id=$1`,
		`UPDATE channel_account_stats_pricing_rules SET account_ids=array_remove(account_ids,$1) WHERE $1=ANY(account_ids)`,
	}
	for _, query := range queries {
		if _, err := exec.ExecContext(ctx, query, accountID); err != nil {
			return err
		}
	}
	return nil
}

func archiveAccountForDelete(ctx context.Context, exec accountDeleteExecutor, accountID int64) error {
	if _, err := exec.ExecContext(ctx, `
UPDATE pool_approval_requests
SET status=CASE WHEN (status='pending' AND action_type<>'DELETE_ACCOUNT') OR (action_type='VIEW_CREDENTIAL' AND status='approved') THEN 'expired' ELSE status END,
    decision_reason=CASE WHEN (status='pending' AND action_type<>'DELETE_ACCOUNT') OR (action_type='VIEW_CREDENTIAL' AND status='approved')
      THEN 'account archived' ELSE decision_reason END,
    decided_at=CASE WHEN (status='pending' AND action_type<>'DELETE_ACCOUNT') OR (action_type='VIEW_CREDENTIAL' AND status='approved')
      THEN COALESCE(decided_at,NOW()) ELSE decided_at END,
    reveal_expires_at=NULL,
    payload=CASE WHEN action_type='DELETE_ACCOUNT' AND status='pending' THEN payload ELSE '{}'::jsonb END,
    updated_at=NOW()
WHERE account_id=$1`, accountID); err != nil {
		return err
	}
	if _, err := exec.ExecContext(ctx, `
UPDATE accounts SET deleted_at=COALESCE(deleted_at,NOW()),updated_at=NOW(),status='inactive',schedulable=FALSE,
  credentials='{}'::jsonb,
  extra=extra-'ollama_cloud_usage_session'-'upstream_billing_probe'-'model_rate_limits',
  proxy_id=NULL,proxy_fallback_origin_id=NULL,error_message='',rate_limited_at=NULL,rate_limit_reset_at=NULL,
  overload_until=NULL,temp_unschedulable_until=NULL,temp_unschedulable_reason=NULL,
  session_window_start=NULL,session_window_end=NULL,session_window_status=NULL
WHERE id=$1 AND deleted_at IS NULL`, accountID); err != nil {
		return err
	}
	_, err := exec.ExecContext(ctx, `INSERT INTO scheduler_outbox(event_type,account_id) VALUES($1,$2)`, service.SchedulerOutboxEventAccountChanged, accountID)
	return err
}

func physicallyDeleteEmptyAccount(ctx context.Context, exec accountDeleteExecutor, accountID int64) error {
	for _, query := range []string{
		`DELETE FROM pool_approval_requests WHERE account_id=$1`,
		`DELETE FROM accounts WHERE id=$1 AND deleted_at IS NULL`,
	} {
		result, err := exec.ExecContext(ctx, query, accountID)
		if err != nil {
			return err
		}
		if strings.HasPrefix(query, "DELETE FROM accounts") {
			affected, err := result.RowsAffected()
			if err != nil {
				return err
			}
			if affected == 0 {
				return service.ErrAccountNotFound
			}
		}
	}
	_, err := exec.ExecContext(ctx, `INSERT INTO scheduler_outbox(event_type,account_id) VALUES($1,$2)`, service.SchedulerOutboxEventAccountChanged, accountID)
	return err
}

func recordAccountRetired(ctx context.Context, exec accountDeleteExecutor, accountID int64, options service.AccountDeleteOptions) error {
	reason := options.Reason
	if reason == "" {
		reason = "account archived"
	}
	_, err := exec.ExecContext(ctx, `
INSERT INTO account_lifecycle_events(account_id,event_type,occurred_at,reason,transferred_cost_minor,created_by_user_id,source)
VALUES($1,'retired',NOW(),$2,0,$3,'manual')`, accountID, reason, options.ActorUserID)
	return err
}

func nullInt64Value(value sql.NullInt64) any {
	if value.Valid {
		return value.Int64
	}
	return nil
}
