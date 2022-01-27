package controllers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/errors"
	tekton "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const (
	operatorPipelinesRepo         = "https://github.com/redhat-openshift-ecosystem/operator-pipelines.git"
	pipelineDependenciesAvailable = "PipelineDependenciesAvailable"
	operatorCIPipelineYml         = "operator-ci-pipeline.yml"
	operatorHostedPipelineYml     = "operator-hosted-pipeline.yml"
	operatorReleasePipelineYml    = "operator-release-pipeline.yml"
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

	gitMount, ok := os.LookupEnv("GIT_REPO_PATH")
	if !ok {
		log.Error(errors.ErrGitRepoPathNotSpecified, "could not find envvar GIT_REPO_PATH")
		return errors.ErrGitRepoPathNotSpecified
	}
	gitPath := filepath.Join(gitMount, "operator-pipeline")
	err := cloneOrPullRepo(gitPath)

	if err != nil {
		log.Error(err, "Couldn't clone the repository for operator-pipelines")
		if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
			return err
		}
		return err
	}

	pipelineManifestsPath := filepath.Join(gitPath, pipelineManifestsPath)

	if pipeline.Spec.ApplyCIPipeline {
		if err := r.applyManifests(ctx, filepath.Join(pipelineManifestsPath, operatorCIPipelineYml), pipeline, new(tekton.Pipeline)); err != nil {
			if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
				return err
			}
			return err
		}
	} else {
		if err := r.deleteManifests(ctx, filepath.Join(pipelineManifestsPath, operatorCIPipelineYml), pipeline, new(tekton.Pipeline)); err != nil {
			return err
		}
	}

	if pipeline.Spec.ApplyHostedPipeline {
		if err := r.applyManifests(ctx, filepath.Join(pipelineManifestsPath, operatorHostedPipelineYml), pipeline, new(tekton.Pipeline)); err != nil {
			if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
				return err
			}
			return err
		}
	} else {
		if err := r.deleteManifests(ctx, filepath.Join(pipelineManifestsPath, operatorHostedPipelineYml), pipeline, new(tekton.Pipeline)); err != nil {
			return err
		}
	}

	if pipeline.Spec.ApplyReleasePipeline {
		if err := r.applyManifests(ctx, filepath.Join(pipelineManifestsPath, operatorReleasePipelineYml), pipeline, new(tekton.Pipeline)); err != nil {
			if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
				return err
			}
			return err
		}
	} else {
		if err := r.deleteManifests(ctx, filepath.Join(pipelineManifestsPath, operatorReleasePipelineYml), pipeline, new(tekton.Pipeline)); err != nil {
			return err
		}
	}

	taskManifestsPath := filepath.Join(gitPath, taskManifestsPath)

	tasks, err := os.ReadDir(taskManifestsPath)
	if err != nil {
		log.Error(err, "could not read tasks directory")
		if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
			return err
		}
		return err
	}

	for _, task := range tasks {
		if !task.IsDir() {
			if err := r.applyManifests(ctx, filepath.Join(taskManifestsPath, task.Name()), pipeline, new(tekton.Task)); err != nil {
				if err := r.updateStatusCondition(ctx, pipeline, pipelineDependenciesAvailable, metav1.ConditionFalse, reconcileFailed, err.Error()); err != nil {
					return err
				}
				return err
			}
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
		if !apierrors.IsNotFound(err) {
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

func (r *OperatorPipelineReconciler) deleteManifests(ctx context.Context, fileName string, owner, obj client.Object) error {

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
	if err := r.Client.Get(ctx, types.NamespacedName{Name: obj.GetName(), Namespace: obj.GetNamespace()}, obj); err != nil && !apierrors.IsNotFound(err) {
		return err
	} else if apierrors.IsNotFound(err) {
		return nil
	}

	if err := r.Client.Delete(ctx, obj); err != nil {
		log.Error(err, fmt.Sprintf("failed to delete pipeline resource for file: %s", fileName))
		return err
	}

	return nil
}

func cloneOrPullRepo(targetPath string) error {
	// Directory does not exist, so let's clone
	_, err := git.PlainClone(targetPath, false, &git.CloneOptions{
		URL: operatorPipelinesRepo,
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		log.Error(err, "could not clone repo")
		return err
	}
	if err == git.ErrRepositoryAlreadyExists {
		// The directory is already there, so let's just update to latest
		r, err := git.PlainOpen(targetPath)
		if err != nil {
			log.Error(err, "could not open existing git dir")
			return err
		}
		w, err := r.Worktree()
		if err != nil {
			log.Error(err, "could not open git worktree")
			return err
		}
		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil && err != git.NoErrAlreadyUpToDate {
			log.Error(err, "could not pull remote git repo")
			return err
		}
		ref, err := r.Head()
		if err != nil {
			log.Error(err, "could not retrieve current head of")
			return err
		}
		log.Info(fmt.Sprintf("Hash of operator-pipelines HEAD: %s", ref.Hash()))
	}
	return nil
}
