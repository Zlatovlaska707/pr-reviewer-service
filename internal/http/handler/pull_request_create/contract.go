package pullrequestcreate

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	CreatePullRequest(ctx context.Context, prID, name, authorID string) (domain.PullRequest, error)
}
