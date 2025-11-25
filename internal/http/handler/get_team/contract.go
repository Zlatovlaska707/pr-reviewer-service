package getteam

import (
	"context"

	"pr-reviewer-service_Avito/internal/domain"
)

type UseCase interface {
	GetTeam(ctx context.Context, name string) (domain.Team, error)
}
