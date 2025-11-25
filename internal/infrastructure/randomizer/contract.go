package randomizer

// Randomizer предоставляет абстракцию для рандомизации.
type Randomizer interface {
	Shuffle(n int, swap func(i, j int))
}
