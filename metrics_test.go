package metrics

import (
	"fmt"
	"io/ioutil"
	"log"
	"sync"
	"testing"
)

const FANOUT = 128

// Stop the compiler from complaining during debugging.
var (
	_ = ioutil.Discard
	_ = log.LstdFlags
)

func BenchmarkMetrics(b *testing.B) {
	r := NewRegistry()
	c := NewRegisteredCounter("counter", r)
	g := NewRegisteredGauge("gauge", r)
	gf := NewRegisteredGaugeFloat64("gaugefloat64", r)
	h := NewRegisteredHistogram("histogram", r, NewUniformSample(100))
	m := NewRegisteredMeter("meter", r)
	t := NewRegisteredTimer("timer", r)
	RegisterDebugGCStats(r)
	RegisterRuntimeMemStats(r)
	b.ResetTimer()
	ch := make(chan bool)

	wgD := &sync.WaitGroup{}
	wgR := &sync.WaitGroup{}

	wgR.Add(1)
	go func() {
		defer wgR.Done()

		for {
			select {
			case <-ch:
				return
			default:
				CaptureRuntimeMemStatsOnce(r)
			}
		}
	}()

	wgW := &sync.WaitGroup{}

	wg := &sync.WaitGroup{}
	wg.Add(FANOUT)
	for i := 0; i < FANOUT; i++ {
		go func(i int) {
			defer wg.Done()

			for i := 0; i < b.N; i++ {
				c.Inc(1)
				g.Update(int64(i))
				gf.Update(float64(i))
				h.Update(int64(i))
				m.Mark(1)
				t.Update(1)
			}
		}(i)
	}
	wg.Wait()
	close(ch)
	wgD.Wait()
	wgR.Wait()
	wgW.Wait()
}

func Example() {
	c := NewCounter()
	Register("money", c)
	c.Inc(17)

	// Threadsafe registration
	t := GetOrRegisterTimer("db.get.latency", nil)
	t.Time(func() {})
	t.Update(1)

	fmt.Println(c.Count())
	fmt.Println(t.Min())
	// Output: 17
	// 1
}
