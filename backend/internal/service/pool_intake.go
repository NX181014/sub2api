package service

import (
	"context"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

// CreateAccountIntakeInput is the one-shot record attached to a newly created
// account. Credentials remain owned by the existing account API.
type CreateAccountIntakeInput struct {
	AccountID          int64
	ProviderIdentity   string
	ContributorUserID  int64
	UploaderUserID     int64
	PurchaseSourceName string
	SourceWebsiteURL   *string
	Cost               CreateAccountCostInput
	ActorUserID        int64
}

type AccountIntakeResult struct {
	Account PoolAccount      `json:"account"`
	Source  PurchaseSource   `json:"source"`
	Cost    AccountCostEntry `json:"cost"`
}

func (s *PoolService) CreateAccountIntake(ctx context.Context, input CreateAccountIntakeInput) (*AccountIntakeResult, error) {
	input.ProviderIdentity = strings.TrimSpace(input.ProviderIdentity)
	input.PurchaseSourceName = normalizePoolName(input.PurchaseSourceName)
	if input.AccountID <= 0 || input.ContributorUserID <= 0 || input.UploaderUserID <= 0 || input.ActorUserID <= 0 {
		return nil, infraerrors.BadRequest("INVALID_POOL_INTAKE_PARTY", "account, contributor, uploader and actor are required")
	}
	if input.ProviderIdentity == "" || len(input.ProviderIdentity) > 255 {
		return nil, infraerrors.BadRequest("INVALID_PROVIDER_IDENTITY", "provider identity is required and must not exceed 255 characters")
	}
	if input.PurchaseSourceName == "" || len(input.PurchaseSourceName) > 100 {
		return nil, infraerrors.BadRequest("INVALID_SOURCE_NAME", "source name is required and must not exceed 100 characters")
	}
	if input.Cost.ExpectedTokenCount == nil || *input.Cost.ExpectedTokenCount <= 0 {
		return nil, infraerrors.BadRequest("INVALID_EXPECTED_TOKEN_COUNT", "expected_token_count must be positive")
	}
	input.Cost.AccountID = input.AccountID
	input.Cost.PayerUserID = input.ContributorUserID
	input.Cost.CreatedByUserID = input.ActorUserID
	input.Cost.PurchaseSourceID = nil
	normalizedCost, err := normalizePoolCostInput(input.Cost)
	if err != nil {
		return nil, err
	}
	input.Cost = normalizedCost
	return s.repo.CreateAccountIntake(ctx, input)
}

func normalizePoolName(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}
