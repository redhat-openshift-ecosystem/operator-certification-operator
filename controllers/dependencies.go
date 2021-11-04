package controllers

import (
	"context"

	"os"

	"path/filepath"

	git "github.com/go-git/go-git/v5"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	OPERATOR_PIPELINES_REPO = "https://github.com/redhat-openshift-ecosystem/operator-pipelines.git"
	REPO_CLONE_PATH         = "/tmp/operator-pipelines/"
	PIPELINE_MANIFESTS_PATH = "ansible/roles/operator-pipeline/templates/openshift/pipelines"
	TASK_MANIFESTS_PATH     = "ansible/roles/operator-pipeline/templates/openshift/tasks"
)

func (r *OperatorPipelineReconciler) reconcilePipelineDependencies(meta metav1.ObjectMeta) error {

	// Cloning operator-pipelines project to retrieve pipelines and tasks
	// yaml manifests that need to be applied beforehand
	// ref: https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#step-6---install-the-certification-pipeline-and-dependencies-into-the-cluster

	_, err := git.PlainClone(REPO_CLONE_PATH, false, &git.CloneOptions{
		URL: OPERATOR_PIPELINES_REPO,
	})
	defer r.removePipelineDependencyFiles(REPO_CLONE_PATH)
	if err != nil {
		log.Error(err, "Couldn't clone the repository for operator-pipelines")
		return err
	}

	paths := []string{PIPELINE_MANIFESTS_PATH, TASK_MANIFESTS_PATH}
	for _, path := range paths {

		// base repository root + specific yaml manifest directory (pipelines or tasks)
		root := REPO_CLONE_PATH + path

		// walking through the each directory (pipelines or tasks)
		err = filepath.Walk(root, func(filePath string, info os.FileInfo, err error) error {

			// For each file NOT directories
			if !info.IsDir() {
				var pipeline tekton.Pipeline
				var task tekton.Task

				// apply pipeline yaml manifests
				if path == PIPELINE_MANIFESTS_PATH {
					if errors := r.applyManifests(filePath, meta.Namespace, &pipeline); errors != nil {
						log.Error(errors, "Couldn't apply pipeline manifest")
						return errors
					}

					// or apply tasks manifests
				} else {
					if errors := r.applyManifests(filePath, meta.Namespace, &task); errors != nil {
						log.Error(errors, "Couldn't apply task manifest")
						return errors
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

func (r *OperatorPipelineReconciler) removePipelineDependencyFiles(filePath string) error {
	if err := os.RemoveAll(filePath); err != nil {
		log.Error(err, "Couldn't remove operator-pipelines directory")
		return err
	}
	return nil
}

func (r *OperatorPipelineReconciler) applyManifests(fileName string, Namespace string, obj client.Object) error {

	b, err := os.ReadFile(fileName)
	if err != nil {
		log.Error(err, "Couldn't read manifest file")
		return err
	}

	if err = yamlutil.Unmarshal(b, &obj); err != nil {
		log.Error(err, "Couldn't unmarshall yaml file")
		return err
	}

	obj.SetNamespace(Namespace)
	if err = r.Client.Create(context.Background(), obj); err != nil {
		log.Error(err, "Couldn't create resource")
		return err
	}

	return nil
}
