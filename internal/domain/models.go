package domain

import "time"

// PRStatus отражает возможные состояния PR.
type PRStatus string

const (
	PRStatusOpen   PRStatus = "OPEN"
	PRStatusMerged PRStatus = "MERGED"
)

// Team описывает команду и её участников.
type Team struct {
	Name    string `json:"team_name"`
	Members []User `json:"members"`
}

// User представляет участника команды.
type User struct {
	ID       string `json:"user_id"`
	Username string `json:"username"`
	TeamName string `json:"team_name"`
	IsActive bool   `json:"is_active"`
}

// PullRequest содержит данные PR.
type PullRequest struct {
	ID                string     `json:"pull_request_id"`
	Name              string     `json:"pull_request_name"`
	AuthorID          string     `json:"author_id"`
	Status            PRStatus   `json:"status"`
	AssignedReviewers []string   `json:"assigned_reviewers"`
	CreatedAt         time.Time  `json:"createdAt"`
	MergedAt          *time.Time `json:"mergedAt,omitempty"`
}

// PullRequestShort используется там, где достаточно укороченного представления.
type PullRequestShort struct {
	ID       string   `json:"pull_request_id"`
	Name     string   `json:"pull_request_name"`
	AuthorID string   `json:"author_id"`
	Status   PRStatus `json:"status"`
}

// AssignmentStats содержит метрики по назначениям.
type AssignmentStats struct {
	PerUser []UserAssignmentStat `json:"per_user"`
	PerPR   []PRAssignmentStat   `json:"per_pull_request"`
}

// UserAssignmentStat хранит информацию о количестве назначений конкретного пользователя.
type UserAssignmentStat struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	TeamName  string `json:"team_name"`
	Assigned  int64  `json:"assigned_total"`
	ActivePRs int64  `json:"active_pull_requests"`
}

// PRAssignmentStat описывает статистику по PR.
type PRAssignmentStat struct {
	PullRequestID string `json:"pull_request_id"`
	ReviewerCount int64  `json:"reviewer_count"`
}
