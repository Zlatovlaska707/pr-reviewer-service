package addteam

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	CreateTeam(ctx context.Context, team domain.Team) (domain.Team, error)
}
