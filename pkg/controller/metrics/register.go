// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package metrics

import (
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/datadog/extendeddaemonset/pkg/controller/httpserver"
)

// Register used to register Metrics handler in the http server
func Register(mux httpserver.Server) {
	var metricsPath = "/metrics"
	handler := promhttp.HandlerFor(metrics.Registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.HTTPErrorOnError,
	})
	mux.Handle(metricsPath, handler)
}
