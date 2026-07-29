package repository

import (
	"encoding/json"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func decodePoolCostTranches(raw []byte) ([]service.AccountCostTranche, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var tranches []service.AccountCostTranche
	if err := json.Unmarshal(raw, &tranches); err != nil {
		return nil, err
	}
	return tranches, nil
}
