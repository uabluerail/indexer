package main

import (
	"slices"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var reposQueued = promauto.NewCounter(prometheus.CounterOpts{
	Name: "indexer_repos_queued_count",
	Help: "Number of repos added to the queue",
})

var queueLenght = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "indexer_queue_length",
	Help: "Current length of indexing queue",
}, []string{"state"})

var reposFetched = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "indexer_repos_fetched_count",
	Help: "Number of repos fetched",
}, []string{"remote", "success"})

var reposIndexed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "indexer_repos_indexed_count",
	Help: "Number of repos indexed",
}, []string{"success"})

var recordsFetched = promauto.NewCounter(prometheus.CounterOpts{
	Name: "indexer_records_fetched_count",
	Help: "Number of records fetched",
})

var recordsInserted = promauto.NewCounter(prometheus.CounterOpts{
	Name: "indexer_records_inserted_count",
	Help: "Number of records inserted into DB",
})

var workerPoolSize = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "indexer_workers_count",
	Help: "Current number of workers running",
})

var repoFetchSize = promauto.NewHistogram(prometheus.HistogramOpts{
	Name: "indexer_repo_fetched_size_bytes",
	Help: "Size of the fetched repo in bytes",
	Buckets: slices.Concat(
		prometheus.LinearBuckets(100*1024, 100*1024, 9),
		prometheus.LinearBuckets(1024*1024, 1024*1024, 9),
		prometheus.LinearBuckets(10*1024*1024, 5*1024*1024, 10)),
})
