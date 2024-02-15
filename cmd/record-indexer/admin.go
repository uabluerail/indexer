package main

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"golang.org/x/time/rate"
)

func AddAdminHandlers(limiter *Limiter, pool *WorkerPool) {
	http.HandleFunc("/rate/set", handleRateSet(limiter))
	http.HandleFunc("/rate/setAll", handleRateSetAll(limiter))
	http.HandleFunc("/pool/resize", handlePoolResize(pool))
}

func handlePoolResize(pool *WorkerPool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := r.FormValue("size")
		if s == "" {
			http.Error(w, "need size", http.StatusBadRequest)
			return
		}

		size, err := strconv.Atoi(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		pool.Resize(context.Background(), size)
		fmt.Fprintln(w, "OK")
	}
}

func handleRateSet(limiter *Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := r.FormValue("limit")
		if s == "" {
			http.Error(w, "need limit", http.StatusBadRequest)
			return
		}
		name := r.FormValue("name")
		if name == "" {
			http.Error(w, "need name", http.StatusBadRequest)
			return
		}

		limit, err := strconv.Atoi(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		limiter.SetLimit(context.Background(), name, rate.Limit(limit))
		fmt.Fprintln(w, "OK")
	}
}

func handleRateSetAll(limiter *Limiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s := r.FormValue("limit")
		if s == "" {
			http.Error(w, "need limit", http.StatusBadRequest)
			return
		}
		limit, err := strconv.Atoi(s)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		limiter.SetAllLimits(context.Background(), rate.Limit(limit))
		fmt.Fprintln(w, "OK")
	}
}
