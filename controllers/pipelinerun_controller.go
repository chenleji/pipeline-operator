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

package controllers

import (
	"context"
	"github.com/chenleji/pipeline-operator/api/v1"
	"github.com/chenleji/pipeline-operator/resources"
	"github.com/astaxie/beego/logs"
	"github.com/go-logr/logr"
	"github.com/knative/build/pkg/apis/build/v1alpha1"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	duckV1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"
)

const (
	YonghuiFinalizer        = "finalizer.ljchen.net"
	SyncBuildStatusInterval = 5 * time.Second
)

// PipelineRunReconciler reconciles a PipelineRun object
type PipelineRunReconciler struct {
	client.Client
	Log         logr.Logger
	BuildClient *buildclientset.Clientset
	KubeClient  *kubernetes.Clientset
}

// +kubebuilder:rbac:groups=controller.ljchen.net,resources=pipelineruns,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=controller.ljchen.net,resources=pipelineruns/status,verbs=get;update;patch

func (r *PipelineRunReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("pipelineRun", req.NamespacedName)
	ctx := context.WithValue(context.Background(), "log", log)

	// Get the pipelineRun resource with this namespace/name
	run := &v1.PipelineRun{}
	err := r.Get(ctx, req.NamespacedName, run)
	if apierrors.IsNotFound(err) {
		log.Info("pipelineRun in work queue no longer exists!")
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	// Don't mutate the informer's copy of our object.
	run = run.DeepCopy()
	originState := run.Status

	// delete
	if run.ObjectMeta.GetDeletionTimestamp() != nil {
		if result := resources.RemoveFinalizer(run, YonghuiFinalizer); result == resources.FinalizerRemoved {
			if err := r.Update(ctx, run); err != nil {
				r.Log.Error(err, "Failed to remove finalizer")
				return reconcile.Result{Requeue: true}, err
			}
			return reconcile.Result{}, err
		}
	}

	// ignore done, expired
	if resources.IsStatusDone(&run.Status.Status) ||
		resources.IsStatusExpired(&run.Status.Status) {
		return ctrl.Result{}, nil
	}

	// failed, retry
	if resources.IsStatusFailed(&run.Status.Status) {
		if err := r.handleFailed(ctx, run); err != nil {
			return ctrl.Result{RequeueAfter: time.Second}, err
		}
		return ctrl.Result{}, nil
	}

	// reconcile
	if run.Status.Cluster == nil {
		err = r.reconcile(ctx, run)
		if err != nil {
			r.Log.Error(err, "Error reconciling run")
		} else {
			r.Log.Info("run reconciled")
		}
	} else {
		if run.Status.Cluster.BuildName != "" {
			build, err := r.BuildClient.BuildV1alpha1().
				Builds(run.Status.Cluster.Namespace).
				Get(run.Status.Cluster.BuildName, metav1.GetOptions{})
			if err != nil {
				r.Log.Error(err, "get build fail....")
				return ctrl.Result{RequeueAfter: time.Second}, nil
			}
			run = resources.MergePipelineRunStatus(run, build)
		}
	}

	//update status
	if equality.Semantic.DeepEqual(originState, run.Status) {
		logs.Debug("[pipelineRun] status are equal... ", run.Namespace, run.Name)
		return reconcile.Result{RequeueAfter: SyncBuildStatusInterval}, nil
	}

	if err := r.Status().Update(ctx, run); err != nil {
		logs.Error("failed to update pipelineRun status, err: %s", err.Error())
		return reconcile.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *PipelineRunReconciler) reconcile(ctx context.Context, run *v1.PipelineRun) error {
	log := ctx.Value("log").(logr.Logger)

	p := r.getRefPipeline(ctx, run)
	if p == nil {
		log.Info("p is null...")
		return nil
	}

	// set owner references
	if len(run.GetOwnerReferences()) == 0 {
		run.SetOwnerReferences([]metav1.OwnerReference{
			*metav1.NewControllerRef(p.GetObjectMeta(), p.GroupVersionKind()),
		})

		if err := r.Client.Update(ctx, run); err != nil {
			logs.Error("[pipelineRun] update ownerReferences err ...", err.Error())
		}
		return nil
	}

	// batch job
	if p.Spec.BatchJob != nil {
		return r.batchJobReconcile(ctx, run, p)
	}

	// stream job
	if p.Spec.StreamJob != nil {
		return r.streamJobReconcile(ctx, run, p)
	}

	return nil
}

func (r *PipelineRunReconciler) batchJobReconcile(ctx context.Context, run *v1.PipelineRun, p *v1.Pipeline) error {
	log := ctx.Value("log").(logr.Logger)

	build := resources.MakeBuild(p, run)
	_, err := r.BuildClient.BuildV1alpha1().Builds(build.Namespace).Get(build.Name, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			if isExpired(p) {
				run.Status.SetCondition(&duckV1alpha1.Condition{
					Type:    v1alpha1.BuildCancelled,
					Status:  corev1.ConditionFalse,
					Reason:  "Pipeline is expired",
					Message: "Skip the build creation",
				})

				log.Info("p is expired... [skip]")
				return nil
			}

			if _, err := r.BuildClient.BuildV1alpha1().Builds(build.Namespace).Create(build); err != nil {
				run.Status.SetCondition(&duckV1alpha1.Condition{
					Type:    v1alpha1.BuildSucceeded,
					Status:  corev1.ConditionFalse,
					Reason:  "Create build failed!",
					Message: "Please check log info.",
				})
				return err
			}

			// update status
			run.Status.Cluster = &v1.ClusterSpec{
				Namespace: build.Namespace,
				BuildName: build.Name,
			}
			run.Status.InitializeConditions()

		} else {
			run.Status.SetCondition(&duckV1alpha1.Condition{
				Type:    v1alpha1.BuildSucceeded,
				Status:  corev1.ConditionFalse,
				Reason:  "Get build failed!",
				Message: "Please check log info.",
			})
			return err
		}
	} else {
		run.Status.SetCondition(&duckV1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionFalse,
			Reason:  "Create build failed!",
			Message: "Build exist, please create new one.",
		})
		return err
	}

	return nil
}

