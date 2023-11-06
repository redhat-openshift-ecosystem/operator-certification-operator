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
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/storer"
	"github.com/go-logr/logr"
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

func (r *PipelineGitRepoReconciler) Reconcile(_ context.Context, pipeline *v1alpha1.OperatorPipeline) (bool, error) {
	log := r.Log.WithName("gitrepo")
	gitMount, ok := os.LookupEnv("GIT_REPO_PATH")
	if !ok {
		log.Error(errors.ErrGitRepoPathNotSpecified, "could not find envvar GIT_REPO_PATH")
		return true, errors.ErrGitRepoPathNotSpecified
	}
	gitPath := filepath.Join(gitMount, "operator-pipeline")
	hash, err := cloneOrPullRepo(gitPath, pipeline.Spec.OperatorPipelinesRelease)
	if err != nil {
		log.Error(err, "Couldn't clone the repository for operator-pipelines")
		return true, err
	}
	log.Info(fmt.Sprintf("Hash of operator-pipelines HEAD: %s", hash))

	return false, nil
}

func cloneOrPullRepo(targetPath string, pipelineRelease string) (string, error) {
	// Try to clone first
	r, err := git.PlainClone(targetPath, false, &git.CloneOptions{
		URL: operatorPipelinesRepo,
	})
	if err != nil && err != git.ErrRepositoryAlreadyExists {
		return "", err
	}
	// The directory is already there, so let's just update to latest
	if r == nil && err == git.ErrRepositoryAlreadyExists {
		var err error
		r, err = git.PlainOpen(targetPath)
		if err != nil {
			return "", err
		}
	}

	// If r is nil both clone and open were unsuccessful, returning err to avoid a later panic
	if r == nil {
		return "", fmt.Errorf("could not clone or open repo")
	}

	// Fetching to ensure repo on disk is up to date before we checkout
	if err := r.Fetch(&git.FetchOptions{Tags: git.AllTags}); err != nil && err != git.NoErrAlreadyUpToDate {
		return "", err
	}

	var ref *plumbing.Reference
	iter, err := r.References()
	if err != nil {
		return "", err
	}

	// Iterating over all the references to ensure the one requested is in the repository
	// and to use the reference's value later properly checkout the requested branch/tag
	if err := iter.ForEach(func(reference *plumbing.Reference) error {
		if !strings.HasSuffix(reference.Name().Short(), pipelineRelease) {
			return nil
		}
		ref = reference
		return storer.ErrStop
	}); err != nil {
		return "", err
	}

	if ref == nil {
		return "", fmt.Errorf("requested release is not in repository")
	}

	// Get the worktree
	w, err := r.Worktree()
	if err != nil {
		return "", err
	}

	// Checking out the hash value of the branch/tag that was requested
	if err := w.Checkout(&git.CheckoutOptions{Hash: ref.Hash()}); err != nil {
		return "", err
	}

	return ref.Hash().String(), nil
}
