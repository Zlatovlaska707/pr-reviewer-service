package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	teamsCreated = promauto.NewCounter(
		prometheusCounterOpts("teams_created_total", "Total number of created teams"),
	)
	usersProcessed = promauto.NewCounter(
		prometheusCounterOpts("users_processed_total", "Total number of users processed via API"),
	)
	prCreated = promauto.NewCounter(
		prometheusCounterOpts("pull_requests_created_total", "Total number of created pull requests"),
	)
	reassignments = promauto.NewCounter(
		prometheusCounterOpts("reviewer_reassignments_total", "Total reviewer reassignments"),
	)
)

// IncTeamsCreated увеличивает счётчик созданных команд.
func IncTeamsCreated() {
	teamsCreated.Inc()
}

// AddUsersProcessed увеличивает счётчик обработанных пользователей.
func AddUsersProcessed(delta int) {
	if delta <= 0 {
		return
	}
	usersProcessed.Add(float64(delta))
}

// IncPullRequestsCreated увеличивает счётчик созданных PR.
func IncPullRequestsCreated() {
	prCreated.Inc()
}

// IncReassignments увеличивает счётчик выполненных переназначений.
func IncReassignments() {
	reassignments.Inc()
}

func prometheusCounterOpts(name, help string) prometheus.CounterOpts {
	return prometheus.CounterOpts{
		Name: name,
		Help: help,
	}
}
