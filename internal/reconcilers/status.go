package reconcilers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/errors"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultKubeconfigSecretName        = "kubeconfig"
	defaultKubeconfigSecretKeyName     = "kubeconfig"
	defaultGithubAPISecretName         = "github-api-token"
	defaultGithubAPISecretKeyName      = "GITHUB_TOKEN"
	defaultPyxisAPISecretName          = "pyxis-api-secret"
	defaultPyxisAPISecretKeyName       = "pyxis_api_key"
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

func (r *StatusReconciler) Reconcile(ctx context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	origPipeline := pipeline.DeepCopy()
	pipeline.Status.ObservedGeneration = pipeline.Generation
	log := r.Log.WithValues("status.observedGeneration", pipeline.Generation)

	// This is here so that we don't have to worry about which one of these has the :=
	var requeue bool
	var err error

	// No matter what, try to commit the current status.
	// Even though defer evaluates the args here, this works since pipeline is a pointer
	defer r.commitStatus(ctx, pipeline, log)

	requeue, err = r.reconcilePipelineGitRepoStatus(ctx, pipeline)
	if requeue || err != nil {
		log.Error(err, "pipelineGitRepoStatus")
		return requeue, err
	}

	kubeconfigSecret := overrideSecretFromSpec(defaultKubeconfigSecretName, pipeline.Spec.KubeconfigSecretName)
	requeue, err = r.reconcileSecretStatus(ctx, pipeline, "KubeconfigSecret", kubeconfigSecret, defaultKubeconfigSecretKeyName)
	if requeue || err != nil {
		log.Error(err, "kubeconfigSecretStatus")
		return requeue, err
	}

	githubAPISecret := overrideSecretFromSpec(defaultGithubAPISecretName, pipeline.Spec.GitHubSecretName)
	requeue, err = r.reconcileSecretStatus(ctx, pipeline, "GithubApiSecret", githubAPISecret, defaultGithubAPISecretKeyName)
	if requeue || err != nil {
		log.Error(err, "githubApiSecretStatus")
		return requeue, err
	}

	if len(pipeline.Spec.GithubSSHSecretName) > 0 {
		requeue, err = r.reconcileSecretStatus(ctx, pipeline, "GithubSSHSecret", pipeline.Spec.GithubSSHSecretName, defaultGithubSSHSecretKeyName)
		if requeue || err != nil {
			log.Error(err, "githubSSHSecretStatus")
			return requeue, err
		}
	}

	pyxisAPISecret := overrideSecretFromSpec(defaultPyxisAPISecretName, pipeline.Spec.PyxisSecretName)
	requeue, err = r.reconcileSecretStatus(ctx, pipeline, "PyxisApiSecret", pyxisAPISecret, defaultPyxisAPISecretKeyName)
	if requeue || err != nil {
		log.Error(err, "pyxisApiSecretStatus")
		return requeue, err
	}

	if len(pipeline.Spec.DockerRegistrySecretName) > 0 {
		requeue, err = r.reconcileSecretStatus(ctx, pipeline, "DockerRegistrySecret", pipeline.Spec.DockerRegistrySecretName, defaultDockerRegistrySecretKeyName)
		if requeue || err != nil {
			log.Error(err, "dockerRegistrySecretStatus")
			return requeue, err
		}
	}

	requeue, err = r.reconcilePipelineStatus(ctx, pipeline, "CIPipeline", operatorCIPipelineYml, pipeline.Spec.ApplyCIPipeline)
	if requeue || err != nil {
		log.Error(err, "ciPipelineStatus")
		return requeue, err
	}

	requeue, err = r.reconcilePipelineStatus(ctx, pipeline, "HostedPipeline", operatorHostedPipelineYml, pipeline.Spec.ApplyHostedPipeline)
	if requeue || err != nil {
		log.Error(err, "hostedPipelineStatus")
		return requeue, err
	}

	requeue, err = r.reconcilePipelineStatus(ctx, pipeline, "ReleasePipeline", operatorReleasePipelineYml, pipeline.Spec.ApplyReleasePipeline)
	if requeue || err != nil {
		log.Error(err, "releasePipelineStatus")
		return requeue, err
	}

	// TODO(bpc): Task status
	requeue, err = r.reconcileTasksStatus(ctx, pipeline)
	if requeue || err != nil {
		log.Error(err, "tasksStatus")
		return requeue, err
	}

	requeue, err = r.reconcileImageStreamStatus(ctx, pipeline, "CertifiedIndex", certifiedIndex)
	if requeue || err != nil {
		log.Error(err, "certifiedIndexStatus")
		return requeue, err
	}

	requeue, err = r.reconcileImageStreamStatus(ctx, pipeline, "MarketplaceIndex", marketplaceIndex)
	if requeue || err != nil {
		log.Error(err, "marketplaceIndexStatus")
		return requeue, err
	}

	if equality.Semantic.DeepEqual(pipeline.Status, origPipeline.Status) {
		return false, nil
	}

	return false, nil
}

