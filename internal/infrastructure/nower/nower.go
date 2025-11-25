package nower

import "time"

type nowerImpl struct{}

// New создаёт реализацию на базе системных часов.
func New() Nower {
	return &nowerImpl{}
}

// Now возвращает текущее системное время.
func (n *nowerImpl) Now() time.Time {
	return time.Now()
}
