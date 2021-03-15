package prometheus

import (
	"github.com/prometheus/client_golang/prometheus"

	"github.com/iotaledger/hive.go/events"
)

var (
	compactions       prometheus.Counter
	compactionRunning prometheus.Gauge
	prunings          prometheus.Counter
	pruningRunning    prometheus.Gauge
)

func configureDatabase() {
	compactions = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "iota_database_compactions",
			Help: "The total amount of database compactions.",
		},
	)

	deps.DatabaseEvents.DatabaseCompaction.Attach(events.NewClosure(func(running bool) {
		if running {
			compactions.Inc()
		}
	}))

	compactionRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "iota_database_compaction_running",
		Help: "Current state of database compaction process.",
	})

	prunings = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "iota_database_prunings",
			Help: "The total amount of database prunings.",
		},
	)

	deps.Storage.Events.PruningStateChanged.Attach(events.NewClosure(func(running bool) {
		if running {
			prunings.Inc()
		}
	}))

	pruningRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "iota_database_pruning_running",
		Help: "Current state of database pruning process.",
	})

	registry.MustRegister(compactions)
	registry.MustRegister(compactionRunning)
	registry.MustRegister(prunings)
	registry.MustRegister(pruningRunning)

	addCollect(collectDatabase)
}

func collectDatabase() {
	compactionRunning.Set(0)
	if deps.DatabaseMetrics.CompactionRunning.Load() {
		compactionRunning.Set(1)
	}

	pruningRunning.Set(0)
	if deps.StorageMetrics.PruningRunning.Load() {
		pruningRunning.Set(1)
	}
}
