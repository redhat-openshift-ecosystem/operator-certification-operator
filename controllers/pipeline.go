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
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	OS_OPERATORS_NAMESPACE               = "openshift-operators"
	PIPELINES_OPERATOR_CHANNEL           = "stable"
	PIPELINES_OPERATOR_CATALOG_SOURCE    = "redhat-operators"
	PIPELINES_OPERATOR_CATALOG_SOURCE_NS = "openshift-marketplace"
	PIPELINES_OPERATOR_NAME              = "redhat-openshift-pipelines"
	PIPELINES_OPERATOR_VERSION           = "v1.5.2"
	PIPELINES_OPERATOR_SUBSCRIPTION      = "openshift-pipelines-operator-rh"
)

// reconcilePipelineOperator will ensure that the resources are present to provision the OpenShift Pipeline Operator
func (r *OperatorPipelineReconciler) reconcilePipelineOperator(meta metav1.ObjectMeta) error {
	log.Log.Info("reconciling pipeline operator")

	key := types.NamespacedName{
		Namespace: OS_OPERATORS_NAMESPACE,
		Name:      PIPELINES_OPERATOR_SUBSCRIPTION,
	}

	sub := newSubscription(key)
	if IsObjectFound(r.Client, key, sub) {
		log.Log.Info("existing subscription found")
		return nil // Existing Subscription found, do nothing...
	}

	sub.Spec = &operatorsv1a1.SubscriptionSpec{
		Channel:                PIPELINES_OPERATOR_CHANNEL,
		InstallPlanApproval:    operatorsv1a1.ApprovalAutomatic, // Use ApprovalManual instead?
		Package:                PIPELINES_OPERATOR_SUBSCRIPTION,
		CatalogSource:          PIPELINES_OPERATOR_CATALOG_SOURCE,
		CatalogSourceNamespace: PIPELINES_OPERATOR_CATALOG_SOURCE_NS,
		StartingCSV:            fmt.Sprintf("%s.%s", PIPELINES_OPERATOR_NAME, PIPELINES_OPERATOR_VERSION),
	}

	log.Log.Info("creating new subscription")
	return r.Client.Create(context.TODO(), sub)
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
