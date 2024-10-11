//
// Copyright 2021 Red Hat Inc.
// SPDX-License-Identifier: Apache-2.0
//

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	configv1 "github.com/openshift/api/config/v1"
	operatorsv1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	metricscfgv1beta1 "github.com/project-koku/koku-metrics-operator/api/v1beta1"
	"github.com/project-koku/koku-metrics-operator/internal/controller"
	// +kubebuilder:scaffold:imports
)

var (
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
	defaultLeaseDuration = 60 * time.Second
	defaultRenewDeadline = 30 * time.Second
	defaultRetryPeriod   = 5 * time.Second
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// Adding the configv1 scheme
	utilruntime.Must(configv1.AddToScheme(scheme))
	// Adding the metricscfgv1beta1 scheme
	utilruntime.Must(metricscfgv1beta1.AddToScheme(scheme))
	// Adding the operatorsv1alpha1 scheme
	utilruntime.Must(operatorsv1alpha1.AddToScheme(scheme))

	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var probeAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// fetch leader election configurations from environment variables
	leaseDuration := getEnvVarDuration("LEADER_ELECTION_LEASE_DURATION", defaultLeaseDuration)
	renewDeadline := getEnvVarDuration("LEADER_ELECTION_RENEW_DEADLINE", defaultRenewDeadline)
	retryPeriod := getEnvVarDuration("LEADER_ELECTION_RETRY_PERIOD", defaultRetryPeriod)

	// validate leader election
	leaseDuration, renewDeadline, retryPeriod = validateLeaderElectionConfig(leaseDuration, renewDeadline, retryPeriod)

	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339NanoTimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	inCluster := false
	if value, ok := os.LookupEnv("IN_CLUSTER"); ok {
		inCluster = value == "true"
	}

	watchNamespace, err := getWatchNamespace()
	if err != nil {
		setupLog.Error(err, "unable to get WatchNamespace, "+
			"the manager will watch and manage resources in all namespaces")
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,

		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "91c624a5.openshift.io",
		LeaseDuration:    &leaseDuration,
		RenewDeadline:    &renewDeadline,
		RetryPeriod:      &retryPeriod,
		Cache:            cache.Options{DefaultNamespaces: map[string]cache.Config{watchNamespace: {}}},
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	clientset, err := controller.GetClientset()
	if err != nil {
		setupLog.Error(err, "unable to get clientset")
		os.Exit(1)
	}

	if err = (&controller.MetricsConfigReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		Clientset: clientset,
		InCluster: inCluster,
		Namespace: watchNamespace,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "MetricsConfig")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// getWatchNamespace returns the Namespace the operator should be watching for changes
func getWatchNamespace() (string, error) {
	// WatchNamespaceEnvVar is the constant for env variable WATCH_NAMESPACE
	// which specifies the Namespace to watch.
	// An empty value means the operator is running with cluster scope.
	var watchNamespaceEnvVar = "WATCH_NAMESPACE"

	ns, found := os.LookupEnv(watchNamespaceEnvVar)
	if !found {
		return "", fmt.Errorf("%s must be set", watchNamespaceEnvVar)
	}
	return ns, nil
}

// GetEnvVar returns the value from an environment variable
// or the provided default value if the variable does not exist.
func getEnvVarString(varName, defaultValue string) string {
	if value, exists := os.LookupEnv(varName); exists {
		return value
	}
	return defaultValue
}

// Returns time.Duration parsed from an env variable
// or the default value if the variable does not exist or does not parse into a duration.
func getEnvVarDuration(varName string, defaultValue time.Duration) time.Duration {
	val := getEnvVarString(varName, "")

	if val == "" {
		return defaultValue
	}

	parsedVal, err := time.ParseDuration(val)
	if err != nil {
		setupLog.Error(err, "Invalid boolean format for environment variable", "Variable", varName, "Value", val)
		return defaultValue
	}
	return parsedVal

}

// validateLeaderElectionConfig returns the Namespace the operator should be watching for changes
func validateLeaderElectionConfig(leaseDuration, renewDeadline, retryPeriod time.Duration) (time.Duration, time.Duration, time.Duration) {

	// validate that renewDeadlne < leaseDuration
	if renewDeadline >= leaseDuration {

		setupLog.Info("Invalid configuration: LEADER_ELECTION_RENEW_DEADLINE must be less that LEADER_ELECTION_LEASE_DURATION; using default values",
			"Provided LEADER_ELECTION_LEASE_DURATION", leaseDuration,
			"Provided LEADER_ELECTION_RENEW_DEADLINE", renewDeadline,
			"Default LEADER_ELECTION_LEASE_DURATION", defaultLeaseDuration,
			"Default LEADER_ELECTION_RENEW_DEADLINE", defaultRenewDeadline,
		)

		leaseDuration = defaultLeaseDuration
		renewDeadline = defaultRenewDeadline
	}

	// validate that retryPeriod < renewDeadlne
	if retryPeriod >= renewDeadline {
		setupLog.Info("Invalid configuration: LEADER_ELECTION_RETRY_PERIOD must be less that LEADER_ELECTION_RENEW_DEADLINE; using default values",
			"Provided LEADER_ELECTION_RETRY_PERIOD", retryPeriod,
			"Provided LEADER_ELECTION_RENEW_DEADLINE", renewDeadline,
			"Default LEADER_ELECTION_RETRY_PERIOD", defaultRetryPeriod,
			"Default LEADER_ELECTION_RENEW_DEADLINE", defaultRenewDeadline,
		)
		retryPeriod = defaultRetryPeriod
		renewDeadline = defaultRenewDeadline
	}

	return leaseDuration, renewDeadline, retryPeriod
}
