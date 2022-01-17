package reconcilers

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewStatusReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *StatusReconciler {
	return &StatusReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *StatusReconciler) Reconcile(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) (bool, error) {
	origPipeline := pipeline.DeepCopy()
	pipeline.Status.ObservedGeneration = pipeline.Generation
	log := r.Log.WithValues("status.observedGeneration", pipeline.Generation)

	requeue, err := r.reconcileImageStreamStatus(ctx, pipeline, "CertifiedIndex", certifiedIndex)
	if requeue || err != nil {
		return requeue, err
	}

	requeue, err = r.reconcileImageStreamStatus(ctx, pipeline, "MarketplaceIndex", marketplaceIndex)
	if requeue || err != nil {
		return requeue, err
	}

	if equality.Semantic.DeepEqual(pipeline.Status, origPipeline.Status) {
		return false, nil
	}

	return r.commitStatus(ctx, pipeline, log)
}

func (r *StatusReconciler) commitStatus(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline, log logr.Logger) (bool, error) {
	err := r.Client.Status().Update(ctx, pipeline, &client.UpdateOptions{})
	if err != nil && apierrors.IsConflict(err) {
		log.Info("conflict updating status, requeuing")
		return true, nil
	}
	if err != nil {
		log.Error(err, "error updating status")
		return true, err
	}

	log.Info("updated status")
	return false, nil
}

func (r *StatusReconciler) setStatusInfo(status v1.ConditionStatus, reason string, message string, condition v1.Condition) v1.Condition {
	condition.Status = status
	condition.Reason = reason
	condition.Message = message
	return condition
}

func (r *StatusReconciler) conditionStatus(b bool) v1.ConditionStatus {
	if b {
		return v1.ConditionTrue
	}
	return v1.ConditionFalse
}

func (r *StatusReconciler) reconcileImageStreamStatus(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline, indexType, indexName string) (bool, error) {
	readyCondition := v1.Condition{
		Type:               fmt.Sprintf("%sReady", indexType),
		ObservedGeneration: pipeline.Generation,
		Status:             v1.ConditionUnknown,
	}
	log := r.Log.WithValues("status.observedGeneration", pipeline.Generation)

	imageStream := &imagev1.ImageStream{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: pipeline.Namespace, Name: indexName}, imageStream)
	if err != nil && !apierrors.IsNotFound(err) {
		log.WithValues("imagestream", types.NamespacedName{Namespace: pipeline.Namespace, Name: indexName}).
			Error(err, "failed to get object")
		return true, err
	}

	if err != nil && apierrors.IsNotFound(err) {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			fmt.Sprintf("%s with name %s not found", indexType, indexName),
			readyCondition))
		requeue, err := r.commitStatus(ctx, pipeline, log)
		if requeue || err != nil {
			return requeue, err
		}
	}

	meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
		r.conditionStatus(true),
		"AsExpected",
		fmt.Sprintf("%s with name %s found", indexType, indexName),
		readyCondition))

	return false, nil
}
