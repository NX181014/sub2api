package repository

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func decodePoolCostTranches(raw []byte) ([]service.AccountCostTranche, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var rows []struct {
		ID               int64  `json:"id"`
		CostMinor        int64  `json:"cost_minor"`
		ExpectedTokens   int64  `json:"expected_tokens"`
		PaidAt           string `json:"paid_at"`
		UsageTokens      int64  `json:"usage_tokens"`
		PayerUserID      int64  `json:"payer_user_id"`
		PurchaseSourceID *int64 `json:"purchase_source_id"`
		ServiceStart     string `json:"service_start"`
		ServiceEnd       string `json:"service_end"`
	}
	if err := json.Unmarshal(raw, &rows); err != nil {
		return nil, err
	}
	tranches := make([]service.AccountCostTranche, len(rows))
	for i, row := range rows {
		paidAt, err := parsePoolCostTime(row.PaidAt)
		if err != nil {
			return nil, fmt.Errorf("decode tranche %d paid_at: %w", row.ID, err)
		}
		serviceStart, err := parsePoolCostTime(row.ServiceStart)
		if err != nil {
			return nil, fmt.Errorf("decode tranche %d service_start: %w", row.ID, err)
		}
		serviceEnd, err := parsePoolCostTime(row.ServiceEnd)
		if err != nil {
			return nil, fmt.Errorf("decode tranche %d service_end: %w", row.ID, err)
		}
		tranches[i] = service.AccountCostTranche{
			ID: row.ID, CostMinor: row.CostMinor, ExpectedTokens: row.ExpectedTokens,
			PaidAt: paidAt, UsageTokens: row.UsageTokens, PayerUserID: row.PayerUserID,
			PurchaseSourceID: row.PurchaseSourceID, ServiceStart: serviceStart, ServiceEnd: serviceEnd,
		}
	}
	return tranches, nil
}

func parsePoolCostTime(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, nil
	}
	if parsed, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return parsed, nil
	}
	return time.Parse(time.DateOnly, value)
}
