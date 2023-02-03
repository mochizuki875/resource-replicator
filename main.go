/*
Copyright 2023.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	replicatev1 "github.com/jnytnai0613/resource-replicator/api/v1"
	"github.com/jnytnai0613/resource-replicator/controllers"
	cli "github.com/jnytnai0613/resource-replicator/pkg/client"
	"github.com/jnytnai0613/resource-replicator/pkg/healthcheck"
	"github.com/jnytnai0613/resource-replicator/pkg/kubeconfig"
	//+kubebuilder:scaffold:imports
)

var (
	LocalClient          client.Client
	enableLeaderElection bool
	metricsAddr          string
	probeAddr            string
	scheme               = runtime.NewScheme()
	setupLog             = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(replicatev1.AddToScheme(scheme))

	//+kubebuilder:scaffold:scheme

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	if err := initCreate(); err != nil {
		setupLog.Error(err, "Failed to initialize ClusterDetector.")
	}
}

// Create a Custom Resource ClusterDetector and register the remote cluster status.
func initCreate() error {
	localClient, _ := cli.CreateLocalClient(setupLog, *scheme)
	_, apiServer, targetCluster, err := kubeconfig.ReadKubeconfig(localClient)
	if err != nil {
		setupLog.Error(err, "Failure to read kubeconfig.")
	}

	var clusterDetector = &replicatev1.ClusterDetector{}

	clusterDetector.SetNamespace("resource-replicator-system")

	ctx := context.Background()

	for _, t := range targetCluster {
		clusterDetector.SetName(t.ContextName)
		/////////////////////////////
		// Create ClusterDetector
		/////////////////////////////

		if op, err := ctrl.CreateOrUpdate(ctx, localClient, clusterDetector, func() error {
			clusterDetector.Spec.Context = t.ContextName
			clusterDetector.Spec.Cluster = t.ClusterName
			clusterDetector.Spec.User = t.UserName

			return nil
		}); op != controllerutil.OperationResultNone {
			setupLog.Info(fmt.Sprintf("ClusterDetector %s\n", op))
		} else if err != nil {
			return err
		}

		/////////////////////////////
		// Update Status
		/////////////////////////////
		for _, a := range apiServer {
			if t.ClusterName == a.Name {
				// Verify that you can communicate with the remote Kubernetes cluster.
				if err := healthcheck.HealthChecks(a); err != nil {
					setupLog.Error(err, fmt.Sprintf("[Cluster: %s] Health Check failed.", a.Name))
					clusterDetector.Status.ClusterStatus = "Unknown"
					break
				}
				clusterDetector.Status.ClusterStatus = "Running"
				break
			}
		}
		if err := localClient.Status().Update(ctx, clusterDetector); err != nil {
			return err
		}
		setupLog.Info(fmt.Sprintf("[ClusterDetector: %s] Complete status update", t.ContextName))
	}

	return nil
}

func main() {
	var resyncPeriod = time.Second * 30

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		SyncPeriod:             &resyncPeriod,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "90fe7186.jnytnai0613.github.io",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ClusterDetectorReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("ClusterDetector"),
		Recorder: mgr.GetEventRecorderFor("ClusterDetector"),
		Scheme:   mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ClusterDetector")
		os.Exit(1)
	}
	//+kubebuilder:scaffold:builder

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
