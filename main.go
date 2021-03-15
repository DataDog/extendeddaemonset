// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2019 Datadog, Inc.

package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	goruntime "runtime"
	"strings"

	"github.com/blang/semver"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	kversion "k8s.io/apimachinery/pkg/version"
	"k8s.io/client-go/discovery"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	datadoghqv1alpha1 "github.com/DataDog/extendeddaemonset/api/v1alpha1"
	"github.com/DataDog/extendeddaemonset/controllers"
	"github.com/DataDog/extendeddaemonset/pkg/config"
	"github.com/DataDog/extendeddaemonset/pkg/controller/debug"
	"github.com/DataDog/extendeddaemonset/pkg/controller/metrics"
	"github.com/DataDog/extendeddaemonset/pkg/version"
	// +kubebuilder:scaffold:imports
)

const (
	maximumGoroutines = 200
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(datadoghqv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", true,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")

	// Custom flags
	var printVersion, pprofActive bool
	var logEncoder string
	flag.StringVar(&logEncoder, "logEncoder", "json", "log encoding ('json' or 'console')")
	logLevel := zap.LevelFlag("loglevel", zapcore.InfoLevel, "Set log level")
	flag.BoolVar(&printVersion, "version", false, "Print version and exit")
	flag.BoolVar(&pprofActive, "pprof", false, "Enable pprof endpoint")

	// Parsing flags
	flag.Parse()

	// Logging setup
	if err := customSetupLogging(*logLevel, logEncoder); err != nil {
		setupLog.Error(err, "unable to setup the logger")
		os.Exit(1)
	}

	// Print version information
	if printVersion {
		version.PrintVersionWriter(os.Stdout)
		os.Exit(0)
	}

	version.PrintVersionLogs(setupLog)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), config.ManagerOptionsWithNamespaces(setupLog, ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		HealthProbeBindAddress: ":8081",
		Port:                   9443,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "extendeddaemonset-lock",
	}))
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Custom setup
	customSetupMetrics(mgr)
	customSetupEnvironment(mgr)
	customSetupHealthChecks(mgr)
	customSetupEndpoints(pprofActive, mgr)

	// Read conf (env + CLI flags)
	nodeAffinityMatchSupport := os.Getenv(config.NodeAffinityMatchSupportEnvVar) == "1"

	// Setup controllers and start manager
	err = controllers.SetupControllers(mgr, nodeAffinityMatchSupport)
	if err != nil {
		setupLog.Error(err, "unable to setup controllers")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func customSetupEnvironment(mgr manager.Manager) {
	// auto discover if Node affinity match is supported in the current cluster
	if os.Getenv(config.NodeAffinityMatchSupportEnvVar) == "" {
		discoveryClient := discovery.NewDiscoveryClientForConfigOrDie(mgr.GetConfig())
		var err error
		var serverVersion *kversion.Info
		if serverVersion, err = discoveryClient.ServerVersion(); err != nil {
			setupLog.Error(err, "")
			os.Exit(1)
		}
		var minServerVersion semver.Version
		if minServerVersion, err = semver.Make("1.12.0"); err != nil {
			setupLog.Error(err, "")
			os.Exit(1)
		}
		var currentServerVersion semver.Version
		if currentServerVersion, err = semver.Make(strings.TrimPrefix(serverVersion.String(), "v")); err != nil {
			setupLog.Error(err, "")
			os.Exit(1)
		}
		if minServerVersion.Compare(currentServerVersion) < 0 {
			// it means the kubernetes cluster support Node assignment with AffinityMatchField
			if err = os.Setenv(config.NodeAffinityMatchSupportEnvVar, "1"); err != nil {
				setupLog.Error(err, "")
				os.Exit(1)
			}
		}
	}
}

func customSetupMetrics(mgr manager.Manager) {
	go func() {
		// This channel is closed when this instance is elected leader
		// Apparently there's no releasing the lease except if application dies
		<-mgr.Elected()
		setupLog.Info("Controller elected - metric changed")
		metrics.SetLeader(true)
	}()
}

func customSetupLogging(logLevel zapcore.Level, logEncoder string) error {
	var encoder zapcore.Encoder
	switch logEncoder {
	case "console":
		encoder = zapcore.NewConsoleEncoder(zap.NewProductionEncoderConfig())
	case "json":
		encoder = zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	default:
		return fmt.Errorf("unknow log encoder: %s", logEncoder)
	}

	ctrl.SetLogger(ctrlzap.New(
		ctrlzap.Encoder(encoder),
		ctrlzap.Level(logLevel),
		ctrlzap.StacktraceLevel(zapcore.PanicLevel)),
	)

	return nil
}

func customSetupHealthChecks(mgr manager.Manager) {
	err := mgr.AddHealthzCheck("goroutines-number", func(req *http.Request) error {
		if goruntime.NumGoroutine() > maximumGoroutines {
			return fmt.Errorf("too much goroutines: %d > limit: %d", goruntime.NumGoroutine(), maximumGoroutines)
		}

		return nil
	})
	if err != nil {
		setupLog.Error(err, "Unable to start ")
	}
}

func customSetupEndpoints(pprofActive bool, mgr manager.Manager) {
	if pprofActive {
		if err := debug.RegisterEndpoint(mgr.AddMetricsExtraHandler, nil); err != nil {
			setupLog.Error(err, "Unable to register pprof endpoint")
		}
	}

	if err := metrics.RegisterEndpoint(mgr, mgr.AddMetricsExtraHandler); err != nil {
		setupLog.Error(err, "Unable to register custom metrics endpoints")
	}
}
