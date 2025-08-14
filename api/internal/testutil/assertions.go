package testutil

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// AssertUUIDEqual compares two UUID values, handling different UUID types
func AssertUUIDEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	expectedStr := normalizeUUID(t, expected)
	actualStr := normalizeUUID(t, actual)

	assert.Equal(t, expectedStr, actualStr, msgAndArgs...)
}

// AssertUUIDNotEmpty verifies that a UUID is not empty/nil
func AssertUUIDNotEmpty(t *testing.T, uuid interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	uuidStr := normalizeUUID(t, uuid)
	assert.NotEmpty(t, uuidStr, msgAndArgs...)
	assert.NotEqual(t, "00000000-0000-0000-0000-000000000000", uuidStr, msgAndArgs...)
}

// normalizeUUID converts various UUID types to string for comparison
func normalizeUUID(t *testing.T, uuid interface{}) string {
	t.Helper()

	switch v := uuid.(type) {
	case string:
		return v
	case pgtype.UUID:
		if !v.Valid {
			return ""
		}
		return fmt.Sprintf("%x-%x-%x-%x-%x", v.Bytes[0:4], v.Bytes[4:6], v.Bytes[6:8], v.Bytes[8:10], v.Bytes[10:16])
	case []byte:
		if len(v) != 16 {
			return ""
		}
		return fmt.Sprintf("%x-%x-%x-%x-%x", v[0:4], v[4:6], v[6:8], v[8:10], v[10:16])
	default:
		// Try to convert to string
		str := fmt.Sprintf("%v", uuid)
		return str
	}
}

// AssertTimestampEqual compares two timestamp values, handling different timestamp types
func AssertTimestampEqual(t *testing.T, expected, actual interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	expectedTime := normalizeTimestamp(t, expected)
	actualTime := normalizeTimestamp(t, actual)

	// Allow for small differences in time comparison (within 1 second)
	timeDiff := expectedTime.Sub(actualTime)
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	assert.True(t, timeDiff < time.Second,
		"Timestamps differ by more than 1 second: expected=%v, actual=%v, diff=%v %v",
		expectedTime, actualTime, timeDiff, msgAndArgs)
}

// AssertTimestampRecent verifies that a timestamp is within a recent time range
func AssertTimestampRecent(t *testing.T, timestamp interface{}, maxAge time.Duration, msgAndArgs ...interface{}) {
	t.Helper()

	ts := normalizeTimestamp(t, timestamp)
	now := time.Now()
	age := now.Sub(ts)

	assert.True(t, age >= 0, "Timestamp is in the future: %v %v", ts, msgAndArgs)
	assert.True(t, age <= maxAge, "Timestamp is too old: %v (age: %v, max: %v) %v", ts, age, maxAge, msgAndArgs)
}

// normalizeTimestamp converts various timestamp types to time.Time
func normalizeTimestamp(t *testing.T, timestamp interface{}) time.Time {
	t.Helper()

	switch v := timestamp.(type) {
	case time.Time:
		return v
	case pgtype.Timestamp:
		if !v.Valid {
			return time.Time{}
		}
		return v.Time
	case string:
		// Try to parse common timestamp formats
		formats := []string{
			time.RFC3339,
			time.RFC3339Nano,
			"2006-01-02T15:04:05Z",
			"2006-01-02 15:04:05",
		}

		for _, format := range formats {
			if parsed, err := time.Parse(format, v); err == nil {
				return parsed
			}
		}

		require.Fail(t, "Unable to parse timestamp string", "Invalid timestamp format: %s", v)
		return time.Time{}
	default:
		require.Fail(t, "Unsupported timestamp type", "Type: %T, Value: %v", timestamp, timestamp)
		return time.Time{}
	}
}

// AssertSliceContains verifies that a slice contains a specific element
func AssertSliceContains(t *testing.T, slice interface{}, element interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	sliceValue := reflect.ValueOf(slice)
	require.True(t, sliceValue.Kind() == reflect.Slice, "First argument must be a slice")

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), element) {
			return // Found the element
		}
	}

	assert.Fail(t, "Slice does not contain element",
		"Slice: %v\nElement: %v\n%v", slice, element, msgAndArgs)
}

// AssertSliceNotContains verifies that a slice does not contain a specific element
func AssertSliceNotContains(t *testing.T, slice interface{}, element interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	sliceValue := reflect.ValueOf(slice)
	require.True(t, sliceValue.Kind() == reflect.Slice, "First argument must be a slice")

	for i := 0; i < sliceValue.Len(); i++ {
		if reflect.DeepEqual(sliceValue.Index(i).Interface(), element) {
			assert.Fail(t, "Slice contains element but should not",
				"Slice: %v\nElement: %v\n%v", slice, element, msgAndArgs)
			return
		}
	}
}

// AssertSliceLength verifies that a slice has the expected length
func AssertSliceLength(t *testing.T, slice interface{}, expectedLength int, msgAndArgs ...interface{}) {
	t.Helper()

	sliceValue := reflect.ValueOf(slice)
	require.True(t, sliceValue.Kind() == reflect.Slice, "Argument must be a slice")

	assert.Equal(t, expectedLength, sliceValue.Len(), msgAndArgs...)
}

// AssertMapContainsKey verifies that a map contains a specific key
func AssertMapContainsKey(t *testing.T, m interface{}, key interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	mapValue := reflect.ValueOf(m)
	require.True(t, mapValue.Kind() == reflect.Map, "First argument must be a map")

	keyValue := reflect.ValueOf(key)
	mapKey := mapValue.MapIndex(keyValue)

	assert.True(t, mapKey.IsValid(), "Map does not contain key: %v %v", key, msgAndArgs)
}

