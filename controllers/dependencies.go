package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	operatorPipelinesRepo         = "https://github.com/redhat-openshift-ecosystem/operator-pipelines.git"
	pipelineDependenciesAvailable = "PipelineDependenciesAvailable"
)

var (
	baseManifestsPath     = filepath.Join("ansible", "roles", "operator-pipeline", "templates", "openshift")
	pipelineManifestsPath = filepath.Join(baseManifestsPath, "pipelines")
	taskManifestsPath     = filepath.Join(baseManifestsPath, "tasks")
)

func (r *OperatorPipelineReconciler) reconcilePipelineDependencies(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) error {
	// Cloning operator-pipelines project to retrieve pipelines and tasks
	// yaml manifests that need to be applied beforehand
	// ref: https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#step-6---install-the-certification-pipeline-and-dependencies-into-the-cluster

	if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionUnknown, reconcileUnknown, ""); err != nil {
		return err
	}

	// creating a tmp director so that each reconcile gets a new directory in case the defer does not execute properly
	tmpRepoClonePath, _ := os.MkdirTemp("", "operator-pipelines-*")

	_, err := git.PlainClone(tmpRepoClonePath, false, &git.CloneOptions{
		URL: operatorPipelinesRepo,
	})

	defer os.RemoveAll(tmpRepoClonePath)

	if err != nil {
		log.Error(err, "Couldn't clone the repository for operator-pipelines")
		if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
			return err
		}
		return err
	}

	paths := []string{pipelineManifestsPath, taskManifestsPath}
	for _, path := range paths {
		// base repository root + specific yaml manifest directory (pipelines or tasks)
		root := filepath.Join(tmpRepoClonePath, path)

		// walking through each directory (pipelines or tasks)
		err = filepath.Walk(root, func(filePath string, info os.FileInfo, err error) error {

			// For each file NOT directories
			if !info.IsDir() {
				switch path {
				case pipelineManifestsPath:
					if err := r.applyManifests(ctx, filePath, pipeline, new(tekton.Pipeline)); err != nil {
						if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
							return err
						}
						return err
					}
				case taskManifestsPath:
					if err := r.applyManifests(ctx, filePath, pipeline, new(tekton.Task)); err != nil {
						if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
							return err
						}
						return err
					}
				default:
					return nil
				}
			}
			return nil
		})
		if err != nil {
			log.Error(err, "Couldn't iterate over operator-pipelines yaml manifest files")
			if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
				return err
			}
			return err
		}
	}

	if err = r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionTrue, reconcileSucceeded, ""); err != nil {
		return err
	}

	return nil
}

func (r *OperatorPipelineReconciler) applyManifests(ctx context.Context, fileName string, owner, obj client.Object) error {
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
