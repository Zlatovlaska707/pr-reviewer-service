package nower

import "time"

// Nower предоставляет абстракцию для получения текущего времени.
// Это для легкого моканья времени в тестах.
type Nower interface {
	Now() time.Time
}
