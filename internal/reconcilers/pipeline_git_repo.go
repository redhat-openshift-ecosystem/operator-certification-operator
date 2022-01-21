package reconcilers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-git/go-git/v5"
	"github.com/go-logr/logr"
	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	"github.com/redhat-openshift-ecosystem/operator-certification-operator/internal/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	operatorPipelinesRepo = "https://github.com/redhat-openshift-ecosystem/operator-pipelines.git"
)

type PipelineGitRepoReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func NewPipelineGitRepoReconciler(client client.Client, log logr.Logger, scheme *runtime.Scheme) *PipelineGitRepoReconciler {
	return &PipelineGitRepoReconciler{
		Client: client,
		Log:    log,
		Scheme: scheme,
	}
}

func (r *PipelineGitRepoReconciler) Reconcile(ctx context.Context, pipeline *certv1alpha1.OperatorPipeline) (bool, error) {
	log := r.Log.WithName("gitrepo")
	gitMount, ok := os.LookupEnv("GIT_REPO_PATH")
	if !ok {
		log.Error(errors.ErrGitRepoPathNotSpecified, "could not find envvar GIT_REPO_PATH")
		return true, errors.ErrGitRepoPathNotSpecified
	}
	gitPath := filepath.Join(gitMount, "operator-pipeline")
	hash, err := cloneOrPullRepo(gitPath)
	if err != nil {
		log.Error(err, "Couldn't clone the repository for operator-pipelines")
		return true, err
	}
	log.Info(fmt.Sprintf("Hash of operator-pipelines HEAD: %s", hash))

	return false, nil
}

func cloneOrPullRepo(targetPath string) (string, error) {
	// Try to clone first
	_, err := git.PlainClone(targetPath, false, &git.CloneOptions{
		URL: operatorPipelinesRepo,
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return "", err
	}
	// The directory is already there, so let's just update to latest
	r, err := git.PlainOpen(targetPath)
	if err != nil {
		return "", err
	}
	w, err := r.Worktree()
	if err != nil {
		return "", err
	}
	err = w.Pull(&git.PullOptions{RemoteName: "origin"})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err
	}
	ref, err := r.Head()
	if err != nil {
		return "", err
	}
	return ref.Hash().String(), nil
}
