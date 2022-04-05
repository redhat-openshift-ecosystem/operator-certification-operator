package reconcilers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	operatorCIPipelineYml      = "operator-ci-pipeline.yml"
	operatorHostedPipelineYml  = "operator-hosted-pipeline.yml"
	operatorReleasePipelineYml = "operator-release-pipeline.yml"
)

var (
	baseManifestsPath     = filepath.Join("ansible", "roles", "operator-pipeline", "templates", "openshift")
	pipelineManifestsPath = filepath.Join(baseManifestsPath, "pipelines")
	taskManifestsPath     = filepath.Join(baseManifestsPath, "tasks")
)

type PipelineDependenciesReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewPipeDependenciesReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *PipelineDependenciesReconciler {
	return &PipelineDependenciesReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *PipelineDependenciesReconciler) Reconcile(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) (bool, error) {
	// Cloning operator-pipelines project to retrieve pipelines and tasks
	// yaml manifests that need to be applied beforehand
	// ref: https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#step-6---install-the-certification-pipeline-and-dependencies-into-the-cluster
	log := r.Log.WithName("pipelinedependencies")

	gitPath := filepath.Join(os.Getenv("GIT_REPO_PATH"), "operator-pipeline")
	// This will check that the repo has been cloned and is valid
	_, err := git.PlainOpen(gitPath)
	if err != nil {
		log.Error(err, "Pipelines repo is not valid")
		return true, err
	}

	pipelineManifestsPath := filepath.Join(gitPath, pipelineManifestsPath)

	if err := r.applyOrDeletePipeline(ctx, pipeline, pipeline.Spec.ApplyCIPipeline, filepath.Join(pipelineManifestsPath, operatorCIPipelineYml)); err != nil {
		return true, err
	}

	if err := r.applyOrDeletePipeline(ctx, pipeline, pipeline.Spec.ApplyHostedPipeline, filepath.Join(pipelineManifestsPath, operatorHostedPipelineYml)); err != nil {
		return true, err
	}

	if err := r.applyOrDeletePipeline(ctx, pipeline, pipeline.Spec.ApplyReleasePipeline, filepath.Join(pipelineManifestsPath, operatorReleasePipelineYml)); err != nil {
		return true, err
	}

	taskManifestsPath := filepath.Join(gitPath, taskManifestsPath)

	tasks, err := os.ReadDir(taskManifestsPath)
	if err != nil {
		log.Error(err, "could not read tasks directory")
		return true, err
	}

	for _, task := range tasks {
		if !task.IsDir() {
			if err := r.applyManifests(ctx, filepath.Join(taskManifestsPath, task.Name()), pipeline, new(tekton.Task)); err != nil {
				return true, err
			}
		}
	}

	return false, nil
}

func (r *PipelineDependenciesReconciler) applyOrDeletePipeline(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline, applyManifest bool, yamlPath string) error {
	if applyManifest {
		return r.applyManifests(ctx, yamlPath, pipeline, new(tekton.Pipeline))
	}
	return r.deleteManifests(ctx, filepath.Join(pipelineManifestsPath, operatorCIPipelineYml), pipeline, new(tekton.Pipeline))
}

func (r *PipelineDependenciesReconciler) applyManifests(ctx context.Context, fileName string, owner, obj client.Object) error {
	log := r.Log.WithName("applyManifests")

	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't read manifest file for: %s", fileName))
		return err
	}

	if err = yamlutil.Unmarshal(b, &obj); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't unmarshall yaml file for: %s", fileName))
		return err
	}

	obj.SetNamespace(owner.GetNamespace())
	err = r.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj)

	if len(obj.GetUID()) > 0 {
		if err := r.Client.Update(ctx, obj); err != nil {
			log.Error(err, fmt.Sprintf("failed to update pipeline resource for file: %s", fileName))
			return err
		}
	}

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		controllerutil.SetControllerReference(owner, obj, r.Scheme)
		if err := r.Client.Create(ctx, obj); err != nil {
			log.Error(err, fmt.Sprintf("failed to create pipeline resource for file: %s", fileName))
			return err
		}
	}

	return nil
}

func (r *PipelineDependenciesReconciler) deleteManifests(ctx context.Context, fileName string, owner, obj client.Object) error {
	log := r.Log.WithName("deleteManifests")
	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't read manifest file for: %s", fileName))
		return err
	}

	if err = yamlutil.Unmarshal(b, &obj); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't unmarshall yaml file for: %s", fileName))
		return err
	}

	obj.SetNamespace(owner.GetNamespace())
	if err := r.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj); err != nil && !errors.IsNotFound(err) {
		return err
	} else if errors.IsNotFound(err) {
		return nil
	}

	if err := r.Client.Delete(ctx, obj); err != nil {
		log.Error(err, fmt.Sprintf("failed to delete pipeline resource for file: %s", fileName))
		return err
	}

	return nil
}
