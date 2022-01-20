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

	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchObject will retrieve the object with the given namespace and name using the Kubernetes API.
// The result will be stored in the given object.
func fetchObject(ctx context.Context, client client.Client, key types.NamespacedName, obj client.Object) error {
	return client.Get(ctx, key, obj)
}

// IsObjectFound will perform a basic check that the given object exists via the Kubernetes API.
// If an error occurs as part of the check, the function will return false.
func isObjectFound(ctx context.Context, client client.Client, key types.NamespacedName, obj client.Object) bool {
	return !apierrors.IsNotFound(fetchObject(ctx, client, key, obj))
}

const (
	reconcileSucceeded = "ReconcileSucceeded"
	reconcileFailed    = "ReconcileFailed"
	reconcileUnknown   = "ReconcileUnknown"
)

// TODO: bcrochet All of this code will go away

// reconcileResources will ensure that all required resources are present and up to date.
func (r *OperatorPipelineReconciler) reconcileResources(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) error {

	if err := r.reconcilePipelineDependencies(ctx, pipeline); err != nil {
		return err
	}

	return nil
}

// updateStatusCondition updates the status/reason/message for each condition for which we reconcile
func (r *OperatorPipelineReconciler) updateStatusCondition(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline,
	conditionType string, status metav1.ConditionStatus, reason, message string) error {
	if err := r.Client.Get(ctx, types.NamespacedName{Name: pipeline.GetName(), Namespace: pipeline.GetNamespace()}, pipeline); err != nil {
		return err
	}
	// checking to see if a given condition type is present, since on each reconcile we do not want to set condition
	// to "Unknown" unless that conditionType has not previously been set to "True" or "False"
	if (meta.IsStatusConditionPresentAndEqual(pipeline.Status.Conditions, conditionType, metav1.ConditionTrue) ||
		meta.IsStatusConditionPresentAndEqual(pipeline.Status.Conditions, conditionType, metav1.ConditionFalse)) && status == metav1.ConditionUnknown {
		return nil
	}

	meta.SetStatusCondition(&pipeline.Status.Conditions, metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	})

	if err := r.Client.Status().Update(ctx, pipeline); err != nil {
		return err
	}

	return nil
}
