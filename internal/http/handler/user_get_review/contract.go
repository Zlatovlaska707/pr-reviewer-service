package usergetreview

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	ListReviewAssignments(ctx context.Context, userID string) ([]domain.PullRequestShort, error)
}
