package chart

import (
	"hash/fnv"
	"sync"
)

// Cacheable is implemented by charts that support render caching.
type Cacheable interface {
	// Render renders the chart at the given width.
	Render(width int) string
}

// Ensure all chart types implement Cacheable.
var (
	_ Cacheable = (*BarChart)(nil)
	_ Cacheable = (*LineChart)(nil)
	_ Cacheable = (*DualLineChart)(nil)
	_ Cacheable = (*StackedBarChart)(nil)
	_ Cacheable = (*SleepStagesChart)(nil)
)

// CachedChart stores a rendered chart along with the parameters used to render it.
type CachedChart struct {
	rendered string
	width    int
	dataHash uint64
}

var chartCache sync.Map // key: string (chartID) -> *CachedChart

// GetCached retrieves a cached chart render if it exists and matches the current parameters.
// Returns the rendered string and true if found and valid, empty string and false otherwise.
func GetCached(id string, width int, dataHash uint64) (string, bool) {
	if id == "" {
		return "", false
	}

	val, ok := chartCache.Load(id)
	if !ok {
		return "", false
	}

	cached := val.(*CachedChart)
	if cached.width == width && cached.dataHash == dataHash {
		return cached.rendered, true
	}

	return "", false
}

// SetCached stores a rendered chart in the cache.
func SetCached(id string, width int, dataHash uint64, rendered string) {
	if id == "" {
		return
	}

	chartCache.Store(id, &CachedChart{
		rendered: rendered,
		width:    width,
		dataHash: dataHash,
	})
}

// ClearCache removes all cached chart renders.
// Should be called when data changes (e.g., date navigation, new data fetch).
func ClearCache() {
	chartCache.Range(func(key, _ any) bool {
		chartCache.Delete(key)
		return true
	})
}

// HashDataPoints computes a hash for a slice of DataPoints.
func HashDataPoints(data []DataPoint) uint64 {
	h := fnv.New64a()
	for _, d := range data {
		_, _ = h.Write([]byte(d.Label))
		// write float64 as bytes
		bits := uint64(d.Value * 1000000) // preserve 6 decimal places
		for i := range 8 {
			_, _ = h.Write([]byte{byte(bits >> (i * 8))})
		}
	}
	return h.Sum64()
}

// HashStackedDataPoints computes a hash for a slice of StackedDataPoints.
func HashStackedDataPoints(data []StackedDataPoint) uint64 {
	h := fnv.New64a()
	for _, d := range data {
		_, _ = h.Write([]byte(d.Label))
		for _, v := range d.Values {
			bits := uint64(v * 1000000)
			for i := range 8 {
				_, _ = h.Write([]byte{byte(bits >> (i * 8))})
			}
		}
	}
	return h.Sum64()
}

// HashSleepStages computes a hash for a slice of SleepStages.
func HashSleepStages(stages []SleepStage, totalDuration, baselineDuration int) uint64 {
	h := fnv.New64a()
	for _, s := range stages {
		_, _ = h.Write([]byte(s.Name))
		bits := uint64(s.Percentage * 1000000)
		for i := range 8 {
			_, _ = h.Write([]byte{byte(bits >> (i * 8))})
		}
		dur := uint64(int64(s.DurationMs)) //nolint:gosec // duration is always non-negative
		for i := range 8 {
			_, _ = h.Write([]byte{byte(dur >> (i * 8))})
		}
	}
	// include total and baseline in hash
	total := uint64(int64(totalDuration))       //nolint:gosec // duration is always non-negative
	baseline := uint64(int64(baselineDuration)) //nolint:gosec // duration is always non-negative
	for i := range 8 {
		_, _ = h.Write([]byte{byte(total >> (i * 8))})
		_, _ = h.Write([]byte{byte(baseline >> (i * 8))})
	}
	return h.Sum64()
}
