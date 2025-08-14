package noise

import (
	"github.com/aquilax/go-perlin"
)

// GeneratorInterface defines the interface for noise generation operations.
// This enables dependency injection and makes services easily testable.
type GeneratorInterface interface {
	GetNoise(x, y float64) float64
	GetTerrainNoise(x, y int, scale float64) float64
	GetSeed() int64
}

// Generator implements the GeneratorInterface using Perlin noise.
type Generator struct {
	noise *perlin.Perlin
	seed  int64
}

// NewGenerator creates a new noise generator with the given seed.
func NewGenerator(seed int64) GeneratorInterface {
	// Create perlin noise with alpha=2, beta=2, n=3
	// These values give good terrain-like noise
	return &Generator{
		noise: perlin.NewPerlin(2, 2, 3, seed),
		seed:  seed,
	}
}

// GetNoise returns a noise value between -1 and 1 for the given coordinates
func (g *Generator) GetNoise(x, y float64) float64 {
	return g.noise.Noise2D(x, y)
}

// GetTerrainNoise returns noise values suitable for terrain generation
// Scale controls the "zoom" level - higher values = more detailed terrain
func (g *Generator) GetTerrainNoise(x, y int, scale float64) float64 {
	fx := float64(x) / scale
	fy := float64(y) / scale
	return g.GetNoise(fx, fy)
}

// GetSeed returns the current seed
func (g *Generator) GetSeed() int64 {
	return g.seed
}