func (r *StatusReconciler) commitStatus(ctx context.Context, pipeline *v1alpha1.OperatorPipeline, log logr.Logger) {
	err := r.Client.Status().Update(ctx, pipeline)
	if err != nil && apierrors.IsConflict(err) {
		log.Info("conflict updating status")
		return
	}
	if err != nil {
		log.Error(err, "error updating status")
		return
	}

	log.Info("updated status")
}

func (r *StatusReconciler) setStatusInfo(status metav1.ConditionStatus, reason string, message string, condition metav1.Condition) metav1.Condition {
	condition.Status = status
	condition.Reason = reason
	condition.Message = message
	return condition
}

func (r *StatusReconciler) conditionStatus(b bool) metav1.ConditionStatus {
	if b {
		return metav1.ConditionTrue
	}
	return metav1.ConditionFalse
}

func (r *StatusReconciler) reconcileImageStreamStatus(ctx context.Context, pipeline *v1alpha1.OperatorPipeline, indexType, indexName string) (bool, error) {
	readyCondition := metav1.Condition{
		Type:               fmt.Sprintf("%sReady", indexType),
		ObservedGeneration: pipeline.Generation,
		Status:             metav1.ConditionUnknown,
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
		return true, err
	}

	meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
		r.conditionStatus(true),
		"AsExpected",
		fmt.Sprintf("%s with name %s found", indexType, indexName),
		readyCondition))

	return false, nil
}

func (r *StatusReconciler) reconcileSecretStatus(ctx context.Context, pipeline *v1alpha1.OperatorPipeline, secretType, secretName, secretKey string) (bool, error) {
	readyCondition := metav1.Condition{
		Type:               fmt.Sprintf("%sReady", secretType),
		ObservedGeneration: pipeline.Generation,
		Status:             metav1.ConditionUnknown,
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
		err = r.Client.Status().Update(ctx, pipeline)
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

func (r *StatusReconciler) reconcilePipelineGitRepoStatus(_ context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	readyCondition := metav1.Condition{
		Type:               "GitRepoReady",
		ObservedGeneration: pipeline.Generation,
		Status:             metav1.ConditionUnknown,
	}

	repo, err := git.PlainOpen(filepath.Join(os.Getenv("GIT_REPO_PATH"), "operator-pipeline"))
	if err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			"Local repo unavailable",
			readyCondition))
		return true, err
	}
	ref, err := repo.Head()
	if err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"Invalid",
			"Local repo invalid",
			readyCondition))
		return true, err
	}

	pipeline.Status.PipelinesRepoHash = ref.Hash().String()

	meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
		r.conditionStatus(true),
		"AsExpected",
		"Git repo is ready",
		readyCondition))

	return false, nil
}

