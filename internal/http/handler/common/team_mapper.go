package common

import (
	"pr-reviewer-service_Avito/internal/api"
	"pr-reviewer-service_Avito/internal/domain"
)

// ToDomainTeam преобразует API DTO to domain model.
func ToDomainTeam(req api.Team) domain.Team {
	members := make([]domain.User, 0, len(req.Members))
	for _, member := range req.Members {
		members = append(members, domain.User{
			ID:       member.UserId,
			Username: member.Username,
			IsActive: member.IsActive,
			TeamName: req.TeamName,
		})
	}
	return domain.Team{
		Name:    req.TeamName,
		Members: members,
	}
}

// FromDomainTeam преобразует domain entity в API DTO.
func FromDomainTeam(team domain.Team) api.Team {
	members := make([]api.TeamMember, 0, len(team.Members))
	for _, member := range team.Members {
		members = append(members, api.TeamMember{
			UserId:   member.ID,
			Username: member.Username,
			IsActive: member.IsActive,
		})
	}
	return api.Team{
		TeamName: team.Name,
		Members:  members,
	}
}
