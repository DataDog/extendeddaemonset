// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package debug

import (
	"net/http/pprof"

	"github.com/datadog/extendeddaemonset/pkg/controller/httpserver"
)

// Options use to provide configuration option
type Options struct {
	CmdLine bool
	Profile bool
	Symbol  bool
	Trace   bool
}

// DefaultOptions returns default options configuration
func DefaultOptions() *Options {
	return &Options{
		CmdLine: true,
		Profile: true,
		Symbol:  true,
		Trace:   true,
	}
}

// Register use to register the different debug endpoint
func Register(mux httpserver.Server, options *Options) {
	mux.HandleFunc("/debug/pprof/", pprof.Index)
	if options == nil {
		options = DefaultOptions()
	}
	if options.CmdLine {
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	}
	if options.Profile {
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	}
	if options.Symbol {
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	}
	if options.Trace {
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
	}
}
