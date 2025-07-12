package noise

import (
	"github.com/aquilax/go-perlin"
)

type Generator struct {
	noise *perlin.Perlin
	seed  int64
}

func NewGenerator(seed int64) *Generator {
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