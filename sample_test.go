package metrics

import (
	"math/rand"
	"runtime"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Benchmark{Compute,Copy}{1000,1000000} demonstrate that, even for relatively
// expensive computations like Variance, the cost of copying the Sample, as
// approximated by a make and copy, is much greater than the cost of the
// computation for small samples and only slightly less for large samples.
func BenchmarkCompute1000(b *testing.B) {
	s := make([]int64, 1000)
	for i := 0; i < len(s); i++ {
		s[i] = int64(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SampleVariance(s)
	}
}
func BenchmarkCompute1000000(b *testing.B) {
	s := make([]int64, 1000000)
	for i := 0; i < len(s); i++ {
		s[i] = int64(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		SampleVariance(s)
	}
}
func BenchmarkCopy1000(b *testing.B) {
	s := make([]int64, 1000)
	for i := 0; i < len(s); i++ {
		s[i] = int64(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sCopy := make([]int64, len(s))
		copy(sCopy, s)
	}
}
func BenchmarkCopy1000000(b *testing.B) {
	s := make([]int64, 1000000)
	for i := 0; i < len(s); i++ {
		s[i] = int64(i)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sCopy := make([]int64, len(s))
		copy(sCopy, s)
	}
}

func BenchmarkExpDecaySample257(b *testing.B) {
	benchmarkSample(b, NewExpDecaySample(WithReservoirSize(257)))
}

func BenchmarkExpDecaySample514(b *testing.B) {
	benchmarkSample(b, NewExpDecaySample(WithReservoirSize(514)))
}

func BenchmarkExpDecaySample1028(b *testing.B) {
	benchmarkSample(b, NewExpDecaySample())
}

func BenchmarkUniformSample257(b *testing.B) {
	benchmarkSample(b, NewUniformSample(257))
}

func BenchmarkUniformSample514(b *testing.B) {
	benchmarkSample(b, NewUniformSample(514))
}

func BenchmarkUniformSample1028(b *testing.B) {
	benchmarkSample(b, NewUniformSample(1028))
}

func TestExpDecaySample__Update(t *testing.T) {
	rand.Seed(1)

	t.Run("10 values with a size of 100", func(t *testing.T) {
		s := NewExpDecaySample(WithReservoirSize(100), WithAlpha(0.99))
		for i := 0; i < 10; i++ {
			s.Update(int64(i))
		}

		assert.EqualValues(t, 10, s.Count())
		assert.EqualValues(t, 10, s.Size())

		require.Len(t, s.Values(), 10)
		for _, v := range s.Values() {
			require.True(t, 0 <= v && v < 10)
		}
	})

	t.Run("100 values with a size of 1000", func(t *testing.T) {
		s := NewExpDecaySample(WithReservoirSize(1000), WithAlpha(0.01))
		for i := 0; i < 100; i++ {
			s.Update(int64(i))
		}

		assert.EqualValues(t, 100, s.Count())
		assert.EqualValues(t, 100, s.Size())

		require.Len(t, s.Values(), 100)
		for _, v := range s.Values() {
			require.True(t, 0 <= v && v < 100)
		}
	})

	t.Run("1000 values with a size of 100", func(t *testing.T) {
		s := NewExpDecaySample(WithReservoirSize(100), WithAlpha(0.99))
		for i := 0; i < 1000; i++ {
			s.Update(int64(i))
		}

		assert.EqualValues(t, 1000, s.Count())
		assert.EqualValues(t, 100, s.Size())

		require.Len(t, s.Values(), 100)
		for _, v := range s.Values() {
			require.True(t, 0 <= v && v < 1000)
		}
	})

}

// This test makes sure that the sample's priority is not amplified by using
// nanosecond duration since start rather than second duration since start.
// The priority becomes +Inf quickly after starting if this is done,
// effectively freezing the set of samples until a rescale step happens.
func TestExpDecaySampleNanosecondRegression(t *testing.T) {
	rand.Seed(1)

	s := NewExpDecaySample(WithReservoirSize(100), WithAlpha(0.99)).(*ExpDecaySample)
	mClock := setupClock(s)

	for i := 0; i < 100; i++ {
		s.Update(10)
	}

	mClock.Add(1 * time.Millisecond)

	for i := 0; i < 100; i++ {
		s.Update(20)
	}

	v := s.Values()
	avg := float64(0)
	for i := 0; i < len(v); i++ {
		avg += float64(v[i])
	}
	avg /= float64(len(v))

	assert.True(t, 14 <= avg && avg <= 16)
}

func TestExpDecaySampleRescale(t *testing.T) {
	t.Run("Size 2 + Alpha 0.001 rescaled after 1 hour", func(t *testing.T) {
		s := NewExpDecaySample(WithReservoirSize(2), WithAlpha(0.001)).(*ExpDecaySample)
		mClock := setupClock(s)

		s.Update(1)
		mClock.Add(time.Hour + time.Microsecond)
		s.Update(1)

		for _, v := range s.values.Values() {
			require.NotZero(t, v.k)
		}
	})

	t.Run("Default but rescaled after 30 minutes", func(t *testing.T) {
		s := NewExpDecaySample(WithRescaleThreshold(30 * time.Minute)).(*ExpDecaySample)
		mClock := setupClock(s)

		s.Update(1)

		heapVals := s.values.Values()
		assert.Len(t, heapVals, 1)
		assert.NotZero(t, heapVals[0].k)

		mClock.Add(31 * time.Minute)

		vals := s.Values()    // Rescale triggered
		assert.Empty(t, vals) // Should forget first value
	})

}

func TestExpDecaySampleSnapshot(t *testing.T) {
	rand.Seed(1)

	now := time.Now()
	s := NewExpDecaySample(WithReservoirSize(100), WithAlpha(0.99)).(*ExpDecaySample)
	for i := 1; i <= 10000; i++ {
		s.update(now.Add(time.Duration(i)), int64(i))
	}

	snapshot := s.Snapshot()
	s.Update(1)

	testExpDecaySampleStatistics(t, snapshot)
}

func TestExpDecaySampleStatistics(t *testing.T) {
	rand.Seed(1)

	now := time.Now()
	s := NewExpDecaySample(WithReservoirSize(100), WithAlpha(0.99)).(*ExpDecaySample)
	for i := 1; i <= 10000; i++ {
		s.update(now.Add(time.Duration(i)), int64(i))
	}

	testExpDecaySampleStatistics(t, s)
}

func TestUniformSample(t *testing.T) {
	rand.Seed(1)

	s := NewUniformSample(100)
	for i := 0; i < 1000; i++ {
		s.Update(int64(i))
	}

	assert.EqualValues(t, 1000, s.Count())
	assert.EqualValues(t, 100, s.Size())

	require.Len(t, s.Values(), 100)
	for _, v := range s.Values() {
		require.True(t, 0 <= v && v < 1000)
	}
}

func TestUniformSampleIncludesTail(t *testing.T) {
	rand.Seed(1)

	s := NewUniformSample(100)
	max := 100
	for i := 0; i < max; i++ {
		s.Update(int64(i))
	}

	v := s.Values()
	sum := 0
	for i := 0; i < len(v); i++ {
		sum += int(v[i])
	}

	exp := (max - 1) * max / 2
	assert.EqualValues(t, exp, sum)
}

func TestUniformSampleSnapshot(t *testing.T) {
	s := NewUniformSample(100)
	for i := 1; i <= 10000; i++ {
		s.Update(int64(i))
	}

	snapshot := s.Snapshot()
	s.Update(1)

	testUniformSampleStatistics(t, snapshot)
}

func TestUniformSampleStatistics(t *testing.T) {
	rand.Seed(1)

	s := NewUniformSample(100)
	for i := 1; i <= 10000; i++ {
		s.Update(int64(i))
	}

	testUniformSampleStatistics(t, s)
}

func benchmarkSample(b *testing.B, s Sample) {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	pauseTotalNs := memStats.PauseTotalNs

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		s.Update(1)
	}

	b.StopTimer()

	runtime.GC()
	runtime.ReadMemStats(&memStats)

	b.Logf("GC cost: %d ns/op", int(memStats.PauseTotalNs-pauseTotalNs)/b.N)
}

func testExpDecaySampleStatistics(t *testing.T, s Sample) {
	assert.EqualValues(t, 10000, s.Count())
	assert.EqualValues(t, 107, s.Min())
	assert.EqualValues(t, 10000, s.Max())
	assert.EqualValues(t, 4965.98, s.Mean())
	assert.EqualValues(t, 2959.825156930727, s.StdDev())
	assert.Equal(t, []float64{4615, 7672, 9998.99},
		s.Percentiles([]float64{0.5, 0.75, 0.99}))
}

func testUniformSampleStatistics(t *testing.T, s Sample) {
	assert.EqualValues(t, 10000, s.Count())
	assert.EqualValues(t, 37, s.Min())
	assert.EqualValues(t, 9989, s.Max())
	assert.EqualValues(t, 4748.14, s.Mean())
	assert.EqualValues(t, 2826.684117548333, s.StdDev())
	assert.Equal(t, []float64{4599, 7380.5, 9986.429999999998},
		s.Percentiles([]float64{0.5, 0.75, 0.99}))
}

// TestUniformSampleConcurrentUpdateCount would expose data race problems with
// concurrent Update and Count calls on Sample when test is called with -race
// argument
func TestUniformSampleConcurrentUpdateCount(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}

	s := NewUniformSample(100)
	for i := 0; i < 100; i++ {
		s.Update(int64(i))
	}

	quit := make(chan struct{})
	go func() {
		t := time.NewTicker(10 * time.Millisecond)
		for {
			select {
			case <-t.C:
				s.Update(rand.Int63())
			case <-quit:
				t.Stop()
				return
			}
		}
	}()

	for i := 0; i < 1000; i++ {
		s.Count()
		time.Sleep(5 * time.Millisecond)
	}
	quit <- struct{}{}
}

func setupClock(s *ExpDecaySample) *clock.Mock {
	mClock := clock.NewMock()
	mClock.Set(time.Date(2021, 6, 1, 0, 0, 0, 0, time.UTC))
	s.clock = mClock
	s.setTime(mClock.Now())
	return mClock
}
