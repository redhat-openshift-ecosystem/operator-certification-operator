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

package controller

import (
	"context"

	"github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/reconcilers"

	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	securityv1 "github.com/openshift/api/security/v1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const operatorPipelineFinalizer = "certification.redhat.com/finalizer"

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
//+kubebuilder:rbac:groups=security.openshift.io,resources=securitycontextconstraints,verbs=*
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings;clusterroles;clusterrolebindings,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelines;tasks,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *OperatorPipelineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	reqLogger := logf.FromContext(ctx, "Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling OperatorPipeline")

	currentPipeline := &v1alpha1.OperatorPipeline{}
	err := r.Client.Get(ctx, req.NamespacedName, currentPipeline)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request. Return and don't
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	// Check if the OperatorPipeline instance is marked to be deleted, which is
	// indicated by the deletion timestamp being set.
	isOperatorPipelineMarkedToBeDeleted := currentPipeline.GetDeletionTimestamp() != nil
	if isOperatorPipelineMarkedToBeDeleted {
		if controllerutil.ContainsFinalizer(currentPipeline, operatorPipelineFinalizer) {
			namespacePipelines := &v1alpha1.OperatorPipelineList{}

			// creating listOptions inorder to know the number of OperatorPipeline resources in the given namespace
			// if last CR in namespace we can remove the ClusterRoleBinding associated with the namespace
			listOptions := client.InNamespace(currentPipeline.Namespace)
			if err := r.Client.List(ctx, namespacePipelines, listOptions); err != nil {
				return ctrl.Result{}, err
			}

			// if the length is 1 we know this is the last CR in a given namespace and can remove the ClusterRoleBinding
			// associated with this namespace
			if len(namespacePipelines.Items) == 1 {
				if err := r.deleteClusterRoleBinding(ctx, log, currentPipeline.Namespace); err != nil {
					return ctrl.Result{}, err
				}
			}

			clusterPipelines := &v1alpha1.OperatorPipelineList{}
			// not creating listOptions since we want to know the total number of OperatorPipelines in the entire cluster
			// if last CR in cluster we can remove the SCC and ClusterRole
			if err := r.Client.List(ctx, clusterPipelines); err != nil {
				return ctrl.Result{}, err
			}

			// if the length is 1 we know this is the last CR in the entire cluster and can remove the SCC and ClusterRole
			if len(clusterPipelines.Items) == 1 {
				if err := r.deleteSCCandClusterRole(ctx, log); err != nil {
					return ctrl.Result{}, err
				}
			}

			// Remove operatorPipelineFinalizer. Once all finalizers have been
			// removed, the object will be deleted.
			controllerutil.RemoveFinalizer(currentPipeline, operatorPipelineFinalizer)
			if err := r.Update(ctx, currentPipeline); err != nil {
				return ctrl.Result{}, err
			}
		}
		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	resourceReconcilers := []reconcilers.Reconciler{
		reconcilers.NewPipelineGitRepoReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewPipeDependenciesReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewCertifiedImageStreamReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewMarketplaceImageStreamReconciler(r.Client, reqLogger, r.Scheme),
		reconcilers.NewStatusReconciler(r.Client, reqLogger, r.Scheme),
	}

	requeueResult := false
	var errResult error = nil
	pipeline := currentPipeline.DeepCopy()
	for _, r := range resourceReconcilers {
		requeue, err := r.Reconcile(ctx, pipeline)
		if err != nil && errResult == nil {
			// Only capture the first error
			log.Error(err, "requeuing with error")
			errResult = err
		}
		requeueResult = requeueResult || requeue
	}

	// Adding finalizer to OperatorPipelines CR
	if !controllerutil.ContainsFinalizer(currentPipeline, operatorPipelineFinalizer) {
		controllerutil.AddFinalizer(currentPipeline, operatorPipelineFinalizer)
		if err := r.Update(ctx, currentPipeline); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Just return the first error reported. It's the most likely issue that needs to be solved.
	return ctrl.Result{Requeue: requeueResult}, errResult
}

func (r *OperatorPipelineReconciler) deleteSCCandClusterRole(ctx context.Context, log logr.Logger) error {
	log.Info("starting deletion of owned SCC and ClusterRole")

	listOption := client.MatchingLabels{
		reconcilers.ClusterResourceLabel: "true",
	}

	crList := &rbacv1.ClusterRoleList{}
	if err := r.Client.List(ctx, crList, listOption); err != nil {
		return err
	}

	for _, cr := range crList.Items {
		if err := r.Client.Delete(ctx, &cr); err != nil {
			return err
		}
	}

	sccList := &securityv1.SecurityContextConstraintsList{}
	if err := r.Client.List(ctx, sccList, listOption); err != nil {
		return err
	}

	for _, scc := range sccList.Items {
		if err := r.Client.Delete(ctx, &scc); err != nil {
			return err
		}
	}

	return nil
}

func (r *OperatorPipelineReconciler) deleteClusterRoleBinding(ctx context.Context, log logr.Logger, nameSpace string) error {
	log.Info("starting deletion of owned ClusterRoleBinding")

	listOption := client.MatchingLabels{
		reconcilers.ClusterResourceLabel: "true",
		reconcilers.NamespaceLabel:       nameSpace,
	}

	crbList := &rbacv1.ClusterRoleBindingList{}
	if err := r.Client.List(ctx, crbList, listOption); err != nil {
		return err
	}

	for _, crb := range crbList.Items {
		if err := r.Client.Delete(ctx, &crb); err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OperatorPipelineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.OperatorPipeline{}).
		Owns(&corev1.Secret{}).
		Owns(&imagev1.ImageStream{}).
		Owns(&tekton.Pipeline{}).
		Owns(&tekton.Task{}).
		Owns(&securityv1.SecurityContextConstraints{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Complete(r)
}
