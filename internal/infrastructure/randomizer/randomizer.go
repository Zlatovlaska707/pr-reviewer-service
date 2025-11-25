package randomizer

import (
	"math/rand"
	"sync"
	"time"
)

type randomizerImpl struct {
	mu  sync.Mutex // Защищает доступ к генератору случайных чисел
	rnd *rand.Rand
}

// New создаёт потокобезопасный randomizer на основе math/rand.
// Использует псевдослучайный генератор, что подходит для бизнес-логики (выбор ревьюверов).
func New() Randomizer {
	return &randomizerImpl{
		rnd: rand.New(rand.NewSource(time.Now().UnixNano())), // #nosec G404
	}
}

// Shuffle перемешивает элементы используя Fisher-Yates shuffle алгоритм.
// Потокобезопасна благодаря мьютексу.
func (r *randomizerImpl) Shuffle(n int, swap func(i, j int)) {
	if n <= 1 {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rnd.Shuffle(n, swap)
}
