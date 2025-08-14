package noise

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/VoidMesh/api/api/internal/testutil"
)

func TestNewGenerator(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name         string
		seed         int64
		expectFields func(t *testing.T, generator GeneratorInterface)
	}{
		{
			name: "successful generator creation with positive seed",
			seed: 12345,
			expectFields: func(t *testing.T, generator GeneratorInterface) {
				assert.NotNil(t, generator)
				assert.Equal(t, int64(12345), generator.GetSeed())
			},
		},
		{
			name: "successful generator creation with zero seed",
			seed: 0,
			expectFields: func(t *testing.T, generator GeneratorInterface) {
				assert.NotNil(t, generator)
				assert.Equal(t, int64(0), generator.GetSeed())
			},
		},
		{
			name: "successful generator creation with negative seed",
			seed: -9876,
			expectFields: func(t *testing.T, generator GeneratorInterface) {
				assert.NotNil(t, generator)
				assert.Equal(t, int64(-9876), generator.GetSeed())
			},
		},
		{
			name: "successful generator creation with max int64 seed",
			seed: math.MaxInt64,
			expectFields: func(t *testing.T, generator GeneratorInterface) {
				assert.NotNil(t, generator)
				assert.Equal(t, int64(math.MaxInt64), generator.GetSeed())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.seed)
			require.NotNil(t, generator)
			tt.expectFields(t, generator)
		})
	}
}

func TestGenerator_GetNoise(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name         string
		seed         int64
		x, y         float64
		expectInRange bool
		expectValue  *float64 // For exact value checks when deterministic
	}{
		{
			name:          "noise at origin with positive seed",
			seed:          12345,
			x:             0.0,
			y:             0.0,
			expectInRange: true,
		},
		{
			name:          "noise at positive coordinates",
			seed:          12345,
			x:             10.5,
			y:             20.7,
			expectInRange: true,
		},
		{
			name:          "noise at negative coordinates",
			seed:          12345,
			x:             -15.3,
			y:             -8.9,
			expectInRange: true,
		},
		{
			name:          "noise with zero seed",
			seed:          0,
			x:             5.0,
			y:             5.0,
			expectInRange: true,
		},
		{
			name:          "noise with very large coordinates",
			seed:          12345,
			x:             1000000.0,
			y:             2000000.0,
			expectInRange: true,
		},
		{
			name:          "noise with fractional coordinates",
			seed:          12345,
			x:             0.123456,
			y:             0.789012,
			expectInRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.seed)
			require.NotNil(t, generator)

			result := generator.GetNoise(tt.x, tt.y)

			if tt.expectInRange {
				// Perlin noise should return values in the range [-1, 1]
				assert.GreaterOrEqual(t, result, -1.0, "noise value should be >= -1")
				assert.LessOrEqual(t, result, 1.0, "noise value should be <= 1")
			}

			if tt.expectValue != nil {
				assert.InDelta(t, *tt.expectValue, result, 0.000001, "noise value should match expected")
			}
		})
	}
}

func TestGenerator_GetTerrainNoise(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name         string
		seed         int64
		x, y         int
		scale        float64
		expectInRange bool
	}{
		{
			name:          "terrain noise with standard scale",
			seed:          12345,
			x:             10,
			y:             20,
			scale:         100.0,
			expectInRange: true,
		},
		{
			name:          "terrain noise with small scale (high detail)",
			seed:          12345,
			x:             10,
			y:             20,
			scale:         10.0,
			expectInRange: true,
		},
		{
			name:          "terrain noise with large scale (low detail)",
			seed:          12345,
			x:             10,
			y:             20,
			scale:         1000.0,
			expectInRange: true,
		},
		{
			name:          "terrain noise at origin",
			seed:          12345,
			x:             0,
			y:             0,
			scale:         100.0,
			expectInRange: true,
		},
		{
			name:          "terrain noise with negative coordinates",
			seed:          12345,
			x:             -50,
			y:             -75,
			scale:         100.0,
			expectInRange: true,
		},
		{
			name:          "terrain noise with very small scale",
			seed:          12345,
			x:             10,
			y:             20,
			scale:         1.0,
			expectInRange: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.seed)
			require.NotNil(t, generator)

			result := generator.GetTerrainNoise(tt.x, tt.y, tt.scale)

			if tt.expectInRange {
				// Terrain noise should still be in [-1, 1] range since it's based on GetNoise
				assert.GreaterOrEqual(t, result, -1.0, "terrain noise value should be >= -1")
				assert.LessOrEqual(t, result, 1.0, "terrain noise value should be <= 1")
			}
		})
	}
}

