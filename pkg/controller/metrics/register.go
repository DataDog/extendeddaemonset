// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"k8s.io/client-go/tools/cache"
	ksmetric "k8s.io/kube-state-metrics/pkg/metric"
	metricsstore "k8s.io/kube-state-metrics/pkg/metrics_store"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/datadog/extendeddaemonset/pkg/controller/httpserver"
)

// NewHandler return new metrics Handler instance
func NewHandler(mux httpserver.Server) Handler {
	handler := &storesHandler{
		mux: mux,
	}

	var ksmetricsPath = "/ksmetrics"
	mux.HandleFunc(ksmetricsPath, handler.ServeHTTP)

	var metricsPath = "/metrics"
	promHandler := promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
	handler.mux.Handle(metricsPath, promHandler)

	return handler
}

func (h *storesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	resHeader := w.Header()
	// 0.0.4 is the exposition format version of prometheus
	// https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format
	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")
	for _, store := range h.stores {
		store.WriteAll(w)
	}
}

func (h *storesHandler) RegisterStore(generators []ksmetric.FamilyGenerator, expectedType interface{}, lw cache.ListerWatcher) error {
	store := newMetricsStore(generators, expectedType, lw)

	h.stores = append(h.stores, store)
	return nil
}

type storesHandler struct {
	mux    httpserver.Server
	stores []*metricsstore.MetricsStore
}
