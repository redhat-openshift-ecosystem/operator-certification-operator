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
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FetchObject will retrieve the object with the given namespace and name using the Kubernetes API.
// The result will be stored in the given object.
func FetchObject(client client.Client, key types.NamespacedName, obj client.Object) error {
	return client.Get(context.TODO(), key, obj)
}

// IsObjectFound will perform a basic check that the given object exists via the Kubernetes API.
// If an error occurs as part of the check, the function will return false.
func IsObjectFound(client client.Client, key types.NamespacedName, obj client.Object) bool {
	return !apierrors.IsNotFound(FetchObject(client, key, obj))
}

// reconcileResources will ensure that all required resources are present and up to date.
func (r *OperatorPipelineReconciler) reconcileResources(pipeline *certv1alpha1.OperatorPipeline) error {

	if err := r.reconcilePipelineOperator(pipeline.ObjectMeta); err != nil {
		return err
	}

	if err := r.reconcilePipelineDependencies(pipeline); err != nil {
		return err
	}

	if err := r.ensureKubeConfigSecret(pipeline.ObjectMeta); err != nil {
		return err
	}

	if err := r.ensureGitHubAPISecret(pipeline.ObjectMeta); err != nil {
		return err
	}

	if err := r.ensurePyxisAPISecret(pipeline.ObjectMeta); err != nil {
		return err
	}

	if err := r.reconcileCertifiedImageStream(pipeline.ObjectMeta); err != nil {
		return err
	}

	if err := r.reconcileMarketplaceImageStream(pipeline.ObjectMeta); err != nil {
		return err
	}

	return nil
}
