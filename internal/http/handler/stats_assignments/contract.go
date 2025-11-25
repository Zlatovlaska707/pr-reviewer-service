package statsassignments

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	Stats(ctx context.Context) (domain.AssignmentStats, error)
}
