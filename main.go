/*

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
	"flag"
	controllerv1 "github.com/chenleji/pipeline-operator/api/v1"
	"github.com/chenleji/pipeline-operator/controllers"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	"github.com/knative/pkg/signals"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	controllerv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

var (
	metricsAddr          = flag.String("metrics-addr", ":8080", "The address the metric endpoint binds to.")
	enableLeaderElection = flag.Bool("enable-leader-election", false,
		"Enable leader election for controller manager. Enabling this will ensure there is only one active controller manager.")
)

func main() {
	flag.Parse()

	ctrl.SetLogger(zap.Logger(true))
	stopCh := signals.SetupSignalHandler()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: *metricsAddr,
		LeaderElection:     *enableLeaderElection,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// build client
	buildClient, err := buildclientset.NewForConfig(ctrl.GetConfigOrDie())
	if err != nil {
		setupLog.Error(err, "Error building Build clientset")
	}

	// kubernetes client
	kubeClient := kubernetes.NewForConfigOrDie(ctrl.GetConfigOrDie())

	// cron handler
	cronHandler := controllers.NewCronHandler(
		ctrl.Log.WithName("cronHandler").WithName("Pipeline"),
		mgr.GetClient(),
		buildClient,
		kubeClient,
		stopCh,
	)
	go cronHandler.CheckCronTask()

	// pipelineRun producer handler
	runProducer := controllers.NewPipelineRunProducer(
		ctrl.Log.WithName("pipeineRunProducer"),
		mgr.GetClient(),
		kubeClient,
		stopCh,
	)
	go runProducer.Handler()

	// pipeline controller
	err = (&controllers.PipelineReconciler{
		Client:      mgr.GetClient(),
		Log:         ctrl.Log.WithName("controllers").WithName("Pipeline"),
		BuildClient: buildClient,
		KubeClient:  kubeClient,
		CronHandler: cronHandler,
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Pipeline")
		os.Exit(1)
	}

	// pipelineRun controller
	err = (&controllers.PipelineRunReconciler{
		Client:      mgr.GetClient(),
		Log:         ctrl.Log.WithName("controllers").WithName("PipelineRun"),
		BuildClient: buildClient,
		KubeClient:  kubeClient,
	}).SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "PipelineRun")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(stopCh); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
