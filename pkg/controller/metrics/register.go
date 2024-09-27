// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/cache"
	generator "k8s.io/kube-state-metrics/v2/pkg/metric_generator"
	metricsstore "k8s.io/kube-state-metrics/v2/pkg/metrics_store"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	metricsHandler          []func(*runtime.Scheme, Handler) error
	ksmExtraMetricsRegistry = prometheus.NewRegistry()
	log                     = ctrl.Log.WithName("ksmetrics")
)

// RegisterHandlerFunc register a function to be added to endpoint when its registered.
func RegisterHandlerFunc(h func(*runtime.Scheme, Handler) error) {
	metricsHandler = append(metricsHandler, h)
}

// GetExtraMetricHandlers return handler for a KSM endpoint.
func GetExtraMetricHandlers(scheme *runtime.Scheme) (map[string]http.Handler, error) {
	handler := &storesHandler{}
	for _, metricsHandler := range metricsHandler {
		if err := metricsHandler(scheme, handler); err != nil {
			return nil, err
		}
	}
	handlers := make(map[string]http.Handler)
	handlers["/ksmetrics"] = http.HandlerFunc(handler.serveKsmHTTP)
	return handlers, nil
}

func (h *storesHandler) serveKsmHTTP(w http.ResponseWriter, r *http.Request) {
	resHeader := w.Header()
	// 0.0.4 is the exposition format version of prometheus
	// https://prometheus.io/docs/instrumenting/exposition_formats/#text-based-format
	resHeader.Set("Content-Type", `text/plain; version=`+"0.0.4")

	// Write KSM families
	if err := metricsstore.NewMetricsWriter(h.stores...).WriteAll(w); err != nil {
		log.Error(err, "Unable to write metrics")
	}

	// Write extra metrics
	metrics, err := ksmExtraMetricsRegistry.Gather()
	if err == nil {
		for _, m := range metrics {
			_, err = expfmt.MetricFamilyToText(w, m)
			if err != nil {
				log.Error(err, "Unable to write metrics", "metricFamily", m.GetName())
			}
		}
	} else {
		log.Error(err, "Unable to export extra metrics")
	}
}

func (h *storesHandler) RegisterStore(generators []generator.FamilyGenerator, expectedType interface{}, lw cache.ListerWatcher) error {
	store := newMetricsStore(generators, expectedType, lw)
	h.stores = append(h.stores, store)

	return nil
}

type storesHandler struct {
	stores []*metricsstore.MetricsStore
}
