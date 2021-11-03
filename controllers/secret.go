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
	"errors"
	"fmt"

	certv1alpha1 "github.com/redhat-openshift-ecosystem/operator-certification-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	KUBECONFIG_SECRET = "kubeconfig"
	GITHUB_API_SECRET = "github-api-token"
	GITHUB_TOKEN      = "GITHUB_TOKEN"
	PYXIS_API_SECRET  = "pyxis-api-secret"
)

// reconcileKubeConfigSecret will ensure that the kubeconfig Secret is present and up to date.
func (r *OperatorPipelineReconciler) reconcileKubeConfigSecret(meta metav1.ObjectMeta) error {
	pipelineName := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      meta.Name,
	}
	pipeline := &certv1alpha1.OperatorPipeline{}
	err := r.Client.Get(context.TODO(), pipelineName, pipeline)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			log.Info("pipeline resource not found. Ignoring since object must be deleted")
			return nil
		}
		log.Error(err, "unable to retrieve pipeline resource in namespace "+meta.Namespace)
		return err
	}
	var secretName = KUBECONFIG_SECRET
	if pipeline.Spec.KubeconfigSecretName != "" {
		secretName = pipeline.Spec.KubeconfigSecretName
	}
	key := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      secretName,
	}
	secret := newSecret(key)
	if !IsObjectFound(r.Client, key, secret) {
		err := errors.New(fmt.Sprintf("An existing secret named %s was not found in namespace %s", secretName, meta.Namespace))
		log.Error(err, fmt.Sprintf("unable to reconcile kubeconfig in namespace %s", meta.Namespace))
		return err
	}
	log.Info(fmt.Sprintf("existing %s secret found in namespace %s", secretName, meta.Namespace))
	kubeConfigSecret := &corev1.Secret{}
	err = r.Client.Get(context.TODO(), key, kubeConfigSecret)
	if err != nil {
		log.Error(err, fmt.Sprintf("unable to get the %s secret", secretName))
		return err
	}
	if kubeConfigSecret.Data[KUBECONFIG_SECRET] == nil {
		err = errors.New(fmt.Sprintf("the kubeconfig key in %s is empty!", secretName))
		log.Error(err, fmt.Sprintf("The %s secret does not contain a kubeconfig", secretName))
		return err
	}
	return nil // Existing Secret found, do nothing...
}

// reconcileGitHubAPISecret will ensure that the GitHub API Secret is present and up to date.
func (r *OperatorPipelineReconciler) reconcileGitHubAPISecret(meta metav1.ObjectMeta) error {

	namespaceName := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      meta.Name,
	}

	operatorPipeline := &certv1alpha1.OperatorPipeline{}
	err := r.Client.Get(context.TODO(), namespaceName, operatorPipeline)
	if err != nil {
		if k8errors.IsNotFound(err) {
			log.Info("pipeline resource not found. Ignoring since object must be deleted")
			return nil
		}

		log.Error(err, fmt.Sprintf("unable to retrieve pipeline resource in namespace: %s ", meta.Namespace))
		return err
	}

	var gitHubSecretName = GITHUB_API_SECRET
	if operatorPipeline.Spec.GitHubSecretName != "" {
		gitHubSecretName = operatorPipeline.Spec.GitHubSecretName
	}

	key := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      gitHubSecretName,
	}

	secret := newSecret(key)
	if !IsObjectFound(r.Client, key, secret) {
		return errors.New(fmt.Sprintf("github api secret not found in namespace: %s", meta.Namespace))
	}

	gitHubSecret := &corev1.Secret{}
	if err = r.Get(context.TODO(), key, gitHubSecret); err != nil {
		log.Error(err, fmt.Sprintf("unable to retrieve github api secret in namespace: %s ", meta.Namespace))
		return err
	}

	// checking to make sure that the key is present in the map before continue to verify the data
	if value, ok := gitHubSecret.Data[GITHUB_TOKEN]; ok {
		if value == nil || string(value) == "" {
			return errors.New(fmt.Sprintf("github api secret found, but has an empty token value for: %s", GITHUB_TOKEN))
		}

		log.Info(fmt.Sprintf("github api secret with proper token name found in namespace: %s", meta.Namespace))
	} else {
		return errors.New(fmt.Sprintf("github api secret found, but has an improper token name, token name should be: %s", GITHUB_TOKEN))
	}

	return nil // Existing Secret found, do nothing...
}

// reconcilePyxisAPISecret will ensure that the Pyxis API Secret is present and up to date.
func (r *OperatorPipelineReconciler) reconcilePyxisAPISecret(meta metav1.ObjectMeta) error {
	key := types.NamespacedName{
		Namespace: meta.Namespace,
		Name:      PYXIS_API_SECRET,
	}

	secret := newSecret(key)
	if IsObjectFound(r.Client, key, secret) {
		log.Info("existing pyxis api secret found")
		return nil // Existing Secret found, do nothing...
	}

	return errors.New("pyxis api secret not found")
}

// newSecret will create and return a new Secret instance using the given Name/Namespace.
func newSecret(key types.NamespacedName) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      key.Name,
			Namespace: key.Namespace,
		},
	}
}
