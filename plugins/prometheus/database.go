package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/iotaledger/hive.go/events"
)

var (
	compactions       prometheus.Counter
	compactionRunning prometheus.Gauge
)

func configureDatabase() {
	compactions = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "iota_database_compactions",
			Help: "The total amount of database compactions.",
		},
	)

	deps.DatabaseEvents.DatabaseCompaction.Attach(events.NewClosure(func(compactionRunning bool) {
		if compactionRunning {
			compactions.Inc()
		}
	}))

	compactionRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "iota_database_compaction_running",
		Help: "Current state of database compaction process.",
	})

	registry.MustRegister(compactions)
	registry.MustRegister(compactionRunning)

	addCollect(collectDatabase)
}

func collectDatabase() {
	compactionRunning.Set(0)
	if deps.DatabaseMetrics.CompactionRunning.Load() {
		compactionRunning.Set(1)
	}
}
