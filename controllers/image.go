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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	certifiedIndex                  = "certified-operator-index"
	certifiedImageStreamAvailable   = "CertifiedImageStreamAvailable"
	marketplaceIndex                = "redhat-marketplace-index"
	marketplaceImageStreamAvailable = "MarketplaceImageStreamAvailable"
)

// reconcileCertifiedImageStream will ensure that the certified operator ImageStream is present and up to date.
func (r *OperatorPipelineReconciler) reconcileCertifiedImageStream(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) error {
	if err := r.updateStatusCondition(ctx, pipeline, certifiedImageStreamAvailable, metav1.ConditionUnknown, reconcileUnknown,
		""); err != nil {
		return err
	}

	key := types.NamespacedName{
		Namespace: pipeline.Namespace,
		Name:      certifiedIndex,
	}

	stream := newImageStream(key)
	if IsObjectFound(ctx, r.Client, key, stream) {
		log.Info("existing certified image stream found")
		return nil // Existing ImageStream found, do nothing...
	}

	imgImport := newImageStreamImport(key)
	imgImport.Spec.Import = true
	imgImport.Spec.Repository = &imagev1.RepositoryImportSpec{
		From: corev1.ObjectReference{
			Kind: "DockerImage",
			Name: "registry.redhat.io/redhat/certified-operator-index",
		},
		ImportPolicy: imagev1.TagImportPolicy{
			Scheduled: true,
		},
		ReferencePolicy: imagev1.TagReferencePolicy{
			Type: imagev1.LocalTagReferencePolicy,
		},
	}

	log.Info("creating new certified image stream import")
	if err := r.Client.Create(ctx, imgImport); err != nil {
		if err := r.updateStatusCondition(ctx, pipeline, certifiedImageStreamAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err := r.updateStatusCondition(ctx, pipeline, certifiedImageStreamAvailable, metav1.ConditionTrue, reconcileSucceeded,
		""); err != nil {
		return err
	}

	return nil
}

// reconcileMarketplaceImageStream will ensure that the Red Hat Marketplace ImageStream is present and up to date.
func (r *OperatorPipelineReconciler) reconcileMarketplaceImageStream(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) error {
	if err := r.updateStatusCondition(ctx, pipeline, marketplaceImageStreamAvailable, metav1.ConditionUnknown, reconcileUnknown,
		""); err != nil {
		return err
	}

	key := types.NamespacedName{
		Namespace: pipeline.Namespace,
		Name:      marketplaceIndex,
	}

	stream := newImageStream(key)
	if IsObjectFound(ctx, r.Client, key, stream) {
		log.Info("existing marketplace image stream found")
		return nil // Existing ImageStream found, do nothing...
	}

	imgImport := newImageStreamImport(key)
	imgImport.Spec.Import = true
	imgImport.Spec.Repository = &imagev1.RepositoryImportSpec{
		From: corev1.ObjectReference{
			Kind: "DockerImage",
			Name: "registry.redhat.io/redhat/redhat-marketplace-index",
		},
		ImportPolicy: imagev1.TagImportPolicy{
			Scheduled: true,
		},
		ReferencePolicy: imagev1.TagReferencePolicy{
			Type: imagev1.LocalTagReferencePolicy,
		},
	}

	log.Info("creating new marketplace image stream import")
	if err := r.Client.Create(ctx, imgImport); err != nil {
		if err := r.updateStatusCondition(ctx, pipeline, marketplaceImageStreamAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err := r.updateStatusCondition(ctx, pipeline, marketplaceImageStreamAvailable, metav1.ConditionTrue, reconcileSucceeded,
		""); err != nil {
		return err
	}

	return nil
}

// newImageStream will create and return a new ImageStream instance using the given Name/Namespace.
func newImageStream(key types.NamespacedName) *imagev1.ImageStream {
	return &imagev1.ImageStream{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
}

// newImageStreamImport will create and return a new ImageStreamImport instance using the given Name/Namespace.
func newImageStreamImport(key types.NamespacedName) *imagev1.ImageStreamImport {
	return &imagev1.ImageStreamImport{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
}
