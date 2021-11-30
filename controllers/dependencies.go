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
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	OPERATOR_PIPELINES_REPO = "https://github.com/redhat-openshift-ecosystem/operator-pipelines.git"
	PIPELINE_MANIFESTS_PATH = "ansible/roles/operator-pipeline/templates/openshift/pipelines"
	TASK_MANIFESTS_PATH     = "ansible/roles/operator-pipeline/templates/openshift/tasks"
)

func (r *OperatorPipelineReconciler) reconcilePipelineDependencies(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) error {

	// Cloning operator-pipelines project to retrieve pipelines and tasks
	// yaml manifests that need to be applied beforehand
	// ref: https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#step-6---install-the-certification-pipeline-and-dependencies-into-the-cluster

	// creating a tmp director so that each reconcile gets a new directory in case the defer does not execute properly
	tmpRepoClonePath, _ := os.MkdirTemp("", "operator-pipelines-*")

	_, err := git.PlainClone(tmpRepoClonePath, false, &git.CloneOptions{
		URL: OPERATOR_PIPELINES_REPO,
	})

	defer os.RemoveAll(tmpRepoClonePath)

	if err != nil {
		log.Error(err, "Couldn't clone the repository for operator-pipelines")
		return err
	}

	paths := []string{PIPELINE_MANIFESTS_PATH, TASK_MANIFESTS_PATH}
	for _, path := range paths {

		// base repository root + specific yaml manifest directory (pipelines or tasks)
		root := filepath.Join(tmpRepoClonePath, path)

		// walking through each directory (pipelines or tasks)
		err = filepath.Walk(root, func(filePath string, info os.FileInfo, err error) error {

			// For each file NOT directories
			if !info.IsDir() {

				// apply pipeline yaml manifests
				if path == PIPELINE_MANIFESTS_PATH {
					if errs := r.applyPipelineManifests(ctx, filePath, pipeline); errs != nil {
						return errs
					}
					// or apply tasks manifests
				} else {
					if errs := r.applyTaskManifests(ctx, filePath, pipeline); errs != nil {
						return errs
					}
				}
			}
			return nil
		})
		if err != nil {
			log.Error(err, "Couldn't iterate over operator-pipelines yaml manifest files")
			return err
		}
	}

	return nil
}

func (r *OperatorPipelineReconciler) applyPipelineManifests(ctx context.Context, fileName string, obj metav1.Object) error {

	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't read manifest file for: %s", fileName))
		return err
	}

	pipeline := new(tekton.Pipeline)

	if err = yamlutil.Unmarshal(b, pipeline); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't unmarshall yaml file for: %s", fileName))
		return err
	}

	pipeline.SetNamespace(obj.GetNamespace())
	err = r.Get(ctx, types.NamespacedName{Name: pipeline.Name, Namespace: pipeline.Namespace}, pipeline)

	if len(pipeline.ObjectMeta.UID) > 0 {
		if err := r.Client.Update(ctx, pipeline); err != nil {
			log.Error(err, fmt.Sprintf("failed to update pipeline resource for file: %s", fileName))
			return err
		}
	}

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		controllerutil.SetControllerReference(obj, pipeline, r.Scheme)
		if err := r.Client.Create(ctx, pipeline); err != nil {
			log.Error(err, fmt.Sprintf("failed to create pipeline resource for file: %s", fileName))
			return err
		}
	}

	return nil
}

func (r *OperatorPipelineReconciler) applyTaskManifests(ctx context.Context, fileName string, obj metav1.Object) error {

	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, fmt.Sprintf("Couldn't read manifest file for: %s", fileName))
		return err
	}

	task := new(tekton.Task)

	if err = yamlutil.Unmarshal(b, task); err != nil {
		log.Error(err, fmt.Sprintf("Couldn't unmarshall yaml file for: %s", fileName))
		return err
	}

	task.SetNamespace(obj.GetNamespace())
	err = r.Get(ctx, types.NamespacedName{Name: task.Name, Namespace: task.Namespace}, task)

	if len(task.ObjectMeta.UID) > 0 {
		if err := r.Client.Update(ctx, task); err != nil {
			log.Error(err, fmt.Sprintf("failed to create task resource for file: %s", fileName))
			return err
		}
	}

	if err != nil {
		if !errors.IsNotFound(err) {
			return err
		}
		controllerutil.SetControllerReference(obj, task, r.Scheme)
		if err := r.Client.Create(ctx, task); err != nil {
			log.Error(err, fmt.Sprintf("failed to update task resource for file: %s", fileName))
			return err
		}
	}

	return nil
}
