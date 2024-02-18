package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var lastEventTimestamp = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "repo_commit_received_timestamp",
	Help: "Timestamp of the last event received from firehose.",
}, []string{"remote"})

var eventCounter = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "repo_commits_received_counter",
	Help: "Counter of events received from each remote.",
}, []string{"remote", "type"})

var reposDiscovered = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "repo_discovered_counter",
	Help: "Counter of newly discovered repos",
}, []string{"remote"})
