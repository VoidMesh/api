package resource_node

import (
	"math/rand"
)

// RandomGenerator implements RandomGeneratorInterface using math/rand.
type RandomGenerator struct {
	rand *rand.Rand
}

// NewRandomGenerator creates a new random generator with the given seed.
func NewRandomGenerator(seed int64) RandomGeneratorInterface {
	source := rand.NewSource(seed)
	return &RandomGenerator{
		rand: rand.New(source),
	}
}

func (r *RandomGenerator) Intn(n int) int {
	return r.rand.Intn(n)
}

func (r *RandomGenerator) Int31n(n int32) int32 {
	return r.rand.Int31n(n)
}

func (r *RandomGenerator) Float32() float32 {
	return r.rand.Float32()
}

func (r *RandomGenerator) Shuffle(n int, swap func(i, j int)) {
	r.rand.Shuffle(n, swap)
}