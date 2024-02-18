package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var reposDiscovered = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "repo_discovered_counter",
	Help: "Counter of newly discovered repos",
}, []string{"remote"})

var reposListed = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "repo_listed_counter",
	Help: "Counter of repos received by listing PDSs.",
}, []string{"remote"})