func (r *PipelineRunReconciler) streamJobReconcile(ctx context.Context, run *v1.PipelineRun, p *v1.Pipeline) error {
	log := ctx.Value("log").(logr.Logger)

	deployNames := make([]string, 0)
	svcNames := make([]string, 0)
	deploys, svcs := resources.MakeDeploysAndServices(p, run)

	// deploy
	for _, deploy := range deploys {
		deployNames = append(deployNames, deploy.Name)
		origin, err := r.KubeClient.AppsV1().Deployments(deploy.Namespace).Get(deploy.Name, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				if _, err = r.KubeClient.AppsV1().Deployments(deploy.Namespace).Create(&deploy); err != nil {
					log.Error(err, "create deploy fail...")
					return err
				}
			}
		}
		if origin != nil {
			continue
		}
	}

	// svc
	for _, svc := range svcs {
		svcNames = append(svcNames, svc.Name)
		origin, err := r.KubeClient.CoreV1().Services(svc.Namespace).Get(svc.Name, metav1.GetOptions{})
		if err != nil {
			if _, err := r.KubeClient.CoreV1().Services(svc.Namespace).Create(&svc); err != nil {
				log.Error(err, "create svc fail...")
				return err
			}
		}
		if origin != nil {
			continue
		}
	}

	// status
	run.Status.Cluster = &v1.ClusterSpec{
		Namespace:       p.Namespace,
		DeploymentNames: deployNames,
		ServiceNames:    svcNames,
	}
	run.Status.SetCondition(&duckV1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionTrue,
		Reason:  "success!",
		Message: "deploy & svc created.",
	})

	return nil
}

func (r *PipelineRunReconciler) getRefPipeline(ctx context.Context, run *v1.PipelineRun) *v1.Pipeline {
	p := &v1.Pipeline{}

	if err := r.Get(ctx, types.NamespacedName{
		Namespace: run.Namespace,
		Name:      run.Spec.RefPipeline,
	}, p); err != nil {
		return nil
	}

	return p
}

func (r *PipelineRunReconciler) handleFailed(ctx context.Context, run *v1.PipelineRun) error {
	maybeRetry := func(run *v1.PipelineRun) bool {
		if run.Spec.MaxRetryCount == 0 {
			return false
		}
		if run.Status.RetryStatus != nil {
			if run.Status.RetryStatus.RetryCount < run.Spec.MaxRetryCount {
				return true
			}
			return false
		}
		return true
	}

	if run.Status.RetryStatus == nil {
		run.Status.RetryStatus = &v1.RetryStatus{
			RetryCount: 0,
			HistoryBuild: []v1.ClusterSpec{
				{
					Namespace: run.Status.Cluster.Namespace,
					BuildName: run.Status.Cluster.BuildName,
				},
			},
		}
	} else {
		run.Status.RetryStatus = &v1.RetryStatus{
			RetryCount: run.Status.RetryStatus.RetryCount + 1,
			HistoryBuild: append(run.Status.RetryStatus.HistoryBuild, v1.ClusterSpec{
				Namespace: run.Status.Cluster.Namespace,
				BuildName: run.Status.Cluster.BuildName,
			}),
		}
	}

	if maybeRetry(run) {
		run.Status.Cluster = nil
		run.Status.SetCondition(&duckV1alpha1.Condition{
			Type:    v1alpha1.BuildSucceeded,
			Status:  corev1.ConditionUnknown,
			Reason:  resources.ReasonRetry,
			Message: resources.ReasonRetry,
		})

		if err := r.Status().Update(ctx, run); err != nil {
			r.Log.Error(err, "Error updating run retry status")
			return err
		}

		logs.Debug("[pipelineRun] prepare to retry ...", run.Namespace, run.Name, run.Status.RetryStatus.RetryCount)
		return nil
	}

	// finally failed
	if resources.IsStatusFailedFinallyfail(&run.Status.Status) {
		logs.Debug("[pipelineRun] announce finally fail ...", run.Namespace, run.Name)
		return nil
	}

	// update retryStatus
	run.Status.SetCondition(&duckV1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Reason:  resources.ReasonFail,
		Message: resources.ReasonFail,
	})

	if err := r.Status().Update(ctx, run); err != nil {
		r.Log.Error(err, "Error updating run failed status")
		return err
	}

	return nil
}

func (r *PipelineRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.PipelineRun{}).
		Complete(r)
}
