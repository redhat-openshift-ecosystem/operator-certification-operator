/*
Copyright 2021.

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

	imagev1 "github.com/openshift/api/image/v1"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/reconcilers"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("controller_operatorpipeline")

// OperatorPipelineReconciler reconciles a OperatorPipeline object
type OperatorPipelineReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=certification.redhat.com,resources=operatorpipelines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=certification.redhat.com,resources=operatorpipelines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=certification.redhat.com,resources=operatorpipelines/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=image.openshift.io,resources=imagestreamimports,verbs=create
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelines;tasks,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OperatorPipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logf.FromContext(ctx, "Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling OperatorPipeline")

	currentPipeline := &certv1alpha1.OperatorPipeline{}
	err := r.Client.Get(ctx, req.NamespacedName, currentPipeline)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	reconcilers := []reconcilers.Reconciler{
		reconcilers.NewPipelineGitRepoReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewPipeDependenciesReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewCertifiedImageStreamReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewMarketplaceImageStreamReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewStatusReconciler(r.Client, reqLogger, r.Scheme),
	}

	requeueResult := false
	pipeline := currentPipeline.DeepCopy()
	for _, r := range reconcilers {
		requeue, err := r.Reconcile(ctx, pipeline)
		if err != nil {
			log.Error(err, "requeuing with error")
			return ctrl.Result{Requeue: true}, err
		}
		requeueResult = requeueResult || requeue
	}

	return ctrl.Result{Requeue: requeueResult}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorPipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&certv1alpha1.OperatorPipeline{}).
		Owns(&corev1.Secret{}).
		Owns(&imagev1.ImageStream{}).
		Owns(&tekton.Pipeline{}).
		Owns(&tekton.Task{}).
		Complete(r)
}
