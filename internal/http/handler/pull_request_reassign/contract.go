package pullrequestreassign

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	ReassignReviewer(ctx context.Context, prID, oldReviewer string) (domain.PullRequest, string, error)
}