func (r *StatusReconciler) reconcilePipelineStatus(ctx context.Context, pipeline *v1alpha1.OperatorPipeline, pipelineType, pipelineYaml string, pipelinePresent bool) (bool, error) {
	readyCondition := metav1.Condition{
		Type:               fmt.Sprintf("%sReady", pipelineType),
		ObservedGeneration: pipeline.Generation,
		Status:             metav1.ConditionUnknown,
	}

	if !pipelinePresent {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(true),
			"AsExpected",
			"Pipeline not requested",
			readyCondition))
		return false, nil
	}

	gitPath := filepath.Join(os.Getenv("GIT_REPO_PATH"), "operator-pipeline")
	// This will check that the repo has been cloned and is valid
	_, err := git.PlainOpen(gitPath)
	if err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			"Local repo unavailable",
			readyCondition))
		return true, err
	}

	fileName := filepath.Join(gitPath, pipelineManifestsPath, pipelineYaml)
	b, err := os.ReadFile(fileName)
	if err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"Invalid",
			"Pipeline YAML could not be read",
			readyCondition))
		return true, err
	}

	obj := new(tekton.Pipeline)
	if err = yamlutil.Unmarshal(b, &obj); err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"Invalid",
			"Pipeline YAML not valid",
			readyCondition))
		return true, err
	}

	obj.SetNamespace(pipeline.ObjectMeta.Namespace)
	err = r.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
	if err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			"Pipeline not found",
			readyCondition))
		return true, err
	}

	meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
		r.conditionStatus(true),
		"AsExpected",
		fmt.Sprintf("%s pipeline is ready", pipelineType),
		readyCondition))

	return false, nil
}

func (r *StatusReconciler) reconcileTasksStatus(ctx context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	readyCondition := metav1.Condition{
		Type:               "TasksReady",
		ObservedGeneration: pipeline.Generation,
		Status:             metav1.ConditionUnknown,
	}

	gitPath := filepath.Join(os.Getenv("GIT_REPO_PATH"), "operator-pipeline")
	// This will check that the repo has been cloned and is valid
	_, err := git.PlainOpen(gitPath)
	if err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			"Local repo unavailable",
			readyCondition))
		return true, err
	}

	fileName := filepath.Join(gitPath, taskManifestsPath)
	directory, err := os.ReadDir(fileName)
	if err != nil {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"Invalid",
			"Tasks YAML directory could not be read",
			readyCondition))
		return true, err
	}

	obj := new(tekton.Task)
	fileErrors := make([]string, 0, 10)
	unmarshalErrors := make([]string, 0, 10)
	getErrors := make([]string, 0, 10)
	for _, entry := range directory {
		if entry.IsDir() {
			continue
		}
		b, err := os.ReadFile(filepath.Join(fileName, entry.Name()))
		if err != nil {
			fileErrors = append(fileErrors, entry.Name())
			continue
		}
		if err = yamlutil.Unmarshal(b, &obj); err != nil {
			unmarshalErrors = append(unmarshalErrors, entry.Name())
			continue
		}
		obj.SetNamespace(pipeline.ObjectMeta.Namespace)
		err = r.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)
		if err != nil && apierrors.IsNotFound(err) {
			getErrors = append(getErrors, obj.GetName())
			continue
		}
	}

	if len(fileErrors) > 0 {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			fmt.Sprintf("Some tasks YAML files could not be read: %x", fileErrors),
			readyCondition))
		return true, nil
	}

	if len(unmarshalErrors) > 0 {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"Invalid",
			fmt.Sprintf("Some tasks YAML files are not valid: %x", unmarshalErrors),
			readyCondition))
		return true, nil
	}

	if len(unmarshalErrors) > 0 {
		meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
			r.conditionStatus(false),
			"NotFound",
			fmt.Sprintf("Some tasks are not present: %x", getErrors),
			readyCondition))
		return true, nil
	}

	meta.SetStatusCondition(&pipeline.Status.Conditions, r.setStatusInfo(
		r.conditionStatus(true),
		"AsExpected",
		"Tasks are ready",
		readyCondition))

	return false, nil
}