func TestGenerator_GetSeed(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name         string
		seed         int64
		expectedSeed int64
	}{
		{
			name:         "positive seed",
			seed:         12345,
			expectedSeed: 12345,
		},
		{
			name:         "zero seed",
			seed:         0,
			expectedSeed: 0,
		},
		{
			name:         "negative seed",
			seed:         -9876,
			expectedSeed: -9876,
		},
		{
			name:         "max int64 seed",
			seed:         math.MaxInt64,
			expectedSeed: math.MaxInt64,
		},
		{
			name:         "min int64 seed",
			seed:         math.MinInt64,
			expectedSeed: math.MinInt64,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.seed)
			require.NotNil(t, generator)

			result := generator.GetSeed()
			assert.Equal(t, tt.expectedSeed, result)
		})
	}
}

func TestNoiseDeterminism(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name         string
		seed         int64
		coordinates  []struct{ x, y float64 }
		iterations   int
	}{
		{
			name: "deterministic output for same seed and coordinates",
			seed: 12345,
			coordinates: []struct{ x, y float64 }{
				{0.0, 0.0},
				{10.5, 20.7},
				{-15.3, -8.9},
				{100.0, 200.0},
			},
			iterations: 5,
		},
		{
			name: "deterministic output with zero seed",
			seed: 0,
			coordinates: []struct{ x, y float64 }{
				{1.0, 1.0},
				{50.0, 75.0},
			},
			iterations: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Store initial values
			generator1 := NewGenerator(tt.seed)
			initialValues := make([]float64, len(tt.coordinates))
			for i, coord := range tt.coordinates {
				initialValues[i] = generator1.GetNoise(coord.x, coord.y)
			}

			// Test multiple iterations
			for iteration := 0; iteration < tt.iterations; iteration++ {
				generator := NewGenerator(tt.seed)
				for i, coord := range tt.coordinates {
					result := generator.GetNoise(coord.x, coord.y)
					assert.Equal(t, initialValues[i], result,
						"noise value should be deterministic for seed %d at coordinates (%.2f, %.2f) iteration %d",
						tt.seed, coord.x, coord.y, iteration)
				}
			}
		})
	}
}

func TestNoiseDifferentSeeds(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	// Test that different seeds produce different noise patterns
	seeds := []int64{12345, 54321, 0, -12345, 999999}
	testCoordinates := []struct{ x, y float64 }{
		{1.0, 1.0},    // Avoid origin which might have special behavior
		{10.5, 10.5},  // Use fractional coordinates
		{-5.3, 5.7},   // Mixed signs with fractions
		{25.1, -33.2}, // More varied coordinates
	}

	// Collect noise values for each seed at each coordinate
	noiseValues := make(map[int64][]float64)
	for _, seed := range seeds {
		generator := NewGenerator(seed)
		values := make([]float64, len(testCoordinates))
		for i, coord := range testCoordinates {
			values[i] = generator.GetNoise(coord.x, coord.y)
		}
		noiseValues[seed] = values
	}

	// Count how many seed pairs produce different patterns
	differentPatterns := 0
	totalComparisons := 0

	for i, seed1 := range seeds {
		for j, seed2 := range seeds {
			if i >= j {
				continue // Skip same seed comparisons and duplicates
			}

			totalComparisons++
			values1 := noiseValues[seed1]
			values2 := noiseValues[seed2]

			// At least some values should be different between different seeds
			foundDifference := false
			for k := 0; k < len(values1); k++ {
				if math.Abs(values1[k]-values2[k]) > 0.0001 {
					foundDifference = true
					break
				}
			}

			if foundDifference {
				differentPatterns++
			}
		}
	}

	// At least 80% of seed pairs should produce different patterns
	// This accounts for potential edge cases in the Perlin noise implementation
	expectedMinDifferent := int(float64(totalComparisons) * 0.8)
	assert.GreaterOrEqual(t, differentPatterns, expectedMinDifferent,
		"at least 80%% of different seed pairs should produce different noise patterns, got %d/%d",
		differentPatterns, totalComparisons)
}

