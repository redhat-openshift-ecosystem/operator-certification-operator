package reconcilers

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultKubeconfigSecretName        = "kubeconfig"
	defaultKubeconfigSecretKeyName     = "kubeconfig"
	defaultGithubApiSecretName         = "github-api-token"
	defaultGithubApiSecretKeyName      = "GITHUB_TOKEN"
	defaultPyxisApiSecretName          = "pyxis-api-secret"
	defaultPyxisApiSecretKeyName       = "pyxis_api_key"
	defaultDockerRegistrySecretKeyName = ".dockerconfigjson"
	defaultGithubSSHSecretKeyName      = "id_rsa"
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

func overrideSecretFromSpec(secretDefault, spec string) string {
	if len(spec) > 0 {
		return spec
	}
	return secretDefault
}

func (r *StatusReconciler) Reconcile(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) (bool, error) {
	origPipeline := pipeline.DeepCopy()
	pipeline.Status.ObservedGeneration = pipeline.Generation
	log := r.Log.WithValues("status.observedGeneration", pipeline.Generation)

	kubeconfigSecret := overrideSecretFromSpec(defaultKubeconfigSecretName, pipeline.Spec.KubeconfigSecretName)
	requeue, err := r.reconcileSecretStatus(ctx, pipeline, "KubeconfigSecret", kubeconfigSecret, defaultKubeconfigSecretKeyName)
	if requeue || err != nil {
		return requeue, err
	}

	githubApiSecret := overrideSecretFromSpec(defaultGithubApiSecretName, pipeline.Spec.GitHubSecretName)
	requeue, err = r.reconcileSecretStatus(ctx, pipeline, "GithubApiSecret", githubApiSecret, defaultGithubApiSecretKeyName)
	if requeue || err != nil {
		return requeue, err
	}

	if len(pipeline.Spec.GithubSSHSecretName) > 0 {
		requeue, err = r.reconcileSecretStatus(ctx, pipeline, "GithubSSHSecret", pipeline.Spec.GithubSSHSecretName, defaultGithubSSHSecretKeyName)
		if requeue || err != nil {
			return requeue, err
		}
	}

	pyxisApiSecret := overrideSecretFromSpec(defaultPyxisApiSecretName, pipeline.Spec.PyxisSecretName)
	requeue, err = r.reconcileSecretStatus(ctx, pipeline, "PyxisApiSecret", pyxisApiSecret, defaultPyxisApiSecretKeyName)
	if requeue || err != nil {
		return requeue, err
	}

	if len(pipeline.Spec.DockerRegistrySecretName) > 0 {
		requeue, err = r.reconcileSecretStatus(ctx, pipeline, "DockerRegistrySecret", pipeline.Spec.DockerRegistrySecretName, defaultDockerRegistrySecretKeyName)
		if requeue || err != nil {
			return requeue, err
		}
	}

	requeue, err = r.reconcileImageStreamStatus(ctx, pipeline, "CertifiedIndex", certifiedIndex)
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

func (r *StatusReconciler) reconcileSecretStatus(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline, secretType, secretName, secretKey string) (bool, error) {
	readyCondition := v1.Condition{
		Type:               fmt.Sprintf("%sReady", secretType),
		ObservedGeneration: pipeline.Generation,
		Status:             v1.ConditionUnknown,
	}
	log := r.Log.WithValues("status.observedGeneration", pipeline.Generation)
	secret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Namespace: pipeline.Namespace, Name: secretName}, secret)
	if err != nil && !apierrors.IsNotFound(err) {
		log.WithValues(strings.ToLower(secretType), types.NamespacedName{Namespace: pipeline.Namespace, Name: secretName}).
			Error(err, "failed to get object")
		return true, err
	}

	value, ok := secret.Data[secretKey]
	if !ok {
		log.Error(errors.ErrInvalidSecret, fmt.Sprintf("the %s secret does not contain the key %s", secretName, secretKey))
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"KeyNotFound",
			fmt.Sprintf("%s key not found in secret %s", secretKey, secretName),
			readyCondition))
		return true, errors.ErrInvalidSecret
	}

	if len(value) == 0 {
		log.Error(errors.ErrInvalidSecret, fmt.Sprintf("the %s secret does not contain a valid value at key %s", secretName, secretKey))
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"KeyDataInvalid",
			fmt.Sprintf("secret data invalid in secret %s", secretName),
			readyCondition))
		return true, errors.ErrInvalidSecret
	}

	if err != nil && apierrors.IsNotFound(err) {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			fmt.Sprintf("%s ecret not found", secretName),
			readyCondition))
		err = r.Client.Status().Update(ctx, pipeline, &client.UpdateOptions{})
		if apierrors.IsConflict(err) {
			log.Info("conflict updating object, requeueing")
			return true, nil
		}
		if err != nil {
			log.Error(err, "failed to update object")
			return true, err
		}
	}

	meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
		r.conditionStatus(true),
		"AsExpected",
		fmt.Sprintf("%s secret found", secretName),
		readyCondition))

	return false, nil
}
