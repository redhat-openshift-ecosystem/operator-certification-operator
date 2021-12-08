/*
Copyright 2021.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	defaultKubeconfigSecretName        = "kubeconfig"
	defaultKubeconfigSecretKeyName     = "kubeconfig"
	defaultDockerRegistrySecretKeyName = ".dockerconfigjson"
	kubeConfigSecretAvailable          = "KubeConfigSecretAvailable"
	defaultGithubApiSecretName         = "github-api-token"
	defaultGithubApiSecretKeyName      = "GITHUB_TOKEN"
	defaultGithubSSHSecretKeyName      = "id_rsa"
	gitHubSecretAvailable              = "GithubSecretAvailable"
	defaultPyxisApiSecretName          = "pyxis-api-secret"
	defaultPyxisApiSecretKeyName       = "pyxis_api_key"
	pyxisApiSecretAvailable            = "PyxisApiSecretAvailable"
)

// ensureKubeConfigSecret will ensure that the kubeconfig Secret is present and up to date.
func (r *OperatorPipelineReconciler) ensureKubeConfigSecret(ctx context.Context, meta metav1.ObjectMeta) error {
	operatorPipeline, err := r.getPipeline(ctx, meta)
	if err != nil {
		log.Error(err, "unable to resolve kubeconfig secret for %s/%s", meta.Namespace, meta.Name)
		if err := r.updateStatusCondition(ctx, operatorPipeline, kubeConfigSecretAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err = r.updateStatusCondition(ctx, operatorPipeline, kubeConfigSecretAvailable, metav1.ConditionUnknown, reconcileUnknown,
		""); err != nil {
		return err
	}

	secretName := defaultKubeconfigSecretName
	if operatorPipeline.Spec.KubeconfigSecretName != "" {
		secretName = operatorPipeline.Spec.KubeconfigSecretName
	}

	if err = r.ensureSecret(ctx, secretName, defaultKubeconfigSecretKeyName, meta); err != nil {
		if err := r.updateStatusCondition(ctx, operatorPipeline, kubeConfigSecretAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err = r.updateStatusCondition(ctx, operatorPipeline, kubeConfigSecretAvailable, metav1.ConditionTrue, reconcileSucceeded,
		""); err != nil {
		return err
	}

	return nil
}

// ensureGitHubAPISecret will ensure that the GitHub API Secret is present and up to date.
func (r *OperatorPipelineReconciler) ensureGitHubAPISecret(ctx context.Context, meta metav1.ObjectMeta) error {
	operatorPipeline, err := r.getPipeline(ctx, meta)
	if err != nil {
		log.Error(err, "unable to resolve github secret for %s/%s", meta.Namespace, meta.Name)
		if err := r.updateStatusCondition(ctx, operatorPipeline, gitHubSecretAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err = r.updateStatusCondition(ctx, operatorPipeline, gitHubSecretAvailable, metav1.ConditionUnknown, reconcileUnknown,
		""); err != nil {
		return err
	}

	secretName := defaultGithubApiSecretName
	if operatorPipeline.Spec.GitHubSecretName != "" {
		secretName = operatorPipeline.Spec.GitHubSecretName
	}

	if err = r.ensureSecret(ctx, secretName, defaultGithubApiSecretKeyName, meta); err != nil {
		if err := r.updateStatusCondition(ctx, operatorPipeline, gitHubSecretAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err = r.updateStatusCondition(ctx, operatorPipeline, gitHubSecretAvailable, metav1.ConditionTrue, reconcileSucceeded,
		""); err != nil {
		return err
	}

	return nil
}

// ensurePyxisAPISecret will ensure that the Pyxis API Secret is present and up to date.
func (r *OperatorPipelineReconciler) ensurePyxisAPISecret(ctx context.Context, meta metav1.ObjectMeta) error {
	operatorPipeline, err := r.getPipeline(ctx, meta)
	if err != nil {
		log.Error(err, "unable to resolve pyxis secret for %s in %s", meta.Name, meta.Namespace)
		if err := r.updateStatusCondition(ctx, operatorPipeline, pyxisApiSecretAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err = r.updateStatusCondition(ctx, operatorPipeline, pyxisApiSecretAvailable, metav1.ConditionUnknown, reconcileUnknown,
		""); err != nil {
		return err
	}

	secretName := defaultPyxisApiSecretName
	if operatorPipeline.Spec.PyxisSecretName != "" {
		secretName = operatorPipeline.Spec.PyxisSecretName
	}

	if err = r.ensureSecret(ctx, secretName, defaultPyxisApiSecretKeyName, meta); err != nil {
		if err := r.updateStatusCondition(ctx, operatorPipeline, pyxisApiSecretAvailable, metav1.ConditionFalse, reconcileFailed,
			err.Error()); err != nil {
			return err
		}
		return err
	}

	if err = r.updateStatusCondition(ctx, operatorPipeline, pyxisApiSecretAvailable, metav1.ConditionTrue, reconcileSucceeded,
		""); err != nil {
		return err
	}

	return nil
}

// ensureDockerRegistrySecret will ensure that the Docker Registry Credentials Secret is present and up to date.
func (r *OperatorPipelineReconciler) ensureDockerRegistrySecret(ctx context.Context, meta metav1.ObjectMeta) error {
	operatorPipeline, err := r.getPipeline(ctx, meta)
	if err != nil {
		log.Error(err, "unable to resolve docker registry credentials secret for %s in %s", meta.Name, meta.Namespace)
		return err
	}

	if operatorPipeline.Spec.DockerRegistrySecretName != "" {
		if err := r.ensureSecret(ctx, operatorPipeline.Spec.DockerRegistrySecretName, defaultDockerRegistrySecretKeyName, meta); err != nil {
			return err
		}
	}

	return nil
}

// ensureGithubSSHSecret will ensure that the Github SSH Secret is present and up to date.
func (r *OperatorPipelineReconciler) ensureGithubSSHSecret(ctx context.Context, meta metav1.ObjectMeta) error {
	operatorPipeline, err := r.getPipeline(ctx, meta)
	if err != nil {
		log.Error(err, "unable to resolve github ssh secret for %s in %s", meta.Name, meta.Namespace)
		return err
	}

	if operatorPipeline.Spec.GithubSSHSecretName != "" {
		if err := r.ensureSecret(ctx, operatorPipeline.Spec.GithubSSHSecretName, defaultGithubSSHSecretKeyName, meta); err != nil {
			return err
		}
	}

	return nil
}

// ensureSecret will ensure that the a secret with the appropriate name and key name are present
func (r *OperatorPipelineReconciler) ensureSecret(ctx context.Context, secretName string, secretKeyName string, meta metav1.ObjectMeta) error {
	namespacedSecretName := newNamespacedName(secretName, meta.Namespace)
	secret := &corev1.Secret{}
	if !isObjectFound(ctx, r.Client, namespacedSecretName, secret) {
		log.Error(ErrSecretNotFound, fmt.Sprintf("could not find existing secret %s/%s", meta.Namespace, secretName))
		return ErrSecretNotFound
	}
	log.Info(fmt.Sprintf("found existing secret %s/%s", meta.Namespace, secretName))
	if value, ok := secret.Data[secretKeyName]; ok {
		if len(value) == 0 {
			log.Error(ErrInvalidSecret, fmt.Sprintf("the %s secret does not contain a valid value at key %s", secretName, secretKeyName))
			return ErrInvalidSecret
		}
		log.Info(fmt.Sprintf("the %s secret contains the key %s", secretName, secretKeyName))
	} else {
		log.Error(ErrInvalidSecret, fmt.Sprintf("the %s secret does not contain the key %s", secretName, secretKeyName))
		return ErrInvalidSecret
	}
	return nil // Existing Secret found, do nothing...
}

// getPipeline retrieves a pipeline cr
func (r *OperatorPipelineReconciler) getPipeline(ctx context.Context, meta metav1.ObjectMeta) (*certv1alpha1.OperatorPipeline, error) {
	pipelineName := newNamespacedName(meta.Name, meta.Namespace)
	pipeline := &certv1alpha1.OperatorPipeline{}
	err := r.Client.Get(ctx, pipelineName, pipeline)
	if err != nil {
		if k8errors.IsNotFound(err) {
			log.Info("pipeline resource not found. Ignoring since cr must be deleted")
			return nil, nil
		}
		log.Error(err, "unable to retrieve the pipeline resource %s/%s", meta.Namespace, meta.Name)
		return nil, err
	}
	return pipeline, nil
}

// newNamespacedName will create and return a new namespaced name instance using the given name and namespace.
func newNamespacedName(name string, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}
