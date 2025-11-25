package pullrequestmerge

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	MergePullRequest(ctx context.Context, prID string) (domain.PullRequest, error)
}
