package common

import (
	"testing"

	"github.com/stretchr/testify/require"

	"pr-reviewer-service_Avito/internal/api"
)

func TestTeamMappingRoundtrip(t *testing.T) {
	apiTeam := api.Team{
		TeamName: "backend",
		Members: []api.TeamMember{
			{UserId: "u1", Username: "Alice", IsActive: true},
		},
	}
	domainTeam := ToDomainTeam(apiTeam)
	require.Equal(t, "backend", domainTeam.Name)
	require.Len(t, domainTeam.Members, 1)
	back := FromDomainTeam(domainTeam)
	require.Equal(t, apiTeam.TeamName, back.TeamName)
	require.Len(t, back.Members, 1)
}
