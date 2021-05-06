# go-metrics

Go port of [Coda Hale's Metrics library](https://github.com/dropwizard/metrics).

## Usage

Create and update metrics:

```go
c := metrics.NewCounter()
metrics.Register("foo", c)
c.Inc(47)

g := metrics.NewGauge()
metrics.Register("bar", g)
g.Update(47)

r := NewRegistry()
g := metrics.NewRegisteredFunctionalGauge("cache-evictions", r, func() int64 { return cache.getEvictionsCount() })

s := metrics.NewExpDecaySample(1028, 0.015) // or metrics.NewUniformSample(1028)
h := metrics.NewHistogram(s)
metrics.Register("baz", h)
h.Update(47)

m := metrics.NewMeter()
metrics.Register("quux", m)
m.Mark(47)

t := metrics.NewTimer()
metrics.Register("bang", t)
t.Time(func() {})
t.Update(47)
```

Register() is not thread-safe. For thread-safe metric registration use
GetOrRegister:

```go
t := metrics.GetOrRegisterTimer("account.create.latency", nil)
t.Time(func() {})
t.Update(47)
```

**NOTE:** Be sure to unregister short-lived meters and timers otherwise they will
leak memory:

```go
// Will call Stop() on the Meter to allow for garbage collection
metrics.Unregister("quux")
// Or similarly for a Timer that embeds a Meter
metrics.Unregister("bang")
```

Periodically log every metric in human-readable form to standard error:

```go
go metrics.Log(metrics.DefaultRegistry, 5 * time.Second, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
```

Periodically log every metric in slightly-more-parsable form to syslog:

```go
w, _ := syslog.Dial("unixgram", "/dev/log", syslog.LOG_INFO, "metrics")
go metrics.Syslog(metrics.DefaultRegistry, 60e9, w)
```

Maintain all metrics along with expvars at `/debug/metrics`:

This uses the same mechanism as [the official expvar](http://golang.org/pkg/expvar/)
but exposed under `/debug/metrics`, which shows a json representation of all your usual expvars
as well as all your go-metrics.

```go
import "github.com/weareyolo/go-metrics/exp"

exp.Exp(metrics.DefaultRegistry)
```

## Installation

```sh
go get github.com/weareyolo/go-metrics
```

## Publishing Metrics

Clients are available for the following destinations:

- [Librato](https://github.com/mihasya/go-metrics-librato)
- [Graphite](https://github.com/cyberdelia/go-metrics-graphite)
- [InfluxDB](https://github.com/vrischmann/go-metrics-influxdb)
- [Ganglia](https://github.com/appscode/metlia)
- [Prometheus](https://github.com/deathowl/go-metrics-prometheus)
- [DataDog](https://github.com/syntaqx/go-metrics-datadog)
- [SignalFX](https://github.com/pascallouisperez/go-metrics-signalfx)
- [Honeycomb](https://github.com/getspine/go-metrics-honeycomb)
- [Wavefront](https://github.com/wavefrontHQ/go-metrics-wavefront)
