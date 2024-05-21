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

var postsByLanguageIndexed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "indexer_posts_by_language_count",
	Help: "Number of posts by language",
}, []string{"remote", "lang"})

var connectionFailures = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "consumer_connection_failures",
	Help: "Counter of firehose connection failures",
}, []string{"remote"})

var pdsOnline = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "consumer_connection_up",
	Help: "Status of a connection. 1 - up and running.",
}, []string{"remote"})
