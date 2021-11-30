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
	"fmt"

	operatorsv1a1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	operatorNamespace                      = "openshift-operators"
	pipelineOperatorChannel                = "stable"
	pipelineOperatorCatalogSource          = "redhat-operators"
	pipelineOperatorCatalogSourceNamespace = "openshift-marketplace"
	pipelineOperatorName                   = "redhat-openshift-pipelines"
	pipelineOperatorVersion                = "v1.5.2"
	pipelineOperatorSubscription           = "openshift-pipelines-operator-rh"
)

// reconcilePipelineOperator will ensure that the resources are present to provision the OpenShift Pipeline Operator
func (r *OperatorPipelineReconciler) reconcilePipelineOperator(ctx context.Context, meta metav1.ObjectMeta) error {
	log.Info("reconciling pipeline operator")

	key := types.NamespacedName{
		Namespace: operatorNamespace,
		Name:      pipelineOperatorSubscription,
	}

	sub := newSubscription(key)
	if IsObjectFound(ctx, r.Client, key, sub) {
		log.Info("existing subscription found")
		return nil // Existing Subscription found, do nothing...
	}

	sub.Spec = &operatorsv1a1.SubscriptionSpec{
		Channel:                pipelineOperatorChannel,
		InstallPlanApproval:    operatorsv1a1.ApprovalAutomatic, // Use ApprovalManual instead?
		Package:                pipelineOperatorSubscription,
		CatalogSource:          pipelineOperatorCatalogSource,
		CatalogSourceNamespace: pipelineOperatorCatalogSourceNamespace,
		StartingCSV:            fmt.Sprintf("%s.%s", pipelineOperatorName, pipelineOperatorVersion),
	}

	log.Info("creating new subscription")
	return r.Client.Create(ctx, sub)
}

// newSubscription will create and return a new Subscription instance using the given Name/Namespace.
func newSubscription(key types.NamespacedName) *operatorsv1a1.Subscription {
	return &operatorsv1a1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
}
