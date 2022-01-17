package reconcilers

import (
	"context"

	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
)

type Reconciler interface {
	Reconcile(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) (bool, error)
}