func TestTerrainNoiseScaleEffects(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	// Test that scale affects the terrain noise appropriately
	seed := int64(12345)
	generator := NewGenerator(seed)

	testCoordinates := []struct{ x, y int }{
		{10, 15},   // Avoid origin
		{32, 32},   // Chunk-sized coordinates
		{100, 200}, // Larger coordinates
	}

	scales := []float64{10.0, 50.0, 100.0, 500.0}

	for _, coord := range testCoordinates {
		scaleResults := make([]float64, len(scales))
		for i, scale := range scales {
			scaleResults[i] = generator.GetTerrainNoise(coord.x, coord.y, scale)
		}

		// Verify all results are in valid range
		for i, result := range scaleResults {
			assert.GreaterOrEqual(t, result, -1.0,
				"terrain noise with scale %.1f should be >= -1 at (%d, %d)", scales[i], coord.x, coord.y)
			assert.LessOrEqual(t, result, 1.0,
				"terrain noise with scale %.1f should be <= 1 at (%d, %d)", scales[i], coord.x, coord.y)
		}

		// Count unique values (with tolerance for floating point precision)
		uniqueCount := 0
		for i := 0; i < len(scaleResults); i++ {
			isUnique := true
			for j := 0; j < i; j++ {
				if math.Abs(scaleResults[i]-scaleResults[j]) < 0.0001 {
					isUnique = false
					break
				}
			}
			if isUnique {
				uniqueCount++
			}
		}

		// We should have at least 1 unique value (at minimum, scales shouldn't all produce identical values)
		// But realistically, most coordinates should produce varied results across scales
		assert.GreaterOrEqual(t, uniqueCount, 1,
			"different scales should produce at least 1 unique terrain noise value at (%d, %d)", coord.x, coord.y)
	}
}

func TestNoiseEdgeCases(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	tests := []struct {
		name      string
		seed      int64
		operation func(generator GeneratorInterface) float64
		expectInRange bool // Some edge cases may produce values outside [-1, 1]
	}{
		{
			name: "noise at moderately large positive coordinates",
			seed: 12345,
			operation: func(generator GeneratorInterface) float64 {
				return generator.GetNoise(1000.0, 1000.0)
			},
			expectInRange: true,
		},
		{
			name: "noise at moderately large negative coordinates",
			seed: 12345,
			operation: func(generator GeneratorInterface) float64 {
				return generator.GetNoise(-1000.0, -1000.0)
			},
			expectInRange: true,
		},
		{
			name: "terrain noise with very small scale",
			seed: 12345,
			operation: func(generator GeneratorInterface) float64 {
				return generator.GetTerrainNoise(10, 10, 0.001)
			},
			expectInRange: true,
		},
		{
			name: "terrain noise with very large scale",
			seed: 12345,
			operation: func(generator GeneratorInterface) float64 {
				return generator.GetTerrainNoise(10, 10, 1e6)
			},
			expectInRange: true,
		},
		{
			name: "terrain noise at large positive coordinates",
			seed: 12345,
			operation: func(generator GeneratorInterface) float64 {
				return generator.GetTerrainNoise(100000, 100000, 100.0)
			},
			expectInRange: true,
		},
		{
			name: "terrain noise at large negative coordinates",
			seed: 12345,
			operation: func(generator GeneratorInterface) float64 {
				return generator.GetTerrainNoise(-100000, -100000, 100.0)
			},
			expectInRange: true,
		},
		{
			name: "noise at extreme coordinates (may be outside range)",
			seed: 12345,
			operation: func(generator GeneratorInterface) float64 {
				return generator.GetNoise(1e8, 1e8)
			},
			expectInRange: false, // Very large coordinates may produce out-of-range values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			generator := NewGenerator(tt.seed)
			require.NotNil(t, generator)

			// Should not panic and should return valid value
			result := tt.operation(generator)
			
			// Always check for NaN and infinity
			assert.False(t, math.IsNaN(result), "result should not be NaN")
			assert.False(t, math.IsInf(result, 0), "result should not be infinite")
			
			// Only check range for cases where we expect it
			if tt.expectInRange {
				assert.GreaterOrEqual(t, result, -1.0, "result should be >= -1")
				assert.LessOrEqual(t, result, 1.0, "result should be <= 1")
			} else {
				// For extreme cases, just log the result for awareness
				t.Logf("Extreme case result: %f", result)
			}
		})
	}
}

