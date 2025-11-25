package randomizer

import (
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShuffleKeepsAllElements(t *testing.T) {
	r := New()
	values := []int{1, 2, 3, 4}
	r.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})

	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	require.Equal(t, []int{1, 2, 3, 4}, sorted)
}

func TestShuffleNoopForShortSlices(t *testing.T) {
	r := New()
	values := []int{1}
	r.Shuffle(len(values), func(i, j int) {
		values[i], values[j] = values[j], values[i]
	})
	require.Equal(t, []int{1}, values)
}
