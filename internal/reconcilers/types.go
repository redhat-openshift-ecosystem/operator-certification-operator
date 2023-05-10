package reconcilers

import (
	"context"

	"github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
)

type Reconciler interface {
	Reconcile(ctx context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error)
}
