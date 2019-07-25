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
	"github.com/go-logr/logr"
	buildclientset "github.com/knative/build/pkg/client/clientset/versioned"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// PipelineReconciler reconciles a Pipeline object
type PipelineReconciler struct {
	client.Client
	Log         logr.Logger
	BuildClient *buildclientset.Clientset
	KubeClient  *kubernetes.Clientset
	CronHandler *CronTaskSet
}

// +kubebuilder:rbac:groups=controller.ljchen.net,resources=pipelines,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=controller.ljchen.net,resources=pipelines/status,verbs=get;update;patch

func (r *PipelineReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("p", req.NamespacedName)
	ctx := context.WithValue(context.Background(), "log", log)

	// your logic here
	p := new(v1.Pipeline)
	err := r.Get(ctx, req.NamespacedName, p)

	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("delete pipeline success!")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Get pipeline fail")
		return ctrl.Result{}, err
	}

	// delete
	if p.ObjectMeta.GetDeletionTimestamp() != nil {
		r.CronHandler.release(p)

		if result := resources.RemoveFinalizer(p, YonghuiFinalizer); result == resources.FinalizerRemoved {
			if err := r.Update(ctx, p); err != nil {
				r.Log.Error(err, "Failed to remove finalizer")
				return reconcile.Result{Requeue: true}, nil
			}
		}

		return ctrl.Result{}, nil
	}

	// create
	if isExpired(p) || isRegistered(p) {
		log.Info("pipeline is registered or expired.", p.Namespace, p.Name)
		return ctrl.Result{}, nil
	}

	go r.CronHandler.wait(p)

	return ctrl.Result{}, nil
}

func (r *PipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pipeline{}).
		Complete(r)
}