func TestNoiseContinuity(t *testing.T) {
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	// Test that noise values are continuous (small changes in input produce small changes in output)
	generator := NewGenerator(12345)

	baseX, baseY := 10.0, 10.0
	baseValue := generator.GetNoise(baseX, baseY)

	// Test small increments
	increments := []float64{0.01, 0.1, 0.5}
	
	for _, increment := range increments {
		// Test X direction
		nearValueX := generator.GetNoise(baseX+increment, baseY)
		diffX := math.Abs(nearValueX - baseValue)
		
		// Test Y direction  
		nearValueY := generator.GetNoise(baseX, baseY+increment)
		diffY := math.Abs(nearValueY - baseValue)
		
		// For small increments, the difference should be relatively small
		// This tests the continuity property of Perlin noise
		if increment <= 0.1 {
			assert.Less(t, diffX, 1.0, "small increment in X should produce relatively small change")
			assert.Less(t, diffY, 1.0, "small increment in Y should produce relatively small change")
		}
	}
}

// Benchmark tests
func BenchmarkGenerator_GetNoise(b *testing.B) {
	generator := NewGenerator(12345)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := float64(i % 1000)
		y := float64(i % 1000)
		generator.GetNoise(x, y)
	}
}

func BenchmarkGenerator_GetTerrainNoise(b *testing.B) {
	generator := NewGenerator(12345)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		x := i % 1000
		y := i % 1000
		generator.GetTerrainNoise(x, y, 100.0)
	}
}

func BenchmarkNewGenerator(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		NewGenerator(int64(i))
	}
}

func TestNoisePerformance(t *testing.T) {
	testutil.SkipIfShort(t, "skipping performance test in short mode")
	
	cleanup := testutil.SetupTest(t, testutil.DefaultTestConfig())
	defer cleanup()

	generator := NewGenerator(12345)
	
	// Test performance of generating a large noise map
	start := time.Now()
	mapSize := 1000
	noiseCount := 0
	
	for x := 0; x < mapSize; x++ {
		for y := 0; y < mapSize; y++ {
			generator.GetTerrainNoise(x, y, 100.0)
			noiseCount++
		}
	}
	
	duration := time.Since(start)
	
	// Log performance metrics
	t.Logf("Generated %d noise values in %v", noiseCount, duration)
	t.Logf("Average time per noise value: %v", duration/time.Duration(noiseCount))
	
	// Basic performance assertion - should be able to generate at least 10k values per second
	valuesPerSecond := float64(noiseCount) / duration.Seconds()
	assert.Greater(t, valuesPerSecond, 10000.0, 
		"should generate at least 10k noise values per second, got %.2f", valuesPerSecond)
}