package domain

import "errors"

// Доменные ошибки, используемые для обработки бизнес-логики.
// Эти ошибки преобразуются в HTTP-ответы в слое обработчиков.
var (
	ErrTeamExists     = errors.New("team already exists")                   // Возникает при попытке создать команду, которая уже существует.
	ErrTeamNotFound   = errors.New("team not found")                        // Возникает при попытке получить несуществующую команду.
	ErrUserNotFound   = errors.New("user not found")                        // Возникает при попытке получить несуществующего пользователя.
	ErrPRExists       = errors.New("pull request already exists")           // Возникает при попытке создать PR с уже существующим ID.
	ErrPRNotFound     = errors.New("pull request not found")                // Возникает при попытке получить несуществующий PR.
	ErrPRMerged       = errors.New("pull request already merged")           // Возникает при попытке выполнить операцию над уже смерженным PR.
	ErrReviewerAbsent = errors.New("reviewer not assigned to pull request") // Возникает при попытке переназначить ревьювера, который не назначен на PR.
	ErrNoCandidate    = errors.New("no candidate available")                // Возникает когда нет доступных кандидатов для назначения ревьювером.
)
