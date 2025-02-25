package main

import (
	"fmt"
	"time"

	"github.com/weareyolo/go-metrics"
)

func main() {
	r := metrics.NewRegistry()
	for i := 0; i < 10000; i++ {
		r.Register(fmt.Sprintf("counter-%d", i), metrics.NewCounter())
		r.Register(fmt.Sprintf("gauge-%d", i), metrics.NewGauge())
		r.Register(fmt.Sprintf("gaugefloat64-%d", i), metrics.NewGaugeFloat64())
		r.Register(fmt.Sprintf("histogram-uniform-%d", i), metrics.NewHistogram(metrics.NewUniformSample(1028)))
		r.Register(fmt.Sprintf("histogram-exp-%d", i), metrics.NewHistogram(metrics.NewExpDecaySample()))
		r.Register(fmt.Sprintf("meter-%d", i), metrics.NewMeter())
	}
	time.Sleep(600e9)
}