// AssertMapContainsValue verifies that a map contains a specific value
func AssertMapContainsValue(t *testing.T, m interface{}, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	mapValue := reflect.ValueOf(m)
	require.True(t, mapValue.Kind() == reflect.Map, "First argument must be a map")

	for _, key := range mapValue.MapKeys() {
		mapVal := mapValue.MapIndex(key)
		if reflect.DeepEqual(mapVal.Interface(), value) {
			return // Found the value
		}
	}

	assert.Fail(t, "Map does not contain value",
		"Map: %v\nValue: %v\n%v", m, value, msgAndArgs)
}

// AssertStringMatches verifies that a string matches a pattern using assert.Regexp
func AssertStringMatches(t *testing.T, pattern string, str string, msgAndArgs ...interface{}) {
	t.Helper()
	assert.Regexp(t, pattern, str, msgAndArgs...)
}

// AssertStringNotMatches verifies that a string does not match a pattern
func AssertStringNotMatches(t *testing.T, pattern string, str string, msgAndArgs ...interface{}) {
	t.Helper()
	assert.NotRegexp(t, pattern, str, msgAndArgs...)
}

// AssertBetween verifies that a value is between min and max (inclusive)
func AssertBetween(t *testing.T, value, min, max interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	assert.True(t, compare(value, min) >= 0, "Value %v should be >= %v %v", value, min, msgAndArgs)
	assert.True(t, compare(value, max) <= 0, "Value %v should be <= %v %v", value, max, msgAndArgs)
}

// compare compares two values and returns -1, 0, or 1
func compare(a, b interface{}) int {
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	// Handle numeric types
	switch va.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		aVal := va.Int()
		bVal := vb.Int()
		if aVal < bVal {
			return -1
		} else if aVal > bVal {
			return 1
		}
		return 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		aVal := va.Uint()
		bVal := vb.Uint()
		if aVal < bVal {
			return -1
		} else if aVal > bVal {
			return 1
		}
		return 0
	case reflect.Float32, reflect.Float64:
		aVal := va.Float()
		bVal := vb.Float()
		if aVal < bVal {
			return -1
		} else if aVal > bVal {
			return 1
		}
		return 0
	case reflect.String:
		aVal := va.String()
		bVal := vb.String()
		if aVal < bVal {
			return -1
		} else if aVal > bVal {
			return 1
		}
		return 0
	}

	// Fallback to string comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	if aStr < bStr {
		return -1
	} else if aStr > bStr {
		return 1
	}
	return 0
}

// AssertPositiveNumber verifies that a number is positive
func AssertPositiveNumber(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		assert.True(t, v.Int() > 0, "Expected positive number, got: %v %v", value, msgAndArgs)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		assert.True(t, v.Uint() > 0, "Expected positive number, got: %v %v", value, msgAndArgs)
	case reflect.Float32, reflect.Float64:
		assert.True(t, v.Float() > 0, "Expected positive number, got: %v %v", value, msgAndArgs)
	default:
		assert.Fail(t, "Value is not a number", "Type: %T, Value: %v %v", value, value, msgAndArgs)
	}
}

// AssertNonNegativeNumber verifies that a number is non-negative (>= 0)
func AssertNonNegativeNumber(t *testing.T, value interface{}, msgAndArgs ...interface{}) {
	t.Helper()

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		assert.True(t, v.Int() >= 0, "Expected non-negative number, got: %v %v", value, msgAndArgs)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// Unsigned integers are always non-negative
		return
	case reflect.Float32, reflect.Float64:
		assert.True(t, v.Float() >= 0, "Expected non-negative number, got: %v %v", value, msgAndArgs)
	default:
		assert.Fail(t, "Value is not a number", "Type: %T, Value: %v %v", value, value, msgAndArgs)
	}
}

// AssertValidCoordinates verifies that coordinates are within valid game bounds
func AssertValidCoordinates(t *testing.T, x, y int32, msgAndArgs ...interface{}) {
	t.Helper()

	// Assuming game coordinates have reasonable bounds
	const maxCoordinate = 1000000
	const minCoordinate = -1000000

	assert.True(t, x >= minCoordinate && x <= maxCoordinate,
		"X coordinate out of bounds: %d (min: %d, max: %d) %v", x, minCoordinate, maxCoordinate, msgAndArgs)
	assert.True(t, y >= minCoordinate && y <= maxCoordinate,
		"Y coordinate out of bounds: %d (min: %d, max: %d) %v", y, minCoordinate, maxCoordinate, msgAndArgs)
}

// AssertValidChunkCoordinates verifies that chunk coordinates are valid
func AssertValidChunkCoordinates(t *testing.T, chunkX, chunkY int32, msgAndArgs ...interface{}) {
	t.Helper()

	// Chunk coordinates should be reasonable
	const maxChunkCoordinate = 10000
	const minChunkCoordinate = -10000

	assert.True(t, chunkX >= minChunkCoordinate && chunkX <= maxChunkCoordinate,
		"Chunk X coordinate out of bounds: %d (min: %d, max: %d) %v", chunkX, minChunkCoordinate, maxChunkCoordinate, msgAndArgs)
	assert.True(t, chunkY >= minChunkCoordinate && chunkY <= maxChunkCoordinate,
		"Chunk Y coordinate out of bounds: %d (min: %d, max: %d) %v", chunkY, minChunkCoordinate, maxChunkCoordinate, msgAndArgs)
}
