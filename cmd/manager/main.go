// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/datadog/extendeddaemonset/pkg/apis"
	edsconfig "github.com/datadog/extendeddaemonset/pkg/config"
	"github.com/datadog/extendeddaemonset/pkg/controller"
	"github.com/datadog/extendeddaemonset/pkg/controller/debug"
	"github.com/datadog/extendeddaemonset/pkg/controller/httpserver"
	"github.com/datadog/extendeddaemonset/pkg/controller/metrics"
	"github.com/datadog/extendeddaemonset/version"

	"github.com/operator-framework/operator-sdk/pkg/k8sutil"
	"github.com/operator-framework/operator-sdk/pkg/leader"
	"github.com/operator-framework/operator-sdk/pkg/log/zap"
	"github.com/operator-framework/operator-sdk/pkg/restmapper"

	"github.com/spf13/pflag"

	"github.com/blang/semver"
	kversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"

	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/runtime/signals"
)

// Change below variables to serve metrics on different host or port.
var (
	bindHost              = "0.0.0.0"
	bindPort        int32 = 8383
	printVersionArg bool
	pprofActive     bool

	log = logf.Log.WithName("cmd")
)

func main() {
	// Add the zap logger flag set to the CLI. The flag set must
	// be added before calling pflag.Parse().
	pflag.CommandLine.AddFlagSet(zap.FlagSet())

	// Add flags registered by imported packages (e.g. glog and
	// controller-runtime)
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.BoolVarP(&printVersionArg, "version", "v", printVersionArg, "print version")
	pflag.BoolVarP(&pprofActive, "pprof", "", false, "enable pprof endpoint")

	pflag.Parse()

	// Use a zap logr.Logger implementation. If none of the zap
	// flags are configured (or if the zap flag set is not being
	// used), this defaults to a production zap logger.
	//
	// The logger instantiated here can be changed to any logger
	// implementing the logr.Logger interface. This logger will
	// be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(zap.Logger())

	if printVersionArg {
		version.PrintVersionWriter(os.Stdout)
		os.Exit(0)
	}

	version.PrintVersionLogs(log)

	namespace, err := k8sutil.GetWatchNamespace()
	if err != nil {
		log.Error(err, "Failed to get watch namespace")
		os.Exit(1)
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	//auto discover if Node affinity match is supported in the current cluster
	if os.Getenv(edsconfig.NodeAffinityMatchSupportEnvVar) == "" {
		discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(cfg)
		var serverVersion *kversion.Info
		if serverVersion, err = discoveryClient.ServerVersion(); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		var minServerVersion semver.Version
		if minServerVersion, err = semver.Make("1.12.0"); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		var currentServerVersion semver.Version
		if currentServerVersion, err = semver.Make(strings.TrimPrefix(serverVersion.String(), "v")); err != nil {
			log.Error(err, "")
			os.Exit(1)
		}
		if minServerVersion.Compare(currentServerVersion) < 0 {
			// it means the kubernetes cluster support Node assignment with AffinityMatchField
			if err = os.Setenv(edsconfig.NodeAffinityMatchSupportEnvVar, "1"); err != nil {
				log.Error(err, "")
				os.Exit(1)
			}
		}
	}

	ctx := context.TODO()

	// Become the leader before proceeding
	err = leader.Become(ctx, "extendeddaemonset-lock")
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{
		Namespace:          namespace,
		MapperProvider:     restmapper.NewDynamicRESTMapper,
		MetricsBindAddress: "0",
	})
	if err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	log.Info("Registering Components.")

	// Setup Scheme for all resources
	if err = apis.AddToScheme(mgr.GetScheme()); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Setup all Controllers
	if err = controller.AddToManager(mgr); err != nil {
		log.Error(err, "")
		os.Exit(1)
	}

	// Create HttpServer handler
	srv := httpserver.New(httpserver.Options{BindAddress: fmt.Sprintf("%s:%d", bindHost, bindPort)})
	metrics.Register(srv)
	if pprofActive {
		debug.Register(srv, debug.DefaultOptions())
	}
	if err = mgr.Add(srv); err != nil {
		log.Error(err, "HTTP server registration error")
		os.Exit(1)
	}

	log.Info("Starting the Cmd.")

	// Start the Cmd
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Error(err, "Manager exited non-zero")
		os.Exit(1)
	}
}
